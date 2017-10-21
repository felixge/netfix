package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/felixge/netfix"
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

	db, err := netfix.OpenDB(os.Args[2])
	if err != nil {
		return err
	}
	defer db.Close()

	if from, to, err := netfix.MigrateDB(db); err != nil {
		return err
	} else if from != to {
		log.Printf("db: migrated from version %d to %d", from, to)
	}

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var (
		ld    = NewLegacyDecoder(lf)
		p     = &netfix.Ping{}
		stats = Stats{Start: time.Now()}
	)

	for stats.Lines = 1; ; stats.Lines++ {
		if err := ld.Read(p); err == io.EOF {
			break
		} else if err != nil {
			stats.Errors++
			continue
		} else if p.Timeout {
			stats.Timeouts++
		}
		if err := p.Insert(tx); err != nil {
			return err
		}
	}
	fmt.Printf("%s\n", stats)
	return tx.Commit()
}

func NewLegacyDecoder(r io.Reader) *LegacyDecoder {
	return &LegacyDecoder{r: bufio.NewReader(r)}
}

type LegacyDecoder struct {
	line int
	r    *bufio.Reader
}

var durationPattern = regexp.MustCompile("time=([0-9.]+) ([a-z]+)")

func (d *LegacyDecoder) Read(p *netfix.Ping) error {
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
	p.End = t

	if strings.Contains(line, "unreachable") {
		p.Start = p.End.Add(-time.Second)
		p.Timeout = true
		return nil
	}
	m := durationPattern.FindStringSubmatch(line)
	if len(m) != 3 {
		return fmt.Errorf("bad time on line: %d: %s", d.line, line)
	} else if duration, err := time.ParseDuration(m[1] + m[2]); err != nil {
		return err
	} else {
		p.Start = p.End.Add(-duration)
	}
	return nil
}

type Stats struct {
	Start    time.Time
	Timeouts int
	Lines    int
	Errors   int
}

func (s Stats) String() string {
	return fmt.Sprintf(
		"lines: %d\ntimeouts: %d\nerrors: %d\nduration: %s",
		s.Lines,
		s.Timeouts,
		s.Errors,
		time.Since(s.Start),
	)
}
