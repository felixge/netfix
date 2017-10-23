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
	go func() { errCh <- recordPings(c, db) }()

	for i := 0; i < 2; i++ {
		if err := <-errCh; err != nil {
			return err
		}
	}
	return nil
}

func EnvConfig() (Config, error) {
	c := Config{}
	if err := nonEmptyString("NF_DB", &c.DB); err != nil {
		return c, err
	} else if err := nonEmptyString("NF_HTTP_ADDR", &c.HttpAddr); err != nil {
		return c, err
	} else if err := nonEmptyString("NF_TARGET", &c.Target); err != nil {
		return c, err
	} else if err := nonEmptyString("NF_IP_VERSION", &c.IPVersion); err != nil {
		return c, err
	} else if err := parseEnvDuration("NF_INTERVAL", &c.Interval); err != nil {
		return c, err
	} else if err := parseEnvDuration("NF_TIMEOUT", &c.Timeout); err != nil {
		return c, err
	}
	return c, nil
}

func nonEmptyString(envVar string, dst *string) error {
	val := os.Getenv(envVar)
	if val == "" {
		return fmt.Errorf("%s: must not be empty", envVar)
	}
	*dst = val
	return nil

}

func parseEnvDuration(envVar string, dst *time.Duration) error {
	val := os.Getenv(envVar)
	d, err := time.ParseDuration(val)
	if err != nil {
		return fmt.Errorf("%s: %s", envVar, err)
	}
	*dst = d
	return nil
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
