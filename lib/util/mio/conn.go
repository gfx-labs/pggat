package mio

import (
	"fmt"
	"net"
	"time"
)

type Conn struct {
	out ReadWriteCloser
	in  ReadWriteCloser
}

func (T *Conn) Close() error {
	if err := T.out.Close(); err != nil {
		return err
	}
	return T.in.Close()
}

type OutwardConn struct {
	*Conn
}

func (T OutwardConn) Read(b []byte) (n int, err error) {
	return T.out.Read(b)
}

func (T OutwardConn) Write(b []byte) (n int, err error) {
	return T.in.Write(b)
}

func (T OutwardConn) LocalAddr() net.Addr {
	return ConnAddr{
		Outward: true,
		Conn:    T.Conn,
	}
}

func (T OutwardConn) RemoteAddr() net.Addr {
	return ConnAddr{
		Outward: false,
		Conn:    T.Conn,
	}
}

func (T OutwardConn) SetDeadline(t time.Time) error {
	if err := T.SetReadDeadline(t); err != nil {
		return err
	}

	return T.SetWriteDeadline(t)
}

func (T OutwardConn) SetReadDeadline(t time.Time) error {
	return T.out.SetReadDeadline(t)
}

func (T OutwardConn) SetWriteDeadline(t time.Time) error {
	return T.in.SetWriteDeadline(t)
}

type InwardConn struct {
	*Conn
}

func (T InwardConn) Read(b []byte) (n int, err error) {
	return T.in.Read(b)
}

func (T InwardConn) Write(b []byte) (n int, err error) {
	return T.out.Write(b)
}

func (T InwardConn) LocalAddr() net.Addr {
	return ConnAddr{
		Outward: false,
		Conn:    T.Conn,
	}
}

func (T InwardConn) RemoteAddr() net.Addr {
	return ConnAddr{
		Outward: true,
		Conn:    T.Conn,
	}
}

func (T InwardConn) SetDeadline(t time.Time) error {
	if err := T.SetReadDeadline(t); err != nil {
		return err
	}

	return T.SetWriteDeadline(t)
}

func (T InwardConn) SetReadDeadline(t time.Time) error {
	return T.in.SetReadDeadline(t)
}

func (T InwardConn) SetWriteDeadline(t time.Time) error {
	return T.out.SetWriteDeadline(t)
}

var _ net.Conn = OutwardConn{}
var _ net.Conn = InwardConn{}

type ConnAddr struct {
	Outward bool
	Conn    *Conn
}

func (T ConnAddr) Network() string {
	return "mio"
}

func (T ConnAddr) String() string {
	return fmt.Sprintf("memory conn(%p)", T.Conn)
}

var _ net.Addr = ConnAddr{}
