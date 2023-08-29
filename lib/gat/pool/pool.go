package pool

import (
	"sync"

	"github.com/google/uuid"
	"tuxpa.in/a/zlog/log"

	"pggat2/lib/auth"
	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/bouncer/bouncers/v2"
	"pggat2/lib/bouncer/frontends/v0"
	"pggat2/lib/middleware"
	"pggat2/lib/middleware/interceptor"
	"pggat2/lib/middleware/middlewares/eqp"
	"pggat2/lib/middleware/middlewares/ps"
	"pggat2/lib/middleware/middlewares/unterminate"
	"pggat2/lib/util/slices"
	"pggat2/lib/util/strutil"
	"pggat2/lib/zap"
	packets "pggat2/lib/zap/packets/v3.0"
)

type poolServer struct {
	conn   zap.Conn
	accept backends.AcceptParams
	recipe string

	// middlewares
	psServer  *ps.Server
	eqpServer *eqp.Server
}

type poolRecipe struct {
	recipe Recipe
	count  int
}

type Pool struct {
	options Options

	maxServers int
	recipes    map[string]*poolRecipe
	servers    map[uuid.UUID]poolServer
	clients    map[uuid.UUID]zap.Conn
	mu         sync.Mutex
}

func NewPool(options Options) *Pool {
	return &Pool{
		options: options,
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
	}

	serverID := uuid.New()
	if T.servers == nil {
		T.servers = make(map[uuid.UUID]poolServer)
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

	T.servers[serverID] = poolServer{
		conn:   server,
		accept: params,
		recipe: name,

		psServer:  psServer,
		eqpServer: eqpServer,
	}
	T.options.Pooler.AddServer(serverID)
}

func (T *Pool) AddRecipe(name string, recipe Recipe) {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.recipes == nil {
		T.recipes = make(map[string]*poolRecipe)
	}
	T.maxServers += recipe.MaxConnections
	T.recipes[name] = &poolRecipe{
		recipe: recipe,
		count:  0,
	}

	for i := 0; i < recipe.MinConnections; i++ {
		T._scaleUpRecipe(name)
	}
}

func (T *Pool) RemoveRecipe(name string) {
	T.mu.Lock()
	defer T.mu.Unlock()

	if r, ok := T.recipes[name]; ok {
		T.maxServers -= r.count
	}
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
		if r.count < r.recipe.MaxConnections {
			T._scaleUpRecipe(name)
			return
		}
	}
}

func (T *Pool) syncInitialParameters(
	client zap.Conn,
	clientParams map[strutil.CIString]string,
	server zap.Conn,
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

		if slices.Contains(T.options.TrackedParameters, key) {
			serverErr = backends.ResetParameter(new(backends.Context), server, key)
			if serverErr != nil {
				return
			}
		} else {
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
	}

	return
}

func (T *Pool) Serve(
	client zap.Conn,
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

	clientID := T.addClient(client)

	var serverID uuid.UUID
	var server poolServer

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
		if clientErr != nil && serverErr != nil {
			clientErr, serverErr = bouncers.Bounce(client, server.conn, packet)
		}
		if serverErr != nil {
			T.removeServer(serverID)
			serverID = uuid.Nil
			server = poolServer{}
			return serverErr
		} else {
			if T.options.Pooler.ReleaseAfterTransaction() {
				T.releaseServer(serverID)
				serverID = uuid.Nil
				server = poolServer{}
			}
		}

		if clientErr != nil {
			return clientErr
		}
	}
}

func (T *Pool) addClient(client zap.Conn) uuid.UUID {
	T.mu.Lock()
	defer T.mu.Unlock()

	clientID := uuid.New()

	if T.clients == nil {
		T.clients = make(map[uuid.UUID]zap.Conn)
	}
	T.clients[clientID] = client
	T.options.Pooler.AddClient(clientID)
	return clientID
}

func (T *Pool) acquireServer(clientID uuid.UUID) (serverID uuid.UUID, server poolServer) {
	serverID = T.options.Pooler.AcquireConcurrent(clientID)
	if serverID == uuid.Nil {
		go T.scaleUp()
		serverID = T.options.Pooler.AcquireAsync(clientID)
	}

	T.mu.Lock()
	defer T.mu.Unlock()
	server = T.servers[serverID]
	return
}

func (T *Pool) releaseServer(serverID uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.options.ServerResetQuery != "" {
		server := T.servers[serverID].conn
		err := backends.QueryString(new(backends.Context), server, T.options.ServerResetQuery)
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
			r.count--
		}
	}
}

func (T *Pool) removeServer(serverID uuid.UUID) {
	T.mu.Lock()
	defer T.mu.Unlock()

	T._removeServer(serverID)
}
