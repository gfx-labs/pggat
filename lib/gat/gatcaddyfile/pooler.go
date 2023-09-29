package gatcaddyfile

import (
	"time"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"gfx.cafe/gfx/pggat/lib/gat/pool"
	"gfx.cafe/gfx/pggat/lib/gat/poolers/session"
	"gfx.cafe/gfx/pggat/lib/gat/poolers/transaction"
	"gfx.cafe/gfx/pggat/lib/util/dur"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

var defaultPoolManagementConfig = pool.ManagementConfig{
	ServerIdleTimeout:          dur.Duration(5 * time.Minute),
	ServerReconnectInitialTime: dur.Duration(5 * time.Second),
	ServerReconnectMaxTime:     dur.Duration(1 * time.Minute),
	TrackedParameters: []strutil.CIString{
		strutil.MakeCIString("client_encoding"),
		strutil.MakeCIString("datestyle"),
		strutil.MakeCIString("timezone"),
		strutil.MakeCIString("standard_conforming_strings"),
		strutil.MakeCIString("application_name"),
	},
}

func init() {
	RegisterDirective(Pooler, "transaction", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		module := transaction.Pool{
			ManagementConfig: defaultPoolManagementConfig,
		}

		if !d.NextBlock(d.Nesting()) {
			return &module, nil
		}

		// TODO(garet)
		panic("TODO(garet)")
	})
	RegisterDirective(Pooler, "session", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		module := session.Pool{
			ManagementConfig: defaultPoolManagementConfig,
		}

		if !d.NextBlock(d.Nesting()) {
			return &module, nil
		}

		// TODO(garet)
		panic("TODO(garet)")
	})
}
