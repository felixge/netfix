package netfix

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/felixge/netfix/pg"
	"github.com/hashicorp/go-multierror"
)

func TestConfig(t *testing.T) Config {
	t.Helper()
	c, err := EnvConfig()
	if err != nil {
		t.Fatal(err)
	}
	return c
}

func EnvConfig() (Config, error) {
	c := Config{DB: pg.Config{
		Pass:  os.Getenv("PGPASSWORD"),
		Extra: os.Getenv("NF_PGEXTRA"),
	}}

	var result *multierror.Error
	if err := nonEmptyString("PGHOST", &c.DB.Host); err != nil {
		result = multierror.Append(err, result)
	}
	if err := nonEmptyString("PGPORT", &c.DB.Port); err != nil {
		result = multierror.Append(err, result)
	}
	if err := nonEmptyString("PGUSER", &c.DB.User); err != nil {
		result = multierror.Append(err, result)
	}
	if err := nonEmptyString("PGDATABASE", &c.DB.DB); err != nil {
		result = multierror.Append(err, result)
	}
	if err := nonEmptyString("PGSSLMODE", &c.DB.SSLMode); err != nil {
		result = multierror.Append(err, result)
	}
	if err := nonEmptyStringSlice("NF_PGSCHEMAS", &c.DB.Schemas); err != nil {
		result = multierror.Append(err, result)
	}
	if err := nonEmptyString("NF_HTTP_ADDR", &c.HttpAddr); err != nil {
		result = multierror.Append(err, result)
	}
	if err := nonEmptyString("NF_TARGET", &c.Target); err != nil {
		result = multierror.Append(err, result)
	}
	if err := nonEmptyString("NF_IP_VERSION", &c.IPVersion); err != nil {
		result = multierror.Append(err, result)
	}
	if err := parseEnvDuration("NF_INTERVAL", &c.Interval); err != nil {
		result = multierror.Append(err, result)
	}
	if err := parseEnvDuration("NF_TIMEOUT", &c.Timeout); err != nil {
		result = multierror.Append(err, result)
	}
	return c, result.ErrorOrNil()
}

func nonEmptyString(envVar string, dst *string) error {
	val := os.Getenv(envVar)
	if val == "" {
		return fmt.Errorf("%s: must not be empty", envVar)
	}
	*dst = val
	return nil
}

func nonEmptyStringSlice(envVar string, dst *[]string) error {
	vals := strings.Split(os.Getenv(envVar), ",")
	for _, v := range vals {
		v = strings.TrimSpace(v)
		if v != "" {
			*dst = append(*dst, v)
		}
	}
	if len(*dst) == 0 {
		return fmt.Errorf("%s: must not be non-empty list", envVar)
	}
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
	DB        pg.Config
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
