package main

import (
	"fmt"
	"log"
	"os"
	"time"

	ndb "github.com/felixge/netfix/db"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 2 {
		return fmt.Errorf("usage: ./netfix <dbfile.sqlite3>")
	}
	db, err := ndb.Open(os.Args[1])
	if err != nil {
		return err
	} else if from, to, err := ndb.Migrate(db); err != nil {
		return err
	} else if from != to {
		log.Printf("db: migrated from version %d to %d", from, to)
	}

	start := time.Now()
	outages, err := ndb.Outages(db, ndb.OutageFilter{
		MinLoss:        0.01,
		OutageLoss:     0.01,
		OutageDuration: 15 * time.Minute,
		OutageGap:      5 * time.Minute,
	})
	if err != nil {
		return err
	}

	for _, o := range outages {
		fmt.Printf("%s\n", o)
	}
	fmt.Printf("%d outages (%s)\n", len(outages), outages.Duration())
	fmt.Printf("%s\n", time.Since(start))

	return nil
}
