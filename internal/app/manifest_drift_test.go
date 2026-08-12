package app

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/futuretea/mcm/internal/manifest"
)

func TestApplyPlannedRejectsManifestDriftWithoutWrites(t *testing.T) {
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
	preview, err := application.Plan([]string{"cursor"}, "")
	if err != nil {
		t.Fatalf("Plan([cursor], \"\") error = %v", err)
	}
	if err := location.Save(manifest.Config{
		Version: 1,
		Servers: map[string]manifest.Server{
			"local": {
				Transport: manifest.TransportStdio,
				Command:   "changed",
			},
		},
	}); err != nil {
		t.Fatalf("Save(changed manifest) error = %v", err)
	}

	if _, err := application.ApplyPlanned(preview); err == nil {
		t.Error("ApplyPlanned(preview) error = nil, want manifest drift error")
	}

	for _, path := range []string{
		filepath.Join(userHome, ".cursor", "mcp.json"),
		filepath.Join(root, "state.json"),
		filepath.Join(root, "journal", "cursor.json"),
	} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s exists after rejected ApplyPlanned: stat error = %v, want not exist", path, err)
		}
	}
}
