package pool

import (
	"sync"
	"time"

	"github.com/google/uuid"

	"pggat2/lib/auth"
	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/bouncer/bouncers/v2"
	"pggat2/lib/fed"
	"pggat2/lib/gat/metrics"
	"pggat2/lib/gat/pool/recipe"
	"pggat2/lib/util/slices"
	"pggat2/lib/util/strutil"
)

type Pool struct {
	options Options

	recipes         map[string]*recipe.Recipe
	clients         map[uuid.UUID]*Client
	clientsByKey    map[[8]byte]*Client
	servers         map[uuid.UUID]*Server
	serversByRecipe map[string][]*Server
	mu              sync.RWMutex
}

func NewPool(options Options) *Pool {
	p := &Pool{
		options: options,
	}

	if options.ServerIdleTimeout != 0 {
		go p.idleLoop()
	}

	return p
}

func (T *Pool) idlest() (server *Server, at time.Time) {
	T.mu.RLock()
	defer T.mu.RUnlock()

	for _, s := range T.servers {
		state, _, since := s.GetState()
		if state != metrics.ConnStateIdle {
			continue
		}

		if at == (time.Time{}) || since.Before(at) {
			server = s
			at = since
		}
	}

	return
}

func (T *Pool) idleLoop() {
	for {
		var wait time.Duration

		now := time.Now()
		var idlest *Server
		var idle time.Time
		for idlest, idle = T.idlest(); idlest != nil && now.Sub(idle) > T.options.ServerIdleTimeout; idlest, idle = T.idlest() {
			T.removeServer(idlest)
		}

		if idlest == nil {
			wait = T.options.ServerIdleTimeout
		} else {
			wait = idle.Add(T.options.ServerIdleTimeout).Sub(now)
		}

		time.Sleep(wait)
	}
}

func (T *Pool) GetCredentials() auth.Credentials {
	return T.options.Credentials
}

func (T *Pool) AddRecipe(name string, r *recipe.Recipe) {
	func() {
		T.mu.Lock()
		defer T.mu.Unlock()

		T.removeRecipe(name)

		if T.recipes == nil {
			T.recipes = make(map[string]*recipe.Recipe)
		}
		T.recipes[name] = r
	}()

	count := r.AllocateInitial()
	for i := 0; i < count; i++ {
		T.scaleUpL1(name, r)
	}
}

func (T *Pool) RemoveRecipe(name string) {
	T.mu.Lock()
	defer T.mu.Unlock()

	T.removeRecipe(name)
}

func (T *Pool) removeRecipe(name string) {
	r, ok := T.recipes[name]
	if !ok {
		return
	}
	delete(T.recipes, name)

	servers := T.serversByRecipe[name]
	delete(T.serversByRecipe, name)

	for _, server := range servers {
		r.Free()
		T.removeServerL1(server)
	}
}

func (T *Pool) scaleUp() {
	name, r := func() (string, *recipe.Recipe) {
		T.mu.RLock()
		defer T.mu.RUnlock()
		for name, r := range T.recipes {
			if r.Allocate() {
				return name, r
			}
		}

		return "", nil
	}()
	if r == nil {
		// no recipe to scale
		return
	}

	T.scaleUpL1(name, r)
}

func (T *Pool) scaleUpL1(name string, r *recipe.Recipe) {
	conn, params, err := r.Dial()
	if err != nil {
		// failed to dial
		r.Free()
		return
	}

	T.mu.Lock()
	defer T.mu.Unlock()
	if T.recipes[name] != r {
		// recipe was removed
		r.Free()
		return
	}

	id := T.options.Pooler.NewServer()
	server := NewServer(
		T.options,
		id,
		name,
		conn,
		params.InitialParameters,
		params.BackendKey,
	)

	if T.servers == nil {
		T.servers = make(map[uuid.UUID]*Server)
	}
	T.servers[id] = server

	if T.serversByRecipe == nil {
		T.serversByRecipe = make(map[string][]*Server)
	}
	T.serversByRecipe[name] = append(T.serversByRecipe[name], server)
}

func (T *Pool) removeServer(server *Server) {
	T.mu.Lock()
	defer T.mu.Unlock()

	T.removeServerL1(server)
}

func (T *Pool) removeServerL1(server *Server) {
	delete(T.servers, server.GetID())
	T.options.Pooler.DeleteServer(server.GetID())
	_ = server.GetConn().Close()
	if T.serversByRecipe != nil {
		T.serversByRecipe[server.GetRecipe()] = slices.Remove(T.serversByRecipe[server.GetRecipe()], server)
	}
}

func (T *Pool) acquireServer(client *Client) *Server {
	client.SetState(metrics.ConnStateAwaitingServer, uuid.Nil)

	serverID := T.options.Pooler.Acquire(client.GetID(), SyncModeNonBlocking)
	if serverID == uuid.Nil {
		// TODO(garet) can this be run on same thread and only create a goroutine if scaling is possible?
		go T.scaleUp()
		serverID = T.options.Pooler.Acquire(client.GetID(), SyncModeBlocking)
	}

	T.mu.RLock()
	defer T.mu.RUnlock()
	return T.servers[serverID]
}

func (T *Pool) releaseServer(server *Server) {
	server.SetState(metrics.ConnStateRunningResetQuery, uuid.Nil)

	if T.options.ServerResetQuery != "" {
		err := backends.QueryString(new(backends.Context), server.GetConn(), T.options.ServerResetQuery)
		if err != nil {
			T.removeServer(server)
			return
		}
	}

	server.SetState(metrics.ConnStateIdle, uuid.Nil)

	T.options.Pooler.Release(server.GetID())
}

func (T *Pool) Serve(
	conn fed.Conn,
	initialParameters map[strutil.CIString]string,
	backendKey [8]byte,
) error {
	defer func() {
		_ = conn.Close()
	}()

	client := NewClient(
		T.options,
		conn,
		initialParameters,
		backendKey,
	)

	return T.serve(client)
}

func (T *Pool) serve(client *Client) error {
	T.addClient(client)
	defer T.removeClient(client)

	var server *Server

	for {
		packet, err := client.GetConn().ReadPacket(true)
		if err != nil {
			if server != nil {
				T.releaseServer(server)
			}
			return err
		}

		var serverErr error
		if server == nil {
			server = T.acquireServer(client)

			err, serverErr = Pair(T.options, client, server)
		}
		if err == nil && serverErr == nil {
			err, serverErr = bouncers.Bounce(client.GetConn(), server.GetConn(), packet)
		}
		if serverErr != nil {
			T.removeServer(server)
			return serverErr
		} else {
			TransactionComplete(client, server)
			if T.options.ReleaseAfterTransaction {
				client.SetState(metrics.ConnStateIdle, uuid.Nil)
				go T.releaseServer(server) // TODO(garet) does this need to be a goroutine
				server = nil
			}
		}

		if err != nil {
			if server != nil {
				T.releaseServer(server)
			}
			return err
		}
	}
}

func (T *Pool) addClient(client *Client) {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.clients == nil {
		T.clients = make(map[uuid.UUID]*Client)
	}
	T.clients[client.GetID()] = client
	if T.clientsByKey == nil {
		T.clientsByKey = make(map[[8]byte]*Client)
	}
	T.clientsByKey[client.GetBackendKey()] = client
}

func (T *Pool) removeClient(client *Client) {
	T.mu.Lock()
	defer T.mu.Unlock()

	delete(T.clients, client.GetID())
	delete(T.clientsByKey, client.GetBackendKey())
}

func (T *Pool) Cancel(key [8]byte) error {
	T.mu.RLock()
	defer T.mu.RUnlock()

	client, ok := T.clientsByKey[key]
	if !ok {
		return nil
	}

	state, peer, _ := client.GetState()
	if state != metrics.ConnStateActive {
		return nil
	}

	server, ok := T.servers[peer]
	if !ok {
		return nil
	}

	r, ok := T.recipes[server.GetRecipe()]
	if !ok {
		return nil
	}

	return r.Cancel(server.GetBackendKey())
}

func (T *Pool) ReadMetrics(m *metrics.Pool) {
	T.mu.RLock()
	defer T.mu.RUnlock()

	if len(T.clients) != 0 && m.Clients == nil {
		m.Clients = make(map[uuid.UUID]metrics.Conn)
	}
	if len(T.servers) != 0 && m.Servers == nil {
		m.Servers = make(map[uuid.UUID]metrics.Conn)
	}

	for id, client := range T.clients {
		var mc metrics.Conn
		client.ReadMetrics(&mc)
		m.Clients[id] = mc
	}

	for id, server := range T.servers {
		var mc metrics.Conn
		server.ReadMetrics(&mc)
		m.Servers[id] = mc
	}
}
