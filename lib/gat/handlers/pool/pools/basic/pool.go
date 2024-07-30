package basic

import (
	"context"
	"fmt"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/tracing"
	"sync"

	"github.com/google/uuid"

	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/bouncer/bouncers/v2"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/eqp"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/ps"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/spool"
	"gfx.cafe/gfx/pggat/lib/gat/metrics"
	"gfx.cafe/gfx/pggat/lib/util/slices"
)

type Pool struct {
	config Config

	servers spool.Pool

	clients map[fed.BackendKey]*Client
	mu      sync.RWMutex
}

func NewPool(config Config) *Pool {
	p := &Pool{
		config:  config,
		servers: spool.MakePool(config.Spool()),
	}
	go p.servers.ScaleLoop()
	return p
}

func (T *Pool) AddRecipe(name string, recipe *pool.Recipe) {
	T.servers.AddRecipe(name, recipe)
}

func (T *Pool) RemoveRecipe(name string) {
	T.servers.RemoveRecipe(name)
}

func (T *Pool) SyncInitialParameters(client *Client, server *spool.Server) (err, serverErr error) {
	clientParams := client.Conn.InitialParameters
	serverParams := server.Conn.InitialParameters

	for key, value := range clientParams {
		// skip already set params
		if serverParams[key] == value {
			p := packets.ParameterStatus{
				Key:   key.String(),
				Value: serverParams[key],
			}
			err = client.Conn.WritePacket(&p)
			if err != nil {
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
		err = client.Conn.WritePacket(&p)
		if err != nil {
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
		err = client.Conn.WritePacket(&p)
		if err != nil {
			return
		}
	}

	return
}

func (T *Pool) Pair(client *Client, server *spool.Server) (err, serverErr error) {
	if T.config.ParameterStatusSync != ParameterStatusSyncNone || T.config.ExtendedQuerySync {
		client.SetState(metrics.ConnStatePairing, server)
		server.SetState(metrics.ConnStatePairing, client.ID)

		switch T.config.ParameterStatusSync {
		case ParameterStatusSyncDynamic:
			err, serverErr = ps.Sync(T.config.TrackedParameters, client.Conn, server.Conn)
		case ParameterStatusSyncInitial:
			err, serverErr = T.SyncInitialParameters(client, server)
		}

		if err != nil || serverErr != nil {
			return
		}

		if T.config.ExtendedQuerySync {
			serverErr = eqp.Sync(client.Conn, server.Conn)
		}

		if serverErr != nil {
			return
		}
	}

	client.SetState(metrics.ConnStateActive, server)
	server.SetState(metrics.ConnStateActive, client.ID)
	return
}

func (T *Pool) addClient(client *Client) {
	T.mu.Lock()
	defer T.mu.Unlock()

	if T.clients == nil {
		T.clients = make(map[fed.BackendKey]*Client)
	}
	T.clients[client.Conn.BackendKey] = client
}

func (T *Pool) removeClient(client *Client) {
	T.mu.Lock()
	defer T.mu.Unlock()

	delete(T.clients, client.Conn.BackendKey)
}

func (T *Pool) Serve(conn *fed.Conn) error {
	if (T.config.PacketTracingOption & TracingOptionClient) != 0 {
		conn.Middleware = append(
			conn.Middleware,
			tracing.NewPacketTrace(context.Background()))
	}

	if (T.config.OtelTracingOption & TracingOptionClient) != 0 {
		conn.Middleware = append(
			conn.Middleware,
			tracing.NewOtelTrace(context.Background()))
	}

	if T.config.ParameterStatusSync == ParameterStatusSyncDynamic {
		conn.Middleware = append(
			conn.Middleware,
			ps.NewClient(conn.InitialParameters),
		)
	}
	if T.config.ExtendedQuerySync {
		conn.Middleware = append(
			conn.Middleware,
			eqp.NewClient(),
		)
	}

	client := NewClient(conn)

	T.addClient(client)
	defer T.removeClient(client)

	T.servers.AddClient(client.ID)
	defer T.servers.RemoveClient(client.ID)

	var err, serverErr error

	var server *spool.Server
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
		client.SetState(metrics.ConnStateAwaitingServer, nil)

		server = T.servers.Acquire(client.ID)
		if server == nil {
			return pool.ErrFailedToAcquirePeer
		}

		err, serverErr = T.Pair(client, server)
		if serverErr != nil {
			return serverErr
		}
		if err != nil {
			return err
		}

		p := packets.ReadyForQuery('I')
		err = client.Conn.WritePacket(&p)
		if err != nil {
			return err
		}

		client.Conn.Ready = true
	}

	for {
		if server != nil && T.config.ReleaseAfterTransaction {
			client.SetState(metrics.ConnStateIdle, nil)
			T.servers.Release(server)
			server = nil
		}

		var packet fed.Packet
		packet, err = client.Conn.ReadPacket(true)
		if err != nil {
			return err
		}

		if server == nil {
			client.SetState(metrics.ConnStateAwaitingServer, nil)

			server = T.servers.Acquire(client.ID)
			if server == nil {
				return pool.ErrFailedToAcquirePeer
			}

			err, serverErr = T.Pair(client, server)
		}
		if err == nil && serverErr == nil {
			err, serverErr = bouncers.Bounce(client.Conn, server.Conn, packet)
		}

		if serverErr != nil {
			return fmt.Errorf("server error: %w", serverErr)
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
	peer := func() *spool.Server {
		T.mu.RLock()
		defer T.mu.RUnlock()

		c, ok := T.clients[key]
		if !ok {
			return nil
		}

		_, _, peer := c.GetState()
		return peer
	}()

	if peer == nil {
		return
	}

	T.servers.Cancel(peer)
}

func (T *Pool) ReadMetrics(m *metrics.Pool) {
	T.servers.ReadMetrics(m)

	T.mu.RLock()
	defer T.mu.RUnlock()

	if m.Clients == nil {
		m.Clients = make(map[uuid.UUID]metrics.Conn)
	}
	for _, client := range T.clients {
		var c metrics.Conn
		client.ReadMetrics(&c)
		m.Clients[client.ID] = c
	}
}

func (T *Pool) Close() {
	T.servers.Close()
}

var _ pool.Pool = (*Pool)(nil)
