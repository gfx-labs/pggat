package zap

import (
	"crypto/tls"
	"encoding/binary"
	"io"
	"net"
)

type Conn struct {
	conn net.Conn
	w    io.Writer
	r    io.Reader

	buffers net.Buffers

	byteBuf [1]byte
}

func WrapNetConn(conn net.Conn) *Conn {
	return &Conn{
		conn: conn,
		w:    conn,
		r:    conn,
	}
}

func (T *Conn) EnableSSLClient(config *tls.Config) error {
	sslConn := tls.Client(T.conn, config)
	T.conn = sslConn
	T.w = sslConn
	T.r = sslConn
	return sslConn.Handshake()
}

func (T *Conn) EnableSSLServer(config *tls.Config) error {
	sslConn := tls.Server(T.conn, config)
	T.conn = sslConn
	T.w = sslConn
	T.r = sslConn
	return sslConn.Handshake()
}

func (T *Conn) flush() error {
	if len(T.buffers) == 0 {
		return nil
	}

	_, err := T.buffers.WriteTo(T.w)
	T.buffers = T.buffers[0:]
	return err
}

func (T *Conn) ReadByte() (byte, error) {
	if err := T.flush(); err != nil {
		return 0, err
	}
	_, err := io.ReadFull(T.r, T.byteBuf[:])
	if err != nil {
		return 0, err
	}
	return T.byteBuf[0], nil
}

func (T *Conn) ReadPacket(typed bool) (Packet, error) {
	if err := T.flush(); err != nil {
		return nil, err
	}
	packet := NewPacket(0)
	if typed {
		_, err := io.ReadFull(T.r, packet)
		if err != nil {
			return nil, err
		}
	} else {
		_, err := io.ReadFull(T.r, packet[1:])
		if err != nil {
			return nil, err
		}
	}

	length := binary.BigEndian.Uint32(packet[1:])
	packet = packet.Grow(int(length) - 4)
	_, err := io.ReadFull(T.r, packet.Payload())
	if err != nil {
		return nil, err
	}
	return packet, nil
}

func (T *Conn) WriteByte(b byte) error {
	T.buffers = append(T.buffers, []byte{b})
	return nil
}

func (T *Conn) WritePacket(packet Packet) error {
	T.buffers = append(T.buffers, packet.Bytes())
	return nil
}

func (T *Conn) Close() error {
	return T.conn.Close()
}

var _ ReadWriter = (*Conn)(nil)
