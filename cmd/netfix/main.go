package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
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
	go func() { errCh <- runPings(c, db) }()

	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			return err
		}
	}
	return nil
}

func EnvConfig() (Config, error) {
	c := Config{
		DB:        os.Getenv("NF_DB"),
		HttpAddr:  os.Getenv("NF_HTTP_ADDR"),
		Target:    os.Getenv("NF_TARGET"),
		IPVersion: os.Getenv("NF_IP_VERSION"),
	}
	if d, err := parseEnvDuration("NF_INTERVAL"); err != nil {
		return c, err
	} else {
		c.Interval = d
	}
	if d, err := parseEnvDuration("NF_TIMEOUT"); err != nil {
		return c, err
	} else {
		c.Timeout = d
	}

	return c, nil
}

func parseEnvDuration(envVar string) (time.Duration, error) {
	val := os.Getenv(envVar)
	d, err := time.ParseDuration(val)
	if err != nil {
		return d, fmt.Errorf("%s: %s", envVar, err)
	}
	return d, err
}

type Config struct {
	DB        string
	HttpAddr  string
	Target    string
	IPVersion string
	Interval  time.Duration
	Timeout   time.Duration
}

func (c Config) String() string {
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		panic(err)
	}
	return string(data)
}
