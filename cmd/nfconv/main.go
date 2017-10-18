package main

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 3 {
		return errors.New("usage: nfconv <legacy_file> <nf_db>")
	}

	lf, err := os.Open(os.Args[1])
	if err != nil {
		return err
	}
	defer lf.Close()

	db, err := sql.Open("sqlite3", os.Args[2])
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
CREATE TABLE pings (
	time integer PRIMARY KEY,
	millisec integer,
	timeout bool
)`)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	r := bufio.NewReader(lf)
	stats := Stats{Start: time.Now()}
	dreg := regexp.MustCompile("time=([0-9.]+) ([a-z]+)")
	for {
		stats.Lines++
		line, err := r.ReadString('\n')
		if err == io.EOF {
			break
		} else if err != nil {
			return err
			//} else if stats.Lines > 1000 {
			//break
		}

		ts := line[0:19]
		t, err := time.ParseInLocation("2006/01/02 15:04:05", ts, time.Local)
		if err != nil {
			stats.Errors++
			continue
		}
		if strings.Contains(line, "unreachable") {

			sql := "INSERT OR REPLACE INTO pings (time, millisec, timeout) VALUES ($1, $2, $3);"
			if _, err := tx.Exec(sql, t.Unix(), 1000, true); err != nil {
				return err
			}
			stats.Unreachable++
			continue
		}
		m := dreg.FindStringSubmatch(line)
		if len(m) != 3 {
			return fmt.Errorf("bad time on line: %d: %s", stats.Lines, line)
		}
		d, err := time.ParseDuration(m[1] + m[2])
		if err != nil {
			return err
		}
		sql := "INSERT OR REPLACE INTO pings (time, millisec) VALUES ($1, $2);"
		if _, err := tx.Exec(sql, t.Unix(), d.Nanoseconds()/int64(time.Millisecond)); err != nil {
			return err
		}
	}
	fmt.Printf("%s\n", stats)
	return tx.Commit()
}

type Stats struct {
	Start       time.Time
	Unreachable int
	Lines       int
	Errors      int
}

func (s Stats) String() string {
	return fmt.Sprintf(
		"lines: %d\nunreachable: %d\nerrors: %d\nduration: %s",
		s.Lines,
		s.Unreachable,
		s.Errors,
		time.Since(s.Start),
	)
}
