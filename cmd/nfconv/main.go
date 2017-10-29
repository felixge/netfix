package main

import (
	"bufio"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/felixge/netfix"
	"github.com/felixge/netfix/pg"
	"github.com/lib/pq"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	if len(os.Args) != 2 {
		return errors.New("usage: nfconv <legacy_file>")
	}

	c, err := netfix.EnvConfig()
	if err != nil {
		return err
	}

	db, err := c.DB.Open()
	if err != nil {
		return err
	}

	if stats, err := Convert(os.Args[1], db); err != nil {
		return err
	} else {
		fmt.Printf("%s\n", stats)
	}
	return nil
}

func Convert(legacyFile string, db *sql.DB) (Stats, error) {
	stats := Stats{Start: time.Now()}
	lf, err := os.Open(legacyFile)
	if err != nil {
		return stats, err
	}
	defer lf.Close()

	tx, err := db.Begin()
	if err != nil {
		return stats, err
	}
	defer tx.Rollback()

	stmt, err := tx.Prepare(pq.CopyIn("pings", "started", "duration_ms", "timeout"))
	if err != nil {
		log.Fatal(err)
	}
	defer stmt.Close()

	var (
		ld = NewLegacyDecoder(lf)
		p  = &pg.Ping{}
	)

	var prevStart time.Time
	for stats.Lines = 1; ; stats.Lines++ {
		if err := ld.Read(p); err == io.EOF {
			break
		} else if err != nil {
			stats.Errors++
			continue
		} else if p.Start.Equal(prevStart) {
			stats.Dupes++
			continue
		} else if p.Timeout {
			stats.Timeouts++
		}
		args := []interface{}{
			p.Start,
			float64(p.Duration.Nanoseconds()) / float64(time.Millisecond),
			p.Timeout,
		}
		if _, err := stmt.Exec(args...); err != nil {
			return stats, err
		}
		prevStart = p.Start
	}

	if _, err := stmt.Exec(); err != nil {
		return stats, err
	} else if err := stmt.Close(); err != nil {
		return stats, err
	}

	return stats, tx.Commit()
}

func NewLegacyDecoder(r io.Reader) *LegacyDecoder {
	return &LegacyDecoder{r: bufio.NewReader(r)}
}

type LegacyDecoder struct {
	line int
	r    *bufio.Reader
}

var durationPattern = regexp.MustCompile("time=([0-9.]+) ([a-z]+)")

func (d *LegacyDecoder) Read(p *pg.Ping) error {
	d.line++
	line, err := d.r.ReadString('\n')
	if err != nil {
		return err
	}

	ts := line[0:19]
	t, err := time.ParseInLocation("2006/01/02 15:04:05", ts, time.Local)
	if err != nil {
		return err
	}

	if strings.Contains(line, "unreachable") {
		p.Duration = time.Second
		p.Start = t.Add(-p.Duration)
		p.Timeout = true
		return nil
	}
	m := durationPattern.FindStringSubmatch(line)
	if len(m) != 3 {
		return fmt.Errorf("bad time on line: %d: %s", d.line, line)
	} else if duration, err := time.ParseDuration(m[1] + m[2]); err != nil {
		return err
	} else {
		p.Timeout = false
		p.Duration = duration
		p.Start = t.Add(-p.Duration)
	}
	return nil
}

type Stats struct {
	Start    time.Time
	Timeouts int
	Dupes    int
	Lines    int
	Errors   int
}

func (s Stats) String() string {
	return fmt.Sprintf(
		"lines: %d\ntimeouts: %d\ndupes: %d\nerrors: %d\nduration: %s",
		s.Lines,
		s.Timeouts,
		s.Dupes,
		s.Errors,
		time.Since(s.Start),
	)
}
