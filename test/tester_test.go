package test_test

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	_ "net/http/pprof"
	"strconv"
	"testing"
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"

	"gfx.cafe/gfx/pggat/lib/auth/credentials"
	"gfx.cafe/gfx/pggat/lib/bouncer"
	"gfx.cafe/gfx/pggat/lib/gat"
	"gfx.cafe/gfx/pggat/lib/gat/gatcaddyfile"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/pool/pools/basic"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/rewrite_password"
	"gfx.cafe/gfx/pggat/lib/gat/matchers"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
	"gfx.cafe/gfx/pggat/test"
	"gfx.cafe/gfx/pggat/test/tests"

	_ "gfx.cafe/gfx/pggat/lib/fed/listeners/netconnlistener"
)

func wrapConfig(conf basic.Config) basic.Config {
	conf.ServerIdleTimeout = caddy.Duration(time.Second)
	conf.TrackedParameters = []strutil.CIString{
		strutil.MakeCIString("application_name"),
	}
	return conf
}

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

func randPassword() (string, error) {
	var b [20]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(b[:]), nil
}

func createServer(parent dialer, pools map[string]caddy.Module) (server gat.ServerConfig, dialers map[string]dialer, err error) {
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

	for name, pp := range pools {
		p := pool.Module{
			Pool: gatcaddyfile.JSONModuleObject(
				pp,
				gatcaddyfile.Pool,
				"pool",
				nil,
			),
			Recipe: pool.Recipe{
				Dialer: pool.Dialer{
					Address:     parent.Address,
					Username:    parent.Username,
					SSLMode:     bouncer.SSLModeDisable,
					RawPassword: parent.Password,
					Database:    parent.Database,
				},
			},
		}

		server.Routes = append(server.Routes, gat.RouteConfig{
			Match: gatcaddyfile.JSONModuleObject(
				&matchers.Database{
					Database: strutil.Matcher(name),
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
		var poolConfig basic.Config
		if i%2 == 0 {
			poolConfig = wrapConfig(basic.Transaction)
		} else {
			poolConfig = wrapConfig(basic.Session)
		}

		server, dialers, err := createServer(control, map[string]caddy.Module{
			"pool": &basic.Factory{
				Config: poolConfig,
			},
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
		Address:  "localhost:5432",
		Username: "postgres",
		SSLMode:  bouncer.SSLModeDisable,
		Credentials: credentials.Cleartext{
			Username: "postgres",
			Password: "postgres",
		},
		Database: "postgres",
	}

	config := gat.Config{}

	parent, err := daisyChain(&config, dialer{
		Address:  "localhost:5432",
		Username: "postgres",
		Password: "postgres",
		Database: "postgres",
	}, 16)
	if err != nil {
		t.Error(err)
		return
	}

	server, dialers, err := createServer(parent, map[string]caddy.Module{
		"transaction": &basic.Factory{
			Config: wrapConfig(basic.Transaction),
		},
		"session": &basic.Factory{
			Config: wrapConfig(basic.Session),
		},
	})
	if err != nil {
		t.Error(err)
		return
	}

	config.Servers = append(config.Servers, server)

	transactionDialer := pool.Dialer{
		Address:  dialers["transaction"].Address,
		Username: dialers["transaction"].Username,
		Credentials: credentials.FromString(
			dialers["transaction"].Username,
			dialers["transaction"].Password,
		),
		Database: "transaction",
		Parameters: map[strutil.CIString]string{
			strutil.MakeCIString("application_name"): "transaction",
		},
	}
	sessionDialer := pool.Dialer{
		Address:  dialers["session"].Address,
		Username: dialers["session"].Username,
		Credentials: credentials.FromString(
			dialers["session"].Username,
			dialers["session"].Password,
		),
		Database: "session",
		Parameters: map[strutil.CIString]string{
			strutil.MakeCIString("application_name"): "session",
		},
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
