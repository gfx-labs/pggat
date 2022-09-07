package gat

import (
	"context"
	"fmt"
	"net"
	"sync"

	"git.tuxpa.in/a/zlog/log"

	"gfx.cafe/gfx/pggat/lib/config"
	"gfx.cafe/gfx/pggat/lib/gat/protocol"
)

type Gatling struct {
	c  *config.Global
	mu sync.RWMutex

	rout *QueryRouter

	csm     map[ClientKey]*ClientInfo
	clients map[string]*Client

	chConfig chan *config.Global

	servers map[string]*Server
	pools   map[string]*ConnectionPool
}

func NewGatling() *Gatling {
	g := &Gatling{
		csm:      map[ClientKey]*ClientInfo{},
		chConfig: make(chan *config.Global, 1),
		servers:  map[string]*Server{},
		clients:  map[string]*Client{},
		pools:    map[string]*ConnectionPool{},
		rout:     &QueryRouter{},
	}
	go g.watchConfigs()
	return g
}

func (g *Gatling) watchConfigs() {
	for {
		c := <-g.chConfig
		err := g.ensureConfig(c)
		if err != nil {
			log.Println("failed to parse config", err)
		}
	}
}

func (g *Gatling) GetClient(s string) (*Client, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	srv, ok := g.clients[s]
	if !ok {
		return nil, fmt.Errorf("client '%s' not found", s)
	}
	return srv, nil
}
func (g *Gatling) GetPool(s string) (*ConnectionPool, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	srv, ok := g.pools[s]
	if !ok {
		return nil, fmt.Errorf("pool '%s' not found", s)
	}
	return srv, nil
}

func (g *Gatling) GetServer(s string) (*Server, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	srv, ok := g.servers[s]
	if !ok {
		return nil, fmt.Errorf("server '%s' not found", s)
	}
	return srv, nil
}

func (g *Gatling) ensureConfig(c *config.Global) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if err := g.ensureGeneral(c); err != nil {
		return err
	}
	if err := g.ensureAdmin(c); err != nil {
		return err
	}
	if err := g.ensureServers(c); err != nil {
		return err
	}
	if err := g.ensurePools(c); err != nil {
		return err
	}

	return nil
}

// TODO: all other settings
func (g *Gatling) ensureGeneral(c *config.Global) error {
	return nil
}

// TODO: should configure the admin things, metrics, etc
func (g *Gatling) ensureAdmin(c *config.Global) error {
	return nil
}

// TODO: should connect to and load servers from config
func (g *Gatling) ensureServers(c *config.Global) error {
	return nil
}

// TODO: should connect to & load pools from config
func (g *Gatling) ensurePools(c *config.Global) error {
	return nil
}

func (g *Gatling) ListenAndServe(ctx context.Context) error {
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", g.c.General.Host, g.c.General.Port))
	if err != nil {
		return err
	}
	for {
		var c net.Conn
		c, err = ln.Accept()
		if err != nil {
			return err
		}
		go func() {
			err := g.handleConnection(ctx, c)
			if err != nil {
				log.Println("disconnected:", err)
			}
		}()
	}
}

// TODO: TLS
func (g *Gatling) handleConnection(ctx context.Context, c net.Conn) error {
	cl := NewClient(g.c, c, false)
	err := cl.Accept(ctx)
	if err != nil {
		log.Println(err.Error())
		switch e := err.(type) {
		case *PostgresError:
			_, err = e.Packet().Write(cl.wr)
			return err
		default:
			pgErr := &PostgresError{
				Severity: Error,
				Code:     InternalError,
				Message:  e.Error(),
			}
			_, err = pgErr.Packet().Write(cl.wr)
			return err
		}
	}
	return nil
}

type QueryRequest struct {
	ctx context.Context
	raw protocol.Packet
	c   *Client
}

func (g *Gatling) handleQuery(ctx context.Context, c *Client, raw protocol.Packet) error {
	// 1. analyze query using the query router
	role, err := g.rout.InferRole(raw)
	if err != nil {
		return err
	}
	pool, err := g.GetPool(g.selectPool(c, role))
	if err != nil {
		return err
	}
	// check config, select a pool
	_ = pool
	// TODO: we need to add some more information to the connectionpools, like current load, selectors, etc
	// perhaps we should just put the server connections in ServerPool and make that responsible for all of that
	srv, err := g.GetServer("some_output")
	if err != nil {
		return err
	}
	// write the packet or maybe send in a channel to the server
	_ = srv

	// send the result back to the client
	_ = c
	return nil
}

func (g *Gatling) selectPool(c *Client, role config.ServerRole) string {
	g.mu.RLock()
	defer g.mu.RUnlock()
	// do some filtering and figure out which pool you want to connect this client to, knowing their rold
	return "some_pool"
}
