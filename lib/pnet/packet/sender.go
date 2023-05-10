package packet

type Sender interface {
	Send(Type, []byte) error
}
