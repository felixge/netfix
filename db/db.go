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
	path string
	lock sync.Mutex
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

	su, eu := start.Unix(), end.Unix()
	var (
		file *os.File
		br   *bufio.Reader
		vals = make([]Val, eu-su)
		buf  = make([]byte, 2)
		err  error
	)
	for u := su; u < eu; u++ {
		path, offset := d.pathOffset(u)
		if file == nil || file.Name() != path {
			if file != nil {
				if err := file.Close(); err != nil {
					return nil, err
				}
			}
			file, err = os.OpenFile(path, os.O_CREATE|os.O_RDONLY, 0600)
			if err != nil {
				return nil, err
			}
			if _, err := file.Seek(offset, 0); err != nil {
				return nil, err
			}
			br = bufio.NewReader(file)
			defer file.Close()
		}
		if _, err := io.ReadFull(br, buf); err != nil {
			return nil, err
		}
		vals[u-su] = Val(binary.LittleEndian.Uint16(buf))
	}
	return vals, file.Close()
}

func (d *DB) Write(start time.Time, vals []Val) error {
	d.lock.Lock()
	defer d.lock.Unlock()

	var (
		su   = start.Unix()
		eu   = su + int64(len(vals))
		file *os.File
		bw   *bufio.Writer
		err  error
	)

	for u := su; u < eu; u++ {
		path, offset := d.pathOffset(u)
		if file == nil || file.Name() != path {
			if file != nil {
				if err := bw.Flush(); err != nil {
					return err
				} else if err := file.Close(); err != nil {
					return err
				}
			}
			file, err = os.OpenFile(path, os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				return err
			}
			if _, err := file.Seek(offset, 0); err != nil {
				return err
			}
			bw = bufio.NewWriter(file)
			defer file.Close()
		}

		defer file.Close()
		buf := make([]byte, 2)
		val := vals[u-su]
		binary.LittleEndian.PutUint16(buf, uint16(val))
		if _, err := bw.Write(buf); err != nil {
			return err
		}
	}
	if err := bw.Flush(); err != nil {
		return err
	} else if err := file.Close(); err != nil {
		return err
	}
	return nil
}

func (d *DB) pathOffset(u int64) (string, int64) {
	day := u - (u % (24 * 60 * 60))
	offset := (u - day) * 2
	path := filepath.Join(d.path, fmt.Sprintf("%d.vals", day))
	return path, offset
}
