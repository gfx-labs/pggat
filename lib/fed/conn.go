package fed

import (
	"crypto/tls"
	"encoding/binary"
	"io"
	"net"
)

type Conn interface {
	ReadWriter

	EnableSSLClient(config *tls.Config) error
	EnableSSLServer(config *tls.Config) error

	Close() error
}

const pktBufSize = 4096

type netConn struct {
	conn net.Conn
	w    io.Writer

	writeBuf net.Buffers

	pktBuf  [pktBufSize]byte
	readBuf []byte

	headerBuf [5]byte
}

func WrapNetConn(conn net.Conn) Conn {
	return &netConn{
		conn: conn,
		w:    conn,
	}
}

func (T *netConn) EnableSSLClient(config *tls.Config) error {
	if err := T.flush(); err != nil {
		return err
	}
	sslConn := tls.Client(T.conn, config)
	T.conn = sslConn
	T.w = sslConn
	return sslConn.Handshake()
}

func (T *netConn) EnableSSLServer(config *tls.Config) error {
	if err := T.flush(); err != nil {
		return err
	}
	sslConn := tls.Server(T.conn, config)
	T.conn = sslConn
	T.w = sslConn
	return sslConn.Handshake()
}

func (T *netConn) flush() error {
	if len(T.writeBuf) == 0 {
		return nil
	}

	_, err := T.writeBuf.WriteTo(T.w)
	T.writeBuf = T.writeBuf[0:]
	return err
}

func (T *netConn) read(buf []byte) (n int, err error) {
	for {
		if len(T.readBuf) > 0 {
			cn := copy(buf, T.readBuf)
			buf = buf[cn:]
			T.readBuf = T.readBuf[cn:]
			n += cn
		}

		if len(buf) == 0 {
			return
		}

		if len(buf) > len(T.pktBuf) {
			var rn int
			rn, err = T.conn.Read(buf)
			n += rn
			if err != nil {
				return
			}
			buf = buf[rn:]
		} else {
			var rn int
			rn, err = T.conn.Read(T.pktBuf[:])
			if err != nil {
				return
			}
			T.readBuf = T.pktBuf[:rn]
		}
	}
}

func (T *netConn) ReadByte() (byte, error) {
	if err := T.flush(); err != nil {
		return 0, err
	}
	var b [1]byte
	_, err := T.read(b[:])
	if err != nil {
		return 0, err
	}
	return b[0], nil
}

func (T *netConn) ReadPacket(typed bool) (Packet, error) {
	if err := T.flush(); err != nil {
		return nil, err
	}
	if typed {
		_, err := T.read(T.headerBuf[:])
		if err != nil {
			return nil, err
		}
	} else {
		_, err := T.read(T.headerBuf[1:])
		if err != nil {
			return nil, err
		}
	}

	length := binary.BigEndian.Uint32(T.headerBuf[1:])

	p := make([]byte, length+1)
	copy(p, T.headerBuf[:])

	packet := Packet(p)
	_, err := T.read(packet.Payload())
	if err != nil {
		return nil, err
	}
	return packet, nil
}

func (T *netConn) WriteByte(b byte) error {
	T.writeBuf = append(T.writeBuf, []byte{b})
	return nil
}

func (T *netConn) WritePacket(packet Packet) error {
	T.writeBuf = append(T.writeBuf, packet.Bytes())
	return nil
}

func (T *netConn) Close() error {
	if err := T.flush(); err != nil {
		return err
	}
	return T.conn.Close()
}

var _ Conn = (*netConn)(nil)
