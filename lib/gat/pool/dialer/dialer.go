package dialer

import (
	"pggat/lib/fed"
)

type Dialer interface {
	Dial() (fed.Conn, backends.AcceptParams, error)
	Cancel(key [8]byte) error
}
