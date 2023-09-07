package pool

import (
	"github.com/google/uuid"

	"pggat2/lib/fed"
	"pggat2/lib/middleware/middlewares/eqp"
	"pggat2/lib/middleware/middlewares/ps"
	"pggat2/lib/util/strutil"
)

type Server struct {
}

func (T *Server) GetID() uuid.UUID {

}

func (T *Server) GetConn() fed.Conn {

}

func (T *Server) GetEQP() *eqp.Server {

}

func (T *Server) GetPS() *ps.Server {

}

func (T *Server) TransactionComplete() {

}

func (T *Server) GetInitialParameters() map[strutil.CIString]string {

}

func (T *Server) SetState(state State, peer uuid.UUID) {

}
