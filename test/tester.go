package test

import (
	"tuxpa.in/a/zlog/log"

	"pggat/test/inst"
)

type Tester struct {
	config Config
}

func NewTester(config Config) *Tester {
	return &Tester{
		config: config,
	}
}

func (T *Tester) run(test Test) error {
	for _, v := range test.Instructions {
		switch i := v.(type) {
		case inst.SimpleQuery:
			log.Println("run", i)
		}
	}
	return nil // TODO(garet)
}

func (T *Tester) Run(tests ...Test) error {
	for _, test := range tests {
		if err := T.run(test); err != nil {
			return err
		}
	}
	return nil
}
