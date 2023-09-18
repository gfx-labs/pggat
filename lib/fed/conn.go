package fed

import (
	"bufio"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"io"
	"net"
)

type Conn interface {
	ReadWriter

	Close() error
}

type netConn struct {
	conn   net.Conn
	writer bufio.Writer
	reader bufio.Reader

	headerBuf [5]byte
}

func WrapNetConn(conn net.Conn) Conn {
	c := &netConn{
		conn: conn,
	}
	c.writer.Reset(conn)
	c.reader.Reset(conn)
	return c
}

func (T *netConn) EnableSSLClient(config *tls.Config) error {
	if err := T.writer.Flush(); err != nil {
		return err
	}
	if T.reader.Buffered() > 0 {
		return errors.New("expected empty read buffer")
	}
	sslConn := tls.Client(T.conn, config)
	T.writer.Reset(sslConn)
	T.reader.Reset(sslConn)
	T.conn = sslConn
	return sslConn.Handshake()
}

func (T *netConn) EnableSSLServer(config *tls.Config) error {
	if err := T.writer.Flush(); err != nil {
		return err
	}
	if T.reader.Buffered() > 0 {
		return errors.New("expected empty read buffer")
	}
	sslConn := tls.Server(T.conn, config)
	T.writer.Reset(sslConn)
	T.reader.Reset(sslConn)
	T.conn = sslConn
	return sslConn.Handshake()
}

func (T *netConn) ReadByte() (byte, error) {
	if err := T.writer.Flush(); err != nil {
		return 0, err
	}
	return T.reader.ReadByte()
}

func (T *netConn) ReadPacket(typed bool) (Packet, error) {
	if err := T.writer.Flush(); err != nil {
		return nil, err
	}
	if typed {
		_, err := io.ReadFull(&T.reader, T.headerBuf[:])
		if err != nil {
			return nil, err
		}
	} else {
		_, err := io.ReadFull(&T.reader, T.headerBuf[1:])
		if err != nil {
			return nil, err
		}
	}

	length := binary.BigEndian.Uint32(T.headerBuf[1:])

	p := make([]byte, length+1)
	copy(p, T.headerBuf[:])

	packet := Packet(p)
	_, err := io.ReadFull(&T.reader, packet.Payload())
	if err != nil {
		return nil, err
	}
	return packet, nil
}

func (T *netConn) WriteByte(b byte) error {
	return T.writer.WriteByte(b)
}

func (T *netConn) WritePacket(packet Packet) error {
	_, err := T.writer.Write(packet.Bytes())
	return err
}

func (T *netConn) Close() error {
	if err := T.writer.Flush(); err != nil {
		return err
	}
	return T.conn.Close()
}

var _ Conn = (*netConn)(nil)
var _ SSLServer = (*netConn)(nil)
var _ SSLClient = (*netConn)(nil)
var _ io.ByteReader = (*netConn)(nil)
var _ io.ByteWriter = (*netConn)(nil)
