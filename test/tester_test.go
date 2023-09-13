package test_test

import (
	"testing"

	"pggat/lib/auth/credentials"
	"pggat/lib/bouncer/backends/v0"
	"pggat/lib/gat/pool"
	"pggat/lib/gat/pool/dialer"
	"pggat/lib/gat/pool/pools/session"
	"pggat/lib/gat/pool/pools/transaction"
	"pggat/test"
	"pggat/test/tests"
)

func TestTester(t *testing.T) {
	tester := test.NewTester(test.Config{
		Modes: map[string]pool.Options{
			"transaction": transaction.Apply(pool.Options{}),
			"session": session.Apply(pool.Options{
				ServerResetQuery: "discard all",
			}),
		},
		Peer: dialer.Net{
			Network: "tcp",
			Address: "localhost:5432",
			AcceptOptions: backends.AcceptOptions{
				Credentials: credentials.Cleartext{
					Username: "postgres",
					Password: "password",
				},
				Database: "postgres",
			},
		},
	})
	if err := tester.Run(
		tests.SimpleQuery,
		tests.Transaction,
		tests.Sync,
	); err != nil {
		t.Error(err)
	}
}
