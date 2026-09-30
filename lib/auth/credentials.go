package auth

type Credentials interface {
	Credentials()
}

type CleartextClient interface {
	Credentials

	EncodeCleartext() string
}

type CleartextServer interface {
	Credentials

	VerifyCleartext(value string) error
}

type MD5Client interface {
	Credentials

	EncodeMD5(salt [4]byte) string
}

type MD5Server interface {
	Credentials

	VerifyMD5(salt [4]byte, value string) error
}

type SASLMechanism = string

const (
	ScramSHA256 SASLMechanism = "SCRAM-SHA-256"
)

// SASLEncoder is the client side of a SASL exchange. Step(nil) returns the initial response.
type SASLEncoder interface {
	Step(in []byte) ([]byte, error)
	Authenticated() bool
}

// SASLVerifier is the server side of a SASL exchange. Once Done reports true
// after a successful Step, the returned bytes are the final server message.
type SASLVerifier interface {
	Step(in []byte) ([]byte, error)
	Done() bool
}

type SASLClient interface {
	Credentials

	EncodeSASL(mechanisms []SASLMechanism) (SASLMechanism, SASLEncoder, error)
}

type SASLServer interface {
	Credentials

	SupportedSASLMechanisms() []SASLMechanism

	VerifySASL(mechanism SASLMechanism) (SASLVerifier, error)
}
