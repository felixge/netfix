package ping

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
	"time"
)

func TestPinger(t *testing.T) {
	src := rand.NewSource(time.Now().Unix())
	id := ProcessID()
	req := &Echo{
		ID:   id,
		Seq:  uint16(src.Int63()),
		Data: []byte(fmt.Sprintf("%d", src.Int63())),
	}
	for _, ipv := range []IPVersion{IPv4, IPv6} {
		p, err := NewPinger(ipv)
		if err != nil {
			t.Fatal(err)
		} else if dst, err := p.Resolve("google.com"); err != nil {
			t.Fatal(err)
		} else if err := p.Send(dst, req); err != nil {
			t.Fatal(err)
		} else if res, err := p.Receive(id); err != nil {
			t.Fatal(err)
		} else if !reflect.DeepEqual(req, res) {
			t.Fatalf("\ngot =%s\nwant=%s", res, req)
		}
	}
}
