package berr

type Server struct {
	Error error
}

func (Server) IsServer() bool {
	return true
}

func (Server) IsClient() bool {
	return false
}

func (Server) err() {}

var _ Error = Server{}
