package test

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"tuxpa.in/a/zlog/log"
)

type Tester struct {
	config Config
}

func NewTester(config Config) *Tester {
	return &Tester{
		config: config,
	}
}

func (T *Tester) Run() error {
	dirEntries, err := os.ReadDir(T.config.TestsPath)
	if err != nil {
		return err
	}

	for _, dirEntry := range dirEntries {
		if dirEntry.IsDir() {
			continue
		}
		path := filepath.Join(T.config.TestsPath, dirEntry.Name())
		log.Printf(`Running test "%s"`, path)

		file, err := os.Open(path)
		if err != nil {
			return err
		}

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()

			fields := strings.Fields(line)
			if len(fields) == 0 {
				continue
			}

			instruction := fields[0]
			arguments := make([]any, 0, len(fields)-1)
			for _, argString := range fields[1:] {
				var arg any
				switch {
				case argString == "true":
					arg = true
				case argString == "false":
					arg = false
				case strings.HasPrefix(argString, `"`), strings.HasPrefix(argString, "`"):
					log.Printf("unquote %s", argString)
					arg, err = strconv.Unquote(argString)
					if err != nil {
						return err
					}
				default:
					return fmt.Errorf(`unknown argument "%s"`, argString)
				}
				arguments = append(arguments, arg)
			}
			log.Print(instruction, " ", arguments)
		}
		if err = scanner.Err(); err != nil {
			return err
		}

		if err = file.Close(); err != nil {
			return err
		}

		log.Print("OK")
	}

	return nil
}
