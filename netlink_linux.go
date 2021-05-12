package lldpd

import (
	"net"
	"syscall"

	"github.com/extrame/raw"

	"github.com/jsimonetti/rtnetlink"
	"github.com/mdlayher/netlink"
	"github.com/pkg/errors"
)

// Listen will start the listener loop
func (l *nlListener) Listen() error {
	nl, err := rtnetlink.Dial(&netlink.Config{Groups: rtnetlink.RTNLGRP_LINK})
	if err != nil {
		errors.Wrap(err, "could not dial rtnetlink")
	}

	//send request for current list of interfaces
	req := &rtnetlink.LinkMessage{}
	nl.Send(req, rtnetlink.RTM_GETLINK, netlink.HeaderFlagsRequest|netlink.HeaderFlagsDump)

	for {
		msgs, omsgs, err := nl.Receive()
		if err != nil {
			return errors.Wrap(err, "netlink receive error")
		}

		for i, msg := range msgs {
			if m, ok := msg.(*rtnetlink.LinkMessage); ok {
				if m.Type != syscall.ARPHRD_ETHER {
					// skip non-ethernet
					continue
				}

				if m.Family != syscall.AF_UNSPEC {
					// skip non-generic
					continue
				}

				if omsgs[i].Header.Type == rtnetlink.RTM_NEWLINK {
					if _, ok := l.list[m.Attributes.Name]; !ok {

						link, _ := net.InterfaceByIndex(int(m.Index))
						l.Messages <- &linkMessage{
							ifi: raw.NewInterface(link),
							op:  IF_ADD,
						}

						l.list[m.Attributes.Name] = 0
						l.log.Info("msg", "netlink reports new interface", "ifname", m.Attributes.Name, "ifindex", m.Index)
					}
					continue
				}
				if omsgs[i].Header.Type == rtnetlink.RTM_DELLINK {
					if _, ok := l.list[m.Attributes.Name]; ok {

						l.Messages <- &linkMessage{
							ifi: raw.NewInterfaceDelegate(int(m.Index), m.Attributes.Name),
							op:  IF_DEL,
						}

						delete(l.list, m.Attributes.Name)
						l.log.Info("msg", "netlink reports deleted interface", "ifname", m.Attributes.Name, "ifindex", m.Index)
					}
					continue
				}
			}
		}
	}
}
