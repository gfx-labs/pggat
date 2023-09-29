package gatcaddyfile

import (
	"fmt"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"gfx.cafe/gfx/pggat/lib/gat/handlers/pgbouncer"
	"gfx.cafe/gfx/pggat/lib/gat/handlers/require_ssl"
)

var handlers map[string]Unmarshaller

func RegisterHandlerDirective(directive string, unmarshaller Unmarshaller) {
	if _, ok := handlers[directive]; ok {
		panic(fmt.Sprintf(`duplicate handler directive "%s"`, directive))
	}
	if handlers == nil {
		handlers = make(map[string]Unmarshaller)
	}
	handlers[directive] = unmarshaller
}

func init() {
	RegisterHandlerDirective("require_ssl", func(d *caddyfile.Dispenser) (caddy.Module, error) {
		var ssl = true
		if d.Next() {
			switch d.Val() {
			case "true":
				ssl = true
			case "false":
				ssl = false
			default:
				return nil, d.SyntaxErr("boolean")
			}
		}
		return &require_ssl.Module{
			SSL: ssl,
		}, nil
	})
	RegisterHandlerDirective("pgbouncer", func(d *caddyfile.Dispenser) (caddy.Module, error) {
		var config = "pgbouncer.ini"
		if d.Next() {
			config = d.Val()
		}
		return &pgbouncer.Module{
			ConfigFile: config,
		}, nil
	})
}
