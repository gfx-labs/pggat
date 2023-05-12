package berr

type Server struct{}

func (Server) err() {}

var _ Error = Server{}
