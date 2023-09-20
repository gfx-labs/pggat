package credentials

import (
	"github.com/xdg-go/scram"

	"pggat/lib/auth"
)

type ConversationScramEncoder struct {
	conversation *scram.ClientConversation
}

func MakeCleartextScramEncoder(username, password string, hashGenerator scram.HashGeneratorFcn) (auth.SASLEncoder, error) {
	client, err := hashGenerator.NewClient(username, password, "")
	if err != nil {
		return nil, err
	}

	return ConversationScramEncoder{
		conversation: client.NewConversation(),
	}, nil
}

func (T ConversationScramEncoder) Write(bytes []byte) ([]byte, error) {
	msg, err := T.conversation.Step(string(bytes))
	if err != nil {
		return nil, err
	}
	return []byte(msg), nil
}

var _ auth.SASLEncoder = ConversationScramEncoder{}

type ConversationScramVerifier struct {
	conversation *scram.ServerConversation
}

func MakeCleartextScramVerifier(username, password string, hashGenerator scram.HashGeneratorFcn) (auth.SASLVerifier, error) {
	client, err := hashGenerator.NewClient(username, password, "")
	if err != nil {
		return nil, err
	}

	kf := scram.KeyFactors{
		Iters: 4096,
	}
	stored := client.GetStoredCredentials(kf)

	return MakeStoredCredentialsScramVerifier(stored, hashGenerator)
}

func MakeStoredCredentialsScramVerifier(credentials scram.StoredCredentials, hashGenerator scram.HashGeneratorFcn) (auth.SASLVerifier, error) {
	server, err := hashGenerator.NewServer(
		func(string) (scram.StoredCredentials, error) {
			return credentials, nil
		},
	)
	if err != nil {
		return nil, err
	}

	return ConversationScramVerifier{
		conversation: server.NewConversation(),
	}, nil
}

func (T ConversationScramVerifier) Write(bytes []byte) ([]byte, error) {
	msg, err := T.conversation.Step(string(bytes))
	if err != nil {
		return nil, err
	}

	if T.conversation.Done() {
		// check if conversation params are valid
		if !T.conversation.Valid() {
			return nil, auth.ErrFailed
		}

		T.conversation.AuthzID()

		// done
		return []byte(msg), auth.ErrSASLComplete
	}

	// there is more
	return []byte(msg), nil
}

var _ auth.SASLVerifier = ConversationScramVerifier{}
