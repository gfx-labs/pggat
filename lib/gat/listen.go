package gat

import (
	"encoding/json"
	"fmt"
	"net"

	"github.com/caddyserver/caddy/v2"
	"tuxpa.in/a/zlog/log"
)

type ListenerConfig struct {
	Network string          `json:"network"`
	Address string          `json:"address"`
	SSL     json.RawMessage `json:"ssl" caddy:"namespace=pggat.ssl.servers inline_key=provider"`
}

type Listener struct {
	ListenerConfig

	ssl SSLServer

	listener net.Listener
}

func (T *Listener) Provision(ctx caddy.Context) error {
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
	T.listener, err = net.Listen(T.Network, T.Address)
	if err != nil {
		return err
	}

	log.Printf("listening on %v", T.listener.Addr())

	return nil
}

func (T *Listener) Stop() error {
	return T.listener.Close()
}

var _ caddy.App = (*Listener)(nil)
var _ caddy.Provisioner = (*Listener)(nil)
