package gatling

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"gfx.cafe/util/go/generic"

	"gfx.cafe/gfx/pggat/lib/gat/admin"
	"gfx.cafe/gfx/pggat/lib/gat/database"
	"gfx.cafe/gfx/pggat/lib/gat/gatling/server"
	"gfx.cafe/gfx/pggat/lib/metrics"

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
	clients generic.Map[gat.ClientID, gat.Client]
}

func NewGatling(conf *config.Global) *Gatling {
	g := &Gatling{
		chConfig: make(chan *config.Global, 1),
		pools:    make(map[string]gat.Database),
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
	c, ok := g.clients.Load(id)
	if !ok {
		return nil
	}
	return c
}

func (g *Gatling) GetClients() (out []gat.Client) {
	g.clients.Range(func(id gat.ClientID, client gat.Client) bool {
		out = append(out, client)
		return true
	})
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
			g.pools[name] = database.New(server.Dial, name, p)
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
				metrics.RecordAcceptConnectionStatus(err)
				close(errch)
				g.handleConnection(ctx, c)
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
func (g *Gatling) handleConnection(ctx context.Context, c net.Conn) {
	cl := client.NewClient(g, g.c, c, false)

	g.clients.Store(cl.GetId(), cl)
	metrics.RecordActiveConnections(1)
	defer func() {
		g.clients.Delete(cl.GetId())
		metrics.RecordActiveConnections(-1)
	}()

	err := cl.Accept(ctx)
	if err != nil {
		metrics.RecordConnectionError(err)
		if !errors.Is(err, io.EOF) {
			log.Println("err in connection:", err.Error())
			_ = cl.Send(pg_error.IntoPacket(err))
			_ = cl.Flush()
		}
	}
	_ = c.Close()
}

var _ gat.Gat = (*Gatling)(nil)
