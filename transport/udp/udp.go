package udp

import (
	"errors"
	"github.com/rs/zerolog"
	"go.dedis.ch/cs438/transport"
	"net"
	"os"
	"strings"
	"sync"
	"time"
)

const bufSize = 65000

var (
	// logout is the logger configuration
	logout = zerolog.ConsoleWriter{
		Out:        os.Stdout,
		TimeFormat: time.RFC3339,
	}
)

// NewUDP returns a new udp transport implementation.
func NewUDP() transport.Transport {
	return &UDP{}
}

// UDP implements a transport layer using UDP
//
// - implements transport.Transport
type UDP struct {
	// Did not find any meaningful information to store here (for now I guess).
}

// CreateSocket implements transport.Transport
func (n *UDP) CreateSocket(address string) (transport.ClosableSocket, error) {
	defaultLevel := zerolog.DebugLevel
	if strings.TrimSpace(os.Getenv("GLOG")) == "no" {
		defaultLevel = zerolog.Disabled
	}

	addr, err := net.ResolveUDPAddr("udp", address)
	if err == nil {
		conn, err := net.ListenUDP("udp", addr)
		if err == nil {
			return &Socket{
				udpConn: conn,
				ins:     packets{},
				outs:    packets{},
				logger: zerolog.New(logout).
					Level(defaultLevel).
					With().Timestamp().Logger().
					With().Caller().Logger().
					With().Str("role", "udp").Logger(),
			}, nil
		}
	}

	return nil, err
}

// Socket implements a network socket using UDP.
//
// - implements transport.Socket
// - implements transport.ClosableSocket
type Socket struct {
	transport.ClosableSocket
	udpConn *net.UDPConn
	ins     packets
	outs    packets
	logger  zerolog.Logger
}

type SocketError struct {
	message string
}

func (e *SocketError) Error() string {
	return e.message
}

func getTime(timeout time.Duration) time.Time {
	var t time.Time
	if timeout.Seconds() > 0 {
		t = time.Now().Add(timeout)
	} else {
		t = time.Time{}
	}

	return t
}

// Close implements transport.Socket. It returns an error if already closed.
func (s *Socket) Close() error {
	return s.udpConn.Close()
}

func isTimeout(err error) bool {
	var netErr net.Error
	return (errors.As(err, &netErr) && netErr.Timeout()) || (errors.Is(err, transport.TimeoutError(0)))
}

func logAndReturnError(logger zerolog.Logger, err error) error {
	if err != nil {
		if !isTimeout(err) {
			logger.Err(err).Msg(err.Error())
		}
	}

	return err
}

func (s *Socket) sendPacket(dest *net.UDPAddr, pkt transport.Packet, timeout time.Duration) error {
	buf, err := pkt.Marshal()
	if err == nil {
		nwrite, err := s.udpConn.WriteToUDP(buf, dest)
		if err == nil {
			if nwrite != len(buf) {
				err := SocketError{message: "Sent less bytes than required for a packet!"}
				return logAndReturnError(s.logger, &err)
			}

			s.outs.add(pkt)
			return nil
		}

		if isTimeout(err) {
			return logAndReturnError(s.logger, transport.TimeoutError(timeout))
		}
	}

	return logAndReturnError(s.logger, err)
}

func (s *Socket) readPacket(timeout time.Duration) (transport.Packet, error) {
	buf := make([]byte, bufSize)
	nread, _, err := s.udpConn.ReadFromUDP(buf)
	if err == nil {
		var pkt transport.Packet
		err := pkt.Unmarshal(buf[:nread])
		if err == nil {
			s.ins.add(pkt)
			return pkt, nil
		}
	}

	if isTimeout(err) {
		err = transport.TimeoutError(timeout)
	}

	return transport.Packet{}, logAndReturnError(s.logger, err)
}

// Send implements transport.Socket
func (s *Socket) Send(dest string, pkt transport.Packet, timeout time.Duration) error {
	addr, err := net.ResolveUDPAddr("udp", dest)
	if err == nil {
		err = s.udpConn.SetWriteDeadline(getTime(timeout))
		if err == nil {
			//s.logger.Info().Msgf("SEND:"+
			//	"pkt: %v\n", pkt.Msg.Type,
			//	"to: %s\n", pkt.Header.Destination)
			return s.sendPacket(addr, pkt, timeout)
		}
	}

	return logAndReturnError(s.logger, err)
}

// Recv implements transport.Socket. It blocks until a packet is received, or
// the timeout is reached. In the case the timeout is reached, return a
// TimeoutErr.
func (s *Socket) Recv(timeout time.Duration) (transport.Packet, error) {
	err := s.udpConn.SetReadDeadline(getTime(timeout))
	if err == nil {
		return s.readPacket(timeout)
	}

	return transport.Packet{}, logAndReturnError(s.logger, err)
}

// GetAddress implements transport.Socket. It returns the address assigned. Can
// be useful in the case one provided a :0 address, which makes the system use a
// random free port.
func (s *Socket) GetAddress() string {
	return s.udpConn.LocalAddr().String()
}

// GetIns implements transport.Socket
func (s *Socket) GetIns() []transport.Packet {
	return s.ins.getAll()
}

// GetOuts implements transport.Socket
func (s *Socket) GetOuts() []transport.Packet {
	return s.outs.getAll()
}

type packets struct {
	sync.Mutex
	data []transport.Packet
}

func (p *packets) add(pkt transport.Packet) {
	p.Lock()
	defer p.Unlock()

	p.data = append(p.data, pkt.Copy())
}

func (p *packets) getAll() []transport.Packet {
	p.Lock()
	defer p.Unlock()

	res := make([]transport.Packet, len(p.data))

	for i, pkt := range p.data {
		res[i] = pkt.Copy()
	}

	return res
}
