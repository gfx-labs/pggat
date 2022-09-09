package gatling

import (
	"context"
	"errors"
	"fmt"
	"net"
	"sync"

	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/gatling/client"
	"gfx.cafe/gfx/pggat/lib/gat/gatling/pool"
	"gfx.cafe/gfx/pggat/lib/gat/protocol/pg_error"

	"git.tuxpa.in/a/zlog/log"

	"gfx.cafe/gfx/pggat/lib/config"
)

type Gatling struct {
	c  *config.Global
	mu sync.RWMutex

	chConfig chan *config.Global

	pools   map[string]*pool.Pool
	clients map[gat.ClientID]*client.Client
}

func NewGatling(conf *config.Global) *Gatling {
	g := &Gatling{
		chConfig: make(chan *config.Global, 1),
		pools:    make(map[string]*pool.Pool),
		clients:  make(map[gat.ClientID]*client.Client),
	}
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

func (g *Gatling) GetPool(name string) (gat.Pool, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	srv, ok := g.pools[name]
	if !ok {
		return nil, fmt.Errorf("pool '%s' not found", name)
	}
	return srv, nil
}

func (g *Gatling) GetClient(id gat.ClientID) (gat.Client, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	c, ok := g.clients[id]
	if !ok {
		return nil, errors.New("client not found")
	}
	return c, nil
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
			g.pools[name] = pool.NewPool(p)
		}
	}
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
			err = g.handleConnection(ctx, c)
			if err != nil {
				log.Println("disconnected:", err)
				return
			}
		}()
	}
}

// TODO: TLS
func (g *Gatling) handleConnection(ctx context.Context, c net.Conn) error {
	err := c.(*net.TCPConn).SetNoDelay(false)
	if err != nil {
		return err
	}

	cl := client.NewClient(g, g.c, c, false)

	func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		g.clients[cl.Id()] = cl
	}()
	defer func() {
		g.mu.Lock()
		defer g.mu.Unlock()
		delete(g.clients, cl.Id())
	}()

	err = cl.Accept(ctx)
	if err != nil {
		log.Println("err in connection:", err.Error())
		_ = cl.Send(pg_error.IntoPacket(err))
	}
	_ = c.Close()
	return nil
}

var _ gat.Gat = (*Gatling)(nil)
