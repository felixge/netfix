package main

import (
	"fmt"
	"log"
	"os"

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
	db, err := c.DB.Open()
	if err != nil {
		return err
	}

	errCh := make(chan error)

	go func() { errCh <- serveHttp(c, db) }()
	go func() { errCh <- recordPings(c, db) }()

	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			return err
		}
	}
	return nil
}
