package matchers

import (
	"fmt"
	"net"

	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat"
)

func init() {
	caddy.RegisterModule((*LocalAddress)(nil))
}

type LocalAddress struct {
	Network string `json:"network"`
	Address string `json:"address"`

	addr net.Addr
}

func (T *LocalAddress) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: "pggat.matchers.local_address",
		New: func() caddy.Module {
			return new(LocalAddress)
		},
	}
}

func (T *LocalAddress) Provision(ctx caddy.Context) error {
	var err error
	switch T.Network {
	case "tcp", "tcp4", "tcp6":
		T.addr, err = net.ResolveTCPAddr(T.Network, T.Address)
	case "udp", "udp4", "udp6":
		T.addr, err = net.ResolveUDPAddr(T.Network, T.Address)
	case "ip", "ip4", "ip6":
		T.addr, err = net.ResolveIPAddr(T.Network, T.Address)
	case "unix", "unixgram", "unixpacket":
		T.addr, err = net.ResolveUnixAddr(T.Network, T.Address)
	default:
		err = fmt.Errorf("unknown network: %s", T.Network)
	}
	return err
}

func (T *LocalAddress) Matches(conn *fed.Conn) bool {
	switch addr := conn.LocalAddr().(type) {
	case *net.TCPAddr:
		expected, ok := T.addr.(*net.TCPAddr)
		if !ok {
			return false
		}
		return addr.Port == expected.Port && addr.Zone == expected.Zone && (expected.IP == nil || addr.IP.Equal(expected.IP))
	case *net.IPAddr:
		expected, ok := T.addr.(*net.IPAddr)
		if !ok {
			return false
		}
		return addr.Zone == expected.Zone && (expected.IP == nil || addr.IP.Equal(expected.IP))
	case *net.UDPAddr:
		expected, ok := T.addr.(*net.UDPAddr)
		if !ok {
			return false
		}
		return addr.Port == expected.Port && addr.Zone == expected.Zone && (expected.IP == nil || addr.IP.Equal(expected.IP))
	case *net.UnixAddr:
		expected, ok := T.addr.(*net.UnixAddr)
		if !ok {
			return false
		}
		return addr.Name == expected.Name && addr.Net == expected.Net
	default:
		return false
	}
}

var _ gat.Matcher = (*LocalAddress)(nil)
var _ caddy.Module = (*LocalAddress)(nil)
var _ caddy.Provisioner = (*LocalAddress)(nil)
