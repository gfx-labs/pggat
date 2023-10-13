package clientpool

import (
	"sync"

	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/bouncer/bouncers/v2"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/eqp"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/ps"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/gat/pool"
	"gfx.cafe/gfx/pggat/lib/gat/pool/recipe"
	"gfx.cafe/gfx/pggat/lib/gat/pool2"
	"gfx.cafe/gfx/pggat/lib/gat/pool2/scalingpool"
	"gfx.cafe/gfx/pggat/lib/util/slices"
)

type Pool struct {
	config Config

	servers scalingpool.Pool

	clientsByBackendKey map[fed.BackendKey]*pool2.Conn
	mu                  sync.RWMutex
}

func MakePool(config Config) Pool {
	return Pool{
		config: config,

		servers: scalingpool.MakePool(config.Config),
	}
}

func (T *Pool) AddRecipe(name string, r *recipe.Recipe) {
	T.servers.AddRecipe(name, r)
}

func (T *Pool) RemoveRecipe(name string) {
	T.servers.RemoveRecipe(name)
}

func (T *Pool) addClient(client *pool2.Conn) {
	T.servers.AddClient(client)

	T.mu.Lock()
	defer T.mu.Unlock()
	if T.clientsByBackendKey == nil {
		T.clientsByBackendKey = make(map[fed.BackendKey]*pool2.Conn)
	}
	T.clientsByBackendKey[client.Conn.BackendKey] = client
}

func (T *Pool) removeClient(client *pool2.Conn) {
	T.servers.RemoveClient(client)

	T.mu.Lock()
	defer T.mu.Unlock()
	delete(T.clientsByBackendKey, client.Conn.BackendKey)
}

func (T *Pool) syncInitialParameters(client, server *pool2.Conn) (clientErr, serverErr error) {
	clientParams := client.Conn.InitialParameters
	serverParams := server.Conn.InitialParameters

	for key, value := range clientParams {
		// skip already set params
		if serverParams[key] == value {
			p := packets.ParameterStatus{
				Key:   key.String(),
				Value: serverParams[key],
			}
			clientErr = client.Conn.WritePacket(&p)
			if clientErr != nil {
				return
			}
			continue
		}

		setServer := slices.Contains(T.config.TrackedParameters, key)

		if !setServer {
			value = serverParams[key]
		}

		p := packets.ParameterStatus{
			Key:   key.String(),
			Value: value,
		}
		clientErr = client.Conn.WritePacket(&p)
		if clientErr != nil {
			return
		}

		if !setServer {
			continue
		}

		serverErr, _ = backends.SetParameter(server.Conn, nil, key, value)
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
		clientErr = client.Conn.WritePacket(&p)
		if clientErr != nil {
			return
		}
	}

	return

}

func (T *Pool) pair(client, server *pool2.Conn) (err, serverErr error) {
	if T.config.ParameterStatusSync != pool.ParameterStatusSyncNone || T.config.ExtendedQuerySync {
		pool2.SetConnState(metrics.ConnStatePairing, client, server)
	}

	switch T.config.ParameterStatusSync {
	case pool.ParameterStatusSyncDynamic:
		err, serverErr = ps.Sync(T.config.TrackedParameters, client.Conn, server.Conn)
	case pool.ParameterStatusSyncInitial:
		err, serverErr = T.syncInitialParameters(client, server)
	}

	if err != nil || serverErr != nil {
		return
	}

	if T.config.ExtendedQuerySync {
		serverErr = eqp.Sync(client.Conn, server.Conn)
	}

	return
}

func (T *Pool) Serve(conn *fed.Conn) error {
	if T.config.ExtendedQuerySync {
		conn.Middleware = append(
			conn.Middleware,
			eqp.NewClient(),
		)
	}

	if T.config.ParameterStatusSync == pool.ParameterStatusSyncDynamic {
		conn.Middleware = append(
			conn.Middleware,
			ps.NewClient(conn.InitialParameters),
		)
	}

	client := pool2.NewConn(conn)

	T.addClient(client)
	defer T.removeClient(client)

	var err error
	var serverErr error

	var server *pool2.Conn
	defer func() {
		if server != nil {
			if serverErr != nil {
				T.servers.RemoveServer(server)
			} else {
				T.servers.Release(server)
			}
			server = nil
		}
	}()

	if !client.Conn.Ready {
		server = T.servers.Acquire(client)
		if server == nil {
			return pool2.ErrClosed
		}

		err, serverErr = T.pair(client, server)
		if serverErr != nil {
			return serverErr
		}
		if err != nil {
			return err
		}
	}

	for {
		if server != nil && T.config.ReleaseAfterTransaction {
			T.servers.Release(server)
			server = nil
		}

		var packet fed.Packet
		packet, err = client.Conn.ReadPacket(true)
		if err != nil {
			return err
		}

		if server == nil {
			server = T.servers.Acquire(client)
			if server == nil {
				return pool2.ErrClosed
			}

			err, serverErr = T.pair(client, server)
		}
		if err == nil && serverErr == nil {
			err, serverErr = bouncers.Bounce(client.Conn, server.Conn, packet)
		}

		if serverErr != nil {
			return serverErr
		} else {
			client.TransactionComplete()
			server.TransactionComplete()
		}

		if err != nil {
			return err
		}
	}
}

func (T *Pool) Cancel(key fed.BackendKey) {
	// TODO(garet)
}

func (T *Pool) Close() {
	T.servers.Close()
}
