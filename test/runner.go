package test

import (
	"errors"
	"fmt"
	"io"

	"pggat/lib/bouncer/bouncers/v2"
	"pggat/lib/fed"
	packets "pggat/lib/fed/packets/v3.0"
	"pggat/lib/gat/pool/dialer"
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
		case inst.Parse:
			p := packets.Parse{
				Destination: v.Destination,
				Query:       v.Query,
			}
			client.Do(&results[i], p.IntoPacket())
		case inst.Bind:
			p := packets.Bind{
				Destination: v.Destination,
				Source:      v.Source,
			}
			client.Do(&results[i], p.IntoPacket())
		case inst.DescribePortal:
			p := packets.Describe{
				Which:  'P',
				Target: string(v),
			}
			client.Do(&results[i], p.IntoPacket())
		case inst.DescribePreparedStatement:
			p := packets.Describe{
				Which:  'S',
				Target: string(v),
			}
			client.Do(&results[i], p.IntoPacket())
		case inst.Execute:
			p := packets.Execute{
				Target: string(v),
			}
			client.Do(&results[i], p.IntoPacket())
		case inst.ClosePortal:
			p := packets.Close{
				Which:  'P',
				Target: string(v),
			}
			client.Do(&results[i], p.IntoPacket())
		case inst.ClosePreparedStatement:
			p := packets.Close{
				Which:  'S',
				Target: string(v),
			}
			client.Do(&results[i], p.IntoPacket())
		}
	}

	return results
}

func (T *Runner) runMode(dialer dialer.Dialer) ([]Capturer, error) {
	server, _, err := dialer.Dial()
	if err != nil {
		return nil, err
	}
	defer func() {
		_ = server.Close()
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

		clientErr, serverErr := bouncers.Bounce(&client, server, p)
		if clientErr != nil {
			return nil, clientErr
		}
		if serverErr != nil {
			return nil, serverErr
		}
	}

	return results, nil
}

func (T *Runner) Run() error {
	var errs []error

	var expected []Capturer

	// modes
	for name, mode := range T.config.Modes {
		actual, err := T.runMode(mode)
		if err != nil {
			errs = append(errs, ErrorIn{
				Name: name,
				Err:  err,
			})
			continue
		}

		if expected == nil {
			expected = actual
			continue
		}

		if len(expected) != len(actual) {
			errs = append(errs, ErrorIn{
				Name: name,
				Err:  fmt.Errorf("wrong number of results! expected %d but got %d", len(expected), len(actual)),
			})
			continue
		}

		var modeErrs []error

		for i, exp := range expected {
			act := actual[i]

			if err = exp.Check(&act); err != nil {
				modeErrs = append(modeErrs, fmt.Errorf("instruction %d: %s", i+1, err.Error()))
			}
		}

		if len(modeErrs) > 0 {
			errs = append(errs, ErrorIn{
				Name: name,
				Err:  Errors(modeErrs),
			})
		}
	}

	if len(errs) > 0 {
		return Errors(errs)
	}
	return nil
}
