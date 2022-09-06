package gat

import (
	"context"
	"fmt"
	"git.tuxpa.in/a/zlog/log"
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
	cl := NewClient(g.c, c, g.csm, false)
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
