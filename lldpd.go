package lldpd

import (
	"bytes"
	"net"
	"sync"

	"github.com/golang/glog"
	"github.com/pkg/errors"

	"github.com/extrame/raw"
	"github.com/mdlayher/ethernet"
	"github.com/mdlayher/lldp"
)

//sudo ip maddr add 01:80:c2:00:00:0e dev eth0
//
var LldpMulticaseAddress = []byte{0x01, 0x80, 0xc2, 0x00, 0x00, 0x0e}

// var LldpMulticaseAddress = []byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff}

// LLDPD is the server for LLDP PDU's
// It will always listen passively. This means, it will
// only send LLDP PDU's in response to a received PDU.
type LLDPD struct {
	filterFn      InterfaceFilterFn
	portLookupFn  PortLookupFn
	handleInputFn HandleInputFn
	sourceAddress SetSourceAddressFn //net.HardwareAddr
	errListenFn   ErrListenFn

	recvChannel chan *Message
	sendChannel chan *Message

	// listenersLock sync.RWMutex
	listeners sync.Map

	// log Logger
}

type packetConn struct {
	conn   *raw.Conn
	packet []byte
}

// New will return a new LLDPD server with the optional
// options configured.
func New(opts ...Option) *LLDPD {
	l := &LLDPD{
		filterFn:      defaultInterfaceFilterFn,
		portLookupFn:  defaultPortLookupFn,
		sourceAddress: defaultSetSourceAddressFn, //[]byte{0xde, 0xad, 0xbe, 0xef, 0xde, 0xad}
		recvChannel:   make(chan *Message, 64),
		sendChannel:   make(chan *Message, 64),
	}

	for _, opt := range opts {
		l.SetOption(opt)
	}

	return l
}

func (l *LLDPD) startNLLoop() {
	nl := NewNLListener()
	nl.Start()

	go func() {
		for {
			select {
			case info := <-nl.Messages:
				switch info.op {
				case IF_ADD:
					if l.filterFn(info.ifi) {
						glog.Error("start listen on ", info.ifi.Name())
						go l.ListenOn(info.ifi)
					}
				case IF_DEL:
					l.CancelListenOn(info.ifi)
				}
			}
		}
	}()
}

// ListenOn will listen on the specified interface for
// LLDP PDU's
func (l *LLDPD) ListenOn(ifi raw.Interface) {
	var err error
	if _, ok := l.listeners.Load(ifi.Name()); !ok {
		// conn, err := raw.ListenPacket(ifi, uint16(0x3), nil)
		var conn *raw.Conn
		conn, err = raw.ListenPacket(ifi, uint16(lldp.EtherType), nil)
		if err != nil {
			err = errors.Wrapf(err, "in listen on [%d]%s", ifi.Index(), ifi.Name())
			goto finish
		}

		l.listeners.Store(ifi.Name(), &packetConn{
			conn: conn,
		})

		glog.Info("msg", "started listener on interface", "ifname", ifi.Name, "ifindex", ifi.Index)

		b := make([]byte, ifi.MTU())
		for {
			var n int
			var src net.Addr
			n, src, err = conn.ReadFrom(b)
			switch err {
			case nil:
				goto handle
			// case unix.EBADF:
			// 	goto finish
			default:
				if isShouldFinishError(err) {
					goto finish
				}
				glog.Error("msg", "error read from interface", "ifname", ifi.Name, "ifindex", ifi.Index, "error", err)
				continue
			}
		handle:
			var frame ethernet.Frame
			frame.UnmarshalBinary(b[:n])
			glog.Info("lldp package ", "received ", "len ", n)
			if frame.EtherType == lldp.EtherType {
				var lldpFrame lldp.Frame
				if err = lldpFrame.UnmarshalBinary(frame.Payload); err == nil {
					l.recvChannel <- &Message{
						Frame: &lldpFrame,
						Ifi:   ifi,
						From:  src.(*raw.Addr),
						To:    &raw.Addr{HardwareAddr: frame.Destination},
					}
				} else {
					//try to minimize -4 and retry
					frame.Payload = frame.Payload[:len(frame.Payload)-4]
					if err = lldpFrame.UnmarshalBinary(frame.Payload); err == nil {
						l.recvChannel <- &Message{
							Frame: &lldpFrame,
							Ifi:   ifi,
							From:  src.(*raw.Addr),
							To:    &raw.Addr{HardwareAddr: frame.Destination},
						}
					} else {
						glog.Error(err)
					}
				}
			}
			//spew.Dump(src, err, b[:n])
		}
	}
finish:
	if err != nil {
		err = errors.Wrapf(err, "in listen on [%d]%s", ifi.Index, ifi.Name)
		if l.errListenFn != nil {
			l.errListenFn(err, ifi)
		}
	}
}

// CancelListenOn will stop listening on the interface
func (l *LLDPD) CancelListenOn(ifi raw.Interface) {
	if pconn, ok := l.listeners.Load(ifi.Name()); ok {
		pconn.(*packetConn).conn.Close()
		l.listeners.Delete(ifi.Name())
		glog.Info("msg", "closed listener on interface", "ifname", ifi.Name, "ifindex", ifi.Index)
	}
}

func (l *LLDPD) Send(msg *Message) error {
	if _, ok := l.listeners.Load(msg.Ifi.Name()); !ok {
		return errors.New("not listened on this interface")
	}
	l.sendChannel <- msg
	return nil
}

// Listen will start the main listener loop
func (l *LLDPD) Listen() error {
	l.startNLLoop()

	go func() {
		for {
			select {
			case msg := <-l.sendChannel:
				pconnRaw, ok := l.listeners.Load(msg.Ifi.Name())

				if !ok {
					continue
				}

				pconn := pconnRaw.(*packetConn)

				if msg.To == nil {
					msg.To = &raw.Addr{
						HardwareAddr: LldpMulticaseAddress,
					}
				}

				b := l.packetFor(msg)

				glog.Info("send msg ", msg, " on ", msg.Ifi.Name)

				_, err := pconn.conn.WriteTo(b, msg.From)
				if err != nil {
					glog.Error("msg", "error sending pdu out on interface", "name", msg.Ifi.Name, "index", msg.Ifi.Index, "error", err)
				}
				continue
			}
		}
	}()

	for {
		select {
		case msg := <-l.recvChannel:
			glog.Info("msg", "incoming pdu on interface", "name", msg.Ifi.Name, "index", msg.Ifi.Index)
			if resp, err := l.handleInputFn(msg); err == nil {
				if resp != nil {
					l.sendChannel <- resp
				}
			} else {
				glog.Info("msg", "respond input error", "error", err)
			}
			continue
		}
		break
	}
	close(l.sendChannel)

	return nil
}

func (l *LLDPD) packetFor(msg *Message) []byte {

	pDescr := l.portLookupFn(msg.Ifi)
	var portDescr bytes.Buffer
	portDescr.WriteString(pDescr)

	sourceAddress, _ := l.sourceAddress(msg.Ifi)

	b, err := msg.Frame.MarshalBinary()
	if err != nil {
		glog.Error("msg", "error marshalling lldp frame", "error", err)
		return nil
	}

	f := &ethernet.Frame{
		Destination: msg.To.HardwareAddr,
		Source:      sourceAddress,
		EtherType:   lldp.EtherType,
		Payload:     b,
	}
	frame, err := f.MarshalBinary()

	if err != nil {
		glog.Error("msg", "error marshalling ethernet frame", "error", err)
		return nil
	}
	return frame
}

type Message struct {
	From  *raw.Addr
	To    *raw.Addr
	Frame *lldp.Frame
	Ifi   raw.Interface
}
