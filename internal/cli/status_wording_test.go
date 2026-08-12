package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/futuretea/mcm/internal/app"
	"github.com/futuretea/mcm/internal/manifest"
)

func TestRunStatusExplainsFileOnlyTargets(t *testing.T) {
	userHome := filepath.Join(t.TempDir(), "user-home")
	root := filepath.Join(t.TempDir(), "mcm-root")
	location := manifest.NewLocation(userHome, root, "")

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
	if err := os.MkdirAll(filepath.Join(userHome, ".qoder"), 0o700); err != nil {
		t.Fatalf("create Qoder directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(userHome, ".config", "mcp"), 0o700); err != nil {
		t.Fatalf("create mcp-cli directory: %v", err)
	}
	if _, err := app.New(userHome, location).Apply([]string{"qoder-ide", "mcp-cli"}, ""); err != nil {
		t.Fatalf("Apply([qoder-ide mcp-cli], \"\") error = %v", err)
	}

	for _, test := range []struct {
		target string
		want   string
	}{
		{target: "qoder-ide", want: "file-only"},
		{target: "mcp-cli", want: "mcp-cli --config"},
	} {
		t.Run(test.target, func(t *testing.T) {
			out.Reset()
			errOut.Reset()
			if exitCode := Run([]string{"--home", root, "status", "--target", test.target}, userHome, strings.NewReader(""), &out, &errOut); exitCode != 0 {
				t.Fatalf("Run(--home %q status --target %s) exit code = %d, want 0; stderr: %s", root, test.target, exitCode, errOut.String())
			}
			if !strings.Contains(out.String(), test.want) {
				t.Errorf("Run(--home %q status --target %s) stdout = %q, want to contain %q", root, test.target, out.String(), test.want)
			}
		})
	}
}
