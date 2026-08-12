package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocationInitPreservesExistingExternalParentMode(t *testing.T) {
	workspace := t.TempDir()
	externalParent := filepath.Join(workspace, "external")
	if err := os.Mkdir(externalParent, 0o755); err != nil {
		t.Fatalf("create external parent: %v", err)
	}
	configPath := filepath.Join(externalParent, "custom.yaml")
	location := NewLocation(filepath.Join(workspace, "home"), filepath.Join(workspace, "mcm-root"), configPath)

	if err := location.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	info, err := os.Stat(externalParent)
	if err != nil {
		t.Fatalf("stat external parent %q: %v", externalParent, err)
	}
	if got := info.Mode().Perm(); got != 0o755 {
		t.Errorf("external parent permissions = %o, want %o", got, 0o755)
	}

	info, err = os.Stat(configPath)
	if err != nil {
		t.Fatalf("stat config file %q: %v", configPath, err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("config file permissions = %o, want %o", got, 0o600)
	}
}
