package test

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"gfx.cafe/gfx/pggat/lib/bouncer/bouncers/v2"
	"gfx.cafe/gfx/pggat/lib/fed"
	"gfx.cafe/gfx/pggat/lib/fed/middlewares/unterminate"
	packets "gfx.cafe/gfx/pggat/lib/fed/packets/v3.0"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/gsql"
	"gfx.cafe/gfx/pggat/lib/util/flip"
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

func (T *Runner) prepare(client *fed.Conn, until int) error {
	for i := 0; i < until; i++ {
		x := T.test.Packets[i]
		if err := client.WritePacket(x); err != nil {
			return err
		}
	}

	if err := client.WritePacket(&packets.Terminate{}); err != nil {
		return err
	}

	return client.Flush()
}

func (T *Runner) runModeL1(dialer pool.Dialer, client *fed.Conn) error {
	server, err := dialer.Dial()
	if err != nil {
		return err
	}
	defer func() {
		_ = server.Close()
	}()

	client.Middleware = append(client.Middleware, unterminate.Unterminate)

	for {
		var p fed.Packet
		p, err = client.ReadPacket(true)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return err
		}

		clientErr, serverErr := bouncers.Bounce(client, server, p)
		if clientErr != nil {
			return clientErr
		}
		if serverErr != nil {
			return serverErr
		}
	}

	return nil
}

func (T *Runner) runModeOnce(dialer pool.Dialer) ([]byte, error) {
	inward, outward, in, _ := gsql.NewPair()
	if err := T.prepare(inward, len(T.test.Packets)); err != nil {
		return nil, err
	}

	if err := T.runModeL1(dialer, outward); err != nil {
		return nil, err
	}

	if err := inward.Close(); err != nil {
		return nil, err
	}

	return io.ReadAll(in)
}

func (T *Runner) runModeFail(dialer pool.Dialer) error {
	for i := 1; i <= len(T.test.Packets); i++ {
		inward, outward, _, _ := gsql.NewPair()
		if err := T.prepare(inward, i); err != nil {
			return err
		}

		if err := T.runModeL1(dialer, outward); err != nil && !errors.Is(err, io.EOF) {
			return err
		}
	}

	return nil
}

func (T *Runner) runMode(dialer pool.Dialer) ([]byte, error) {
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
			if !bytes.Equal(expected, actual) {
				return fmt.Errorf("mismatched results: expected %v but got %v", expected, actual)
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

	var expected []byte

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

		if !bytes.Equal(expected, actual) {
			errs = append(errs, ErrorIn{
				Name: name,
				Err:  fmt.Errorf("mismatched results: expected %v but got %v", expected, actual),
			})
			continue
		}
	}

	if len(errs) > 0 {
		return Errors(errs)
	}
	return nil
}
