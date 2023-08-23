package zap

import (
	"crypto/tls"
	"io"
	"net"
)

type Conn struct {
	conn net.Conn
	buf  [1]byte
}

func WrapNetConn(conn net.Conn) *Conn {
	return &Conn{
		conn: conn,
	}
}

func (T *Conn) EnableSSLClient(config *tls.Config) error {
	sslConn := tls.Client(T.conn, config)
	T.conn = sslConn
	return sslConn.Handshake()
}

func (T *Conn) EnableSSLServer(config *tls.Config) error {
	sslConn := tls.Server(T.conn, config)
	T.conn = sslConn
	return sslConn.Handshake()
}

func (T *Conn) ReadByte() (byte, error) {
	_, err := io.ReadFull(T.conn, T.buf[:])
	if err != nil {
		return 0, err
	}
	return T.buf[0], nil
}

func (T *Conn) Read(packet *Packet) error {
	_, err := packet.ReadFrom(T.conn)
	return err
}

func (T *Conn) ReadUntyped(packet *UntypedPacket) error {
	_, err := packet.ReadFrom(T.conn)
	return err
}

func (T *Conn) WriteByte(b byte) error {
	T.buf[0] = b
	_, err := T.conn.Write(T.buf[:])
	return err
}

func (T *Conn) Write(packet *Packet) error {
	_, err := packet.WriteTo(T.conn)
	return err
}

func (T *Conn) WriteUntyped(packet *UntypedPacket) error {
	_, err := packet.WriteTo(T.conn)
	return err
}

func (T *Conn) WriteV(packets *Packets) error {
	_, err := packets.WriteTo(T.conn)
	return err
}

func (T *Conn) Close() error {
	return T.conn.Close()
}

var _ ReadWriter = (*Conn)(nil)
