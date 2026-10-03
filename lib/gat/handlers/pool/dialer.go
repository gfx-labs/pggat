package pool

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"strings"
	"time"

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

// cancelTimeout bounds one forwarded CancelRequest, from dial to PostgreSQL closing the socket.
const cancelTimeout = 10 * time.Second

// ErrCancelUnconfirmed means a CancelRequest may have reached PostgreSQL without its close being seen,
// so the backend may still receive the cancel later.
var ErrCancelUnconfirmed = errors.New("cancel request not confirmed by server")

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

func (T *Dialer) dial(ctx context.Context) (net.Conn, error) {
	var d net.Dialer
	if strings.HasPrefix(T.Address, "/") {
		return d.DialContext(ctx, "unix", T.Address)
	} else {
		return d.DialContext(ctx, "tcp", T.Address)
	}
}

func (T *Dialer) Dial() (*fed.Conn, error) {
	c, err := T.dial(context.Background())
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

// Cancel forwards a CancelRequest and waits for PostgreSQL to close the connection, which it does
// after signaling the backend. It returns ErrCancelUnconfirmed if the request may have been delivered
// but that close was not observed.
func (T *Dialer) Cancel(ctx context.Context, key fed.BackendKey) error {
	ctx, cancel := context.WithTimeout(ctx, cancelTimeout)
	defer cancel()

	c, err := T.dial(ctx)
	if err != nil {
		// nothing was sent
		return err
	}
	defer func() {
		_ = c.Close()
	}()
	// The codec ignores ctx, so closing the socket is what interrupts a blocked write or read.
	stop := context.AfterFunc(ctx, func() {
		_ = c.Close()
	})
	defer stop()
	if deadline, ok := ctx.Deadline(); ok {
		if err = c.SetDeadline(deadline); err != nil {
			return err
		}
	}
	conn := fed.NewConn(netconncodec.NewCodec(c))
	if err = backends.Cancel(ctx, conn, key); err != nil {
		return errors.Join(ErrCancelUnconfirmed, err)
	}
	if err = conn.Flush(ctx); err != nil {
		return errors.Join(ErrCancelUnconfirmed, err)
	}

	// PostgreSQL signals the backend before closing, so a clean close means the cancel is pending there.
	if _, err = conn.ReadPacket(ctx, true); !errors.Is(err, io.EOF) {
		return errors.Join(ErrCancelUnconfirmed, err)
	}
	return nil
}

var _ caddy.Provisioner = (*gat.Listener)(nil)
