package pool

import (
	"sync"
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

type poolServer struct {
	conn   fed.Conn
	accept backends.AcceptParams
	recipe string

	// middlewares
	psServer  *ps.Server
	eqpServer *eqp.Server

	metrics ServerMetrics
	mu      sync.Mutex
}

type poolRecipe struct {
	recipe Recipe
	count  atomic.Int64
}

type poolClient struct {
	conn fed.Conn
	key  [8]byte

	metrics ClientMetrics
	mu      sync.Mutex
}

type Pool struct {
	options Options

	recipes map[string]*poolRecipe
	servers map[uuid.UUID]*poolServer
	clients map[uuid.UUID]*poolClient
	mu      sync.Mutex
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
	T.mu.Lock()
	defer T.mu.Unlock()

	for serverID, server := range T.servers {
		func() {
			server.mu.Lock()
			defer server.mu.Unlock()

			if server.metrics.Peer != uuid.Nil {
				return
			}

			if idle != (time.Time{}) && server.metrics.Since.After(idle) {
				return
			}

			idlest = serverID
			idle = server.metrics.Since
		}()
	}

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
	r := T.recipes[name]

	server, params, err := r.recipe.Dialer.Dial()
	if err != nil {
		log.Printf("failed to dial server: %v", err)
		return
	}

	serverID := uuid.New()
	if T.servers == nil {
		T.servers = make(map[uuid.UUID]*poolServer)
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

	T.servers[serverID] = &poolServer{
		conn:   server,
		accept: params,
		recipe: name,

		psServer:  psServer,
		eqpServer: eqpServer,

		metrics: MakeServerMetrics(),
	}
	T.options.Pooler.AddServer(serverID)
}

func (T *Pool) AddRecipe(name string, recipe Recipe) {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.recipes == nil {
		T.recipes = make(map[string]*poolRecipe)
	}
	T.recipes[name] = &poolRecipe{
		recipe: recipe,
	}

	for i := 0; i < recipe.MinConnections; i++ {
		T._scaleUpRecipe(name)
	}
}

func (T *Pool) RemoveRecipe(name string) {
	T.mu.Lock()
	defer T.mu.Unlock()

	delete(T.recipes, name)

	// close all servers with this recipe
	for id, server := range T.servers {
		if server.recipe == name {
			_ = server.conn.Close()
			T.options.Pooler.RemoveServer(id)
			delete(T.servers, id)
		}
	}
}

func (T *Pool) scaleUp() {
	T.mu.Lock()
	defer T.mu.Unlock()

	for name, r := range T.recipes {
		if r.recipe.MaxConnections == 0 || int(r.count.Load()) < r.recipe.MaxConnections {
			T._scaleUpRecipe(name)
			return
		}
	}

	log.Println("warning: tried to scale up pool but no space was available")
}

func (T *Pool) syncInitialParameters(
	client fed.Conn,
	clientParams map[strutil.CIString]string,
	server fed.Conn,
	serverParams map[strutil.CIString]string,
) (clientErr, serverErr error) {
	for key, value := range clientParams {
		setServer := slices.Contains(T.options.TrackedParameters, key)

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
	var server *poolServer

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
				clientErr, serverErr = ps.Sync(T.options.TrackedParameters, client, psClient, server.conn, server.psServer)
			case ParameterStatusSyncInitial:
				clientErr, serverErr = T.syncInitialParameters(client, accept.InitialParameters, server.conn, server.accept.InitialParameters)
			}

			if T.options.ExtendedQuerySync {
				server.eqpServer.SetClient(eqpClient)
			}
		}
		if clientErr == nil && serverErr == nil {
			clientErr, serverErr = bouncers.Bounce(client, server.conn, packet)
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
			}
		}

		if clientErr != nil {
			return clientErr
		}
	}
}

func (T *Pool) addClient(client fed.Conn, key [8]byte) uuid.UUID {
	T.mu.Lock()
	defer T.mu.Unlock()

	clientID := uuid.New()

	if T.clients == nil {
		T.clients = make(map[uuid.UUID]*poolClient)
	}
	T.clients[clientID] = &poolClient{
		conn: client,
		key:  key,

		metrics: MakeClientMetrics(),
	}
	T.options.Pooler.AddClient(clientID)
	return clientID
}

func (T *Pool) removeClient(clientID uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()

	delete(T.clients, clientID)
	T.options.Pooler.RemoveClient(clientID)
}

func (T *Pool) acquireServer(clientID uuid.UUID) (serverID uuid.UUID, server *poolServer) {
	serverID = T.options.Pooler.Acquire(clientID, SyncModeNonBlocking)
	if serverID == uuid.Nil {
		go T.scaleUp()
		serverID = T.options.Pooler.Acquire(clientID, SyncModeBlocking)
	}

	T.mu.Lock()
	defer T.mu.Unlock()
	server = T.servers[serverID]
	client := T.clients[clientID]
	if server != nil {
		server.mu.Lock()
		defer server.mu.Unlock()
		server.metrics.SetPeer(clientID)
	}
	if client != nil {
		client.mu.Lock()
		defer client.mu.Unlock()
		client.metrics.SetPeer(serverID)
	}
	return
}

func (T *Pool) releaseServer(serverID uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()

	server := T.servers[serverID]
	if server == nil {
		return
	}

	var clientID uuid.UUID

	func() {
		server.mu.Lock()
		defer server.mu.Unlock()
		clientID = server.metrics.Peer
		server.metrics.SetPeer(uuid.Nil)
	}()

	if clientID != uuid.Nil {
		client := T.clients[clientID]
		if client != nil {
			func() {
				client.mu.Lock()
				defer client.mu.Unlock()
				client.metrics.SetPeer(uuid.Nil)
			}()
		}
	}

	if T.options.ServerResetQuery != "" {
		err := backends.QueryString(new(backends.Context), server.conn, T.options.ServerResetQuery)
		if err != nil {
			T._removeServer(serverID)
			return
		}
	}
	T.options.Pooler.Release(serverID)
}

func (T *Pool) _removeServer(serverID uuid.UUID) {
	if server, ok := T.servers[serverID]; ok {
		_ = server.conn.Close()
		delete(T.servers, serverID)
		T.options.Pooler.RemoveServer(serverID)
		r := T.recipes[server.recipe]
		if r != nil {
			r.count.Add(-1)
		}
	}
}

func (T *Pool) removeServer(serverID uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()

	T._removeServer(serverID)
}

func (T *Pool) Cancel(key [8]byte) error {
	dialer, backendKey := func() (Dialer, [8]byte) {
		T.mu.Lock()
		defer T.mu.Unlock()

		var clientID uuid.UUID
		for id, client := range T.clients {
			if client.key == key {
				clientID = id
				break
			}
		}

		if clientID == uuid.Nil {
			return nil, [8]byte{}
		}

		// get peer
		var recipe string
		var serverKey [8]byte
		var ok bool
		for _, server := range T.servers {
			func() {
				server.mu.Lock()
				defer server.mu.Unlock()

				if server.metrics.Peer == clientID {
					recipe = server.recipe
					serverKey = server.accept.BackendKey
					ok = true
					return
				}
			}()
			if ok {
				break
			}
		}

		if !ok {
			return nil, [8]byte{}
		}

		r, ok := T.recipes[recipe]
		if !ok {
			return nil, [8]byte{}
		}

		return r.recipe.Dialer, serverKey
	}()

	if dialer == nil {
		return nil
	}

	return dialer.Cancel(backendKey)
}
