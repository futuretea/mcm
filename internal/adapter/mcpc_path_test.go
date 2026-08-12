package adapter

import (
	"path/filepath"
	"testing"
)

func TestResolvePathMCPCRejectsExplicitPaths(t *testing.T) {
	workspace := t.TempDir()
	tempUser := filepath.Join(workspace, "user")
	tempRoot := filepath.Join(workspace, "mcm")
	overrideAbs := filepath.Join(workspace, "override.json")
	configuredAbs := filepath.Join(workspace, "configured.json")

	t.Run("override", func(t *testing.T) {
		if _, err := ResolvePath("mcpc", tempUser, tempRoot, overrideAbs, ""); err == nil {
			t.Error("ResolvePath(mcpc) with override returned nil error")
		}
	})

	t.Run("configured", func(t *testing.T) {
		if _, err := ResolvePath("mcpc", tempUser, tempRoot, "", configuredAbs); err == nil {
			t.Error("ResolvePath(mcpc) with configured path returned nil error")
		}
	})
}
