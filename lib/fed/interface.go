package fed

import (
	"crypto/tls"
	"net"

	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type Conn interface {
	Flush() error
	ReadPacket(typed bool) (Packet, error)
	WritePacket(packet Packet) error
	WriteByte(b byte) error
	ReadByte() (byte, error)
	EnableSSL(config *tls.Config, isClient bool) error
	Close() error

	User() string
	Database() string
	SetUser(string)
	SetDatabase(string)
	LocalAddr() net.Addr
	BackendKey() BackendKey
	SetBackendKey(BackendKey)
	SSL() bool

	Middleware() []Middleware
	AddMiddleware(...Middleware)
	InitialParameters() map[strutil.CIString]string
	SetInitialParameters(map[strutil.CIString]string)
	SetAuthenticated(bool)
	Authenticated() bool
	Ready() bool
	SetReady(bool)
}
