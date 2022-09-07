package gatling

import (
	"context"
	"fmt"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/gatling/client"
	"gfx.cafe/gfx/pggat/lib/gat/gatling/pool"
	"gfx.cafe/gfx/pggat/lib/gat/protocol/pg_error"
	"net"
	"sync"

	"git.tuxpa.in/a/zlog/log"

	"gfx.cafe/gfx/pggat/lib/config"
)

type Gatling struct {
	c  *config.Global
	mu sync.RWMutex

	chConfig chan *config.Global

	pools map[string]*pool.Pool
}

func NewGatling(conf *config.Global) *Gatling {
	g := &Gatling{
		chConfig: make(chan *config.Global, 1),
		pools:    map[string]*pool.Pool{},
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
			existing.EnsureConfig(&p)
		} else {
			g.pools[name] = pool.NewPool(&p)
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
			err := g.handleConnection(ctx, c)
			if err != nil {
				log.Println("disconnected:", err)
			}
		}()
	}
}

// TODO: TLS
func (g *Gatling) handleConnection(ctx context.Context, c net.Conn) error {
	cl := client.NewClient(g, g.c, c, false)
	err := cl.Accept(ctx)
	if err != nil {
		log.Println(err.Error())
		switch e := err.(type) {
		case *pg_error.Error:
			return cl.Send(e.Packet())
		default:
			pgErr := &pg_error.Error{
				Severity: pg_error.Err,
				Code:     pg_error.InternalError,
				Message:  e.Error(),
			}
			return cl.Send(pgErr.Packet())
		}
	}
	return nil
}

var _ gat.Gat = (*Gatling)(nil)
