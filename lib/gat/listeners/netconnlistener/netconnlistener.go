package netconnlistener

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/fed/codecs/netconncodec"
	"gfx.cafe/gfx/pggat/lib/gat/listeners"
	"gfx.cafe/gfx/pggat/lib/perror"
	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule((*Listener)(nil))
}

func (T *Listener) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID: caddy.ModuleID(listeners.WithNamespace("gnet")),
		New: func() caddy.Module {
			return new(Listener)
		},
	}
}

func (T *Listener) TLSConfig() (bool, *tls.Config) {
	if T.ssl == nil {
		return false, nil
	}
	return true, T.ssl.ServerTLSConfig()
}
func (T *Listener) Accept(fn func(*fed.Conn)) error {
	raw, err := T.listener.Accept()
	if err != nil {
		return err
	}
	fedConn := fed.NewConn(netconncodec.NewCodec(raw))
	if T.MaxConnections != 0 {
		count := T.open.Add(1)
		defer T.open.Add(-1)
		if int(count) > T.MaxConnections {
			_ = fedConn.WritePacket(
				perror.ToPacket(perror.New(
					perror.FATAL,
					perror.TooManyConnections,
					"Too many connections, sorry",
				)),
			)
			return nil
		}
		fn(fedConn)
		return nil
	}
	return nil
}

type Listener struct {
	Address        string          `json:"address"`
	SSL            json.RawMessage `json:"ssl,omitempty" caddy:"namespace=pggat.ssl.servers inline_key=provider"`
	MaxConnections int             `json:"max_connections,omitempty"`

	networkAddress caddy.NetworkAddress
	ssl            listeners.SSLServer

	listener net.Listener
	open     atomic.Int64

	log *zap.Logger
}

func (T *Listener) Provision(ctx caddy.Context) error {
	T.log = ctx.Logger()

	if strings.HasPrefix(T.Address, "/") {
		// unix address
		T.networkAddress = caddy.NetworkAddress{
			Network: "unix",
			Host:    T.Address,
		}
	} else {
		// tcp address
		host, rawPort, ok := strings.Cut(T.Address, ":")

		var port = 5432
		if ok {
			var err error
			port, err = strconv.Atoi(rawPort)
			if err != nil {
				return fmt.Errorf("parsing port: %v", err)
			}
		}

		T.networkAddress = caddy.NetworkAddress{
			Network:   "tcp",
			Host:      host,
			StartPort: uint(port),
			EndPort:   uint(port),
		}
	}

	if T.SSL != nil {
		val, err := ctx.LoadModule(T, "SSL")
		if err != nil {
			return fmt.Errorf("loading ssl module: %v", err)
		}
		T.ssl = val.(listeners.SSLServer)
	}

	return nil
}

func (T *Listener) Start() error {
	if T.networkAddress.Network == "unix" {
		if err := os.MkdirAll(filepath.Dir(T.networkAddress.Host), 0o660); err != nil {
			return err
		}
	}
	listener, err := T.networkAddress.Listen(context.Background(), 0, net.ListenConfig{})
	if err != nil {
		return err
	}
	T.listener = listener.(net.Listener)

	T.log.Info("listening", zap.String("address", T.listener.Addr().String()))

	return nil
}

func (T *Listener) Stop() error {
	return T.listener.Close()
}

var _ caddy.App = (*Listener)(nil)
var _ caddy.Provisioner = (*Listener)(nil)
var _ listeners.Listener = (*Listener)(nil)
