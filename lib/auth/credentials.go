package auth

type Credentials interface {
	GetUsername() string
}

type Cleartext interface {
	Credentials

	EncodeCleartext() string
	VerifyCleartext(value string) error
}

type MD5 interface {
	Credentials

	EncodeMD5(salt [4]byte) string
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

type SASL interface {
	Credentials

	SupportedSASLMechanisms() []SASLMechanism

	EncodeSASL(mechanisms []SASLMechanism) (SASLMechanism, SASLEncoder, error)
	VerifySASL(mechanism SASLMechanism) (SASLVerifier, error)
}
