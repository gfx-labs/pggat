package test_test

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"strconv"
	"testing"

	"pggat/lib/auth"
	"pggat/lib/auth/credentials"
	"pggat/lib/bouncer/backends/v0"
	"pggat/lib/bouncer/frontends/v0"
	"pggat/lib/gat"
	"pggat/lib/gat/pool"
	"pggat/lib/gat/pool/dialer"
	"pggat/lib/gat/pool/pools/session"
	"pggat/lib/gat/pool/pools/transaction"
	"pggat/lib/gat/pool/recipe"
	"pggat/test"
	"pggat/test/tests"
)

func daisyChain(creds auth.Credentials, control dialer.Net, n int) (dialer.Net, error) {
	for i := 0; i < n; i++ {
		var g gat.PoolsMap

		var options = pool.Options{
			Credentials: creds,
		}
		if i%2 == 0 {
			options = transaction.Apply(options)
		} else {
			options.ServerResetQuery = "DISCARD ALL"
			options = session.Apply(options)
		}

		p := pool.NewPool(options)
		p.AddRecipe("runner", recipe.NewRecipe(recipe.Options{
			Dialer: control,
		}))
		g.Add("runner", "pool", p)

		listener, err := gat.Listen("tcp", ":0", frontends.AcceptOptions{})
		if err != nil {
			return dialer.Net{}, err
		}
		port := listener.Listener.Addr().(*net.TCPAddr).Port

		go func() {
			err := gat.Serve(listener, &g)
			if err != nil {
				panic(err)
			}
		}()

		control = dialer.Net{
			Network: "tcp",
			Address: ":" + strconv.Itoa(port),
			AcceptOptions: backends.AcceptOptions{
				Credentials: creds,
				Database:    "pool",
			},
		}
	}

	return control, nil
}

func TestTester(t *testing.T) {
	control := dialer.Net{
		Network: "tcp",
		Address: "localhost:5432",
		AcceptOptions: backends.AcceptOptions{
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

	var g gat.PoolsMap

	transactionPool := pool.NewPool(transaction.Apply(pool.Options{
		Credentials: creds,
	}))
	transactionPool.AddRecipe("runner", recipe.NewRecipe(recipe.Options{
		Dialer: parent,
	}))
	g.Add("runner", "transaction", transactionPool)

	sessionPool := pool.NewPool(session.Apply(pool.Options{
		Credentials:      creds,
		ServerResetQuery: "discard all",
	}))
	sessionPool.AddRecipe("runner", recipe.NewRecipe(recipe.Options{
		Dialer: parent,
	}))
	g.Add("runner", "session", sessionPool)

	listener, err := gat.Listen("tcp", ":0", frontends.AcceptOptions{})
	if err != nil {
		t.Error(err)
		return
	}
	port := listener.Listener.Addr().(*net.TCPAddr).Port

	go func() {
		err := gat.Serve(listener, &g)
		if err != nil {
			t.Error(err)
		}
	}()

	transactionDialer := dialer.Net{
		Network: "tcp",
		Address: ":" + strconv.Itoa(port),
		AcceptOptions: backends.AcceptOptions{
			Credentials: creds,
			Database:    "transaction",
		},
	}
	sessionDialer := dialer.Net{
		Network: "tcp",
		Address: ":" + strconv.Itoa(port),
		AcceptOptions: backends.AcceptOptions{
			Credentials: creds,
			Database:    "session",
		},
	}

	tester := test.NewTester(test.Config{
		Stress: 16,

		Modes: map[string]dialer.Dialer{
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
		tests.DiscardAll,
	); err != nil {
		fmt.Print(err.Error())
		t.Fail()
	}
}
