package cli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/futuretea/mcm/internal/manifest"
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

func TestRunServerAddTTYPromptsOnlyForMissingURLServerName(t *testing.T) {
	userHome := t.TempDir()
	root := filepath.Join(t.TempDir(), "mcm-root")
	location := manifest.NewLocation(userHome, root, "")
	var out bytes.Buffer
	var errOut bytes.Buffer

	if exitCode := Run([]string{"--home", root, "init"}, userHome, strings.NewReader(""), &out, &errOut); exitCode != 0 {
		t.Fatalf("Run(--home %q init) exit code = %d, want 0; stderr: %s", root, exitCode, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if exitCode := Run(
		[]string{"--home", root, "server", "add", "--url", "https://example.test/mcp"},
		userHome,
		terminalInput{strings.NewReader("remote\n")},
		&out,
		&errOut,
	); exitCode != 0 {
		t.Fatalf("Run(server add --url) exit code = %d, want 0; stderr: %s", exitCode, errOut.String())
	}

	config, err := location.Load()
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	server := config.Servers["remote"]
	if server.Transport != manifest.TransportStreamableHTTP {
		t.Errorf("server transport = %q, want %q", server.Transport, manifest.TransportStreamableHTTP)
	}
	if server.URL != "https://example.test/mcp" {
		t.Errorf("server URL = %q, want preserved URL", server.URL)
	}
	for _, unexpected := range []string{"transport (stdio/streamable-http): ", "command: ", "arg (blank to finish): "} {
		if strings.Contains(out.String(), unexpected) {
			t.Errorf("stdout = %q, must not prompt %q", out.String(), unexpected)
		}
	}
	if !strings.Contains(out.String(), "name: ") {
		t.Errorf("stdout = %q, want name prompt", out.String())
	}
}

func TestRunServerAddTTYPromptsForStreamableHTTPServer(t *testing.T) {
	userHome := t.TempDir()
	root := filepath.Join(t.TempDir(), "mcm-root")
	location := manifest.NewLocation(userHome, root, "")
	var out bytes.Buffer
	var errOut bytes.Buffer

	if exitCode := Run([]string{"--home", root, "init"}, userHome, strings.NewReader(""), &out, &errOut); exitCode != 0 {
		t.Fatalf("Run(--home %q init) exit code = %d, want 0; stderr: %s", root, exitCode, errOut.String())
	}

	out.Reset()
	errOut.Reset()
	if exitCode := Run(
		[]string{"--home", root, "server", "add"},
		userHome,
		terminalInput{strings.NewReader("streamable-http\nremote\nhttps://example.test/mcp\n")},
		&out,
		&errOut,
	); exitCode != 0 {
		t.Fatalf("Run(server add) exit code = %d, want 0; stderr: %s", exitCode, errOut.String())
	}

	config, err := location.Load()
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	server := config.Servers["remote"]
	if server.Transport != manifest.TransportStreamableHTTP || server.URL != "https://example.test/mcp" {
		t.Errorf("server = %#v, want streamable HTTP remote", server)
	}
	if strings.Index(out.String(), "name: ") > strings.Index(out.String(), "url: ") {
		t.Errorf("stdout = %q, want name prompt before URL prompt", out.String())
	}
}
