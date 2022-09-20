package cmux

import (
	"log"
	"testing"
)

func TestFsm(t *testing.T) {
	m := NewFsmMux[any, error]()

	m.Register([]string{"set", "shard", "to"}, func(_ any, s []string) error {
		log.Println(s)
		return nil
	})
	m.Register([]string{"set", "sharding", "key", "to"}, func(_ any, s []string) error {
		log.Println(s)
		return nil
	})

	m.Call(nil, []string{"set", "shard", "to", "doggo", "wow", "this", "works"})
	m.Call(nil, []string{"set", "sharding", "key", "to", "doggo", "wow", "this", "works2"})

}
