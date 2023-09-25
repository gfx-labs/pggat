package gat

import (
	"errors"
	"io"
	"net"

	"tuxpa.in/a/zlog/log"

	"pggat/lib/bouncer/frontends/v0"
	"pggat/lib/fed"
	"pggat/lib/gat/metrics"
	"pggat/lib/util/beforeexit"
	"pggat/lib/util/flip"
	"pggat/lib/util/maps"
)

type Server struct {
	modules   []Module
	providers []Provider
	listeners []Listener

	keys maps.RWLocked[[8]byte, *Pool]
}

func (T *Server) AddModule(module Module) {
	T.modules = append(T.modules, module)
	if provider, ok := module.(Provider); ok {
		T.providers = append(T.providers, provider)
	}
	if listener, ok := module.(Listener); ok {
		T.listeners = append(T.listeners, listener)
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

func (T *Server) accept(raw net.Conn, acceptOptions FrontendAcceptOptions) {
	conn := fed.WrapNetConn(raw)

	defer func() {
		_ = conn.Close()
	}()

	ctx := frontends.AcceptContext{
		Conn:    conn,
		Options: acceptOptions,
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
}

func (T *Server) listenAndServe(endpoint Endpoint) error {
	listener, err := net.Listen(endpoint.Network, endpoint.Address)
	if err != nil {
		return err
	}
	if endpoint.Network == "unix" {
		beforeexit.Run(func() {
			_ = listener.Close()
		})
	}

	log.Printf("listening on %s(%s)", endpoint.Network, endpoint.Address)

	for {
		raw, err := listener.Accept()
		if err != nil {
			if errors.Is(err, net.ErrClosed) {
				break
			}
		}

		go T.accept(raw, endpoint.AcceptOptions)
	}

	return nil
}

func (T *Server) ListenAndServe() error {
	var b flip.Bank

	if len(T.listeners) > 0 {
		l := T.listeners[0]
		endpoints := l.Endpoints()
		for _, endpoint := range endpoints {
			e := endpoint
			b.Queue(func() error {
				return T.listenAndServe(e)
			})
		}
	}

	return b.Wait()
}

func (T *Server) ReadMetrics(m *metrics.Server) {
	for _, provider := range T.providers {
		provider.ReadMetrics(&m.Pools)
	}
}
