package lldpd

import (
	"net"

	"github.com/golang/glog"
)

type nlListener struct {
	Messages chan *linkMessage
	list     map[uint32]int32
}

// NewNLListener listens on rtnetlink for addition and removal
// of interfaces and inform users on the Messages channel.
func NewNLListener() *nlListener {
	l := &nlListener{
		Messages: make(chan *linkMessage, 64),
		list:     make(map[uint32]int32),
	}
	return l
}

// Start will start the listener
func (l *nlListener) Start() {
	go func() {
		err := l.Listen()
		if err != nil {
			glog.Error("msg", "could not listen", "error", err)
		}
	}()
}

type linkMessage struct {
	ifi *net.Interface
	op  linkOp
}

type linkOp uint8

const (
	IF_ADD linkOp = 1
	IF_DEL linkOp = 2
)

func (l linkOp) String() string {
	switch l {
	case IF_ADD:
		return "ADD"
	case IF_DEL:
		return "DEL"
	default:
		return "UNKNOWN"
	}
}
