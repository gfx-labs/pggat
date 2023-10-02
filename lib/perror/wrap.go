package perror

func Wrap(err error) Error {
	if err == nil {
		return nil
	}
	if perr, ok := err.(Error); ok {
		return perr
	}
	return New(
		FATAL,
		InternalError,
		err.Error(),
	)
}
