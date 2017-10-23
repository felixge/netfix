package main

import (
	"database/sql"
	"log"
	"net"
	"time"

	"github.com/felixge/netfix/ping"
)

func recordPings(c Config, db *sql.DB) error {
	p, err := ping.NewPinger(ping.IPVersion(c.IPVersion))
	if err != nil {
		return err
	}
	dst, err := p.Resolve(c.Target)
	if err != nil {
		return err
	}
	log.Printf("resolved %s to %s", c.Target, dst)

	id := ping.ProcessID()
	errCh := make(chan error)
	stopCh := make(chan struct{})
	go func() { errCh <- pingRoutine(p, c.Interval, dst, id, stopCh) }()
	go func() { errCh <- receiveRoutine(p, id, stopCh) }()

	return <-errCh
}

func pingRoutine(p *ping.Pinger, interval time.Duration, dst net.Addr, id uint16, stopCh <-chan struct{}) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for seq := uint16(0); ; seq++ {
		start := time.Now()

		echo := &ping.Echo{
			ID:   id,
			Seq:  seq,
			Data: []byte(start.Format(time.RFC3339Nano)),
		}
		if err := p.Send(dst, echo); err != nil {
			return err
		}
		select {
		case <-ticker.C:
		case <-stopCh:
			return nil
		}
	}
	return nil
}

func receiveRoutine(p *ping.Pinger, id uint16, stopCh <-chan struct{}) error {
	for {
		select {
		case <-stopCh:
			return nil
		default:
		}
		echo, err := p.Receive()
		if err != nil {
			if ping.IsTemporary(err) {
				if !ping.IsTimeout(err) {
					log.Printf("%s", err)
				}
				continue
			}
			return err
		} else if echo.ID != id {
			continue
		}

		start, err := time.Parse(time.RFC3339Nano, string(echo.Data))
		if err != nil {
			return err
		}
		dt := time.Since(start)
		log.Printf("%s - %s", echo, dt)
	}
}
