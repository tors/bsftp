package main

import (
	"bytes"
	"log"

	"github.com/secsy/goftp"
)

func Example() {
	conn, err := goftp.DialConfig(goftp.Config{
		User:     "user",
		Password: "password",
	}, "127.0.0.1:3003")

	if err != nil {
		log.Fatal(err)
	}

	data := bytes.NewBufferString("hello world")
	err = conn.Store("test-file.txt", data)
	if err != nil {
		log.Fatal(err)
	}

	// if err := conn.Quit(); err != nil {
	// 	log.Fatal(err)
	// }

	// Output: Ok!
}
