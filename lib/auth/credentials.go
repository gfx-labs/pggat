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

type SASLEncoder interface {
	Write([]byte) ([]byte, error)
}

type SASLVerifier interface {
	Write(bytes []byte) ([]byte, error)
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
