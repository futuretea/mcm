package cli

import (
	"bytes"
	"reflect"
	"path/filepath"
	"strings"
	"testing"

	"github.com/futuretea/mcm/internal/manifest"
)

func TestRunServerAddTTYPartialFlagsPromptForMissingStdioFields(t *testing.T) {
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
	input := terminalInput{strings.NewReader("stdio\nnode\n--inspect\n\n")}
	if exitCode := Run([]string{"--home", root, "server", "add", "--name", "local"}, userHome, input, &out, &errOut); exitCode != 0 {
		t.Fatalf("Run(server add --name local) exit code = %d, want 0; stderr: %s", exitCode, errOut.String())
	}

	config, err := location.Load()
	if err != nil {
		t.Fatalf("load manifest: %v", err)
	}
	server, ok := config.Servers["local"]
	if !ok {
		t.Fatal("manifest does not contain the supplied server name")
	}
	if server.Transport != manifest.TransportStdio {
		t.Errorf("server transport = %q, want %q", server.Transport, manifest.TransportStdio)
	}
	if server.Command != "node" {
		t.Errorf("server command = %q, want %q", server.Command, "node")
	}
	if !reflect.DeepEqual(server.Args, []string{"--inspect"}) {
		t.Errorf("server args = %#v, want %#v", server.Args, []string{"--inspect"})
	}

	for _, prompt := range []string{"transport (stdio/streamable-http): ", "command: ", "arg (blank to finish): "} {
		if !strings.Contains(out.String(), prompt) {
			t.Errorf("stdout = %q, want prompt %q", out.String(), prompt)
		}
	}
	if strings.Contains(out.String(), "name: ") {
		t.Errorf("stdout = %q, must not prompt for a supplied name", out.String())
	}
}
