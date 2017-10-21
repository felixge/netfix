package netfix

import (
	"database/sql"
	"fmt"
	"time"
)

func OpenDB(dsn string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dsn)
	if err != nil {
		return nil, err
	} else if _, err := db.Exec("PRAGMA journal_mode=WAL"); err != nil {
		return nil, err
	}

	return db, nil
}

func MigrateDB(db *sql.DB) (from, to int, err error) {
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

type DBOrTx interface {
	Exec(query string, args ...interface{}) (sql.Result, error)
}

type Ping struct {
	ID      int
	Seq     int
	Start   time.Time
	End     time.Time
	Timeout bool
}

func (p Ping) String() string {
	var d time.Duration
	if !p.End.IsZero() {
		d = p.End.Sub(p.Start)
	}
	return fmt.Sprintf(
		"id=%d seq=%d start=%s end=%s duration=%s timeout=%t",
		p.ID,
		p.Seq,
		p.Start,
		p.End,
		d,
		p.Timeout,
	)
}

func (p Ping) Insert(db DBOrTx) error {
	const sql = `
INSERT OR REPLACE INTO pings (start, duration, timeout)
VALUES ($1, $2, $3)
`

	start := float64(p.Start.UnixNano()) / float64(time.Second)
	duration := float64(p.End.Sub(p.Start).Nanoseconds()) / float64(time.Millisecond)
	_, err := db.Exec(sql, start, duration, p.Timeout)
	return err
}
