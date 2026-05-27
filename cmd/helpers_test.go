package cmd

import (
	"testing"
)

func TestConfigDir(t *testing.T) {
	dir := configDir()
	if dir == "" {
		t.Fatal("Expected non-empty config dir")
	}
}
