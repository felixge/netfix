package main

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"net"
	"os"
	"time"

	"github.com/felixge/netfix"
	"github.com/kelseyhightower/envconfig"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

const (
	ProtocolICMP     = 1  // iana.ProtocolICMP
	ProtocolIPv6ICMP = 58 // iana.ProtocolIPv6ICMP
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run() error {
	c := EnvConfig()
	db, err := c.OpenDB()
	if err != nil {
		return fmt.Errorf("db: open: %s", err)
	} else if from, to, err := netfix.Migrate(db); err != nil {
		return err
	} else if from != to {
		log.Printf("db: migrated from version %d to %d", from, to)
	}

	p := &Pinger{
		Network:    c.Network,
		LocalAddr:  c.LocalAddr,
		RemoteAddr: c.RemoteAddr,
		Interval:   c.Interval,
		Timeout:    c.Timeout,

		Protocol: ProtocolICMP,
		ICMPType: ipv4.ICMPTypeEcho,
		DB:       db,
	}
	return p.Run()
}

type Pinger struct {
	Network    string
	LocalAddr  string
	RemoteAddr string
	Protocol   int
	ICMPType   icmp.Type
	Interval   time.Duration
	Timeout    time.Duration
	DB         *sql.DB
}

func (p *Pinger) Run() error {
	c, err := icmp.ListenPacket(p.Network, p.LocalAddr)
	if err != nil {
		return err
	}
	defer c.Close()

	dst, err := resolve(p.RemoteAddr, c, p.Protocol)
	if err != nil {
		return err
	}
	log.Printf("resolved %s to %s", p.RemoteAddr, dst)

	if p.Network != "udp6" && p.Protocol == ProtocolIPv6ICMP {
		var f ipv6.ICMPFilter
		f.SetAll(true)
		f.Accept(ipv6.ICMPTypeDestinationUnreachable)
		f.Accept(ipv6.ICMPTypePacketTooBig)
		f.Accept(ipv6.ICMPTypeTimeExceeded)
		f.Accept(ipv6.ICMPTypeParameterProblem)
		f.Accept(ipv6.ICMPTypeEchoReply)
		if err := c.IPv6PacketConn().SetICMPFilter(&f); err != nil {
			return err
		}
	}

	var (
		errCh = make(chan error)
		// @TODO (how big should this buffer be, deal with slow sqlite?)
		pingCh = make(chan Ping, 1024)
		id     = os.Getpid() & 0xffff
	)

	// ping sender
	go func() {
		for seq := 0; ; seq++ {
			time.Sleep(p.Interval)
			ping := Ping{
				ID:    id,
				Seq:   seq,
				Start: time.Now().UTC(),
			}

			wm := icmp.Message{
				Type: p.ICMPType,
				Code: 0,
				Body: &icmp.Echo{
					ID:   id,
					Seq:  seq,
					Data: []byte(ping.Start.Format(time.RFC3339Nano)),
				},
			}

			wb, err := wm.Marshal(nil)
			if err != nil {
				errCh <- err
				return
			}

			pingCh <- ping
			time.AfterFunc(p.Timeout, func() {
				ping.End = time.Now()
				ping.Timeout = true
				pingCh <- ping
			})
			if n, err := c.WriteTo(wb, dst); err != nil {
				fmt.Printf("%s\n", err)
				continue
			} else if n != len(wb) {
				errCh <- fmt.Errorf("got %v; want %v", n, len(wb))
				return
			}
		}
	}()

	// ping receiver
	go func() {
		for {
			rb := make([]byte, 1500)
			n, peer, err := c.ReadFrom(rb)
			if err != nil {
				errCh <- err
				return
			}
			end := time.Now()
			rm, err := icmp.ParseMessage(p.Protocol, rb[:n])
			if err != nil {
				errCh <- err
				return
			}
			switch rm.Type {
			case ipv4.ICMPTypeEchoReply, ipv6.ICMPTypeEchoReply:
				data := rm.Body.(*icmp.Echo)
				// Discard ping replies that are not meant for us.
				if data.ID != id {
					continue
				}
				start, err := time.Parse(time.RFC3339Nano, string(data.Data))
				if err != nil {
					errCh <- err
					return
				}
				ping := Ping{
					ID:    data.ID,
					Seq:   data.Seq,
					Start: start,
					End:   end,
				}
				pingCh <- ping
				continue
			case ipv4.ICMPTypeDestinationUnreachable, ipv6.ICMPTypeDestinationUnreachable:
				// @TODO log / track in stats (timeout already gets recored anyway)
				data := rm.Body.(*icmp.DstUnreach)
				fmt.Printf("%#v: %s\n", data, data.Data)
				continue
			default:
				// @TODO log / track in stats
				errCh <- fmt.Errorf("got %#v from %v; want echo reply", rm, peer)
				return
			}
		}
	}()

	for {
		select {
		case ping := <-pingCh:
			start := ping.Start.UnixNano()
			if ping.End.IsZero() {
				sql := "INSERT INTO pings (start) VALUES ($1);"
				if _, err := p.DB.Exec(sql, start); err != nil {
					return err
				}
			} else {
				duration := float64(ping.End.Sub(ping.Start).Nanoseconds()) / float64(time.Millisecond)
				sql := `
INSERT OR REPLACE INTO pings (start, duration, timeout)
SELECT
	$1,
	coalesce(pings.duration, $2),
	coalesce(pings.timeout, $3)
FROM pings
WHERE start = $1;
`
				if _, err := p.DB.Exec(sql, start, duration, ping.Timeout); err != nil {
					return err
				}
			}
		case err := <-errCh:
			return err
		}
	}

	return nil
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

func resolve(host string, c *icmp.PacketConn, protocol int) (net.Addr, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}
	netaddr := func(ip net.IP) (net.Addr, error) {
		switch c.LocalAddr().(type) {
		case *net.UDPAddr:
			return &net.UDPAddr{IP: ip}, nil
		case *net.IPAddr:
			return &net.IPAddr{IP: ip}, nil
		default:
			return nil, errors.New("neither UDPAddr nor IPAddr")
		}
	}
	for _, ip := range ips {
		switch protocol {
		case ProtocolICMP:
			if ip.To4() != nil {
				return netaddr(ip)
			}
		case ProtocolIPv6ICMP:
			if ip.To16() != nil && ip.To4() == nil {
				return netaddr(ip)
			}
		}
	}
	return nil, errors.New("no A or AAAA record")
}

var helpFlags = map[string]bool{
	"-h":     true,
	"--help": true,
	"-help":  true,
}

func EnvConfig() Config {
	var c Config
	err := envconfig.Process("nf", &c)
	if len(os.Args) > 1 && helpFlags[os.Args[1]] {
		envconfig.Usage("nf", &c)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	return c
}

type Config struct {
	DB         string        `required:"true"`
	Network    string        `required:"true"`
	LocalAddr  string        `required:"true"`
	RemoteAddr string        `required:"true"`
	Interval   time.Duration `required:"true"`
	Timeout    time.Duration `required:"true"`
}

func (c Config) OpenDB() (*sql.DB, error) {
	return sql.Open("sqlite3", c.DB)
}
