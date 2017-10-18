package netfix

import (
	"database/sql"
	"fmt"
	"os"

	"github.com/kelseyhightower/envconfig"
)

var helpFlags = map[string]bool{
	"-h":     true,
	"--help": true,
	"-help":  true,
}

func EnvConfig() Config {
	var c Config
	err := envconfig.Process("nf", &c)
	if len(os.Args) > 1 && helpFlags[os.Args[1]] {
		envconfig.Usage("nf", &c)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return c
}

type Config struct {
	DB string `required:"true"`
}

func (c Config) OpenDB() (*sql.DB, error) {
	return sql.Open("sqlite3", c.DB)
}
