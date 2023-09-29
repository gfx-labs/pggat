package gat

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/caddyserver/caddy/v2"
	"github.com/libp2p/go-reuseport"
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/fed"
)

type ListenerConfig struct {
	Network string          `json:"network"`
	Address string          `json:"address"`
	SSL     json.RawMessage `json:"ssl,omitempty" caddy:"namespace=pggat.ssl.servers inline_key=provider"`
}

type Listener struct {
	ListenerConfig

	ssl SSLServer

	listener net.Listener

	log *zap.Logger
}

func (T *Listener) accept() (*fed.Conn, error) {
	raw, err := T.listener.Accept()
	if err != nil {
		return nil, err
	}
	return fed.NewConn(
		fed.NewNetConn(raw),
	), nil
}

func (T *Listener) Provision(ctx caddy.Context) error {
	T.log = ctx.Logger()

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
	var err error
	T.listener, err = reuseport.Listen(T.Network, T.Address)
	if err != nil {
		return err
	}

	T.log.Info("listening", zap.String("address", T.listener.Addr().String()))

	return nil
}

func (T *Listener) Stop() error {
	return T.listener.Close()
}

var _ caddy.App = (*Listener)(nil)
var _ caddy.Provisioner = (*Listener)(nil)
