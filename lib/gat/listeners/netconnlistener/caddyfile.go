package netconnlistener

import (
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
)

// UnmarshalCaddyfile implements caddyfile.Unmarshaler.
func (T *Listener) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	if !d.NextArg() {
		return d.ArgErr()
	}
	for nesting := d.Nesting(); d.NextBlock(nesting); {
		opt := d.Val()
		switch opt {
		// TODO: configurable client (via name)
		//	case "client":
		//		if err := h.Args(&d.Client) {
		//			return err
		//		}
		case "address":
			if !d.NextArg() {
				return d.ArgErr()
			}
			T.Address = d.Val()
		default:
			return d.Errf("unrecognized option: %s", opt)
		}
	}
	return nil
}
