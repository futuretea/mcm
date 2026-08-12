package main

import (
	"bytes"
	"errors"
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

func TestRunRejectsInvalidHomeWithoutMutatingWorkingDirectory(t *testing.T) {
	for _, home := range []string{"", "relative-home"} {
		t.Run(home, func(t *testing.T) {
			workspace := t.TempDir()
			t.Chdir(workspace)
			t.Setenv("HOME", home)
			var errOut bytes.Buffer

			if exitCode := run([]string{"init"}, strings.NewReader(""), io.Discard, &errOut); exitCode == 0 {
				t.Fatalf("run(init) exit code = 0, want failure when HOME=%q", home)
			}

			path := filepath.Join(workspace, ".mcm")
			if home != "" {
				path = filepath.Join(workspace, home, ".mcm")
			}
			if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("MCM root %q exists after invalid HOME: %v", path, err)
			}
		})
	}
}
