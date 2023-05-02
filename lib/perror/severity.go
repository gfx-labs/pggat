package perror

type Severity string

const (
	ERROR   Severity = "ERROR"
	FATAL   Severity = "FATAL"
	PANIC   Severity = "PANIC"
	WARNING Severity = "WARNING"
	NOTICE  Severity = "NOTICE"
	DEBUG   Severity = "DEBUG"
	INFO    Severity = "INFO"
	LOG     Severity = "LOG"
)
