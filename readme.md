# bsftp
> SFTP server backed by Google Cloud Storage

### Requirements

- Google Cloud Storage
- Go 1.18.3

### Generate a key

```sh
ssh-keygen -t rsa -b 4096 -f id_rsa
```

### Run

Make sure env vars are set. I recommend using
[direnv](https://github.com/direnv/direnv) for local dev.

Refer to `env.sample` for the required env vars.

```sh
go build . -o bsftp
./bsftp
```

### References

- [Go and SSH](https://github.com/jpillora/go-and-ssh)
- [AWS SDK Go V2](https://github.com/aws/aws-sdk-go-v2)
