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

	outages, err := ndb.Outages(db, 0.01, 5*time.Minute)
	if err != nil {
		return err
	}

	var (
		duration time.Duration
		count    int
	)
	for _, outage := range outages {
		d := outage.Duration()
		if d >= 5*time.Minute && outage.Loss() >= 0.01 {
			fmt.Printf("%s\n", outage)
			count++
			duration += d
		}
	}
	fmt.Printf("%d outages (%s)\n", count, duration)

	return nil
}
