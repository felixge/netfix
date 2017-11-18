// +build !linux

package ping

import "os"

func (p *Pinger) ID() uint16 {
	// On non-linux systems the convention is to use the process id as the ICMP
	// echo request ID.
	return uint16(os.Getpid() & 0xffff)
}
