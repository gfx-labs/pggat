package gatcaddyfile

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

type Unmarshaller func(*caddyfile.Dispenser) (caddy.Module, error)

func (T Unmarshaller) JSONModuleObject(
	d *caddyfile.Dispenser,
	namespace string,
	inlineKey string,
	warnings *[]caddyconfig.Warning,
) (json.RawMessage, error) {
	module, err := T(d)
	if err != nil {
		return nil, err
	}

	rawModuleID := string(module.CaddyModule().ID)
	dotModuleID := strings.TrimPrefix(rawModuleID, namespace)
	moduleID := strings.TrimPrefix(dotModuleID, ".")
	if rawModuleID == dotModuleID || dotModuleID == moduleID {
		return nil, fmt.Errorf(`expected item in namespace "%s" but got "%s"`, namespace, rawModuleID)
	}

	return caddyconfig.JSONModuleObject(
		module,
		inlineKey,
		moduleID,
		warnings,
	), nil
}
