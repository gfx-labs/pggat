package gatling

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"gfx.cafe/gfx/pggat/lib/gat/admin"
	"gfx.cafe/gfx/pggat/lib/gat/database"
	"gfx.cafe/gfx/pggat/lib/gat/gatling/server"

	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/gatling/client"
	"gfx.cafe/gfx/pggat/lib/gat/protocol/pg_error"

	"git.tuxpa.in/a/zlog/log"

	"gfx.cafe/gfx/pggat/lib/config"
)

// a Gatling is the main runtime process for the proxy
type Gatling struct {
	// config and config mutex
	c  *config.Global
	mu sync.RWMutex
	// channel that new config are delivered
	chConfig chan *config.Global

	pools   map[string]gat.Database
	clients map[gat.ClientID]gat.Client
}

func NewGatling(conf *config.Global) *Gatling {
	g := &Gatling{
		chConfig: make(chan *config.Global, 1),
		pools:    make(map[string]gat.Database),
		clients:  make(map[gat.ClientID]gat.Client),
	}
	// add admin pool
	adminPool := admin.New(g)
	g.pools["pgbouncer"] = adminPool
	g.pools["pggat"] = adminPool

	err := g.ensureConfig(conf)
	if err != nil {
		log.Println("failed to parse config", err)
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

func (g *Gatling) GetVersion() string {
	return "PgGat Gatling 0.0.1"
}

func (g *Gatling) GetConfig() *config.Global {
	return g.c
}

func (g *Gatling) GetDatabase(name string) gat.Database {
	g.mu.RLock()
	defer g.mu.RUnlock()
	srv, ok := g.pools[name]
	if !ok {
		return nil
	}
	return srv
}

func (g *Gatling) GetDatabases() map[string]gat.Database {
	g.mu.RLock()
	defer g.mu.RUnlock()
	return g.pools
}

func (g *Gatling) GetClient(id gat.ClientID) gat.Client {
	g.mu.RLock()
	defer g.mu.RUnlock()
	c, ok := g.clients[id]
	if !ok {
		return nil
	}
	return c
}

func (g *Gatling) GetClients() []gat.Client {
	g.mu.RLock()
	defer g.mu.RUnlock()
	out := make([]gat.Client, len(g.clients))
	idx := 0
	for _, p := range g.clients {
		out[idx] = p
		idx += 1
	}
	return out
}

func (g *Gatling) ensureConfig(c *config.Global) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.c = c

	if err := g.ensureGeneral(c); err != nil {
		return err
	}
	if err := g.ensureAdmin(c); err != nil {
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

func (g *Gatling) ensurePools(c *config.Global) error {
	for name, p := range c.Pools {
		if existing, ok := g.pools[name]; ok {
			existing.EnsureConfig(p)
		} else {
			g.pools[name] = database.New(server.Dial, p)
		}
	}
	return nil
}

func (g *Gatling) ListenAndServe(ctx context.Context) error {
	ln, err := net.Listen("tcp", fmt.Sprintf("%s:%d", g.c.General.Host, g.c.General.Port))
	if err != nil {
		return err
	}
	go func() {
		for {
			errch := make(chan error)
			go func() {
				c, err := ln.Accept()
				if err != nil {
					errch <- err
				}
				close(errch)
				err = g.handleConnection(ctx, c)
				if err != nil {
					if err != io.EOF {
						log.Println("disconnected:", err)
					}
				}
			}()

			err = <-errch
			if err != nil {
				log.Println("failed to accept connection:", err)
			}
		}
	}()
	return nil
}

// TODO: TLS
func (g *Gatling) handleConnection(ctx context.Context, c net.Conn) error {
	cl := client.NewClient(g, g.c, c, false)

	func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		g.clients[cl.GetId()] = cl
	}()
	defer func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		delete(g.clients, cl.GetId())
	}()

	err := cl.Accept(ctx)
	if err != nil {
		if !errors.Is(err, io.EOF) {
			log.Println("err in connection:", err.Error())
			_ = cl.Send(pg_error.IntoPacket(err))
			_ = cl.Flush()
		}
	}
	_ = c.Close()
	return nil
}

var _ gat.Gat = (*Gatling)(nil)
