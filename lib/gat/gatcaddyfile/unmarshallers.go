package gatcaddyfile

import (
	"encoding/json"
	"fmt"

	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"

	"gfx.cafe/gfx/pggat/lib/util/maps"
)

const (
	Discoverer         = "pggat.handlers.discovery.discoverers"
	DigitaloceanFilter = "pggat.handlers.discovery.discoverers.digitalocean.filters"
	Handler            = "pggat.handlers"
	Matcher            = "pggat.matchers"
	Pool               = "pggat.handlers.pool.pools"
	Pooler             = "pggat.handlers.pool.poolers"
	SSLServer          = "pggat.ssl.servers"
	SSLClient          = "pggat.ssl.clients"
)

var unmarshallers maps.TwoKey[string, string, Unmarshaller]

func RegisterDirective(namespace, directive string, unmarshaller Unmarshaller) {
	if _, ok := unmarshallers.Load(namespace, directive); ok {
		panic(fmt.Sprintf(`directive "%s" already exists`, directive))
	}
	unmarshallers.Store(namespace, directive, unmarshaller)
}

func LookupDirective(namespace, directive string) (Unmarshaller, bool) {
	return unmarshallers.Load(namespace, directive)
}

func UnmarshalDirectiveJSONModuleObject(
	d *caddyfile.Dispenser,
	namespace string,
	inlineKey string,
	warnings *[]caddyconfig.Warning,
) (json.RawMessage, error) {
	unmarshaller, ok := LookupDirective(namespace, d.Val())
	if !ok {
		return nil, d.Errf(`unknown directive in %s: "%s"`, namespace, d.Val())
	}

	return unmarshaller.JSONModuleObject(
		d,
		namespace,
		inlineKey,
		warnings,
	)
}
