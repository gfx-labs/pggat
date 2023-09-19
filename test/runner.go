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
	"pggat/lib/util/flip"
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

func (T *Runner) prepare(client *gsql.Client, until int) []Capturer {
	results := make([]Capturer, until)

	for i := 0; i < until; i++ {
		x := T.test.Instructions[i]
		switch v := x.(type) {
		case inst.SimpleQuery:
			q := packets.Query(v)
			client.Do(&results[i], q.IntoPacket(nil))
		case inst.Sync:
			client.Do(&results[i], fed.NewPacket(packets.TypeSync))
		case inst.Parse:
			p := packets.Parse{
				Destination: v.Destination,
				Query:       v.Query,
			}
			client.Do(&results[i], p.IntoPacket(nil))
		case inst.Bind:
			p := packets.Bind{
				Destination: v.Destination,
				Source:      v.Source,
			}
			client.Do(&results[i], p.IntoPacket(nil))
		case inst.DescribePortal:
			p := packets.Describe{
				Which:  'P',
				Target: string(v),
			}
			client.Do(&results[i], p.IntoPacket(nil))
		case inst.DescribePreparedStatement:
			p := packets.Describe{
				Which:  'S',
				Target: string(v),
			}
			client.Do(&results[i], p.IntoPacket(nil))
		case inst.Execute:
			p := packets.Execute{
				Target: string(v),
			}
			client.Do(&results[i], p.IntoPacket(nil))
		case inst.ClosePortal:
			p := packets.Close{
				Which:  'P',
				Target: string(v),
			}
			client.Do(&results[i], p.IntoPacket(nil))
		case inst.ClosePreparedStatement:
			p := packets.Close{
				Which:  'S',
				Target: string(v),
			}
			client.Do(&results[i], p.IntoPacket(nil))
		case inst.CopyData:
			p := packets.CopyData(v)
			client.Do(&results[i], p.IntoPacket(nil))
		case inst.CopyDone:
			client.Do(&results[i], fed.NewPacket(packets.TypeCopyDone))
		}
	}

	return results
}

func (T *Runner) runModeL1(dialer dialer.Dialer, client *gsql.Client) error {
	server, _, err := dialer.Dial()
	if err != nil {
		return err
	}
	defer func() {
		_ = server.Close()
	}()

	for {
		var p fed.Packet
		p, err = client.ReadPacket(true, p)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}

		_, clientErr, serverErr := bouncers.Bounce(client, server, p)
		if clientErr != nil {
			return clientErr
		}
		if serverErr != nil {
			return serverErr
		}
	}

	return nil
}

func (T *Runner) runModeOnce(dialer dialer.Dialer) ([]Capturer, error) {
	var client gsql.Client
	results := T.prepare(&client, len(T.test.Instructions))
	if err := client.Close(); err != nil {
		return nil, err
	}

	if err := T.runModeL1(dialer, &client); err != nil {
		return nil, err
	}

	return results, nil
}

func (T *Runner) runModeFail(dialer dialer.Dialer) error {
	for i := 1; i < len(T.test.Instructions)+1; i++ {
		var client gsql.Client
		T.prepare(&client, i)
		if err := client.Close(); err != nil {
			return err
		}

		if err := T.runModeL1(dialer, &client); err != nil && !errors.Is(err, io.EOF) {
			return err
		}
	}

	return nil
}

func (T *Runner) runMode(dialer dialer.Dialer) ([]Capturer, error) {
	instances := T.config.Stress
	if instances < 1 || T.test.SideEffects {
		return T.runModeOnce(dialer)
	}

	expected, err := T.runModeOnce(dialer)
	if err != nil {
		return nil, err
	}

	// fail testing
	if err = T.runModeFail(dialer); err != nil {
		return nil, err
	}

	// stress test
	var b flip.Bank

	for i := 0; i < instances-1; i++ {
		b.Queue(func() error {
			actual, err := T.runModeOnce(dialer)
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
			return nil
		})
	}

	if err = b.Wait(); err != nil {
		return nil, err
	}

	return expected, nil
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
