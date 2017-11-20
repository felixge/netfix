package main

import (
	"fmt"
	"log"
	"net"
	"os"

	"github.com/coreos/go-systemd/daemon"
	"github.com/felixge/netfix"
)

// version is populated by the Makefile
var version = "?"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	c, err := netfix.EnvConfig()
	if err != nil {
		return err
	}
	log.SetOutput(os.Stdout)
	log.Printf("Starting up netfix version=%s config=%s", version, c)

	log.Printf("Listening on %s", c.HttpAddr)
	ln, err := net.Listen("tcp", c.HttpAddr)
	if err != nil {
		return err
	}
	ok, err := daemon.SdNotify(false, "READY=1")
	log.Printf("Sent systemd readiness notification. sent=%t err=%v", ok, err)

	log.Printf("Open db and running migrations")
	db, err := c.DB.Open()
	if err != nil {
		return err
	}

	errCh := make(chan error)

	go func() { errCh <- serveHttp(c, ln, db) }()
	go func() { errCh <- recordPings(c, db) }()

	log.Printf("Recording pings and serving clients")
	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			return err
		}
	}
	return nil
}
