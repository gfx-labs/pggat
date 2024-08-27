package basic

import (
	"context"
	"fmt"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/tracing"
	"sync"
	"time"

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
	"gfx.cafe/gfx/pggat/lib/instrumentation/prom"
	"gfx.cafe/gfx/pggat/lib/util/slices"
)

type Pool struct {
	config Config

	servers spool.Pool

	clients map[fed.BackendKey]*Client
	mu      sync.RWMutex
}

func NewPool(ctx context.Context, config Config) *Pool {
	p := &Pool{
		config:  config,
		servers: spool.MakePool(config.Spool()),
	}
	go p.servers.ScaleLoop(ctx)
	return p
}

func (T *Pool) AddRecipe(ctx context.Context, name string, recipe *pool.Recipe) {
	T.servers.AddRecipe(ctx, name, recipe)
}

func (T *Pool) RemoveRecipe(ctx context.Context, name string) {
	T.servers.RemoveRecipe(ctx, name)
}

func (T *Pool) SyncInitialParameters(ctx context.Context, client *Client, server *spool.Server) (err, serverErr error) {
	clientParams := client.Conn.InitialParameters
	serverParams := server.Conn.InitialParameters

	for key, value := range clientParams {
		// skip already set params
		if serverParams[key] == value {
			p := packets.ParameterStatus{
				Key:   key.String(),
				Value: serverParams[key],
			}
			err = client.Conn.WritePacket(ctx, &p)
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
		err = client.Conn.WritePacket(ctx, &p)
		if err != nil {
			return
		}

		if !setServer {
			continue
		}

		serverErr, _ = backends.SetParameter(ctx, server.Conn, nil, key, value)
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
		err = client.Conn.WritePacket(ctx, &p)
		if err != nil {
			return
		}
	}

	return
}

func (T *Pool) Pair(ctx context.Context, client *Client, server *spool.Server) (err, serverErr error) {
	if T.config.ParameterStatusSync != ParameterStatusSyncNone || T.config.ExtendedQuerySync {
		client.SetState(metrics.ConnStatePairing, server)
		server.SetState(metrics.ConnStatePairing, client.ID)

		switch T.config.ParameterStatusSync {
		case ParameterStatusSyncDynamic:
			err, serverErr = ps.Sync(ctx, T.config.TrackedParameters, client.Conn, server.Conn)
		case ParameterStatusSyncInitial:
			err, serverErr = T.SyncInitialParameters(ctx, client, server)
		}

		if err != nil || serverErr != nil {
			return
		}

		if T.config.ExtendedQuerySync {
			serverErr = eqp.Sync(ctx, client.Conn, server.Conn)
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

func (T *Pool) Serve(ctx context.Context, conn *fed.Conn) error {
	if (T.config.PacketTracingOption & TracingOptionClient) != 0 {
		conn.Middleware = append(
			conn.Middleware,
			tracing.NewPacketTrace())
	}

	if (T.config.OtelTracingOption & TracingOptionClient) != 0 {
		conn.Middleware = append(
			conn.Middleware,
			tracing.NewOtelTrace())
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
				T.servers.RemoveServer(ctx, server)
			} else {
				T.servers.Release(ctx, server)
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

		err, serverErr = T.Pair(ctx, client, server)
		if serverErr != nil {
			return serverErr
		}
		if err != nil {
			return err
		}

		p := packets.ReadyForQuery('I')
		err = client.Conn.WritePacket(ctx, &p)
		if err != nil {
			return err
		}

		client.Conn.Ready = true
	}

	poolLabels := prom.PoolSimpleLabels{
		Database: conn.Database,
		User:     conn.User,
	}
	{
		if T.config.ReleaseAfterTransaction {
			poolLabels.Mode = "transaction"
		} else {
			poolLabels.Mode = "session"
		}
		prom.PoolSimple.Accepted(poolLabels).Inc()
		prom.PoolSimple.Current(poolLabels).Inc()
		defer prom.PoolSimple.Current(poolLabels).Dec()
	}
	opLabels := poolLabels.ToOperation()
	for {
		if server != nil && T.config.ReleaseAfterTransaction {
			client.SetState(metrics.ConnStateIdle, nil)
			T.servers.Release(ctx, server)
			server = nil
		}

		var packet fed.Packet
		packet, err = client.Conn.ReadPacket(ctx, true)
		if err != nil {
			return err
		}

		if server == nil {
			start := time.Now()
			client.SetState(metrics.ConnStateAwaitingServer, nil)

			server = T.servers.Acquire(client.ID)
			if server == nil {
				return pool.ErrFailedToAcquirePeer
			}

			err, serverErr = T.Pair(ctx, client, server)
			dur := time.Since(start)
			if err == nil && serverErr == nil {
				prom.OperationSimple.Acquire(opLabels).Observe(float64(dur) / float64(time.Millisecond))
			}
		}
		if err == nil && serverErr == nil {
			{
				start := time.Now()
				err, serverErr = bouncers.Bounce(ctx, client.Conn, server.Conn, packet)
				if serverErr == nil {
					dur := time.Since(start)
					prom.OperationSimple.Execution(opLabels).Observe(float64(dur) / float64(time.Millisecond))
				}
			}
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

func (T *Pool) Cancel(ctx context.Context, key fed.BackendKey) {
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

	T.servers.Cancel(ctx, peer)
}

func (T *Pool) ReadMetrics(ctx context.Context,m *metrics.Pool) {
	T.servers.ReadMetrics(ctx,m)

	T.mu.RLock()
	defer T.mu.RUnlock()

	if m.Clients == nil {
		m.Clients = make(map[uuid.UUID]metrics.Conn)
	}
	for _, client := range T.clients {
		var c metrics.Conn
		client.ReadMetrics(ctx,&c)
		m.Clients[client.ID] = c
	}
}

func (T *Pool) Close(ctx context.Context) {
	T.servers.Close(ctx)
}

var _ pool.Pool = (*Pool)(nil)
