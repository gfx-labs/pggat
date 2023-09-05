package dialer

import (
	"pggat2/lib/bouncer/backends/v0"
	"pggat2/lib/fed"
)

type Dialer interface {
	Dial() (fed.Conn, backends.AcceptParams, error)
	Cancel(cancelKey [8]byte) error
}
