package cli

import (
	"bytes"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var _ func([]string, string, io.Reader, io.Writer, io.Writer) int = Run

func TestRunInitUsesExplicitHome(t *testing.T) {
	userHome := t.TempDir()
	root := filepath.Join(t.TempDir(), "mcm-root")
	var out bytes.Buffer
	var errOut bytes.Buffer

	exitCode := Run([]string{"--home", root, "init"}, userHome, strings.NewReader(""), &out, &errOut)

	if exitCode != 0 {
		t.Fatalf("Run(--home %q init) exit code = %d, want 0; stderr: %s", root, exitCode, errOut.String())
	}
	if _, err := os.Stat(filepath.Join(root, "config.yaml")); err != nil {
		t.Fatalf("explicit config does not exist: %v", err)
	}
	if _, err := os.Stat(filepath.Join(userHome, ".mcm")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("default MCM home exists: stat error = %v, want not exist", err)
	}
}

func TestRunRejectsGlobalFlagAfterCommand(t *testing.T) {
	userHome := t.TempDir()
	root := filepath.Join(t.TempDir(), "mcm-root")
	var out bytes.Buffer
	var errOut bytes.Buffer

	exitCode := Run([]string{"init", "--home", root}, userHome, strings.NewReader(""), &out, &errOut)

	if exitCode == 0 {
		t.Fatalf("Run(init --home %q) exit code = 0, want nonzero", root)
	}
}
