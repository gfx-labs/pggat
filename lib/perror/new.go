package perror

func New(severity Severity, code Code, message string, extra ...ExtraField) Error {
	return err{
		severity: severity,
		code:     code,
		message:  message,
		extra:    extra,
	}
}

type err struct {
	severity Severity
	code     Code
	message  string
	extra    []ExtraField
}

func (T err) Severity() Severity {
	return T.severity
}

func (T err) Code() Code {
	return T.code
}

func (T err) Message() string {
	return T.message
}

func (T err) Extra() []ExtraField {
	return T.extra
}

func (T err) String() string {
	return string(T.severity) + ": " + T.message
}

var _ Error = err{}
