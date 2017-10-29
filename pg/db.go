package pg

import (
	"database/sql"
	"fmt"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/lib/pq"
	"github.com/pkg/errors"
)

type Config struct {
	Host    string
	Port    string
	User    string
	Pass    string
	DB      string
	SSLMode string
	AppName string
	Extra   string
	Schemas []string
}

func (c Config) Open() (*sql.DB, error) {
	return c.open(false)
}

func (c Config) OpenTest(t *testing.T) *sql.DB {
	t.Helper()
	db, err := c.open(true)
	if err != nil {
		t.Fatal(err)
	}
	return db
}

func (c Config) open(migrateClean bool) (*sql.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s application_name=%s %s",
		c.Host,
		c.Port,
		c.User,
		c.Pass,
		c.DB,
		c.AppName,
		c.Extra,
	)

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return nil, errors.Wrap(err, "open db")
	} else if err := migrate(db, migrateClean); err != nil {
		return nil, errors.Wrap(err, "open db")
	} else if err := setSearchPath(db, c.Schemas); err != nil {
		return nil, errors.Wrap(err, "open db")
	} else {
		return db, nil
	}
}

// setSearchPath alters the default search_path of the current database to the
// given schemas.
func setSearchPath(db *sql.DB, schemas []string) error {
	dbName, err := currentDB(db)
	if err != nil {
		return errors.Wrap(err, "set search_path")
	}
	setSP := `SET search_path TO ` + strings.Join(quoteIdentifiers(schemas), ",")
	sql := setSP + `; ALTER DATABASE ` + pq.QuoteIdentifier(dbName) + setSP
	_, err = db.Exec(sql)
	err = errors.Wrap(err, "set search_path")
	return err
}

func quoteIdentifiers(s []string) []string {
	quoted := make([]string, len(s))
	for i, _ := range s {
		quoted[i] = pq.QuoteIdentifier(s[i])
	}
	return quoted
}

// currentDB returns the name of the current db. This is useful for queries
// where expressions can't be used.
func currentDB(db *sql.DB) (string, error) {
	var currentDB string
	row := db.QueryRow("SELECT current_database()")
	if err := row.Scan(&currentDB); err != nil {
		return "", errors.Wrap(err, "current db")
	}
	return currentDB, nil
}

//func Migrate(db *sql.DB) error {
//return migrate(db, false)
//}

func migrate(db *sql.DB, clean bool) error {
	var args []string
	if clean {
		args = append(args, "clean")
	}
	args = append(args, "migrate")

	cmd := exec.Command("flyway.sh", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return errors.Wrap(err, "migrate: "+string(out))
	}
	return nil

}

type OutageFilter struct {
	MinLoss        float64
	OutageLoss     float64
	OutageDuration time.Duration
	OutageGap      time.Duration
}

func (o OutageFilter) String() string {
	return fmt.Sprintf(
		"MinLoss=%.2f OutageLoss=%.2f OutageDuration=%s OutageGap=%s",
		o.MinLoss,
		o.OutageLoss,
		o.OutageDuration,
		o.OutageGap,
	)
}

func Outages(db *sql.DB, f OutageFilter) (OutageList, error) {
	const sql = `
WITH timeout_mins AS (
  SELECT DISTINCT cast(start / 60 AS integer) * 60 AS start
  FROM pings
  WHERE timeout = 1
  GROUP BY 1
),

count_mins AS (
  SELECT
    timeout_mins.start,
    count(case when pings.timeout = 1 then 1 else null end) AS lost,
    count(1) as total
  FROM timeout_mins
  JOIN pings ON 
		pings.start >= timeout_mins.start
		AND pings.start < timeout_mins.start + 60
		AND pings.duration IS NOT NULL
  GROUP BY 1
),

loss_mins AS (
  SELECT
    strftime('%Y-%m-%d %H:%M:%S', start, 'unixepoch') AS start,
    lost,
    total,
    lost*1.0/total*100 AS loss
  FROM count_mins
)

SELECT start, lost, total
FROM loss_mins
WHERE loss > $1
ORDER BY 1 ASC;
`

	rows, err := db.Query(sql, f.MinLoss)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var outages OutageList
	for rows.Next() {
		var min LossMin
		var minS string
		if err := rows.Scan(&minS, &min.Lost, &min.Total); err != nil {
			return nil, err
		} else if minT, err := time.Parse("2006-01-02 15:04:05", minS); err != nil {
			return nil, err
		} else {
			min.Start = minT
		}

		var outage *Outage
		if len(outages) > 0 {
			outage = outages[len(outages)-1]
			if min.Start.Sub(outage.End) >= f.OutageGap {
				outage = nil
			}
		}
		if outage == nil {
			outage = &Outage{Start: min.Start}
			outages = append(outages, outage)
		}
		outage.End = min.Start.Add(time.Minute)
		outage.Lost += min.Lost
		outage.Total += min.Total
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	var filtered OutageList
	for _, outage := range outages {
		if outage.Duration() >= f.OutageDuration && outage.Loss() >= f.OutageLoss {
			filtered = append(filtered, outage)
		}
	}
	return filtered, nil
}

type OutageList []*Outage

func (ol OutageList) Duration() time.Duration {
	var total time.Duration
	for _, o := range ol {
		total += o.Duration()
	}
	return total
}

type Outage struct {
	Start time.Time
	End   time.Time
	Lost  int
	Total int
}

func (o Outage) Loss() float64 {
	return float64(o.Lost) / float64(o.Total)
}

func (o Outage) String() string {
	return fmt.Sprintf(
		"start=%s duration=%s lost=%d total=%d loss=%f",
		o.Start,
		o.Duration(),
		o.Lost,
		o.Total,
		o.Loss()*100,
	)
}

func (o Outage) Duration() time.Duration {
	return o.End.Sub(o.Start)
}

type LossMin struct {
	Start time.Time
	Lost  int
	Total int
}

func (m LossMin) String() string {
	return fmt.Sprintf(
		"start=%s lost=%d total=%d loss=%f",
		m.Start,
		m.Lost,
		m.Total,
		float64(m.Lost)/float64(m.Total)*100,
	)
}

type DBOrTx interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

type Ping struct {
	Start    time.Time
	Duration time.Duration
	Timeout  bool
}

func (p Ping) String() string {
	return fmt.Sprintf(
		"start=%s duration=%s timeout=%t",
		p.Start,
		p.Duration,
		p.Timeout,
	)
}

func (p Ping) InsertOrIgnore(db DBOrTx) error {
	sql := `
INSERT INTO pings (started, duration_ms, timeout)
VALUES ($1, $2, $3)
ON CONFLICT (started) DO NOTHING;
`

	_, err := db.Exec(sql, p.sqlArgs()...)
	return err
}

func (p Ping) Finalize(db DBOrTx) error {
	// @TODO(fg) not sure why SET start = $1 is needed here, but without it the
	// query fails. Might be something odd about how the go lib for sqlite3
	// does placeholder variables.
	const sql = `UPDATE pings SET start = $1, duration = $2, timeout = $3 WHERE start = $1 AND duration IS NULL;`
	_, err := db.Exec(sql, p.sqlArgs()...)
	return err
}

func (p Ping) sqlArgs() []interface{} {
	args := []interface{}{p.Start, nil, p.Timeout}
	duration := float64(p.Duration.Nanoseconds()) / float64(time.Millisecond)
	if duration != 0 {
		args[1] = duration
	}
	return args
}
