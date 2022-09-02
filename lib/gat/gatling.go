package gat

import (
	"context"
	"net"

	"gfx.cafe/gfx/pggat/lib/config"
)

type Gatling struct {
	c *config.Global

	csm      map[ClientKey]ClientInfo
	chConfig chan *config.Global
}

func NewGatling() *Gatling {
	return &Gatling{
		csm:      map[ClientKey]ClientInfo{},
		chConfig: make(chan *config.Global, 1),
	}
}

func (g *Gatling) ApplyConfig(c *config.Global) error {
	if g.c == nil {
		g.c = c
	} else {
		// TODO: dynamic config reload
		g.c = c
	}
	return nil
}

func (g *Gatling) ListenAndServe(ctx context.Context) error {
	ln, err := net.Listen("tcp", g.c.General.Host)
	if err != nil {
		return err
	}
	for {
		c, err := ln.Accept()
		if err != nil {
			return err
		}
		go g.handleConnection(ctx, c)
	}
}

// TODO: TLS
func (g *Gatling) handleConnection(ctx context.Context, c net.Conn) error {
	cl := NewClient(g.c, c, g.csm, false)
	return cl.Accept(ctx)
}
