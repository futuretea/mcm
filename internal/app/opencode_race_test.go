package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/futuretea/mcm/internal/manifest"
)

func TestPlanRejectsOpenCodeJSONCAddedAfterResolution(t *testing.T) {
	workspace := t.TempDir()
	userHome := filepath.Join(workspace, "user-home")
	root := filepath.Join(workspace, "mcm-root")
	location := manifest.NewLocation(userHome, root, "")
	if err := location.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := location.Save(manifest.Config{
		Version: 1,
		Servers: map[string]manifest.Server{
			"local": {
				Transport: manifest.TransportStdio,
				Command:   "local-server",
			},
		},
	}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	configDir := filepath.Join(userHome, ".config", "opencode")
	jsonPath := filepath.Join(configDir, "opencode.json")
	jsoncPath := filepath.Join(configDir, "opencode.jsonc")
	initialJSON := []byte("{\"mcp\":{\"servers\":{}}}\n")
	if err := os.MkdirAll(configDir, 0o700); err != nil {
		t.Fatalf("create OpenCode config directory: %v", err)
	}
	if err := os.WriteFile(jsonPath, initialJSON, 0o600); err != nil {
		t.Fatalf("write OpenCode JSON config: %v", err)
	}

	previousHook := afterOpenCodeResolution
	afterOpenCodeResolution = func(path string) error {
		if path != jsonPath {
			t.Fatalf("resolved path = %q, want %q", path, jsonPath)
		}
		return os.WriteFile(jsoncPath, []byte("{}\n"), 0o600)
	}
	defer func() { afterOpenCodeResolution = previousHook }()

	application := New(userHome, location)
	if _, err := application.Plan([]string{"opencode"}, ""); err == nil {
		t.Fatal("Plan([opencode], \"\") error = nil, want JSON/JSONC ambiguity error")
	}
	got, err := os.ReadFile(jsonPath)
	if err != nil {
		t.Fatalf("read OpenCode JSON config: %v", err)
	}
	if string(got) != string(initialJSON) {
		t.Errorf("OpenCode JSON changed during Plan: got %q, want %q", got, initialJSON)
	}
}
