package scram

import (
	"errors"

	"github.com/xdg-go/scram"
)

type Server struct {
	conversation *scram.ServerConversation
}

func NewServer(mechanism, username, password string) (*Server, error) {
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

	kf := scram.KeyFactors{
		Iters: 1,
	}
	stored := client.GetStoredCredentials(kf)

	server, err := generator.NewServer(
		func(string) (scram.StoredCredentials, error) {
			return stored, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return &Server{
		conversation: server.NewConversation(),
	}, nil
}

func (T *Server) InitialResponse(bytes []byte) ([]byte, bool, error) {
	return T.Continue(bytes)
}

func (T *Server) Continue(bytes []byte) ([]byte, bool, error) {
	msg, err := T.conversation.Step(string(bytes))
	if err != nil {
		return nil, false, err
	}

	if T.conversation.Done() {
		// check if conversation params are valid
		if !T.conversation.Valid() {
			return nil, false, errors.New("SCRAM conversation failed")
		}

		// done
		return []byte(msg), true, nil
	}

	// there is more
	return []byte(msg), false, nil
}
