package berr

type Client struct {
	error
}

func MakeClient(err error) Client {
	return Client{err}
}

func (Client) Source() Source {
	return CLIENT
}
