package pool

import (
	"sync"
	"time"

	"pggat2/lib/util/maps"

	"github.com/google/uuid"
	"tuxpa.in/a/zlog/log"

	"pggat2/lib/auth"
	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/bouncer/bouncers/v2"
	"pggat2/lib/bouncer/frontends/v0"
	"pggat2/lib/fed"
	packets "pggat2/lib/fed/packets/v3.0"
	"pggat2/lib/middleware"
	"pggat2/lib/middleware/interceptor"
	"pggat2/lib/middleware/middlewares/eqp"
	"pggat2/lib/middleware/middlewares/ps"
	"pggat2/lib/middleware/middlewares/unterminate"
	"pggat2/lib/util/slices"
	"pggat2/lib/util/strutil"
)

type poolRecipe struct {
	recipe Recipe

	deleted bool
	servers map[uuid.UUID]*Server
	mu      sync.RWMutex
}

func (T *poolRecipe) AddServer(serverID uuid.UUID, server *Server) bool {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.deleted {
		return false
	}

	if T.recipe.MaxConnections != 0 && len(T.servers)+1 > T.recipe.MaxConnections {
		return false
	}

	if T.servers == nil {
		T.servers = make(map[uuid.UUID]*Server)
	}
	T.servers[serverID] = server
	return true
}

func (T *poolRecipe) GetServer(serverID uuid.UUID) *Server {
	T.mu.RLock()
	defer T.mu.RUnlock()

	if T.deleted {
		return nil
	}

	return T.servers[serverID]
}

func (T *poolRecipe) DeleteServer(serverID uuid.UUID) *Server {
	T.mu.RLock()
	defer T.mu.RUnlock()

	if T.deleted {
		return nil
	}

	server := T.servers[serverID]
	delete(T.servers, serverID)
	return server
}

func (T *poolRecipe) Size() int {
	T.mu.RLock()
	defer T.mu.RUnlock()

	return len(T.servers)
}

func (T *poolRecipe) RangeRLock(fn func(serverID uuid.UUID, server *Server) bool) bool {
	T.mu.RLock()
	defer T.mu.RUnlock()

	for serverID, server := range T.servers {
		if !fn(serverID, server) {
			return false
		}
	}

	return true
}

func (T *poolRecipe) Delete(fn func(serverID uuid.UUID, server *Server)) {
	T.mu.Lock()
	defer T.mu.Unlock()

	T.deleted = true
	for serverID, server := range T.servers {
		fn(serverID, server)
		delete(T.servers, serverID)
	}
}

type Pool struct {
	options Options

	recipes maps.RWLocked[string, *poolRecipe]
	servers maps.RWLocked[uuid.UUID, *poolRecipe]
	clients maps.RWLocked[uuid.UUID, *Client]
}

func NewPool(options Options) *Pool {
	p := &Pool{
		options: options,
	}

	if options.ServerIdleTimeout != 0 {
		go p.idleTimeoutLoop()
	}

	return p
}

func (T *Pool) GetServer(serverID uuid.UUID) *Server {
	recipe, _ := T.servers.Load(serverID)
	if recipe == nil {
		return nil
	}
	return recipe.GetServer(serverID)
}

func (T *Pool) idlest() (idlest uuid.UUID, idle time.Time) {
	T.recipes.Range(func(_ string, recipe *poolRecipe) bool {
		recipe.RangeRLock(func(serverID uuid.UUID, server *Server) bool {
			peer, since := server.GetConnection()
			if peer != uuid.Nil {
				return true
			}

			if idle != (time.Time{}) && since.After(idle) {
				return true
			}

			idlest = serverID
			idle = since
			return true
		})
		return true
	})

	return
}

func (T *Pool) idleTimeoutLoop() {
	for {
		var wait time.Duration

		now := time.Now()
		var idlest uuid.UUID
		var idle time.Time
		for idlest, idle = T.idlest(); idlest != uuid.Nil && now.Sub(idle) > T.options.ServerIdleTimeout; idlest, idle = T.idlest() {
			T.removeServer(idlest)
		}

		if idlest == uuid.Nil {
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

func (T *Pool) scaleUpRecipe(r *poolRecipe) {
	server, params, err := r.recipe.Dialer.Dial()
	if err != nil {
		log.Printf("failed to dial server: %v", err)
		return
	}

	var middlewares []middleware.Middleware

	var psServer *ps.Server
	if T.options.ParameterStatusSync == ParameterStatusSyncDynamic {
		// add ps middleware
		psServer = ps.NewServer(params.InitialParameters)
		middlewares = append(middlewares, psServer)
	}

	var eqpServer *eqp.Server
	if T.options.ExtendedQuerySync {
		// add eqp middleware
		eqpServer = eqp.NewServer()
		middlewares = append(middlewares, eqpServer)
	}

	if len(middlewares) > 0 {
		server = interceptor.NewInterceptor(
			server,
			middlewares...,
		)
	}

	serverID := T.options.Pooler.NewServer()
	ok := r.AddServer(serverID, NewServer(
		server,
		params.BackendKey,
		params.InitialParameters,

		psServer,
		eqpServer,
	))
	if !ok {
		_ = server.Close()
		T.options.Pooler.DeleteServer(serverID)
		return
	}
	T.servers.Store(serverID, r)
}

func (T *Pool) AddRecipe(name string, recipe Recipe) {
	r := &poolRecipe{
		recipe: recipe,
	}
	old, _ := T.recipes.Swap(name, r)
	if old != nil {
		old.Delete(func(serverID uuid.UUID, server *Server) {
			_ = server.GetConn().Close()
			T.options.Pooler.DeleteServer(serverID)
			T.servers.Delete(serverID)
		})
	}

	for i := 0; i < recipe.MinConnections; i++ {
		T.scaleUpRecipe(r)
	}
}

func (T *Pool) RemoveRecipe(name string) {
	old, _ := T.recipes.LoadAndDelete(name)

	if old == nil {
		return
	}

	// close all servers with this recipe

	old.Delete(func(serverID uuid.UUID, server *Server) {
		_ = server.GetConn().Close()
		T.options.Pooler.DeleteServer(serverID)
		T.servers.Delete(serverID)
	})
}

func (T *Pool) ScaleUp() {
	failed := T.recipes.Range(func(_ string, r *poolRecipe) bool {
		// this can race, but it will just dial an extra server and disconnect it in worst case
		if r.recipe.MaxConnections == 0 || r.Size() < r.recipe.MaxConnections {
			T.scaleUpRecipe(r)
			return false
		}

		return true
	})
	if failed {
		log.Println("No available recipe found to scale up")
	}
}

func syncInitialParameters(
	trackedParameters []strutil.CIString,
	client fed.Conn,
	clientParams map[strutil.CIString]string,
	server fed.Conn,
	serverParams map[strutil.CIString]string,
) (clientErr, serverErr error) {
	for key, value := range clientParams {
		setServer := slices.Contains(trackedParameters, key)

		// skip already set params
		if serverParams[key] == value {
			setServer = false
		} else if !setServer {
			value = serverParams[key]
		}

		p := packets.ParameterStatus{
			Key:   key.String(),
			Value: serverParams[key],
		}
		clientErr = client.WritePacket(p.IntoPacket())
		if clientErr != nil {
			return
		}

		if !setServer {
			continue
		}

		serverErr = backends.SetParameter(new(backends.Context), server, key, value)
		if serverErr != nil {
			return
		}
	}

	for key, value := range serverParams {
		if _, ok := clientParams[key]; ok {
			continue
		}

		// Don't need to run reset on server because it will reset it to the initial value

		// send to client
		p := packets.ParameterStatus{
			Key:   key.String(),
			Value: value,
		}
		clientErr = client.WritePacket(p.IntoPacket())
		if clientErr != nil {
			return
		}
	}

	return
}

func (T *Pool) Serve(
	client fed.Conn,
	accept frontends.AcceptParams,
	auth frontends.AuthenticateParams,
) error {
	defer func() {
		_ = client.Close()
	}()

	middlewares := []middleware.Middleware{
		unterminate.Unterminate,
	}

	var psClient *ps.Client
	if T.options.ParameterStatusSync == ParameterStatusSyncDynamic {
		// add ps middleware
		psClient = ps.NewClient(accept.InitialParameters)
		middlewares = append(middlewares, psClient)
	}

	var eqpClient *eqp.Client
	if T.options.ExtendedQuerySync {
		// add eqp middleware
		eqpClient = eqp.NewClient()
		middlewares = append(middlewares, eqpClient)
	}

	client = interceptor.NewInterceptor(
		client,
		middlewares...,
	)

	clientID := T.addClient(client, auth.BackendKey)
	defer T.removeClient(clientID)

	var serverID uuid.UUID
	var server *Server

	defer func() {
		if serverID != uuid.Nil {
			T.releaseServer(serverID)
		}
	}()

	for {
		packet, err := client.ReadPacket(true)
		if err != nil {
			return err
		}

		var clientErr, serverErr error
		if serverID == uuid.Nil {
			serverID, server = T.acquireServer(clientID)

			switch T.options.ParameterStatusSync {
			case ParameterStatusSyncDynamic:
				clientErr, serverErr = ps.Sync(T.options.TrackedParameters, client, psClient, server.GetConn(), server.GetPSServer())
			case ParameterStatusSyncInitial:
				clientErr, serverErr = syncInitialParameters(T.options.TrackedParameters, client, accept.InitialParameters, server.GetConn(), server.GetInitialParameters())
			}

			if T.options.ExtendedQuerySync {
				server.GetEQPServer().SetClient(eqpClient)
			}
		}
		if clientErr == nil && serverErr == nil {
			clientErr, serverErr = bouncers.Bounce(client, server.GetConn(), packet)
		}
		if serverErr != nil {
			T.removeServer(serverID)
			serverID = uuid.Nil
			server = nil
			return serverErr
		} else {
			T.transactionComplete(clientID, serverID)
			if T.options.Pooler.ReleaseAfterTransaction() {
				T.releaseServer(serverID)
				serverID = uuid.Nil
				server = nil
			}
		}

		if clientErr != nil {
			return clientErr
		}
	}
}

func (T *Pool) addClient(client fed.Conn, key [8]byte) uuid.UUID {
	clientID := T.options.Pooler.NewClient()

	T.clients.Store(clientID, NewClient(
		client,
		key,
	))
	return clientID
}

func (T *Pool) removeClient(clientID uuid.UUID) {
	T.clients.Delete(clientID)
	T.options.Pooler.DeleteClient(clientID)
}

func (T *Pool) acquireServer(clientID uuid.UUID) (serverID uuid.UUID, server *Server) {
	client, _ := T.clients.Load(clientID)
	if client != nil {
		client.SetPeer(Stalling)
	}

	serverID = T.options.Pooler.Acquire(clientID, SyncModeNonBlocking)
	if serverID == uuid.Nil {
		go T.ScaleUp()
		serverID = T.options.Pooler.Acquire(clientID, SyncModeBlocking)
	}

	server = T.GetServer(serverID)
	if server != nil {
		server.SetPeer(clientID)
	}
	if client != nil {
		client.SetPeer(serverID)
	}
	return
}

func (T *Pool) releaseServer(serverID uuid.UUID) {
	server := T.GetServer(serverID)
	if server == nil {
		return
	}

	clientID := server.SetPeer(Stalling)

	if clientID != uuid.Nil {
		client, _ := T.clients.Load(clientID)
		if client != nil {
			client.SetPeer(uuid.Nil)
		}
	}

	if T.options.ServerResetQuery != "" {
		err := backends.QueryString(new(backends.Context), server.GetConn(), T.options.ServerResetQuery)
		if err != nil {
			T.removeServer(serverID)
			return
		}
	}

	server.SetPeer(uuid.Nil)

	T.options.Pooler.Release(serverID)
}

func (T *Pool) transactionComplete(clientID, serverID uuid.UUID) {
	func() {
		server := T.GetServer(serverID)
		if server == nil {
			return
		}

		server.TransactionComplete()
	}()

	client, _ := T.clients.Load(clientID)
	if client == nil {
		return
	}

	client.TransactionComplete()
}

func (T *Pool) removeServer(serverID uuid.UUID) {
	recipe, _ := T.servers.LoadAndDelete(serverID)
	if recipe == nil {
		return
	}
	server := recipe.DeleteServer(serverID)
	T.options.Pooler.DeleteServer(serverID)
	if server == nil {
		return
	}
	_ = server.GetConn().Close()
}

func (T *Pool) Cancel(key [8]byte) error {
	var clientID uuid.UUID
	T.clients.Range(func(id uuid.UUID, client *Client) bool {
		if client.GetBackendKey() == key {
			clientID = id
			return false
		}
		return true
	})

	if clientID == uuid.Nil {
		return nil
	}

	// get peer
	var r *poolRecipe
	var serverKey [8]byte
	if T.recipes.Range(func(_ string, recipe *poolRecipe) bool {
		return recipe.RangeRLock(func(_ uuid.UUID, server *Server) bool {
			if server.GetPeer() == clientID {
				r = recipe
				serverKey = server.GetBackendKey()
				return false
			}
			return true
		})
	}) {
		return nil
	}

	return r.recipe.Dialer.Cancel(serverKey)
}

func (T *Pool) ReadMetrics(metrics *Metrics) {
	if metrics.Servers == nil {
		metrics.Servers = make(map[uuid.UUID]ItemMetrics)
	}
	if metrics.Clients == nil {
		metrics.Clients = make(map[uuid.UUID]ItemMetrics)
	}

	T.recipes.Range(func(_ string, recipe *poolRecipe) bool {
		recipe.RangeRLock(func(serverID uuid.UUID, server *Server) bool {
			var m ItemMetrics
			server.ReadMetrics(&m)
			metrics.Servers[serverID] = m
			return true
		})
		return true
	})

	T.clients.Range(func(clientID uuid.UUID, client *Client) bool {
		var m ItemMetrics
		client.ReadMetrics(&m)
		metrics.Clients[clientID] = m
		return true
	})
}
