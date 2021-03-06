// +build !linux

package lldpd

import (
	"math/rand"
	"net"
	"time"

	"github.com/extrame/raw"
)

type bpf struct {
	// tap    *bpf_module.NetworkTap
	buflen int
	buf    []byte
	self   net.HardwareAddr
}

func (l *nlListener) Listen() error {
	if err := l.Search(); err != nil {
		return err
	}
	for {
		select {
		case <-time.After(2 * time.Second):
			if err := l.Search(); err != nil {
				return err
			}
		}
	}
}

func (l *nlListener) Search() error {
	var random = rand.Int31n(100000)
	if ifis, err := raw.Interfaces(); err == nil {
		for index := 0; index < len(ifis); index++ {
			var ifi = ifis[index]
			// if ifi.Flags()&net.FlagUp != 0 {
			if _, ok := l.list[ifi.Name()]; !ok {
				// if len(ifi.HardwareAddr) != 0 {
				l.Messages <- &linkMessage{
					ifi: ifi,
					op:  IF_ADD,
				}
				l.log.Info("msg", "netlink reports new interface", "ifname", ifi.Name, "ifindex", ifi.Index)
				// }
			}
			l.list[ifi.Name()] = random
			// }
		}
	} else {
		return err
	}
	for i, set := range l.list {
		if set != random {
			//is not refreshed
			if ifi, err := raw.InterfaceByName(i); err == nil {
				l.Messages <- &linkMessage{
					ifi: ifi,
					op:  IF_DEL,
				}
			} else {
				return err
			}
		}
	}
	return nil
}
