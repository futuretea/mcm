package main

import (
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var _ = run

func TestRunUsesHomeFromEnvironment(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	run([]string{"init"}, strings.NewReader(""), io.Discard, io.Discard)

	if _, err := os.Stat(filepath.Join(home, ".mcm", "config.yaml")); err != nil {
		t.Fatalf("expected config in HOME: %v", err)
	}
}
