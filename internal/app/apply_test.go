package app

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/futuretea/mcm/internal/manifest"
)

func TestApplyWritesTargetAndOwnershipState(t *testing.T) {
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
	if err := os.MkdirAll(filepath.Join(userHome, ".cursor"), 0o700); err != nil {
		t.Fatalf("create cursor directory: %v", err)
	}

	application := New(userHome, location)
	items, err := application.Apply([]string{"cursor"}, "")
	if err != nil {
		t.Fatalf("Apply([cursor], \"\") error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Apply([cursor], \"\") returned %d items, want 1", len(items))
	}

	targetPath := filepath.Join(userHome, ".cursor", "mcp.json")
	data, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read cursor target: %v", err)
	}
	var document struct {
		MCPServers map[string]struct {
			Command string `json:"command"`
		} `json:"mcpServers"`
	}
	if err := json.Unmarshal(data, &document); err != nil {
		t.Fatalf("decode cursor target: %v", err)
	}
	if got := document.MCPServers["local"].Command; got != "local-server" {
		t.Errorf("cursor local command = %q, want local-server", got)
	}
	if _, err := os.Stat(filepath.Join(root, "state.json")); err != nil {
		t.Errorf("ownership state: %v", err)
	}
}
