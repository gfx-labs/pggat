package fed

import (
	"bufio"
	"crypto/tls"
	"encoding/binary"
	"errors"
	"io"
	"net"

	"gfx.cafe/gfx/pggat/lib/util/slices"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type Conn interface {
	ReadWriter

	LocalAddr() net.Addr
	RemoteAddr() net.Addr

	SSLEnabled() bool
	User() string
	Database() string
	InitialParameters() map[strutil.CIString]string
	BackendKey() [8]byte

	Close() error
}

type NetConn struct {
	conn       net.Conn
	writer     bufio.Writer
	reader     bufio.Reader
	sslEnabled bool

	user              string
	database          string
	initialParameters map[strutil.CIString]string
	backendKey        [8]byte

	headerBuf [5]byte
}

func WrapNetConn(conn net.Conn) *NetConn {
	c := &NetConn{
		conn: conn,
	}
	c.writer.Reset(conn)
	c.reader.Reset(conn)
	return c
}

func (T *NetConn) LocalAddr() net.Addr {
	return T.conn.LocalAddr()
}

func (T *NetConn) RemoteAddr() net.Addr {
	return T.conn.RemoteAddr()
}

func (T *NetConn) SSLEnabled() bool {
	return T.sslEnabled
}

func (T *NetConn) User() string {
	return T.user
}

func (T *NetConn) SetUser(user string) {
	T.user = user
}

func (T *NetConn) Database() string {
	return T.database
}

func (T *NetConn) SetDatabase(database string) {
	T.database = database
}

func (T *NetConn) InitialParameters() map[strutil.CIString]string {
	return T.initialParameters
}

func (T *NetConn) SetInitialParameters(initialParameters map[strutil.CIString]string) {
	T.initialParameters = initialParameters
}

func (T *NetConn) BackendKey() [8]byte {
	return T.backendKey
}

func (T *NetConn) SetBackendKey(backendKey [8]byte) {
	T.backendKey = backendKey
}

var errSSLAlreadyEnabled = errors.New("ssl is already enabled")

func (T *NetConn) EnableSSLClient(config *tls.Config) error {
	if T.sslEnabled {
		return errSSLAlreadyEnabled
	}
	T.sslEnabled = true

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

func (T *NetConn) EnableSSLServer(config *tls.Config) error {
	if T.sslEnabled {
		return errSSLAlreadyEnabled
	}
	T.sslEnabled = true

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

func (T *NetConn) ReadByte() (byte, error) {
	if err := T.writer.Flush(); err != nil {
		return 0, err
	}
	return T.reader.ReadByte()
}

func (T *NetConn) ReadPacket(typed bool, buffer Packet) (packet Packet, err error) {
	packet = buffer

	if err = T.writer.Flush(); err != nil {
		return
	}

	if typed {
		_, err = io.ReadFull(&T.reader, T.headerBuf[:])
		if err != nil {
			return
		}
	} else {
		_, err = io.ReadFull(&T.reader, T.headerBuf[1:])
		if err != nil {
			return
		}
	}

	length := binary.BigEndian.Uint32(T.headerBuf[1:])

	packet = slices.Resize(buffer, int(length)+1)
	copy(packet, T.headerBuf[:])

	_, err = io.ReadFull(&T.reader, packet.Payload())
	if err != nil {
		return
	}
	return
}

func (T *NetConn) WriteByte(b byte) error {
	return T.writer.WriteByte(b)
}

func (T *NetConn) WritePacket(packet Packet) error {
	_, err := T.writer.Write(packet.Bytes())
	return err
}

func (T *NetConn) Close() error {
	if err := T.writer.Flush(); err != nil {
		return err
	}
	return T.conn.Close()
}

var _ Conn = (*NetConn)(nil)
var _ SSLServer = (*NetConn)(nil)
var _ SSLClient = (*NetConn)(nil)
var _ io.ByteReader = (*NetConn)(nil)
var _ io.ByteWriter = (*NetConn)(nil)
