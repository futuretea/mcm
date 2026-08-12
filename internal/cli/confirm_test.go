package cli

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

type terminalInput struct {
	*strings.Reader
}

func (terminalInput) Stat() (os.FileInfo, error) {
	return terminalFileInfo{}, nil
}

type terminalFileInfo struct{}

func (terminalFileInfo) Name() string       { return "terminal" }
func (terminalFileInfo) Size() int64        { return 0 }
func (terminalFileInfo) Mode() fs.FileMode  { return fs.ModeCharDevice }
func (terminalFileInfo) ModTime() time.Time { return time.Time{} }
func (terminalFileInfo) IsDir() bool        { return false }
func (terminalFileInfo) Sys() any           { return nil }

func TestRunApplyInteractiveDeclineDoesNotWrite(t *testing.T) {
	userHome := t.TempDir()
	root := filepath.Join(t.TempDir(), "mcm-root")
	targetPath := filepath.Join(userHome, ".cursor", "mcp.json")
	var out bytes.Buffer
	var errOut bytes.Buffer

	if exitCode := Run([]string{"--home", root, "init"}, userHome, strings.NewReader(""), &out, &errOut); exitCode != 0 {
		t.Fatalf("Run(--home %q init) exit code = %d, want 0; stderr: %s", root, exitCode, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if exitCode := Run([]string{"--home", root, "server", "add", "--name", "local", "--command", "node"}, userHome, strings.NewReader(""), &out, &errOut); exitCode != 0 {
		t.Fatalf("Run(--home %q server add) exit code = %d, want 0; stderr: %s", root, exitCode, errOut.String())
	}

	if err := os.Mkdir(filepath.Dir(targetPath), 0o700); err != nil {
		t.Fatalf("create cursor directory: %v", err)
	}

	out.Reset()
	errOut.Reset()
	exitCode := Run([]string{"--home", root, "apply", "--target", "cursor"}, userHome, terminalInput{strings.NewReader("no\n")}, &out, &errOut)

	if exitCode == 0 {
		t.Fatal("Run(apply --target cursor) exit code = 0, want nonzero after declining confirmation")
	}
	if !strings.Contains(errOut.String(), "confirmation cancelled") {
		t.Errorf("Run(apply --target cursor) stderr = %q, want to contain %q", errOut.String(), "confirmation cancelled")
	}
	if _, err := os.Stat(targetPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("cursor target exists after declined apply: stat %q error = %v, want not exist", targetPath, err)
	}
}
