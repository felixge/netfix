package db

import (
	"bufio"
	"encoding/binary"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type Val uint16

func (v Val) IsNil() bool {
	return v == (256*256)-1
}

type DB struct {
	path   string
	lock   sync.Mutex
	file   *os.File
	bw     *bufio.Writer
	br     *bufio.Reader
	offset int64
	stats  Stats
}

type Stats struct {
	Seek  int64
	Open  int64
	Close int64
}

func Open(path string) (*DB, error) {
	if err := os.MkdirAll(path, 0700); err != nil {
		return nil, err
	}
	return &DB{path: path}, nil
}

// Read returns all vals with a timestamp >= start and < end or an error.
func (d *DB) Read(start, end time.Time) ([]Val, error) {
	d.lock.Lock()
	defer d.lock.Unlock()

	var (
		su, eu = start.Unix(), end.Unix()
		vals   = make([]Val, eu-su)
		buf    = make([]byte, 2)
	)

	for u := su; u < eu; u++ {
		if err := d.open(u); err != nil {
			return nil, err
		} else if _, err := io.ReadFull(d.br, buf); err != nil {
			return nil, err
		}
		d.offset += int64(len(buf))
		vals[u-su] = Val(binary.LittleEndian.Uint16(buf))
	}

	return vals, nil
}

func (d *DB) Write(start time.Time, vals ...Val) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	var (
		su = start.Unix()
		eu = su + int64(len(vals))
	)

	for u := su; u < eu; u++ {
		if err := d.open(u); err != nil {
			return err
		}
		buf := make([]byte, 2)
		val := vals[u-su]
		binary.LittleEndian.PutUint16(buf, uint16(val))
		if _, err := d.bw.Write(buf); err != nil {
			return err
		}
		d.offset += int64(len(buf))
	}

	return nil
}

func (d *DB) Stats() Stats {
	d.lock.Lock()
	defer d.lock.Unlock()
	return d.stats
}

func (d *DB) open(u int64) error {
	var err error
	path, offset := d.pathOffset(u)
	if d.file == nil || d.file.Name() != path {
		if d.file != nil {
			if err := d.bw.Flush(); err != nil {
				return err
			} else if err := d.file.Close(); err != nil {
				return err
			}
			d.stats.Close++
		}
		d.stats.Open++
		d.file, err = os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0600)
		if err != nil {
			return err
		}
		d.offset = 0
		d.bw = bufio.NewWriter(d.file)
		d.br = bufio.NewReader(d.file)
	}
	if d.offset != offset {
		if _, err := d.file.Seek(offset, 0); err != nil {
			return err
		}
		d.stats.Seek++
		d.offset = offset
	}
	return nil
}

func (d *DB) pathOffset(u int64) (string, int64) {
	day := u - (u % (24 * 60 * 60))
	offset := (u - day) * 2
	path := filepath.Join(d.path, fmt.Sprintf("%d.vals", day))
	return path, offset
}

func (d *DB) Flush() error {
	if d.file == nil {
		return nil
	} else if err := d.bw.Flush(); err != nil {
		return err
	}
	return nil
}

func (d *DB) Close() error {
	if d.file == nil {
		return nil
	} else if err := d.Flush(); err != nil {
		return err
	} else {
		return d.file.Close()
	}

}
