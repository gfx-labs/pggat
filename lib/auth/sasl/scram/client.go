package scram

import (
	"errors"

	"github.com/xdg-go/scram"
)

var ErrUnsupportedMethod = errors.New("unsupported SCRAM method")

const (
	SHA256 = "SCRAM-SHA-256"
)

type Client struct {
	name         string
	conversation *scram.ClientConversation
}

func NewClient(method, username, password string) (*Client, error) {
	var client *scram.Client

	switch method {
	case SHA256:
		var err error
		client, err = scram.SHA256.NewClient(username, password, "")
		if err != nil {
			return nil, err
		}
	default:
		return nil, ErrUnsupportedMethod
	}

	return &Client{
		name:         method,
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
