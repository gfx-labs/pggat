package gatcaddyfile

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

type Unmarshaller func(*caddyfile.Dispenser, *[]caddyconfig.Warning) (caddy.Module, error)

func (T Unmarshaller) JSONModuleObject(
	d *caddyfile.Dispenser,
	namespace string,
	inlineKey string,
	warnings *[]caddyconfig.Warning,
) (json.RawMessage, error) {
	module, err := T(d, warnings)
	if err != nil {
		return nil, err
	}

	return JSONModuleObject(
		module,
		namespace,
		inlineKey,
		warnings,
	), nil
}

func JSONModuleObject(
	module caddy.Module,
	namespace string,
	inlineKey string,
	warnings *[]caddyconfig.Warning,
) json.RawMessage {
	rawModuleID := string(module.CaddyModule().ID)
	dotModuleID := strings.TrimPrefix(rawModuleID, namespace)
	moduleID := strings.TrimPrefix(dotModuleID, ".")
	if rawModuleID == dotModuleID || dotModuleID == moduleID {
		if warnings != nil {
			*warnings = append(*warnings, caddyconfig.Warning{
				Message: fmt.Sprintf(`expected item in namespace "%s" but got "%s"`, namespace, rawModuleID),
			})
		}
		return nil
	}

	return caddyconfig.JSONModuleObject(
		module,
		inlineKey,
		moduleID,
		warnings,
	)
}
