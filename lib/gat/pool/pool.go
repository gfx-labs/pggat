package pool

import (
	"sync"

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
	servers         map[uuid.UUID]*Server
	serversByRecipe map[string][]*Server
	mu              sync.RWMutex
}

func NewPool(options Options) *Pool {
	return &Pool{
		options: options,
	}
}

func (T *Pool) GetCredentials() auth.Credentials {
	return T.options.Credentials
}

func (T *Pool) AddRecipe(name string, r *recipe.Recipe) {
	T.mu.Lock()
	defer T.mu.Unlock()

	T.removeRecipe(name)

	if T.recipes == nil {
		T.recipes = make(map[string]*recipe.Recipe)
	}
	T.recipes[name] = r

	// TODO(garet) allocate servers until at the min
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
	client.SetState(StateAwaitingServer, uuid.Nil)

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
	server.SetState(StateRunningResetQuery, uuid.Nil)

	if T.options.ServerResetQuery != "" {
		err := backends.QueryString(new(backends.Context), server.GetConn(), T.options.ServerResetQuery)
		if err != nil {
			T.removeServer(server)
			return
		}
	}

	server.SetState(StateIdle, uuid.Nil)

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
				client.SetState(StateIdle, uuid.Nil)
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
}

func (T *Pool) removeClient(client *Client) {
	T.mu.Lock()
	defer T.mu.Unlock()

	delete(T.clients, client.GetID())
}

func (T *Pool) Cancel(key [8]byte) error {
	return nil // TODO(garet)
}

func (T *Pool) ReadMetrics(metrics *metrics.Pool) {
	// TODO(garet)
}
