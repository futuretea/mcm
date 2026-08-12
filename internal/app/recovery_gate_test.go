package app

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/futuretea/mcm/internal/manifest"
	"github.com/futuretea/mcm/internal/store"
)

func TestPlanAndStatusRequireRecoveryBeforeRead(t *testing.T) {
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

	emptyState := store.State{Version: 1, Targets: map[string]store.TargetState{}}
	if err := store.New(root).WriteIntent(store.Intent{
		Target:        "cursor",
		Path:          filepath.Join(workspace, "missing-target.json"),
		DesiredDigest: "deadbeef",
		OldState:      emptyState,
		NewState:      emptyState,
	}); err != nil {
		t.Fatalf("WriteIntent() error = %v", err)
	}

	application := New(userHome, location)
	if _, err := application.Plan([]string{"cursor"}, ""); err == nil || !strings.Contains(err.Error(), "recovery required") {
		t.Errorf("Plan([cursor], \"\") error = %v, want recovery required", err)
	}
	if _, err := application.Status([]string{"cursor"}, ""); err == nil || !strings.Contains(err.Error(), "recovery required") {
		t.Errorf("Status([cursor], \"\") error = %v, want recovery required", err)
	}
}
