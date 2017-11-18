package ping

import "net"

func (p *Pinger) ID() uint16 {
	// Linux forces the local port to be used as the ICMP echo request ID.
	// See https://joekuan.wordpress.com/2017/05/30/behaviour-of-identifier-field-in-icmp-ping-as-udp-between-linux-and-osx/
	addr := p.conn.LocalAddr().(*net.UDPAddr)
	return uint16(addr.Port)
}
