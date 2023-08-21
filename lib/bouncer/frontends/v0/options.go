package frontends

import (
	"pggat2/lib/bouncer"
	"pggat2/lib/util/strutil"
)

type AcceptOptions struct {
	Pooler                bouncer.Pooler
	AllowedStartupOptions []strutil.CIString
}
