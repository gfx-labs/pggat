package bouncer

type SSLMode string

const (
	SSLModeDisable    SSLMode = "disable"
	SSLModeAllow      SSLMode = "allow"
	SSLModePrefer     SSLMode = "prefer"
	SSLModeRequire    SSLMode = "require"
	SSLModeVerifyCa   SSLMode = "verify-ca"
	SSLModeVerifyFull SSLMode = "verify-full"
)

func (T SSLMode) ShouldAttempt() bool {
	switch T {
	case SSLModeDisable:
		return false
	default:
		return true
	}
}

func (T SSLMode) IsRequired() bool {
	switch T {
	case SSLModeDisable, SSLModeAllow, SSLModePrefer, "":
		return false
	default:
		return true
	}
}

func (T SSLMode) VerifyCertificates() bool {
	switch T {
	case SSLModeVerifyCa, SSLModeVerifyFull:
		return true
	default:
		return false
	}
}
