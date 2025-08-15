package gatcaddyfile

import (
	"encoding/json"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"gfx.cafe/gfx/pggat/lib/gat/matchers"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

func init() {
	RegisterDirective(Matcher, "user", func(d *caddyfile.Dispenser, _ *[]caddyconfig.Warning) (caddy.Module, error) {
		if !d.NextArg() {
			return nil, d.ArgErr()
		}
		user := d.Val()
		return &matchers.User{
			User: strutil.Matcher(user),
		}, nil
	})
	RegisterDirective(Matcher, "database", func(d *caddyfile.Dispenser, _ *[]caddyconfig.Warning) (caddy.Module, error) {
		if !d.NextArg() {
			return nil, d.ArgErr()
		}
		database := d.Val()
		return &matchers.Database{
			Database: strutil.Matcher(database),
		}, nil
	})
	RegisterDirective(Matcher, "local_address", func(d *caddyfile.Dispenser, _ *[]caddyconfig.Warning) (caddy.Module, error) {
		if !d.NextArg() {
			return nil, d.ArgErr()
		}
		address := d.Val()
		var network string
		if strings.HasPrefix(address, "/") {
			network = "unix"
		} else {
			network = "tcp"
		}
		return &matchers.LocalAddress{
			Network: network,
			Address: address,
		}, nil
	})
	RegisterDirective(Matcher, "parameter", func(d *caddyfile.Dispenser, _ *[]caddyconfig.Warning) (caddy.Module, error) {
		if !d.NextArg() {
			return nil, d.ArgErr()
		}
		keyValue := d.Val()
		key, value, ok := strings.Cut(keyValue, "=")
		if !ok {
			return nil, d.SyntaxErr("key=value")
		}
		return &matchers.StartupParameter{
			Key:   strutil.MakeCIString(key),
			Value: strutil.Matcher(value),
		}, nil
	})
	RegisterDirective(Matcher, "ssl", func(d *caddyfile.Dispenser, _ *[]caddyconfig.Warning) (caddy.Module, error) {
		var ssl = true
		if d.NextArg() {
			val := d.Val()
			switch val {
			case "true":
				ssl = true
			case "false":
				ssl = false
			default:
				return nil, d.SyntaxErr("boolean")
			}
		}

		return &matchers.SSL{
			SSL: ssl,
		}, nil
	})
	RegisterDirective(Matcher, "not", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		if !d.NextArg() {
			return nil, d.SyntaxErr("matcher directive")
		}

		matcher, err := UnmarshalDirectiveJSONModuleObject(
			d,
			Matcher,
			"matcher",
			warnings,
		)
		if err != nil {
			return nil, err
		}

		return &matchers.Not{
			Not: matcher,
		}, nil
	})
	RegisterDirective(Matcher, "and", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		if !d.NextBlock(d.Nesting()) {
			return nil, d.ArgErr()
		}

		var and []caddy.Module

		for d.Val() != "}" {

			unmarshaller, ok := LookupDirective(Matcher, d.Val())
			if !ok {
				return nil, d.Errf(`unknown matcher "%s"`, d.Val())
			}

			val, err := unmarshaller(d, warnings)
			if err != nil {
				return nil, err
			}
			and = append(and, val)

			if !d.NextLine() {
				return nil, d.EOFErr()
			}
		}

		if len(and) == 0 {
			return nil, nil
		}
		if len(and) == 1 {
			return and[0], nil
		}

		var raw = make([]json.RawMessage, 0, len(and))
		for _, val := range and {
			raw = append(raw, JSONModuleObject(
				val,
				Matcher,
				"matcher",
				warnings,
			))
		}
		return &matchers.And{
			And: raw,
		}, nil
	})
	RegisterDirective(Matcher, "or", func(d *caddyfile.Dispenser, warnings *[]caddyconfig.Warning) (caddy.Module, error) {
		if !d.NextBlock(d.Nesting()) {
			return nil, d.ArgErr()
		}

		var or []caddy.Module

		for d.Val() != "}" {

			unmarshaller, ok := LookupDirective(Matcher, d.Val())
			if !ok {
				return nil, d.Errf(`unknown matcher "%s"`, d.Val())
			}

			val, err := unmarshaller(d, warnings)
			if err != nil {
				return nil, err
			}
			or = append(or, val)

			if !d.NextLine() {
				return nil, d.EOFErr()
			}
		}

		if len(or) == 1 {
			return or[0], nil
		}

		var raw = make([]json.RawMessage, 0, len(or))
		for _, val := range or {
			raw = append(raw, JSONModuleObject(
				val,
				Matcher,
				"matcher",
				warnings,
			))
		}
		return &matchers.Or{
			Or: raw,
		}, nil
	})
}
