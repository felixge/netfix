package main

import (
	"fmt"
	"log"
	"os"

	ndb "github.com/felixge/netfix/db"
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
	c, err := EnvConfig()
	if err != nil {
		return err
	}
	log.SetOutput(os.Stdout)
	log.Printf("Starting up netfix version=%s config=%s", version, c)
	db, err := ndb.Open(c.DB)
	if err != nil {
		return err
	} else if from, to, err := ndb.Migrate(db); err != nil {
		return err
	} else if from != to {
		log.Printf("db: migrated from version %d to %d", from, to)
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
