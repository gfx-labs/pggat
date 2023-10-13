package test_test

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	_ "net/http/pprof"
	"strconv"
	"strings"
	"testing"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"

	"gfx.cafe/gfx/pggat/lib/auth/credentials"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/gatcaddyfile"
	pool_handler "gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/rewrite_password"
	"gfx.cafe/gfx/pggat/lib/gat/matchers"
	"gfx.cafe/gfx/pggat/lib/gat/pool"
	"gfx.cafe/gfx/pggat/lib/gat/poolers/session"
	"gfx.cafe/gfx/pggat/lib/gat/poolers/transaction"
	"gfx.cafe/gfx/pggat/test"
	"gfx.cafe/gfx/pggat/test/tests"
)

type dialer struct {
	Address  string
	Username string
	Password string
	Database string
}

var nextPort int

func randAddress() string {
	nextPort++
	return "/tmp/.s.PGGAT." + strconv.Itoa(nextPort)
}

func resolveNetwork(address string) string {
	if strings.HasPrefix(address, "/") {
		return "unix"
	} else {
		return "tcp"
	}
}

func randPassword() (string, error) {
	var b [20]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b[:]), nil
}

func createServer(parent dialer, poolers map[string]caddy.Module) (server gat.ServerConfig, dialers map[string]dialer, err error) {
	address := randAddress()

	server.Listen = []gat.ListenerConfig{
		{
			Address: address,
		},
	}

	var password string
	password, err = randPassword()
	if err != nil {
		return
	}

	server.Routes = append(
		server.Routes,
		gat.RouteConfig{
			Handle: gatcaddyfile.JSONModuleObject(
				&rewrite_password.Module{
					Password: password,
				},
				gatcaddyfile.Handler,
				"handler",
				nil,
			),
		},
	)

	for name, pooler := range poolers {
		p := pool_handler.Module{
			Config: pool_handler.Config{
				Pooler: gatcaddyfile.JSONModuleObject(
					pooler,
					gatcaddyfile.Pooler,
					"pooler",
					nil,
				),

				ServerAddress: parent.Address,

				ServerUsername: parent.Username,
				ServerPassword: parent.Password,
				ServerDatabase: parent.Database,
			},
		}

		server.Routes = append(server.Routes, gat.RouteConfig{
			Match: gatcaddyfile.JSONModuleObject(
				&matchers.Database{
					Database: name,
				},
				gatcaddyfile.Matcher,
				"matcher",
				nil,
			),
			Handle: gatcaddyfile.JSONModuleObject(
				&p,
				gatcaddyfile.Handler,
				"handler",
				nil,
			),
		})

		if dialers == nil {
			dialers = make(map[string]dialer)
		}
		dialers[name] = dialer{
			Address:  address,
			Username: "pooler",
			Password: password,
			Database: name,
		}
	}

	return
}

func daisyChain(config *gat.Config, control dialer, n int) (dialer, error) {
	for i := 0; i < n; i++ {
		poolConfig := pool.ManagementConfig{}
		var pooler caddy.Module
		if i%2 == 0 {
			pooler = &transaction.Module{
				ManagementConfig: poolConfig,
			}
		} else {
			poolConfig.ServerResetQuery = "DISCARD ALL"
			pooler = &session.Module{
				ManagementConfig: poolConfig,
			}
		}

		server, dialers, err := createServer(control, map[string]caddy.Module{
			"pool": pooler,
		})

		if err != nil {
			return dialer{}, err
		}

		control = dialers["pool"]
		config.Servers = append(config.Servers, server)
	}

	return control, nil
}

func TestTester(t *testing.T) {
	control := pool.Dialer{
		Network:  "tcp",
		Address:  "localhost:5432",
		Username: "postgres",
		Credentials: credentials.Cleartext{
			Username: "postgres",
			Password: "password",
		},
		Database: "postgres",
	}

	config := gat.Config{}

	parent, err := daisyChain(&config, dialer{
		Address:  "localhost:5432",
		Username: "postgres",
		Password: "password",
		Database: "postgres",
	}, 16)
	if err != nil {
		t.Error(err)
		return
	}

	server, dialers, err := createServer(parent, map[string]caddy.Module{
		"transaction": &transaction.Module{},
		"session": &session.Module{
			ManagementConfig: pool.ManagementConfig{
				ServerResetQuery: "discard all",
			},
		},
	})
	if err != nil {
		t.Error(err)
		return
	}

	config.Servers = append(config.Servers, server)

	transactionDialer := pool.Dialer{
		Network:  resolveNetwork(dialers["transaction"].Address),
		Address:  dialers["transaction"].Address,
		Username: dialers["transaction"].Username,
		Credentials: credentials.FromString(
			dialers["transaction"].Username,
			dialers["transaction"].Password,
		),
		Database: "transaction",
	}
	sessionDialer := pool.Dialer{
		Network:  resolveNetwork(dialers["transaction"].Address),
		Address:  dialers["session"].Address,
		Username: dialers["session"].Username,
		Credentials: credentials.FromString(
			dialers["session"].Username,
			dialers["session"].Password,
		),
		Database: "session",
	}

	caddyConfig := caddy.Config{
		AppsRaw: caddy.ModuleMap{
			"pggat": caddyconfig.JSON(config, nil),
		},
	}

	if err = caddy.Run(&caddyConfig); err != nil {
		t.Error(err)
		return
	}

	defer func() {
		_ = caddy.Stop()
	}()

	tester := test.NewTester(test.Config{
		Stress: 8,

		Modes: map[string]pool.Dialer{
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
