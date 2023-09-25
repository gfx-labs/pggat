package discovery

import (
	"crypto/tls"
	"time"

	"gfx.cafe/gfx/pggat/lib/bouncer"
	"gfx.cafe/gfx/pggat/lib/util/strutil"
)

type Config struct {
	// ReconcilePeriod is how often the module should check for changes. 0 = disable
	ReconcilePeriod time.Duration

	Discoverer Discoverer

	ServerSSLMode              bouncer.SSLMode
	ServerSSLConfig            *tls.Config
	ServerStartupParameters    map[strutil.CIString]string
	ServerReconnectInitialTime time.Duration
	ServerReconnectMaxTime     time.Duration
	ServerIdleTimeout          time.Duration
	ServerResetQuery           string

	TrackedParameters []strutil.CIString

	PoolMode string
}
