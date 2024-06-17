package gat

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"

	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/fed/codecs/gnetcodec"
)

type GListener struct {
	ListenerConfig

	networkAddress caddy.NetworkAddress
	ssl            SSLServer

	server *gnetcodec.Server
	open   atomic.Int64

	log *zap.Logger
}

func (T *GListener) accept() (*fed.Conn, error) {
	raw, err := T.server.Accept()
	if err != nil {
		return nil, err
	}
	return fed.NewConn(raw), nil
}

func (T *GListener) Provision(ctx caddy.Context) error {
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
	T.server = &gnetcodec.Server{}

	return T.server.Provision(ctx)
}

func (T *GListener) Start() error {
	if T.networkAddress.Network == "unix" {
		if err := os.MkdirAll(filepath.Dir(T.networkAddress.Host), 0o660); err != nil {
			return err
		}
	}
	ctx, cn := context.WithTimeout(context.Background(), 5*time.Second)
	defer cn()
	bindAddr := caddy.JoinNetworkAddress(
		T.networkAddress.Network,
		T.networkAddress.Host,
		strconv.Itoa(int(T.networkAddress.StartPort)),
	)
	err := T.server.StartServer(ctx, bindAddr)
	if err != nil {
		return err
	}
	T.log.Info("listening", zap.String("address", bindAddr))

	return nil
}

func (T *GListener) Stop() error {
	return T.server.Shutdown(context.Background())
}

var _ caddy.App = (*Listener)(nil)
var _ caddy.Provisioner = (*Listener)(nil)
