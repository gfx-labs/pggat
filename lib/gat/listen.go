package gat

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync/atomic"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/fed"
)

var networkTypes = map[string]ListenerFunc{}

type ListenerFunc func(ctx context.Context, addr caddy.NetworkAddress, config *tls.Config) (fed.Listener, error)

func RegisterNetwork(network string, getListener ListenerFunc) {
	network = strings.TrimSpace(strings.ToLower(network))

	if network == "tcp" || network == "tcp4" || network == "tcp6" ||
		network == "udp" || network == "udp4" || network == "udp6" ||
		network == "unix" || network == "unixpacket" || network == "unixgram" ||
		strings.HasPrefix("ip:", network) || strings.HasPrefix("ip4:", network) || strings.HasPrefix("ip6:", network) {
		panic("network type " + network + " is reserved")
	}

	if _, ok := networkTypes[strings.ToLower(network)]; ok {
		panic("network type " + network + " is already registered")
	}
	networkTypes[network] = getListener
}

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
	listenerFunc, ok := networkTypes[T.networkAddress.Network]
	if !ok {
		listenerFunc, ok = networkTypes["default"]
		if !ok {
			return fmt.Errorf("no default listenerFunc registered. forgot to import gfx.cafe/gfx/pggat/lib/fed/listeners/netconnlistener ?")
		}
	}
	var tlsConfig *tls.Config
	if T.ssl != nil {
		tlsConfig = T.ssl.ServerTLSConfig()
	}
	listener, err := listenerFunc(context.Background(), T.networkAddress, tlsConfig)
	if err != nil {
		return err
	}
	T.listener = listener

	T.log.Info("listening", zap.String("address", T.networkAddress.String()))

	return nil
}

func (T *Listener) Stop() error {
	return T.listener.Close()
}

var _ caddy.App = (*Listener)(nil)
var _ caddy.Provisioner = (*Listener)(nil)
