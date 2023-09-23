package gat

import (
	"errors"
	"io"

	"tuxpa.in/a/zlog/log"

	"pggat/lib/bouncer/frontends/v0"
	"pggat/lib/fed"
	"pggat/lib/gat/metrics"
	"pggat/lib/util/flip"
	"pggat/lib/util/maps"
)

type Server struct {
	modules   []Module
	providers []Provider

	keys maps.RWLocked[[8]byte, *Pool]
}

func (T *Server) AddModule(module Module) {
	T.modules = append(T.modules, module)
	if provider, ok := module.(Provider); ok {
		T.providers = append(T.providers, provider)
	}
}

func (T *Server) cancel(key [8]byte) error {
	p, ok := T.keys.Load(key)
	if !ok {
		return nil
	}

	return p.Cancel(key)
}

func (T *Server) lookupPool(user, database string) *Pool {
	for _, provider := range T.providers {
		p := provider.Lookup(user, database)
		if p != nil {
			return p
		}
	}

	return nil
}

func (T *Server) registerKey(key [8]byte, p *Pool) {
	T.keys.Store(key, p)
}

func (T *Server) unregisterKey(key [8]byte) {
	T.keys.Delete(key)
}

func (T *Server) serve(conn fed.Conn, params frontends.AcceptParams) error {
	if params.CancelKey != [8]byte{} {
		return T.cancel(params.CancelKey)
	}

	p := T.lookupPool(params.User, params.Database)
	if p == nil {
		return errPoolNotFound{
			User:     params.User,
			Database: params.Database,
		}
	}

	ctx := frontends.AuthenticateContext{
		Conn: conn,
		Options: frontends.AuthenticateOptions{
			Credentials: p.GetCredentials(),
		},
	}
	auth, err := frontends.Authenticate(&ctx)
	if err != nil {
		return err
	}

	T.registerKey(auth.BackendKey, p)
	defer T.unregisterKey(auth.BackendKey)

	return p.Serve(conn, params.InitialParameters, auth.BackendKey)
}

func (T *Server) Serve(listener Listener) error {
	raw, err := listener.Listener.Accept()
	if err != nil {
		return err
	}
	conn := fed.WrapNetConn(raw)

	go func() {
		defer func() {
			_ = conn.Close()
		}()

		ctx := frontends.AcceptContext{
			Conn:    conn,
			Options: listener.Options,
		}
		params, err2 := frontends.Accept(&ctx)
		if err2 != nil {
			log.Print("error accepting client: ", err2)
			return
		}

		err := T.serve(conn, params)
		if err != nil && !errors.Is(err, io.EOF) {
			log.Print("error serving client: ", err)
			return
		}
	}()
	return nil
}

func (T *Server) ListenAndServe() error {
	var b flip.Bank

	// TODO(garet) add listeners to bank

	return b.Wait()
}

func (T *Server) ReadMetrics(m *metrics.Server) {
	for _, provider := range T.providers {
		provider.ReadMetrics(&m.Pools)
	}
}
