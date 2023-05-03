package scram

import (
	"github.com/xdg-go/scram"
)

type Client struct {
	name         string
	conversation *scram.ClientConversation
}

func NewClient(mechanism, username, password string) (*Client, error) {
	var generator scram.HashGeneratorFcn

	switch mechanism {
	case SHA256:
		generator = scram.SHA256
	default:
		return nil, ErrUnsupportedMethod
	}

	client, err := generator.NewClient(username, password, "")
	if err != nil {
		return nil, err
	}

	return &Client{
		name:         mechanism,
		conversation: client.NewConversation(),
	}, nil
}

func (T *Client) Name() string {
	return T.name
}

func (T *Client) InitialResponse() []byte {
	return nil
}

func (T *Client) Continue(bytes []byte) ([]byte, error) {
	msg, err := T.conversation.Step(string(bytes))
	if err != nil {
		return nil, err
	}
	return []byte(msg), nil
}

func (T *Client) Final(bytes []byte) error {
	_, err := T.Continue(bytes)
	return err
}
