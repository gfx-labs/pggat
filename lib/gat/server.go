package gat

import (
	"errors"
	"io"

	"tuxpa.in/a/zlog/log"

	"gfx.cafe/gfx/pggat/lib/bouncer/frontends/v0"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/util/maps"
)

type Server struct {
	modules   []Module
	providers []Provider
	listeners []Listener
	starters  []Starter
	stoppers  []Stopper

	done chan struct{}

	keys maps.RWLocked[[8]byte, *Pool]
}

func NewServer(modules ...Module) *Server {
	server := new(Server)

	for _, module := range modules {
		server.modules = append(server.modules, module)
		if provider, ok := module.(Provider); ok {
			server.providers = append(server.providers, provider)
		}
		if listener, ok := module.(Listener); ok {
			server.listeners = append(server.listeners, listener)
		}
		if starter, ok := module.(Starter); ok {
			server.starters = append(server.starters, starter)
		}
		if stopper, ok := module.(Stopper); ok {
			server.stoppers = append(server.stoppers, stopper)
		}
	}

	return server
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

func (T *Server) ReadMetrics(m *metrics.Server) {
	for _, provider := range T.providers {
		provider.ReadMetrics(&m.Pools)
	}
}

func (T *Server) Start() error {
	for _, starter := range T.starters {
		if err := starter.Start(); err != nil {
			return err
		}
	}

	T.done = make(chan struct{})

	go T.acceptLoop()

	return nil
}

func (T *Server) acceptLoop() {
	accept := make(chan AcceptedConn)

	for _, listener := range T.listeners {
		listener.Listen(accept)
	}

	for {
		select {
		case accepted := <-accept:
			go func() {
				if err := T.serve(accepted.Conn, accepted.Params); err != nil && !errors.Is(err, io.EOF) {
					log.Printf("failed to serve client: %v", err)
				}
			}()
		case <-T.done:
			return
		}
	}
}

func (T *Server) Stop() error {
	var err error
	for _, stopper := range T.stoppers {
		if err2 := stopper.Stop(); err2 != nil {
			err = err2
		}
	}

	close(T.done)

	return err
}
