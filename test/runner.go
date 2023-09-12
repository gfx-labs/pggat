package test

import (
	"errors"
	"io"

	"tuxpa.in/a/zlog/log"

	"pggat/lib/bouncer/bouncers/v2"
	"pggat/lib/fed"
	packets "pggat/lib/fed/packets/v3.0"
	"pggat/lib/gat/pool"
	"pggat/lib/gat/pool/recipe"
	"pggat/lib/gsql"
	"pggat/lib/util/maps"
	"pggat/test/inst"
)

type Runner struct {
	config Config
	test   Test

	pools map[string]*pool.Pool
}

func MakeRunner(config Config, test Test) Runner {
	return Runner{
		config: config,
		test:   test,
	}
}

func (T *Runner) setup() error {
	// get pools ready
	maps.Clear(T.pools)
	if T.pools == nil {
		T.pools = make(map[string]*pool.Pool)
	}

	for name, options := range T.config.Modes {
		opts := options
		// allowing ps sync would mess up testing
		opts.ParameterStatusSync = pool.ParameterStatusSyncNone
		p := pool.NewPool(opts)
		p.AddRecipe("server", recipe.NewRecipe(
			recipe.Options{
				Dialer: T.config.Peer,
			},
		))
		T.pools[name] = p
	}

	return nil
}

type logWriter struct{}

func (logWriter) WritePacket(pkt fed.Packet) error {
	log.Print("got packet ", pkt)
	return nil
}

func (T *Runner) run(pkts ...fed.Packet) error {
	// expected
	{
		log.Print("expected packets")

		var client gsql.Client
		client.Do(logWriter{}, pkts...)
		if err := client.Close(); err != nil {
			return err
		}

		server, _, err := T.config.Peer.Dial()
		if err != nil {
			return err
		}

		for {
			p, err := client.ReadPacket(true)
			if err != nil {
				if errors.Is(err, io.EOF) {
					break
				}
				return err
			}

			clientErr, serverErr := bouncers.Bounce(&client, server, p)
			if clientErr != nil {
				return clientErr
			}
			if serverErr != nil {
				return serverErr
			}
		}
	}

	// actual
	for name, p := range T.pools {
		log.Print()
		log.Print("pool ", name)

		var client gsql.Client
		client.Do(logWriter{}, pkts...)
		if err := client.Close(); err != nil {
			return err
		}

		if err := p.Serve(&client, nil, [8]byte{}); err != nil && !errors.Is(err, io.EOF) {
			return err
		}

		_ = name
	}

	return nil
}

func (T *Runner) Run() error {
	if err := T.setup(); err != nil {
		return err
	}

	for _, i := range T.test.Instructions {
		switch v := i.(type) {
		case inst.SimpleQuery:
			q := packets.Query(v)
			if err := T.run(q.IntoPacket()); err != nil {
				return err
			}
		}
	}

	return nil
}
