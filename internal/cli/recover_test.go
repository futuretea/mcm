package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRecoverWithoutJournalIsSafeNoOp(t *testing.T) {
	userHome := t.TempDir()
	root := t.TempDir()
	var out bytes.Buffer
	var errOut bytes.Buffer

	exitCode := Run([]string{"--home", root, "recover"}, userHome, strings.NewReader(""), &out, &errOut)

	if exitCode != 0 {
		t.Fatalf("Run(--home %q recover) exit code = %d, want 0; stderr: %s", root, exitCode, errOut.String())
	}
	if got := out.String(); got != "recovered\n" {
		t.Errorf("Run(--home %q recover) stdout = %q, want %q", root, got, "recovered\n")
	}
	if _, err := os.Stat(filepath.Join(userHome, ".mcm")); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("default MCM home exists: stat error = %v, want not exist", err)
	}
	entries, err := os.ReadDir(userHome)
	if err != nil {
		t.Fatalf("read user home: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("user home entries = %v, want none", entries)
	}
}
