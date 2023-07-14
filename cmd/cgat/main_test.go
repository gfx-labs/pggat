package main

import (
	"testing"
	"time"
)

func Test(t *testing.T) {
	go func() {
		main()
	}()
	time.Sleep(10 * time.Second)
}
