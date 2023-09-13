package test

type Tester struct {
	config Config
}

func NewTester(config Config) *Tester {
	return &Tester{
		config: config,
	}
}

func (T *Tester) Run(tests ...Test) error {
	var errors []error
	for _, test := range tests {
		runner := MakeRunner(T.config, test)
		if err := runner.Run(); err != nil {
			errors = append(errors, ErrorIn{
				Name: test.Name,
				Err:  err,
			})
		}
	}
	if len(errors) > 0 {
		return Errors(errors)
	}
	return nil
}
