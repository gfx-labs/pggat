package test

import (
	"errors"
	"fmt"
	"io"

	"pggat/lib/bouncer/bouncers/v2"
	"pggat/lib/fed"
	packets "pggat/lib/fed/packets/v3.0"
	"pggat/lib/gat/pool"
	"pggat/lib/gat/pool/recipe"
	"pggat/lib/gsql"
	"pggat/test/inst"
)

type Runner struct {
	config Config
	test   Test
}

func MakeRunner(config Config, test Test) Runner {
	return Runner{
		config: config,
		test:   test,
	}
}

func (T *Runner) prepare(client *gsql.Client) []Capturer {
	results := make([]Capturer, len(T.test.Instructions))

	for i, x := range T.test.Instructions {
		switch v := x.(type) {
		case inst.SimpleQuery:
			q := packets.Query(v)
			client.Do(&results[i], q.IntoPacket())
		case inst.Sync:
			client.Do(&results[i], fed.NewPacket(packets.TypeSync))
		}
	}

	return results
}

func (T *Runner) runControl() ([]Capturer, error) {
	control, _, err := T.config.Peer.Dial()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = control.Close()
	}()

	var client gsql.Client
	results := T.prepare(&client)
	if err = client.Close(); err != nil {
		return nil, err
	}

	for {
		var p fed.Packet
		p, err = client.ReadPacket(true)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}

		clientErr, serverErr := bouncers.Bounce(&client, control, p)
		if clientErr != nil {
			return nil, clientErr
		}
		if serverErr != nil {
			return nil, serverErr
		}
	}

	return results, nil
}

func (T *Runner) runMode(options pool.Options) ([]Capturer, error) {
	opts := options
	// allowing ps sync would mess up testing
	opts.ParameterStatusSync = pool.ParameterStatusSyncNone
	p := pool.NewPool(opts)
	defer p.Close()
	p.AddRecipe("server", recipe.NewRecipe(
		recipe.Options{
			Dialer: T.config.Peer,
		},
	))

	var client gsql.Client
	results := T.prepare(&client)
	if err := client.Close(); err != nil {
		return nil, err
	}

	if err := p.Serve(&client, nil, [8]byte{}); err != nil && !errors.Is(err, io.EOF) {
		return nil, err
	}

	return results, nil
}

func (T *Runner) Run() error {
	// control
	expected, err := T.runControl()
	if err != nil {
		return err
	}

	// modes
	for name, mode := range T.config.Modes {
		actual, err := T.runMode(mode)
		if err != nil {
			return err
		}

		if len(expected) != len(actual) {
			return fmt.Errorf("wrong number of results! expected %d but got %d", len(expected), len(actual))
		}

		for i, exp := range expected {
			act := actual[i]

			if err = exp.Check(&act); err != nil {
				return err
			}
		}

		_ = name
	}

	return nil
}
