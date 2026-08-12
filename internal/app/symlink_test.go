package app

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/futuretea/mcm/internal/manifest"
)

func TestPlanRejectsSymlinkTargetWithoutChangingSentinel(t *testing.T) {
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

	targetPath := filepath.Join(userHome, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		t.Fatalf("create cursor directory: %v", err)
	}
	sentinelPath := filepath.Join(workspace, "fixture-only-sensitive-sentinel")
	sentinel := []byte("fixture-only-sensitive-sentinel")
	if err := os.WriteFile(sentinelPath, sentinel, 0o600); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}
	if err := os.Symlink(sentinelPath, targetPath); err != nil {
		t.Fatalf("create target symlink: %v", err)
	}

	application := New(userHome, location)
	if _, err := application.Plan([]string{"cursor"}, targetPath); err == nil {
		t.Error("Plan([cursor], symlink target) returned nil error")
	}

	got, err := os.ReadFile(sentinelPath)
	if err != nil {
		t.Fatalf("read sentinel after Plan: %v", err)
	}
	if !bytes.Equal(got, sentinel) {
		t.Errorf("sentinel after Plan = %q, want %q", got, sentinel)
	}
}
