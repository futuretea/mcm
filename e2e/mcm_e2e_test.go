package e2e

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type mcmRunner func(testing.TB, ...string) (string, string)

type target struct {
	name string
	path string
}

func TestDockerCLIWorkflow(t *testing.T) {
	if os.Getenv("MCM_E2E_IN_DOCKER") != "1" {
		t.Skip("set MCM_E2E_IN_DOCKER=1 to run the Docker E2E workflow")
	}
	run, mcmRoot, workspace, userHome := setupDockerWorkflow(t)
	initializeManifest(t, run, mcmRoot)
	assertTargets(t, run, mcmRoot, workspace, userHome)
	if stdout, _ := run(t, "--home", mcmRoot, "recover"); stdout != "recovered\n" {
		t.Fatalf("mcm recover stdout = %q, want recovered", stdout)
	}
}

func setupDockerWorkflow(t *testing.T) (mcmRunner, string, string, string) {
	t.Helper()
	binary := filepath.Join(t.TempDir(), "mcm")
	build := exec.Command("go", "build", "-o", binary, "./cmd/mcm")
	build.Dir = repositoryRoot(t)
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("go build ./cmd/mcm: %v\n%s", err, output)
	}
	workspace := t.TempDir()
	userHome := filepath.Join(workspace, "home")
	if err := os.Mkdir(userHome, 0o700); err != nil {
		t.Fatalf("create isolated HOME: %v", err)
	}
	return newMCMRunner(binary, userHome), filepath.Join(workspace, "mcm-root"), workspace, userHome
}

func initializeManifest(t *testing.T, run mcmRunner, mcmRoot string) {
	t.Helper()
	if stdout, _ := run(t, "--home", mcmRoot, "init"); stdout != "initialized\n" {
		t.Fatalf("mcm init stdout = %q, want initialized", stdout)
	}
	if stdout, _ := run(t, "--home", mcmRoot, "server", "add", "--name", "local", "--command", "node", "--arg", "x"); stdout != "saved\n" {
		t.Fatalf("mcm server add local stdout = %q, want saved", stdout)
	}
	if stdout, _ := run(t, "--home", mcmRoot, "server", "add", "--name", "remote", "--url", "https://example.test/mcp"); stdout != "saved\n" {
		t.Fatalf("mcm server add remote stdout = %q, want saved", stdout)
	}
	if stdout, _ := run(t, "--home", mcmRoot, "validate"); stdout != "valid\n" {
		t.Fatalf("mcm validate stdout = %q, want valid", stdout)
	}
	if stdout, _ := run(t, "--home", mcmRoot, "server", "list"); !strings.Contains(stdout, "local\n") || !strings.Contains(stdout, "remote\n") {
		t.Fatalf("mcm server list stdout = %q, want local and remote", stdout)
	}
}

func assertTargets(t *testing.T, run mcmRunner, mcmRoot, workspace, userHome string) {
	t.Helper()
	for _, target := range targets(workspace, userHome, mcmRoot) {
		t.Run(target.name, func(t *testing.T) {
			assertTarget(t, run, mcmRoot, target)
		})
	}
}

func targets(workspace, userHome, mcmRoot string) []target {
	return []target{
		{"cursor", filepath.Join(userHome, ".cursor", "mcp.json")},
		{"claude-code", filepath.Join(userHome, ".claude.json")},
		{"codex", filepath.Join(userHome, ".codex", "config.toml")},
		{"vs-code", filepath.Join(workspace, "vscode", "mcp.json")},
		{"qoder-cli", filepath.Join(userHome, ".qoder", "settings.json")},
		{"qoder-ide", filepath.Join(userHome, ".qoder", "mcp.json")},
		{"opencode", filepath.Join(userHome, ".config", "opencode", "opencode.json")},
		{"mcp-cli", filepath.Join(userHome, ".config", "mcp", "mcp_servers.json")},
		{"mcpc", filepath.Join(mcmRoot, "exports", "mcpc.json")},
	}
}

func assertTarget(t *testing.T, run mcmRunner, mcmRoot string, target target) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(target.path), 0o700); err != nil {
		t.Fatalf("create %s parent: %v", target.name, err)
	}
	args := targetArgs(target)
	planStdout, _ := run(t, append([]string{"--home", mcmRoot, "plan"}, args...)...)
	if !strings.Contains(planStdout, target.name) || !strings.Contains(planStdout, target.path) {
		t.Fatalf("mcm plan %s stdout = %q, want target and path", target.name, planStdout)
	}
	applyStdout, applyStderr := run(t, append(append([]string{"--home", mcmRoot, "apply"}, args...), "--yes")...)
	assertApplyOutput(t, target.name, applyStdout, applyStderr)
	statusStdout, _ := run(t, append([]string{"--home", mcmRoot, "status"}, args...)...)
	if !strings.Contains(statusStdout, target.name) || !strings.Contains(statusStdout, target.path) || !strings.Contains(statusStdout, "synchronized") {
		t.Fatalf("mcm status %s stdout = %q, want synchronized target and path", target.name, statusStdout)
	}
	assertRegularNonEmptyFile(t, target.path)
	if target.name == "cursor" {
		assertCursorServers(t, target.path)
	}
}

func targetArgs(target target) []string {
	args := []string{"--target", target.name}
	if target.name == "vs-code" {
		return append(args, "--path", target.path)
	}
	return args
}

func assertApplyOutput(t *testing.T, target, stdout, stderr string) {
	t.Helper()
	planIndex := strings.Index(stdout, target+"  ")
	appliedIndex := strings.LastIndex(stdout, "applied")
	if planIndex < 0 || appliedIndex < planIndex {
		t.Fatalf("mcm apply --yes %s stdout = %q, want plan before final applied", target, stdout)
	}
	if strings.Contains(strings.ToLower(stdout), "warning") {
		t.Fatalf("mcm apply --yes %s stdout = %q, want no warning", target, stdout)
	}
	if target == "cursor" {
		assertCursorWarnings(t, stderr)
	}
}

func newMCMRunner(binary, home string) mcmRunner {
	return func(t testing.TB, args ...string) (string, string) {
		t.Helper()
		command := exec.Command(binary, args...)
		command.Env = withHome(home)
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		command.Stdout = &stdout
		command.Stderr = &stderr
		if err := command.Run(); err != nil {
			t.Fatalf("mcm %s: %v\nstdout:\n%s\nstderr:\n%s", strings.Join(args, " "), err, stdout.String(), stderr.String())
		}
		return stdout.String(), stderr.String()
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	directory, err := os.Getwd()
	if err != nil {
		t.Fatalf("get test working directory: %v", err)
	}
	for {
		if _, err := os.Stat(filepath.Join(directory, "go.mod")); err == nil {
			return directory
		} else if !os.IsNotExist(err) {
			t.Fatalf("inspect repository root: %v", err)
		}
		parent := filepath.Dir(directory)
		if parent == directory {
			t.Fatal("could not find repository go.mod")
		}
		directory = parent
	}
}

func withHome(home string) []string {
	environment := make([]string, 0, len(os.Environ())+1)
	for _, value := range os.Environ() {
		if !strings.HasPrefix(value, "HOME=") {
			environment = append(environment, value)
		}
	}
	return append(environment, "HOME="+home)
}

func assertCursorWarnings(t *testing.T, stderr string) {
	t.Helper()
	warnings := strings.ToLower(stderr)
	if !strings.Contains(warnings, "external writers") || !strings.Contains(warnings, "after final verification") {
		t.Fatalf("cursor apply stderr = %q, want external-writer race warning", stderr)
	}
	if !strings.Contains(warnings, "reserialized") || !strings.Contains(warnings, "formatting and comments may change") {
		t.Fatalf("cursor apply stderr = %q, want reserialization warning", stderr)
	}
}

func assertRegularNonEmptyFile(t *testing.T, path string) {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat output %s: %v", path, err)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 {
		t.Fatalf("output %s mode=%s size=%d, want non-empty regular file", path, info.Mode(), info.Size())
	}
}

func assertCursorServers(t *testing.T, path string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read Cursor output: %v", err)
	}
	var document struct {
		MCPServers map[string]struct {
			Type    string `json:"type"`
			Command string `json:"command"`
			URL     string `json:"url"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("parse Cursor JSON: %v\n%s", err, data)
	}
	if local, ok := document.MCPServers["local"]; !ok || local.Type != "stdio" || local.Command != "node" {
		t.Fatalf("Cursor local server = %#v, want stdio node", local)
	}
	if remote, ok := document.MCPServers["remote"]; !ok || remote.URL != "https://example.test/mcp" {
		t.Fatalf("Cursor remote server = %#v, want streamable HTTP URL", remote)
	}
}
