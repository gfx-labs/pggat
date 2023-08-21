package bouncer

import (
	"pggat2/lib/auth"
)

type Pooler interface {
	GetUserCredentials(user, database string) auth.Credentials
	Cancel(cancellationKey [8]byte)
}
