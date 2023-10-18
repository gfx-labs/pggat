package hybrid

type ErrReadOnly struct{}

func (ErrReadOnly) Error() string {
	return "read only txn"
}

var _ error = ErrReadOnly{}
