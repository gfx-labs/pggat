package test_test

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	_ "net/http/pprof"
	"strconv"
	"testing"

	"gfx.cafe/gfx/pggat/lib/auth"
	"gfx.cafe/gfx/pggat/lib/auth/credentials"
	"gfx.cafe/gfx/pggat/lib/bouncer/backends/v0"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/pool"
	"gfx.cafe/gfx/pggat/lib/gat/pool/recipe"
	"gfx.cafe/gfx/pggat/test"
	"gfx.cafe/gfx/pggat/test/tests"
)

func daisyChain(creds auth.Credentials, control recipe.Dialer, n int) (recipe.Dialer, error) {
	for i := 0; i < n; i++ {
		var options = pool.Config{
			Credentials: creds,
		}
		if i%2 == 0 {
			options = transaction.Apply(options)
		} else {
			options.ServerResetQuery = "DISCARD ALL"
			options = session.Apply(options)
		}

		p := pool.NewPool(options)
		p.AddRecipe("runner", recipe.NewRecipe(recipe.Config{
			Dialer: control,
		}))

		m := new(raw_pools.Module)
		m.Add("runner", "pool", p)

		l := &net_listener.Module{
			Config: net_listener.Config{
				Network: "tcp",
				Address: ":0",
			},
		}
		if err := l.Start(); err != nil {
			return recipe.Dialer{}, err
		}
		port := l.Addr().(*net.TCPAddr).Port

		server := gat.NewServer(m, l)

		if err := server.Start(); err != nil {
			panic(err)
		}

		control = recipe.Dialer{
			Network: "tcp",
			Address: ":" + strconv.Itoa(port),
			AcceptOptions: backends.acceptOptions{
				Username:    "runner",
				Credentials: creds,
				Database:    "pool",
			},
		}
	}

	return control, nil
}

func TestTester(t *testing.T) {
	control := recipe.Dialer{
		Network: "tcp",
		Address: "localhost:5432",
		AcceptOptions: backends.acceptOptions{
			Username: "postgres",
			Credentials: credentials.Cleartext{
				Username: "postgres",
				Password: "password",
			},
			Database: "postgres",
		},
	}

	// generate random password for testing
	var raw [32]byte
	_, err := rand.Read(raw[:])
	if err != nil {
		t.Error(err)
		return
	}
	password := hex.EncodeToString(raw[:])
	creds := credentials.Cleartext{
		Username: "runner",
		Password: password,
	}

	parent, err := daisyChain(creds, control, 16)
	if err != nil {
		t.Error(err)
		return
	}

	m := new(raw_pools.Module)
	transactionPool := pool.NewPool(transaction.Apply(pool.Config{
		Credentials: creds,
	}))
	transactionPool.AddRecipe("runner", recipe.NewRecipe(recipe.Config{
		Dialer: parent,
	}))
	m.Add("runner", "transaction", transactionPool)

	sessionPool := pool.NewPool(session.Apply(pool.Config{
		Credentials:      creds,
		ServerResetQuery: "discard all",
	}))
	sessionPool.AddRecipe("runner", recipe.NewRecipe(recipe.Config{
		Dialer: parent,
	}))
	m.Add("runner", "session", sessionPool)

	l := &net_listener.Module{
		Config: net_listener.Config{
			Network: "tcp",
			Address: ":0",
		},
	}
	if err = l.Start(); err != nil {
		t.Error(err)
		return
	}
	port := l.Addr().(*net.TCPAddr).Port

	server := gat.NewServer(m, l)

	if err = server.Start(); err != nil {
		t.Error(err)
		return
	}

	transactionDialer := recipe.Dialer{
		Network: "tcp",
		Address: ":" + strconv.Itoa(port),
		AcceptOptions: backends.acceptOptions{
			Username:    "runner",
			Credentials: creds,
			Database:    "transaction",
		},
	}
	sessionDialer := recipe.Dialer{
		Network: "tcp",
		Address: ":" + strconv.Itoa(port),
		AcceptOptions: backends.acceptOptions{
			Username:    "runner",
			Credentials: creds,
			Database:    "session",
		},
	}

	tester := test.NewTester(test.Config{
		Stress: 8,

		Modes: map[string]recipe.Dialer{
			"control":     control,
			"transaction": transactionDialer,
			"session":     sessionDialer,
		},
	})
	if err = tester.Run(
		tests.SimpleQuery,
		tests.Transaction,
		tests.Sync,
		tests.EQP0,
		tests.EQP1,
		tests.EQP2,
		tests.EQP3,
		tests.EQP4,
		tests.EQP5,
		tests.EQP6,
		tests.EQP7,
		tests.EQP8,
		tests.CopyOut0,
		tests.CopyOut1,
		tests.CopyIn0,
		tests.CopyIn1,
		tests.DiscardAll,
	); err != nil {
		fmt.Print(err.Error())
		t.Fail()
	}
}
