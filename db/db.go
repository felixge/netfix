package netfix

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func Open(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	} else if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, err
	}

	return db, nil
}

func Migrate(db *sql.DB) (from, to int, err error) {
	tx, err := db.Begin()
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback()

	migrations := []func() error{
		func() error {
			_, err = tx.Exec(`
CREATE TABLE schema_versions (
	version int
);

CREATE TABLE pings (
	start REAL PRIMARY KEY,
	duration REAL,
	timeout bool
);`,
			)
			return err
		},
		func() error {
			_, err = tx.Exec(`CREATE INDEX pings_timeout_idx ON pings(timeout);`)
			return err
		},
	}

	_ = tx.QueryRow("SELECT max(version) FROM schema_versions").Scan(&from)
	if from > len(migrations) {
		return 0, 0, fmt.Errorf("unknown schema_version: %d", from)
	}
	for i, mig := range migrations[from:] {
		if err := mig(); err != nil {
			return 0, 0, err
		} else if _, err := tx.Exec("INSERT INTO schema_versions VALUES ($1);", i+from+1); err != nil {
			return 0, 0, err
		}
	}
	to = len(migrations)
	return from, to, tx.Commit()
}

type OutageFilter struct {
	MinLoss        float64
	OutageLoss     float64
	OutageDuration time.Duration
	OutageGap      time.Duration
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
  JOIN pings ON pings.start >= timeout_mins.start AND pings.start < timeout_mins.start + 60
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

func (p Ping) Insert(db DBOrTx) error {
	const sql = `
INSERT OR REPLACE INTO pings (start, duration, timeout)
VALUES ($1, $2, $3)
`

	start := float64(p.Start.UnixNano()) / float64(time.Second)
	duration := float64(p.Duration.Nanoseconds()) / float64(time.Millisecond)
	_, err := db.Exec(sql, start, duration, p.Timeout)
	return err
}
