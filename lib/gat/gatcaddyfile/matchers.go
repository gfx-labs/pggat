package gatcaddyfile

import (
	"encoding/json"
	"strings"

	"github.com/caddyserver/caddy/v2/caddyconfig"

	"gfx.cafe/gfx/pggat/lib/gat/matchers"
)

func MatcherFromConnectionStrings(strs []string, warnings *[]caddyconfig.Warning) json.RawMessage {
	var or matchers.Or

	for _, str := range strs {
		val := MatcherFromConnectionString(str, warnings)
		if val != nil {
			or.Or = append(or.Or, val)
		}
	}

	if len(or.Or) == 0 {
		return nil
	}
	if len(or.Or) == 1 {
		return or.Or[0]
	}

	return caddyconfig.JSONModuleObject(
		or,
		"matcher",
		"and",
		warnings,
	)
}

// MatcherFromConnectionString converts from the postgres connection string format to a bunch of matchers.
// Example: postgres://user@address:port/database?parameter_key=parameter_value
func MatcherFromConnectionString(str string, warnings *[]caddyconfig.Warning) json.RawMessage {
	// strip optional postgres://
	str = strings.TrimPrefix(str, "postgres://")

	if str == "" {
		return nil
	}

	var and matchers.And

	var parametersString string
	str, parametersString, _ = strings.Cut(str, "?")

	if parametersString != "" {
		var parameters matchers.StartupParameters
		parameters.Parameters = make(map[string]string)
		kvs := strings.Split(parametersString, "&")
		for _, kv := range kvs {
			k, v, _ := strings.Cut(kv, "=")
			parameters.Parameters[k] = v
		}
		and.And = append(
			and.And,
			caddyconfig.JSONModuleObject(
				parameters,
				"matcher",
				"startup_parameters",
				warnings,
			),
		)
	}

	var database matchers.Database
	str, database.Database, _ = strings.Cut(str, "/")
	if database.Database != "" {
		and.And = append(
			and.And,
			caddyconfig.JSONModuleObject(
				database,
				"matcher",
				"database",
				warnings,
			),
		)
	}

	var address = matchers.LocalAddress{
		Network: "tcp",
	}
	var user matchers.User
	var ok bool
	user.User, address.Address, ok = strings.Cut(str, "@")
	if !ok {
		address.Address = user.User
		user.User = ""
	} else if user.User != "" {
		and.And = append(
			and.And,
			caddyconfig.JSONModuleObject(
				user,
				"matcher",
				"user",
				warnings,
			),
		)
	}
	if address.Address != "" {
		and.And = append(
			and.And,
			caddyconfig.JSONModuleObject(
				address,
				"matcher",
				"local_address",
				warnings,
			),
		)
	}

	if len(and.And) == 0 {
		return nil
	}
	if len(and.And) == 1 {
		return and.And[0]
	}

	return caddyconfig.JSONModuleObject(
		and,
		"matcher",
		"and",
		warnings,
	)
}
