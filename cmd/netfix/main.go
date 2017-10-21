package main

import (
	"encoding/json"
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

	return serveHttp(c, db)
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
