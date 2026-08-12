package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/futuretea/mcm/internal/manifest"
)

func TestStatusReportsSynchronizedAndModifiedTargets(t *testing.T) {
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
	if _, err := application.Apply([]string{"cursor"}, ""); err != nil {
		t.Fatalf("Apply([cursor], \"\") error = %v", err)
	}

	items, err := application.Status([]string{"cursor"}, "")
	if err != nil {
		t.Fatalf("Status([cursor], \"\") error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Status([cursor], \"\") returned %d items, want 1", len(items))
	}
	if got := items[0].Target; got != "cursor" {
		t.Errorf("Status([cursor], \"\") target = %q, want cursor", got)
	}
	if got := items[0].State; got != "synchronized" {
		t.Errorf("Status([cursor], \"\") state = %q, want synchronized", got)
	}

	targetPath := filepath.Join(userHome, ".cursor", "mcp.json")
	if err := os.WriteFile(targetPath, []byte(`{"mcpServers":{"different":{"command":"different-server"}}}`), 0o600); err != nil {
		t.Fatalf("modify cursor target: %v", err)
	}

	items, err = application.Status([]string{"cursor"}, "")
	if err != nil {
		t.Fatalf("Status([cursor], \"\") after modification error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Status([cursor], \"\") after modification returned %d items, want 1", len(items))
	}
	if got := items[0].State; got != "modified" {
		t.Errorf("Status([cursor], \"\") after modification state = %q, want modified", got)
	}
}
