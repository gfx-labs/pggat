package gat

import (
	"github.com/google/uuid"
	"tuxpa.in/a/zlog/log"

	"pggat2/lib/auth"
	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/bouncer/bouncers/v2"
	"pggat2/lib/bouncer/frontends/v0"
	"pggat2/lib/middleware/interceptor"
	"pggat2/lib/middleware/middlewares/unterminate"
	"pggat2/lib/util/maths"
	"pggat2/lib/zap"
)

type poolRecipe struct {
	recipe  Recipe
	servers map[uuid.UUID]struct{}
}

type Pool struct {
	options PoolOptions

	recipes map[string]*poolRecipe

	servers map[uuid.UUID]zap.Conn
	clients map[uuid.UUID]zap.Conn
}

type PoolOptions struct {
	Credentials      auth.Credentials
	Pooler           Pooler
	ServerResetQuery string
}

func NewPool(options PoolOptions) *Pool {
	return &Pool{
		options: options,
	}
}

func (T *Pool) GetCredentials() auth.Credentials {
	return T.options.Credentials
}

func (T *Pool) scale(name string, amount int) {
	recipe := T.recipes[name]
	if recipe == nil {
		return
	}

	target := maths.Clamp(len(recipe.servers)+amount, recipe.recipe.MinConnections, recipe.recipe.MaxConnections)
	diff := target - len(recipe.servers)

	for diff > 0 {
		diff--

		// add server
		server, params, err := recipe.recipe.Dialer.Dial()
		if err != nil {
			log.Printf("failed to connect to server: %v", err)
			continue
		}

		_ = params // TODO(garet)

		serverID := T.addServer(server)
		if recipe.servers == nil {
			recipe.servers = make(map[uuid.UUID]struct{})
		}
		recipe.servers[serverID] = struct{}{}
	}

	for diff < 0 {
		diff++

		// remove server
		for s := range recipe.servers {
			T.removeServer(s)
			break
		}
	}
}

func (T *Pool) AddRecipe(name string, recipe Recipe) {
	if T.recipes == nil {
		T.recipes = make(map[string]*poolRecipe)
	}

	T.recipes[name] = &poolRecipe{
		recipe: recipe,
	}

	T.scale(name, 0)
}

func (T *Pool) RemoveRecipe(name string) {
	if recipe, ok := T.recipes[name]; ok {
		recipe.recipe.MaxConnections = 0
		T.scale(name, 0)
		delete(T.recipes, name)
	}
}

func (T *Pool) addClient(
	client zap.Conn,
) uuid.UUID {
	clientID := uuid.New()
	T.options.Pooler.AddClient(clientID)

	if T.clients == nil {
		T.clients = make(map[uuid.UUID]zap.Conn)
	}
	T.clients[clientID] = client
	return clientID
}

func (T *Pool) removeClient(
	clientID uuid.UUID,
) {
	T.options.Pooler.RemoveClient(clientID)
	if client, ok := T.clients[clientID]; ok {
		_ = client.Close()
		delete(T.clients, clientID)
	}
}

func (T *Pool) addServer(
	server zap.Conn,
) uuid.UUID {
	serverID := uuid.New()
	T.options.Pooler.AddServer(serverID)

	if T.servers == nil {
		T.servers = make(map[uuid.UUID]zap.Conn)
	}
	T.servers[serverID] = server
	return serverID
}

func (T *Pool) acquireServer(
	clientID uuid.UUID,
) (serverID uuid.UUID, server zap.Conn) {
	serverID = T.options.Pooler.AcquireConcurrent(clientID)
	if serverID == uuid.Nil {
		// TODO(garet) scale up
		serverID = T.options.Pooler.AcquireAsync(clientID)
	}

	server = T.servers[serverID]
	return
}

func (T *Pool) removeServer(
	serverID uuid.UUID,
) {
	T.options.Pooler.RemoveServer(serverID)
	if server, ok := T.servers[serverID]; ok {
		_ = server.Close()
		delete(T.servers, serverID)
	}
}

func (T *Pool) tryReleaseServer(
	serverID uuid.UUID,
) bool {
	if !T.options.Pooler.CanRelease(serverID) {
		return false
	}
	T.releaseServer(serverID)
	return true
}

func (T *Pool) releaseServer(
	serverID uuid.UUID,
) {
	if T.options.ServerResetQuery != "" {
		server := T.servers[serverID]
		err := backends.QueryString(new(backends.Context), server, T.options.ServerResetQuery)
		if err != nil {
			T.removeServer(serverID)
			return
		}
	}
	T.options.Pooler.Release(serverID)
}

func (T *Pool) Serve(
	client zap.Conn,
	acceptParams frontends.AcceptParams,
	authParams frontends.AuthenticateParams,
) error {
	client = interceptor.NewInterceptor(
		client,
		unterminate.Unterminate,
		// TODO(garet) add middlewares based on Pool.options
	)

	defer func() {
		_ = client.Close()
	}()

	clientID := T.addClient(client)

	var serverID uuid.UUID
	var server zap.Conn

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

		if serverID == uuid.Nil {
			serverID, server = T.acquireServer(clientID)
		}
		clientErr, serverErr := bouncers.Bounce(client, server, packet)
		if serverErr != nil {
			T.removeServer(serverID)
			serverID = uuid.Nil
			server = nil
			return serverErr
		} else {
			if T.tryReleaseServer(serverID) {
				serverID = uuid.Nil
				server = nil
			}
		}

		if clientErr != nil {
			return clientErr
		}
	}
}
