package test_test

import (
	"testing"

	"pggat/lib/auth/credentials"
	"pggat/lib/bouncer/backends/v0"
	"pggat/lib/gat/pool/dialer"
	"pggat/test"
	"pggat/test/tests"
)

func TestTester(t *testing.T) {
	tester := test.NewTester(test.Config{
		Peer: dialer.Net{
			Network: "tcp",
			Address: "localhost:5432",
			AcceptOptions: backends.AcceptOptions{
				Credentials: credentials.Cleartext{
					Username: "postgres",
					Password: "password",
				},
				Database: "pggat",
			},
		},
	})
	if err := tester.Run(tests.SimpleQuery); err != nil {
		t.Error(err)
	}
}
