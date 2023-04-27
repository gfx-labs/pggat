package v0

type work struct {
	source  *Source
	payload any
}

func newWork(source *Source, payload any) *work {
	return &work{
		source:  source,
		payload: payload,
	}
}
