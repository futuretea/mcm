package app

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveOpenCodePathUsesJSONCWhenPresent(t *testing.T) {
	userHome := t.TempDir()
	jsoncPath := filepath.Join(userHome, ".config", "opencode", "opencode.jsonc")
	if err := os.MkdirAll(filepath.Dir(jsoncPath), 0o700); err != nil {
		t.Fatalf("create OpenCode config directory: %v", err)
	}
	if err := os.WriteFile(jsoncPath, []byte("{}"), 0o600); err != nil {
		t.Fatalf("write OpenCode JSONC config: %v", err)
	}

	got, err := ResolveOpenCodePath(userHome)
	if err != nil {
		t.Fatalf("ResolveOpenCodePath() error = %v", err)
	}
	if got != jsoncPath {
		t.Errorf("ResolveOpenCodePath() = %q, want %q", got, jsoncPath)
	}
}

func TestResolveOpenCodePathRejectsJSONAndJSONC(t *testing.T) {
	userHome := t.TempDir()
	configDir := filepath.Join(userHome, ".config", "opencode")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("create OpenCode config directory: %v", err)
	}
	for _, name := range []string{"opencode.json", "opencode.jsonc"} {
		if err := os.WriteFile(filepath.Join(configDir, name), []byte("{}"), 0o600); err != nil {
			t.Fatalf("write OpenCode config %q: %v", name, err)
		}
	}

	if _, err := ResolveOpenCodePath(userHome); err == nil {
		t.Error("ResolveOpenCodePath() with JSON and JSONC configs returned nil error")
	}
}
