package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net"
	"os"

	"cloud.google.com/go/storage"
	"github.com/pkg/sftp"
	"github.com/rs/zerolog/log"
	"golang.org/x/crypto/ssh"
	"google.golang.org/api/option"
)

var SFTP []byte = []byte("sftp")

func main() {

	// Initialize SSH server. Get the ssh.Signer by
	// parsing the private key from envvar.

	sshConfig := getSSHConfig()
	privateKeyBytes := []byte(mustGetenv("PRIVATE_KEY"))
	privateKey, err := ssh.ParsePrivateKey(privateKeyBytes)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("invalid private key")
	}
	sshConfig.AddHostKey(privateKey)
	listener, err := net.Listen("tcp", "0.0.0.0:3003")
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("unable to initialize listener")
	}
	log.Info().Msgf("listening on %v", listener.Addr())

	// Initialize Google Cloud Storage bucket. For now, use
	// a service account cred file to access the bucket.

	bucketName := mustGetenv("BUCKET_NAME")
	ctx := context.Background()
	storageCreds := mustGetenv("SERVICE_ACCOUNT_KEY")
	storageClient, err := storage.NewClient(
		ctx, option.WithCredentialsFile(storageCreds),
	)
	if err != nil {
		log.Fatal().
			Err(err).
			Msg("unable to initialize storage client")
	}
	bucket := storageClient.Bucket(bucketName)
	log.Info().Msgf("using bucket with acl (%v)", bucket.ACL())
	handler := &Handler{bucket}

	// Accept *all* incoming connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Err(err).Msg("failed to accept incoming connection")
		}

		sshcon, channels, requests, err := ssh.NewServerConn(conn, sshConfig)
		if err != nil {
			log.Err(err).Msg("handshake failed")
			continue
		}
		log.Info().Str("addr", sshcon.RemoteAddr().String()).Msg("new connection")
		go ssh.DiscardRequests(requests)
		go handleChannels(channels, handler)
	}
}

func handleChannels(chans <-chan ssh.NewChannel, h *Handler) {
	for newChannel := range chans {
		go handleChannel(newChannel, h)
	}
}

func handleChannel(newChannel ssh.NewChannel, h *Handler) {
	// Only handle channel types that are `session` and reject `x11`,
	// `direct-tcpip` and `forwarded-tcpip`.
	if newChannel.ChannelType() != "session" {
		newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
		return
	}

	// Attempt to accept the channel
	channel, requests, err := newChannel.Accept()
	if err != nil {
		log.Err(err).Msg("could not accept channel")
		return
	}

	go func(in <-chan *ssh.Request) {
		for req := range in {
			// Sessions have out-of-band requests such as `shell`, `pty-req` and
			// `env`. Handle only `subsystem` requests.
			if req.Type == "subsystem" &&
				bytes.Compare(req.Payload[4:], SFTP) == 0 {
				req.Reply(true, nil)
				continue
			}
			req.Reply(false, nil)
		}
	}(requests)

	server := sftp.NewRequestServer(channel, sftp.Handlers{h, h, h, h})
	err = server.Serve()
	if err == io.EOF {
		server.Close()
		log.Info().Msg("session ended")
	}

	if err != nil {
		log.Err(err).Msg("")
	}
}

func getSSHConfig() *ssh.ServerConfig {
	passwordBytes := []byte(mustGetenv("PASSWORD"))
	user := mustGetenv("USER")

	return &ssh.ServerConfig{
		NoClientAuth:  false,
		ServerVersion: "SSH-2.0-BSFTP",
		AuthLogCallback: func(conn ssh.ConnMetadata, method string, err error) {
			zlog := log.Info().Str("user", conn.User()).
				Str("addr", conn.RemoteAddr().String()).
				Str("method", method)
			if err != nil {
				zlog.Msg("auth failed")
				return
			}
			zlog.Msg("auth ok")
		},
		PasswordCallback: func(c ssh.ConnMetadata, p []byte) (*ssh.Permissions, error) {
			if c.User() == user && bytes.Compare(p, passwordBytes) == 0 {
				return nil, nil
			}
			return nil, errors.New("invalid credentials")
		},
	}
}

func mustGetenv(e string) string {
	n := os.Getenv(e)
	if n == "" {
		log.Fatal().Msgf("missing env %s", e)
	}
	return n
}
