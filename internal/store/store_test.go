package store

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadRoundTripsState(t *testing.T) {
	root := t.TempDir()
	targetPath := filepath.Join(root, "target.json")
	store := New(root)
	state := State{
		Targets: map[string]TargetState{
			"cursor": {
				Path:  targetPath,
				Names: []string{"local"},
			},
		},
	}

	if err := store.Save(state); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	target, ok := loaded.Targets["cursor"]
	if !ok {
		t.Fatal("loaded state does not contain cursor target")
	}
	if got := target.Path; got != targetPath {
		t.Errorf("loaded target path = %q, want %q", got, targetPath)
	}
	if len(target.Names) != 1 || target.Names[0] != "local" {
		t.Errorf("loaded target names = %q, want %q", target.Names, []string{"local"})
	}

	info, err := os.Stat(filepath.Join(root, "state.json"))
	if err != nil {
		t.Fatalf("stat state file: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("state file permissions = %o, want %o", got, 0o600)
	}
}
