package frontend

import (
	"pggat2/lib/eqp"
	"pggat2/lib/pnet"
)

type Client interface {
	pnet.ReadWriter
	GetPortal(string) (eqp.Portal, bool)
	GetPreparedStatement(string) (eqp.PreparedStatement, bool)
}
