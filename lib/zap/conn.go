package zap

import (
	"crypto/tls"
	"encoding/binary"
	"io"
	"net"
)

const pktBufSize = 4096

type Conn struct {
	conn net.Conn
	w    io.Writer

	writeBuf net.Buffers

	pktBuf  [pktBufSize]byte
	readBuf []byte
}

func WrapNetConn(conn net.Conn) *Conn {
	return &Conn{
		conn: conn,
		w:    conn,
	}
}

func (T *Conn) EnableSSLClient(config *tls.Config) error {
	sslConn := tls.Client(T.conn, config)
	T.conn = sslConn
	T.w = sslConn
	return sslConn.Handshake()
}

func (T *Conn) EnableSSLServer(config *tls.Config) error {
	sslConn := tls.Server(T.conn, config)
	T.conn = sslConn
	T.w = sslConn
	return sslConn.Handshake()
}

func (T *Conn) flush() error {
	if len(T.writeBuf) == 0 {
		return nil
	}

	_, err := T.writeBuf.WriteTo(T.w)
	T.writeBuf = T.writeBuf[0:]
	return err
}

func (T *Conn) read(buf []byte) (n int, err error) {
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

func (T *Conn) ReadByte() (byte, error) {
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

func (T *Conn) ReadPacket(typed bool) (Packet, error) {
	if err := T.flush(); err != nil {
		return nil, err
	}
	packet := NewPacket(0)
	if typed {
		_, err := T.read(packet)
		if err != nil {
			return nil, err
		}
	} else {
		_, err := T.read(packet[1:])
		if err != nil {
			return nil, err
		}
	}

	length := binary.BigEndian.Uint32(packet[1:])
	packet = packet.Grow(int(length) - 4)
	_, err := T.read(packet.Payload())
	if err != nil {
		return nil, err
	}
	return packet, nil
}

func (T *Conn) WriteByte(b byte) error {
	T.writeBuf = append(T.writeBuf, []byte{b})
	return nil
}

func (T *Conn) WritePacket(packet Packet) error {
	T.writeBuf = append(T.writeBuf, packet.Bytes())
	return nil
}

func (T *Conn) Close() error {
	return T.conn.Close()
}

var _ ReadWriter = (*Conn)(nil)
