package lldpd

import (
	"github.com/extrame/raw"
	"github.com/mdlayher/lldp"
)

// InterfaceFilterFn is the function used to filter interface
// This function is called once for every interface the daemon
// can potentially listen on. It should return true if the
// daemon should listen on the interface.
type InterfaceFilterFn func(raw.Interface) bool

var defaultInterfaceFilterFn InterfaceFilterFn = func(_ raw.Interface) bool { return true }

// InterfaceFilter allows a user to filter interfaces
func InterfaceFilter(fn InterfaceFilterFn) Option {
	return func(l *LLDPD) error {
		l.filterFn = fn
		return nil
	}
}

// ReplyUnicast instructs the daemon to send lldp PDU's to the
// src mac address, instead of the lldp broadcast address
// func ReplyUnicast() Option {
// 	return func(l *LLDPD) error {
// 		l.replyUnicast = true
// 		return nil
// 	}
// }

// SourceAddress sets the ethernet source address to use
// for LLDP PDU's
func SourceAddress(fn SetSourceAddressFn) Option {
	return func(l *LLDPD) error {
		l.sourceAddress = fn
		return nil
	}
}

// PortLookupFn is the function used to respond with a different
// port description. This function is called once, on first receive
// of an LLDP PDU on a port and the reply is cached untill restart.
type PortLookupFn func(raw.Interface) string

type HandleInputFn func(*Message) (*Message, error)

type SetSourceAddressFn func(raw.Interface) ([]byte, lldp.ChassisIDSubtype)

type ErrListenFn func(err error, ifi raw.Interface)

type SuccessListenFn func(ifi raw.Interface)

func defaultSetSourceAddressFn(raw.Interface) ([]byte, lldp.ChassisIDSubtype) {
	return []byte{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad}, lldp.ChassisIDSubtypeMACAddress
}

var defaultPortLookupFn PortLookupFn = func(ifi raw.Interface) string { return ifi.Name() }

// PortLookup allows a user to use a different port description
// lookup mechanism
func PortLookup(fn PortLookupFn) Option {
	return func(l *LLDPD) error {
		l.portLookupFn = fn
		return nil
	}
}

func OnListenErr(fn ErrListenFn) Option {
	return func(l *LLDPD) error {
		l.errListenFn = fn
		return nil
	}
}

func OnListenSuccess(fn SuccessListenFn) Option {
	return func(l *LLDPD) error {
		l.successListenFn = fn
		return nil
	}
}

// Option is a functional option handler for LLDPD.
type Option func(*LLDPD) error

// SetOption runs a functional option against LLDPD.
func (p *LLDPD) SetOption(option Option) error {
	return option(p)
}

// PortLookup allows a user to use a different port description
// lookup mechanism
func HandleInput(fn HandleInputFn) Option {
	return func(l *LLDPD) error {
		l.handleInputFn = fn
		return nil
	}
}
