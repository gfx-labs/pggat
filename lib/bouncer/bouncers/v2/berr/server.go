package berr

type Server struct {
	error
}

func MakeServer(err error) Server {
	return Server{err}
}

func (Server) err() {}
