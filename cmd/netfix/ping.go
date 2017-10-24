package main

import (
	"database/sql"
	"log"
	"net"
	"time"

	ndb "github.com/felixge/netfix/db"
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

	var (
		id     = ping.ProcessID()
		errCh  = make(chan error)
		stopCh = make(chan struct{})
		pingCh = make(chan ndb.Ping)

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
	pingCh   chan<- ndb.Ping
	stopCh   <-chan struct{}
}

func (p *pingRoutine) Run() error {
	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()
	for seq := uint16(0); ; seq++ {
		start := time.Now()

		echo := &ping.Echo{
			ID:   p.id,
			Seq:  seq,
			Data: []byte(start.Format(time.RFC3339Nano)),
		}
		if err := p.p.Send(p.dst, echo); err != nil {
			return err
		}
		pg := ndb.Ping{Start: start}
		select {
		case p.pingCh <- pg:
		case <-p.stopCh:
			return nil
		}
		time.AfterFunc(p.timeout, func() {
			pg.Duration = time.Since(start)
			pg.Timeout = true
			select {
			case p.pingCh <- pg:
			case <-p.stopCh:
			}
		})

		select {
		case <-ticker.C:
		case <-p.stopCh:
			return nil
		}
	}
	return nil
}

type receiveRoutine struct {
	p      *ping.Pinger
	id     uint16
	pingCh chan<- ndb.Ping
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
		select {
		case r.pingCh <- ndb.Ping{Start: start, Duration: dt}:
		case <-r.stopCh:
			return nil
		default:
		}
	}
}

type storeRoutine struct {
	db     *sql.DB
	pingCh <-chan ndb.Ping
	stopCh <-chan struct{}
}

func (s *storeRoutine) Run() error {
	for {
		select {
		case <-s.stopCh:
			return nil
		case pg := <-s.pingCh:
			if pg.Duration == 0 {
				if err := pg.InsertOr(s.db, ndb.OrAbort); err != nil {
					return err
				}
			} else {
				if err := pg.Finalize(s.db); err != nil {
					return err
				}
			}
		}
	}
}

func store(db *sql.DB, pg ndb.Ping) error {
	if pg.Duration == 0 {
		if err := pg.InsertOr(db, ndb.OrAbort); err != nil {
			return err
		}
	} else {
		if err := pg.Finalize(db); err != nil {
			return err
		}
	}
	return nil
}
