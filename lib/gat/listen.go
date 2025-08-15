package gat

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/fed/listeners/netconnlistener"
)

type ListenerConfig struct {
	Address        string          `json:"address"`
	SSL            json.RawMessage `json:"ssl,omitempty" caddy:"namespace=pggat.ssl.servers inline_key=provider"`
	MaxConnections int             `json:"max_connections,omitempty"`
}

type Listener struct {
	ListenerConfig

	networkAddress caddy.NetworkAddress
	ssl            SSLServer

	listener fed.Listener
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
		T.ssl = val.(SSLServer)
	}

	return nil
}

func (T *Listener) Start() error {
	addr := T.networkAddress
	if addr.Network == "unix" {
		if err := os.MkdirAll(filepath.Dir(addr.Host), 0o750); err != nil {
			return err
		}
	}
	listener, err := addr.Listen(context.Background(), 0, net.ListenConfig{})
	if err != nil {
		return err
	}
	if netListener, ok := listener.(net.Listener); ok {
		T.listener = &netconnlistener.Listener{Listener: netListener}
	} else if fedListener, ok := listener.(fed.Listener); ok {
		T.listener = fedListener
	}
	T.log.Info("listening", zap.String("address", T.networkAddress.String()))
	return nil
}

func (T *Listener) Stop() error {
	return T.listener.Close()
}

var _ caddy.App = (*Listener)(nil)
var _ caddy.Provisioner = (*Listener)(nil)
