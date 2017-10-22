package ping

// Todo:
// - Could I make a single pinger that can ping IPv4 and IPv6 targets? Can I detect if IPv6 is available?
// - Does it make sense to discard non-echo-response replies (e.g. ICMPTypeDestinationUnreachable)?
// - Figure out how ICMP over UDP really works?

import (
	"fmt"
	"io"
	"net"

	"golang.org/x/net/icmp"
	"golang.org/x/net/ipv4"
	"golang.org/x/net/ipv6"
)

type IPVersion int

const (
	IPv4 IPVersion = iota
	IPv6
)

const (
	protocolICMP     = 1  // iana.ProtocolICMP
	protocolIPv6ICMP = 58 // iana.ProtocolIPv6ICMP
)

func NewPinger(ipv IPVersion) (*Pinger, error) {
	var (
		network string
		p       = &Pinger{ipv: ipv}
	)

	switch ipv {
	case IPv4:
		network = "udp4"
		p.icmpType = ipv4.ICMPTypeEcho
		p.protocol = protocolICMP
	case IPv6:
		network = "udp6"
		p.icmpType = ipv6.ICMPTypeEchoRequest
		p.protocol = protocolIPv6ICMP
	default:
		return nil, fmt.Errorf("invalid ip version: %d", ipv)
	}

	c, err := icmp.ListenPacket(network, "")
	if err != nil {
		return nil, err
	}
	p.conn = c
	return p, nil
}

type Echo struct {
	ID   uint16
	Seq  uint16
	Data []byte
}

func (e Echo) String() string {
	return fmt.Sprintf("id=%d seq=%d data=%s", e.ID, e.Seq, e.Data)
}

type Pinger struct {
	conn     *icmp.PacketConn
	ipv      IPVersion
	icmpType icmp.Type
	protocol int
}

func (p *Pinger) Resolve(host string) (net.Addr, error) {
	ips, err := net.LookupIP(host)
	if err != nil {
		return nil, err
	}

	var first net.IP
	for _, ip := range ips {
		switch p.ipv {
		case IPv4:
			if ip.To4() != nil {
				first = ip
				break
			}
		case IPv6:
			if ip.To16() != nil && ip.To4() == nil {
				first = ip
				break
			}
		default:
			panic("bug")
		}
	}

	return &net.UDPAddr{IP: first}, nil
}

func (p *Pinger) Send(dst net.Addr, e *Echo) error {
	wm := icmp.Message{
		Type: p.icmpType,
		Code: 0,
		Body: &icmp.Echo{
			ID:   int(e.ID),
			Seq:  int(e.Seq),
			Data: e.Data,
		},
	}
	wb, err := wm.Marshal(nil)
	if err != nil {
		return err
	}
	if n, err := p.conn.WriteTo(wb, dst); err != nil {
		return err
	} else if n != len(wb) {
		return io.ErrShortWrite
	}
	return nil
}

func (p *Pinger) Receive(id uint16) (*Echo, error) {
	for {
		rb := make([]byte, 1500)
		n, _, err := p.conn.ReadFrom(rb)
		if err != nil {
			return nil, err
		}

		rm, err := icmp.ParseMessage(p.protocol, rb[:n])
		if err != nil {
			return nil, err
		}
		switch rm.Type {
		case ipv4.ICMPTypeEchoReply, ipv6.ICMPTypeEchoReply:
			e := rm.Body.(*icmp.Echo)
			if uint16(e.ID) != id {
				continue
			}
			return &Echo{
				ID:   uint16(e.ID),
				Seq:  uint16(e.Seq),
				Data: e.Data,
			}, nil
		default:
			// For now we just ignore ipv4.ICMPTypeDestinationUnreachable,
			// ipv6.ICMPTypeDestinationUnreachable etc. here. But in the future
			// it might be worth to capture them.
			continue
		}
	}
	return nil, nil
}
