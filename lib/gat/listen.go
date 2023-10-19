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

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/fed"
)

type ListenerConfig struct {
	Address string          `json:"address"`
	SSL     json.RawMessage `json:"ssl,omitempty" caddy:"namespace=pggat.ssl.servers inline_key=provider"`
}

type Listener struct {
	ListenerConfig

	networkAddress caddy.NetworkAddress
	ssl            SSLServer

	listener net.Listener

	log *zap.Logger
}

func (T *Listener) accept() (*fed.Conn, error) {
	raw, err := T.listener.Accept()
	if err != nil {
		return nil, err
	}
	return fed.NewConn(raw), nil
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
