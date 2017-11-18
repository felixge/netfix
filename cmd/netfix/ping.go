package main

import (
	"database/sql"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/felixge/netfix"
	"github.com/felixge/netfix/pg"
	"github.com/felixge/netfix/ping"
)

func recordPings(c netfix.Config, db *sql.DB) error {
	p, err := ping.NewPinger(ping.IPVersion(c.IPVersion))
	if err != nil {
		return err
	}
	dst, err := p.Resolve(c.Target)
	if err != nil {
		return err
	}
	log.Printf("resolved %s to %s", c.Target, dst)

	var (
		id     = p.ID()
		errCh  = make(chan error)
		stopCh = make(chan struct{})
		pingCh = make(chan pg.Ping)

		pr = &pingRoutine{
			p:        p,
			interval: c.Interval,
			timeout:  c.Timeout,
			dst:      dst,
			id:       id,
			pingCh:   pingCh,
			stopCh:   stopCh,
		}

		rr = &receiveRoutine{
			p:      p,
			id:     id,
			pingCh: pingCh,
			stopCh: stopCh,
		}

		sr = &storeRoutine{
			db:     db,
			pingCh: pingCh,
			stopCh: stopCh,
		}
	)

	go func() { errCh <- pr.Run() }()
	go func() { errCh <- rr.Run() }()
	go func() { errCh <- sr.Run() }()

	var firstErr error
	for i := 0; i < 3; i++ {
		if err := <-errCh; err != nil && firstErr == nil {
			firstErr = err
			close(stopCh)
		}
	}

	return firstErr
}

type pingRoutine struct {
	p        *ping.Pinger
	interval time.Duration
	timeout  time.Duration
	dst      net.Addr
	id       uint16
	pingCh   chan<- pg.Ping
	stopCh   <-chan struct{}
}

func (pr *pingRoutine) Run() error {
	ticker := time.NewTicker(pr.interval)
	defer ticker.Stop()
	for seq := uint16(0); ; seq++ {
		start := time.Now()

		echo := &ping.Echo{
			ID:   pr.id,
			Seq:  seq,
			Data: []byte(start.Format(time.RFC3339Nano)),
		}
		if err := pr.p.Send(pr.dst, echo); err != nil {
			return err
		}
		p := pg.Ping{Start: start}
		select {
		case pr.pingCh <- p:
		case <-pr.stopCh:
			return nil
		}
		time.AfterFunc(pr.timeout, func() {
			p.Duration = time.Since(start)
			p.Timeout = true
			select {
			case pr.pingCh <- p:
			case <-pr.stopCh:
			}
		})

		select {
		case <-ticker.C:
		case <-pr.stopCh:
			return nil
		}
	}
	return nil
}

type receiveRoutine struct {
	p      *ping.Pinger
	id     uint16
	pingCh chan<- pg.Ping
	stopCh <-chan struct{}
}

func (r *receiveRoutine) Run() error {
	for {
		select {
		case <-r.stopCh:
			return nil
		default:
		}
		echo, err := r.p.Receive()
		if err != nil {
			if ping.IsTemporary(err) {
				if !ping.IsTimeout(err) {
					log.Printf("receive error: %s", err)
				}
				continue
			}
			return err
		} else if echo.ID != r.id {
			continue
		}

		start, err := time.Parse(time.RFC3339Nano, string(echo.Data))
		if err != nil {
			return err
		}
		dt := time.Since(start)
		fmt.Printf("%s - %s\n", echo, dt)
		select {
		case r.pingCh <- pg.Ping{Start: start, Duration: dt}:
		case <-r.stopCh:
			return nil
		}
	}
}

type storeRoutine struct {
	db     *sql.DB
	pingCh <-chan pg.Ping
	stopCh <-chan struct{}
}

func (s *storeRoutine) Run() error {
	for {
		select {
		case <-s.stopCh:
			return nil
		case p := <-s.pingCh:
			sql := `
INSERT INTO pings (started, duration_ms, timeout)
VALUES ($1, $2, $3)
ON CONFLICT (started) DO UPDATE
SET
	duration_ms = EXCLUDED.duration_ms,
	timeout = EXCLUDED.timeout
WHERE pings.duration_ms IS NULL;
`
			var duration interface{}
			if p.Duration > 0 {
				duration = float64(p.Duration.Nanoseconds()) / float64(time.Millisecond)
			}
			if _, err := s.db.Exec(sql, p.Start, duration, p.Timeout); err != nil {
				return err
			}
		}
	}
}
