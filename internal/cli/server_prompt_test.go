package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunServerAddPromptsForStdioServer(t *testing.T) {
	userHome := t.TempDir()
	root := filepath.Join(t.TempDir(), "mcm-root")
	var out bytes.Buffer
	var errOut bytes.Buffer

	if exitCode := Run([]string{"--home", root, "init"}, userHome, strings.NewReader(""), &out, &errOut); exitCode != 0 {
		t.Fatalf("Run(--home %q init) exit code = %d, want 0; stderr: %s", root, exitCode, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if exitCode := Run([]string{"--home", root, "server", "add"}, userHome, terminalInput{strings.NewReader("stdio\nlocal\nnode\n\n")}, &out, &errOut); exitCode != 0 {
		t.Fatalf("Run(--home %q server add) exit code = %d, want 0; stderr: %s", root, exitCode, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if exitCode := Run([]string{"--home", root, "server", "list"}, userHome, strings.NewReader(""), &out, &errOut); exitCode != 0 {
		t.Fatalf("Run(--home %q server list) exit code = %d, want 0; stderr: %s", root, exitCode, errOut.String())
	}
	if !strings.Contains(out.String(), "local") {
		t.Fatalf("Run(--home %q server list) stdout = %q, want to contain %q", root, out.String(), "local")
	}
}
