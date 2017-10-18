package main

import (
	"fmt"
	"log"
	"os"

	"github.com/felixge/netfix"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	c := netfix.EnvConfig()
	db, err := c.OpenDB()
	if err != nil {
		return fmt.Errorf("db: open: %s", err)
	} else if from, to, err := netfix.Migrate(db); err != nil {
		return err
	} else if from != to {
		log.Printf("db: migrated from version %d to %d", from, to)
	}

	_ = db
	fmt.Printf("%#v\n", c)
	return nil
}
