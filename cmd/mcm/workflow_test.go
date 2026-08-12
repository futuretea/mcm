package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLIWorkflow(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "mcm")
	build := exec.Command("go", "build", "-o", binary, ".")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build CLI: %v\n%s", err, output)
	}

	workspace := t.TempDir()
	home := filepath.Join(workspace, "home")
	if err := os.Mkdir(home, 0o700); err != nil {
		t.Fatalf("create HOME: %v", err)
	}
	run := func(args ...string) string {
		t.Helper()
		command := exec.Command(binary, args...)
		command.Env = withHome(home)
		output, err := command.CombinedOutput()
		if err != nil {
			t.Fatalf("mcm %s: %v\n%s", strings.Join(args, " "), err, output)
		}
		return string(output)
	}

	if output := run("init"); output != "initialized\n" {
		t.Fatalf("mcm init output = %q, want %q", output, "initialized\n")
	}
	if output := run("server", "add", "--name", "local", "--command", "node", "--arg", "x"); output != "saved\n" {
		t.Fatalf("mcm server add output = %q, want %q", output, "saved\n")
	}
	if output := run("validate"); output != "valid\n" {
		t.Fatalf("mcm validate output = %q, want %q", output, "valid\n")
	}
	if output := run("server", "list"); output != "local\n" {
		t.Fatalf("mcm server list output = %q, want %q", output, "local\n")
	}

	vsCodePath := filepath.Join(workspace, "vscode", "mcp.json")
	for _, test := range []struct {
		target string
		path   string
	}{
		{"cursor", filepath.Join(home, ".cursor", "mcp.json")},
		{"claude-code", filepath.Join(home, ".claude.json")},
		{"codex", filepath.Join(home, ".codex", "config.toml")},
		{"vs-code", vsCodePath},
		{"qoder-cli", filepath.Join(home, ".qoder", "settings.json")},
		{"qoder-ide", filepath.Join(home, ".qoder", "mcp.json")},
		{"opencode", filepath.Join(home, ".config", "opencode", "opencode.json")},
		{"mcp-cli", filepath.Join(home, ".config", "mcp", "mcp_servers.json")},
		{"mcpc", filepath.Join(home, ".mcm", "exports", "mcpc.json")},
	} {
		t.Run(test.target, func(t *testing.T) {
			if test.target != "mcpc" {
				if err := os.MkdirAll(filepath.Dir(test.path), 0o700); err != nil {
					t.Fatalf("create target parent: %v", err)
				}
			}
			args := []string{"--target", test.target}
			if test.target == "vs-code" {
				args = append(args, "--path", test.path)
			}
			if output := run(append([]string{"plan"}, args...)...); !strings.Contains(output, test.target) {
				t.Errorf("mcm plan output = %q, want target %q", output, test.target)
			}
			applyArgs := append(append([]string{"apply"}, args...), "--yes")
			if output := run(applyArgs...); !strings.Contains(output, test.target) {
				t.Errorf("mcm apply output = %q, want target %q", output, test.target)
			}
			if output := run(append([]string{"status"}, args...)...); !strings.Contains(output, "synchronized") {
				t.Errorf("mcm status output = %q, want synchronized state", output)
			}
		})
	}
	if output := run("recover"); output != "recovered\n" {
		t.Errorf("mcm recover output = %q, want %q", output, "recovered\n")
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
