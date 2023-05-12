package bouncers

import "pggat2/lib/perror"

type Error interface {
	bounceError()
}

type ClientError struct {
	Error perror.Error
}

func makeClientError(err perror.Error) ClientError {
	return ClientError{
		Error: err,
	}
}

func wrapClientError(err error) ClientError {
	return makeClientError(perror.Wrap(err))
}

func (ClientError) bounceError() {}

type ServerError struct {
	Error error
}

func makeServerError(err error) ServerError {
	return ServerError{
		Error: err,
	}
}

func (ServerError) bounceError() {}
