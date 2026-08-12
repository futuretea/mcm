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

func TestRunRejectsInvalidDefaultHomeBeforeEveryCommand(t *testing.T) {
	for _, args := range [][]string{
		{"init"},
		{"validate"},
		{"server", "list"},
		{"plan", "--target", "cursor"},
		{"apply", "--target", "cursor", "--yes"},
		{"status", "--target", "cursor"},
		{"recover"},
	} {
		t.Run(args[0], func(t *testing.T) {
			workspace := t.TempDir()
			t.Chdir(workspace)
			var out bytes.Buffer
			var errOut bytes.Buffer
			if exitCode := Run(args, "", strings.NewReader(""), &out, &errOut); exitCode != 2 {
				t.Fatalf("Run(%q) exit code = %d, want 2; stderr: %s", args, exitCode, errOut.String())
			}
			if _, err := os.Lstat(filepath.Join(workspace, ".mcm")); !errors.Is(err, os.ErrNotExist) {
				t.Errorf("MCM root exists after Run(%q): %v", args, err)
			}
		})
	}
}
