package main

import (
	"flag"

	"pggat/test"
)

func main() {
	var config test.Config

	flag.StringVar(&config.TestsPath, "path", "test/tests", "path to the tests to run")

	flag.BoolVar(&config.Offline, "offline", false, "if true, existing test results will be used")

	flag.StringVar(&config.Host, "host", "localhost", "postgres host")
	flag.IntVar(&config.Port, "port", 5432, "postgres port")
	flag.StringVar(&config.Database, "database", "pggat", "postgres database")
	flag.StringVar(&config.User, "user", "postgres", "postgres user")
	flag.StringVar(&config.Password, "password", "password", "postgres password")

	flag.Parse()

	tester := test.NewTester(config)
	if err := tester.Run(); err != nil {
		panic(err)
	}
}
