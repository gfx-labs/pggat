package cmux

import (
	"log"
	"testing"
)

func TestFsm(t *testing.T) {
	m := NewFsmMux[error]()

	m.Register([]string{"set", "shard", "to"}, func(s []string) error {
		log.Println(s)
		return nil
	})
	m.Register([]string{"set", "sharding", "key", "to"}, func(s []string) error {
		log.Println(s)
		return nil
	})

	m.Call([]string{"set", "shard", "to", "doggo", "wow", "this", "works"})

}
