package perror

type Error interface {
	Severity() Severity
	Code() Code
	Message() string
	Extra() []ExtraField
	String() string
	Error() string
}
