package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/futuretea/mcm/internal/manifest"
)

func TestPlanRequiresTargetAndDoesNotWrite(t *testing.T) {
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

	application := New(userHome, location)
	targetPath := filepath.Join(userHome, ".cursor", "mcp.json")
	if _, err := application.Plan(nil, ""); err == nil {
		t.Error("Plan(nil, \"\") returned nil error")
	}
	if _, err := os.Stat(targetPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("cursor target exists after invalid Plan: stat %q error = %v, want not exist", targetPath, err)
	}

	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		t.Fatalf("create cursor directory: %v", err)
	}
	items, err := application.Plan([]string{"cursor"}, "")
	if err != nil {
		t.Fatalf("Plan([cursor], \"\") error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("Plan([cursor], \"\") returned %d items, want 1", len(items))
	}
	if got := items[0].Target; got != "cursor" {
		t.Errorf("Plan([cursor], \"\") item target = %q, want cursor", got)
	}
	if got := len(items[0].Changes); got != 1 {
		t.Errorf("Plan([cursor], \"\") item changes = %d, want 1", got)
	}
	if _, err := os.Stat(targetPath); !errors.Is(err, os.ErrNotExist) {
		t.Errorf("cursor target exists after Plan: stat %q error = %v, want not exist", targetPath, err)
	}
}
