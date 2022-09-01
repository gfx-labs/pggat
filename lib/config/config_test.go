package config

import (
	"os"
	"testing"

	"github.com/BurntSushi/toml"
)

func TestParseToml(t *testing.T) {
	var c Global
	bts, err := os.ReadFile("./config_data.toml")
	if err != nil {
		t.Error(err)
	}
	err = toml.Unmarshal(bts, &c)
	if err != nil {
		t.Error(err)
	}

	// TODO: write the rest of this test
	if len(c.Pools) != 2 {
		t.Errorf("expect 2 pool, got %d", len(c.Pools))
	}
	if c.General.Host != "0.0.0.0" {
		t.Errorf("expect host %s, got %s", "0.0.0.0", c.General.Host)
	}
}
