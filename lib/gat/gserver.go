package gat

import (
	"crypto/tls"
	"errors"
	"io"

	"gfx.cafe/gfx/pggat/lib/bouncer/frontends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gfed"
	"gfx.cafe/gfx/pggat/lib/perror"
	"go.uber.org/zap"
)

func (T *Server) gaccept(listener *GListener, conn *gfed.Codec) {
	defer func() {
		_ = conn.Close()
	}()

	var tlsConfig *tls.Config
	if listener.ssl != nil {
		tlsConfig = listener.ssl.ServerTLSConfig()
	}

	var cancelKey fed.BackendKey
	var isCanceling bool
	var err error
	cancelKey, isCanceling, err = frontends.Accept(conn, tlsConfig)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			T.log.Warn("error accepting client", zap.Error(err))
		}
		return
	}

	if isCanceling {
		T.Cancel(cancelKey)
		return
	}

	count := listener.open.Add(1)
	defer listener.open.Add(-1)

	if listener.MaxConnections != 0 && int(count) > listener.MaxConnections {
		_ = conn.WritePacket(
			perror.ToPacket(perror.New(
				perror.FATAL,
				perror.TooManyConnections,
				"Too many connections, sorry",
			)),
		)
		return
	}

	T.Serve(conn)
}
