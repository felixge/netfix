package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"time"

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
	c := EnvConfig()
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

	server := &http.Server{
		Addr:         c.HttpAddr,
		Handler:      OutagesHandler(db),
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	return server.ListenAndServe()
}

func OutagesHandler(db *sql.DB) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		f := ndb.OutageFilter{
			MinLoss:        0.01,
			OutageLoss:     0.01,
			OutageDuration: 2 * time.Minute,
			OutageGap:      5 * time.Minute,
		}
		fmt.Sscan(r.URL.Query().Get("min_loss"), &f.MinLoss)
		fmt.Sscan(r.URL.Query().Get("outage_loss"), &f.OutageLoss)
		if d, err := time.ParseDuration(r.URL.Query().Get("outage_duration")); err == nil {
			f.OutageDuration = d
		}
		if d, err := time.ParseDuration(r.URL.Query().Get("outage_gap")); err == nil {
			f.OutageGap = d
		}

		outages, err := ndb.Outages(db, f)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
			fmt.Fprintf(w, "%s\n", err)
			return
		}

		fmt.Fprintf(w, "%s\n", f)
		fmt.Fprintf(w, "%d outages (%s):\n\n", len(outages), outages.Duration())
		sort.Slice(outages, func(i, j int) bool {
			return outages[i].Start.After(outages[j].Start)
		})
		for _, o := range outages {
			fmt.Fprintf(w, "%s\n", o)
		}
	})
}

func EnvConfig() Config {
	return Config{
		DB:       os.Getenv("NF_DB"),
		HttpAddr: os.Getenv("NF_HTTP_ADDR"),
	}
}

type Config struct {
	DB       string
	HttpAddr string
}

func (c Config) String() string {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(data)
}
