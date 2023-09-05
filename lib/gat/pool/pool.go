package pool

import (
	"pggat2/lib/util/maps"
	"sync/atomic"
	"time"

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
	count  atomic.Int64
}

type Pool struct {
	options Options

	recipes maps.RWLocked[string, *poolRecipe]
	servers maps.RWLocked[uuid.UUID, *Server]
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

func (T *Pool) idlest() (idlest uuid.UUID, idle time.Time) {
	T.servers.Range(func(serverID uuid.UUID, server *Server) bool {
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

func (T *Pool) _scaleUpRecipe(name string) {
	r, ok := T.recipes.Load(name)
	if !ok {
		return
	}

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

	r.count.Add(1)
	serverID := uuid.New()
	T.servers.Store(serverID, NewServer(
		server,
		params.BackendKey,
		params.InitialParameters,
		name,
		psServer,
		eqpServer,
	))
	T.options.Pooler.AddServer(serverID)
}

func (T *Pool) AddRecipe(name string, recipe Recipe) {
	_, hasOld := T.recipes.Swap(name, &poolRecipe{
		recipe: recipe,
	})
	if hasOld {
		T.servers.Range(func(serverID uuid.UUID, server *Server) bool {
			if server.GetRecipe() == name {
				_ = server.GetConn().Close()
				T.options.Pooler.RemoveServer(serverID)
				T.servers.Delete(serverID)
			}
			return true
		})
	}

	for i := 0; i < recipe.MinConnections; i++ {
		T._scaleUpRecipe(name)
	}
}

func (T *Pool) RemoveRecipe(name string) {
	T.recipes.Delete(name)

	// close all servers with this recipe

	T.servers.Range(func(serverID uuid.UUID, server *Server) bool {
		if server.GetRecipe() == name {
			_ = server.GetConn().Close()
			T.options.Pooler.RemoveServer(serverID)
			T.servers.Delete(serverID)
		}
		return true
	})
}

func (T *Pool) ScaleUp() {
	T.recipes.Range(func(name string, r *poolRecipe) bool {
		if r.recipe.MaxConnections == 0 || int(r.count.Load()) < r.recipe.MaxConnections {
			T._scaleUpRecipe(name)
			return false
		}

		return true
	})
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
			if T.options.Pooler.ReleaseAfterTransaction() {
				T.releaseServer(serverID)
				serverID = uuid.Nil
				server = nil
			} else {
				T.transactionComplete(serverID)
			}
		}

		if clientErr != nil {
			return clientErr
		}
	}
}

func (T *Pool) addClient(client fed.Conn, key [8]byte) uuid.UUID {
	clientID := uuid.New()

	T.clients.Store(clientID, NewClient(
		client,
		key,
	))
	T.options.Pooler.AddClient(clientID)
	return clientID
}

func (T *Pool) removeClient(clientID uuid.UUID) {
	T.clients.Delete(clientID)
	T.options.Pooler.RemoveClient(clientID)
}

func (T *Pool) acquireServer(clientID uuid.UUID) (serverID uuid.UUID, server *Server) {
	serverID = T.options.Pooler.Acquire(clientID, SyncModeNonBlocking)
	if serverID == uuid.Nil {
		go T.ScaleUp()
		serverID = T.options.Pooler.Acquire(clientID, SyncModeBlocking)
	}

	server, _ = T.servers.Load(serverID)
	client, _ := T.clients.Load(clientID)
	if server != nil {
		server.SetPeer(clientID)
	}
	if client != nil {
		client.SetPeer(serverID)
	}
	return
}

func (T *Pool) releaseServer(serverID uuid.UUID) {
	server, _ := T.servers.Load(serverID)
	if server == nil {
		return
	}

	clientID := server.SetPeer(uuid.Nil)

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
	T.options.Pooler.Release(serverID)
}

func (T *Pool) transactionComplete(serverID uuid.UUID) {

}

func (T *Pool) removeServer(serverID uuid.UUID) {
	server, _ := T.servers.LoadAndDelete(serverID)
	if server == nil {
		return
	}
	_ = server.GetConn().Close()
	T.options.Pooler.RemoveServer(serverID)
	r, _ := T.recipes.Load(server.GetRecipe())
	if r != nil {
		r.count.Add(-1)
	}
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
	var recipe string
	var serverKey [8]byte
	if T.servers.Range(func(_ uuid.UUID, server *Server) bool {
		if server.GetPeer() == clientID {
			recipe = server.GetRecipe()
			serverKey = server.GetBackendKey()
			return false
		}
		return true
	}) {
		return nil
	}

	r, _ := T.recipes.Load(recipe)
	if r == nil {
		return nil
	}

	return r.recipe.Dialer.Cancel(serverKey)
}

func (T *Pool) ReadMetrics(metrics *Metrics) {
	maps.Clear(metrics.Servers)
	maps.Clear(metrics.Clients)

	T.servers.Range(func(serverID uuid.UUID, server *Server) bool {
		var m ServerMetrics
		server.ReadMetrics(&m)
		metrics.Servers[serverID] = m
		return true
	})

	T.clients.Range(func(clientID uuid.UUID, client *Client) bool {
		var m ClientMetrics
		client.ReadMetrics(&m)
		metrics.Clients[clientID] = m
		return true
	})
}
