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
	for _, test := range tests {
		runner := MakeRunner(T.config, test)
		if err := runner.Run(); err != nil {
			return err
		}
	}
	return nil
}
