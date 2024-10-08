package pool

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"github.com/caddyserver/caddy/v2"

	"gfx.cafe/gfx/pggat/lib/auth"
	"gfx.cafe/gfx/pggat/lib/auth/credentials"
	"gfx.cafe/gfx/pggat/lib/bouncer"
	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/fed/codecs/netconncodec"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type Dialer struct {
	Address  string          `json:"address"`
	SSLMode  bouncer.SSLMode `json:"ssl_mode"`
	Username string          `json:"username"`
	Database string          `json:"database"`

	RawSSL        json.RawMessage   `json:"ssl,omitempty" caddy:"namespace=pggat.ssl.clients inline_key=provider"`
	RawPassword   string            `json:"password"`
	RawParameters map[string]string `json:"parameters,omitempty"`

	SSLConfig   *tls.Config                 `json:"-"`
	Credentials auth.Credentials            `json:"-"`
	Parameters  map[strutil.CIString]string `json:"-"`
}

func (T *Dialer) Provision(ctx caddy.Context) error {
	if T.RawSSL != nil {
		val, err := ctx.LoadModule(T, "RawSSL")
		if err != nil {
			return fmt.Errorf("loading ssl module: %v", err)
		}
		T.SSLConfig = val.(gat.SSLClient).ClientTLSConfig()
	}

	T.Credentials = credentials.FromString(T.Username, T.RawPassword)

	T.Parameters = make(map[strutil.CIString]string, len(T.RawParameters))
	for key, value := range T.RawParameters {
		T.Parameters[strutil.MakeCIString(key)] = value
	}

	return nil
}

func (T *Dialer) dial() (net.Conn, error) {
	if strings.HasPrefix(T.Address, "/") {
		return net.Dial("unix", T.Address)
	} else {
		return net.Dial("tcp", T.Address)
	}
}

func (T *Dialer) Dial() (*fed.Conn, error) {
	c, err := T.dial()
	if err != nil {
		return nil, err
	}
	conn := fed.NewConn(netconncodec.NewCodec(c))
	conn.User = T.Username
	conn.Database = T.Database
	err = backends.Accept(
		context.Background(),
		conn,
		T.SSLMode,
		T.SSLConfig,
		T.Username,
		T.Credentials,
		T.Database,
		T.Parameters,
	)
	if err != nil {
		return nil, err
	}
	conn.Ready = true
	return conn, nil
}

func (T *Dialer) Cancel(ctx context.Context, key fed.BackendKey) {
	c, err := T.dial()
	if err != nil {
		return
	}
	conn := fed.NewConn(netconncodec.NewCodec(c))
	defer func() {
		_ = conn.Close(ctx)
	}()
	if err = backends.Cancel(ctx, conn, key); err != nil {
		return
	}

	// wait for server to close the connection, this means that the server received it ok
	_, _ = conn.ReadPacket(ctx, true)
}

var _ caddy.Provisioner = (*gat.Listener)(nil)
