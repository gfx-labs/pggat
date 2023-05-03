package perror

func WrapError(err error) Error {
	if err == nil {
		return nil
	}
	return New(
		FATAL,
		InternalError,
		err.Error(),
	)
}
