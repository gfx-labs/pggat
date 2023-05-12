package berr

type Client struct{}

func (Client) err() {}

var _ Error = Client{}
