package adapter

import (
	"path/filepath"
	"testing"
)

var _ func(string, string, string, string, string) (string, error) = ResolvePath

func TestResolvePathTargetDefaults(t *testing.T) {
	workspace := t.TempDir()
	userHome := filepath.Join(workspace, "user-home")
	mcmRoot := filepath.Join(workspace, "mcm-root")

	tests := []struct {
		target string
		want   string
	}{
		{target: "cursor", want: filepath.Join(userHome, ".cursor", "mcp.json")},
		{target: "codex", want: filepath.Join(userHome, ".codex", "config.toml")},
		{target: "qoder-cli", want: filepath.Join(userHome, ".qoder", "settings.json")},
		{target: "qoder-ide", want: filepath.Join(userHome, ".qoder", "mcp.json")},
		{target: "mcp-cli", want: filepath.Join(userHome, ".config", "mcp", "mcp_servers.json")},
		{target: "mcpc", want: filepath.Join(mcmRoot, "exports", "mcpc.json")},
	}

	for _, test := range tests {
		t.Run(test.target, func(t *testing.T) {
			got, err := ResolvePath(test.target, userHome, mcmRoot, "", "")
			if err != nil {
				t.Fatalf("ResolvePath(%q) error: %v", test.target, err)
			}
			if got != test.want {
				t.Errorf("ResolvePath(%q) = %q, want %q", test.target, got, test.want)
			}
		})
	}
}

func TestResolvePathVSCode(t *testing.T) {
	workspace := t.TempDir()
	userHome := filepath.Join(workspace, "user-home")
	mcmRoot := filepath.Join(workspace, "mcm-root")
	configured := filepath.Join(workspace, "workspace", ".vscode", "mcp.json")

	if _, err := ResolvePath("vs-code", userHome, mcmRoot, "", ""); err == nil {
		t.Error("ResolvePath(vs-code) with empty override and configured path returned nil error")
	}

	got, err := ResolvePath("vs-code", userHome, mcmRoot, "", configured)
	if err != nil {
		t.Fatalf("ResolvePath(vs-code) with configured path error: %v", err)
	}
	if got != configured {
		t.Errorf("ResolvePath(vs-code) = %q, want configured path %q", got, configured)
	}
}
