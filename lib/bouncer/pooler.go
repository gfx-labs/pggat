package bouncer

import (
	"pggat2/lib/auth"
)

type Pooler interface {
	GetUserCredentials(user, database string) auth.Credentials
}
