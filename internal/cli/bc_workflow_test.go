package cli

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunBCWorkflowWithExternalManifest(t *testing.T) {
	workspace := t.TempDir()
	userHome := filepath.Join(workspace, "user-home")
	mcmRoot := filepath.Join(workspace, "mcm-root")
	externalParent := filepath.Join(workspace, "external")
	if err := os.Mkdir(externalParent, 0o755); err != nil {
		t.Fatalf("create external manifest parent: %v", err)
	}
	if err := os.Chmod(externalParent, 0o755); err != nil {
		t.Fatalf("set external manifest parent mode: %v", err)
	}
	manifestPath := filepath.Join(externalParent, "mcm.yaml")

	run := func(args ...string) (string, string) {
		t.Helper()
		var out bytes.Buffer
		var errOut bytes.Buffer
		if exitCode := Run(args, userHome, strings.NewReader(""), &out, &errOut); exitCode != 0 {
			t.Fatalf("Run(%q) exit code = %d, want 0; stderr: %s", args, exitCode, errOut.String())
		}
		return out.String(), errOut.String()
	}

	globalFlags := []string{"--home", mcmRoot, "--config", manifestPath}
	run(append(globalFlags, "init")...)
	run(append(globalFlags, "server", "add", "--name", "local", "--command", "node")...)
	run(append(globalFlags, "validate")...)
	serverList, _ := run(append(globalFlags, "server", "list")...)
	if !strings.Contains(serverList, "local") {
		t.Fatalf("Run(server list) stdout = %q, want to contain %q", serverList, "local")
	}
	run(append(globalFlags, "plan", "--target", "mcpc")...)
	applyOutput, _ := run(append(globalFlags, "apply", "--target", "mcpc", "--yes")...)
	wantConnectHint := "mcpc connect '" + filepath.Join(mcmRoot, "exports", "mcpc.json") + ":local'"
	run(append(globalFlags, "status", "--target", "mcpc")...)
	if !strings.Contains(applyOutput, wantConnectHint) {
		t.Errorf("Run(apply --target mcpc --yes) stdout = %q, want to contain %q", applyOutput, wantConnectHint)
	}

	for _, path := range []string{
		manifestPath,
		filepath.Join(mcmRoot, "state.json"),
		filepath.Join(mcmRoot, "exports", "mcpc.json"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("stat expected file %q: %v", path, err)
		}
	}
	for _, path := range []string{
		filepath.Join(userHome, ".mcm"),
		filepath.Join(userHome, ".config", "mcp", "mcp_servers.json"),
	} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("unexpected default path %q: stat error = %v, want not exist", path, err)
		}
	}
}
