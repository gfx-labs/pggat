package flip

type Bank []func() error

func (T *Bank) Queue(fn func() error) {
	*T = append(*T, fn)
}

func (T *Bank) Wait() error {
	if len(*T) == 0 {
		return nil
	}

	if len(*T) == 1 {
		return (*T)[0]()
	}

	ch := make(chan error, len(*T))

	for _, pending := range *T {
		go func(pending func() error) {
			ch <- pending()
		}(pending)
	}

	for i := 0; i < len(*T); i++ {
		err := <-ch
		if err != nil {
			return err
		}
	}

	return nil
}
