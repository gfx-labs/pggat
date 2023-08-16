package frontends

import (
	"pggat2/lib/auth"
	"pggat2/lib/util/strutil"
)

type Acceptor interface {
	GetUserCredentials(user, database string) auth.Credentials
	IsStartupParameterAllowed(parameter strutil.CIString) bool
}
