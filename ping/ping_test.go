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
	for _, ipv := range []IPVersion{IPv4, IPv6} {
		p, err := NewPinger(ipv)
		if err != nil {
			t.Fatal(err)
		}
		id := p.ID()
		req := &Echo{
			ID:   id,
			Seq:  uint16(src.Int63()),
			Data: []byte(fmt.Sprintf("%d", src.Int63())),
		}
		if dst, err := p.Resolve("google.com"); err != nil {
			t.Fatal(err)
		} else if err := p.Send(dst, req); err != nil {
			t.Fatal(err)
		}
		for {
			res, err := p.Receive()
			if err != nil {
				if !IsTemporary(err) {
					t.Fatal(err)
				}
				continue
			} else if res.ID != id {
				continue
			} else if !reflect.DeepEqual(req, res) {
				t.Fatalf("\ngot =%s\nwant=%s", res, req)
			}
			break
		}
	}
}
