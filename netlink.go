package lldpd

import (
	"github.com/extrame/raw"
	"github.com/golang/glog"
)

type nlListener struct {
	Messages chan *linkMessage
	list     map[string]int32
}

// NewNLListener listens on rtnetlink for addition and removal
// of interfaces and inform users on the Messages channel.
func NewNLListener() *nlListener {
	l := &nlListener{
		Messages: make(chan *linkMessage, 64),
		list:     make(map[string]int32),
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
	ifi raw.Interface
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
