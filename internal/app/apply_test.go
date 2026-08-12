package app

import (
	"bytes"
	"encoding/json"
	"errors"
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

func TestApplyPlannedRejectsTargetSymlinkSwapWithoutWrites(t *testing.T) {
	workspace := t.TempDir()
	userHome := filepath.Join(workspace, "user-home")
	root := filepath.Join(workspace, "mcm-root")
	location := manifest.NewLocation(userHome, root, "")
	if err := location.Init(); err != nil {
		t.Fatalf("Init() error: %v", err)
	}
	if err := location.Save(manifest.Config{Version: 1, Servers: map[string]manifest.Server{
		"local": {Transport: manifest.TransportStdio, Command: "local-server"},
	}}); err != nil {
		t.Fatalf("Save() error: %v", err)
	}
	targetPath := filepath.Join(userHome, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		t.Fatalf("create target parent: %v", err)
	}
	if err := os.WriteFile(targetPath, []byte(`{"mcpServers":{}}`), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}

	application := New(userHome, location)
	preview, err := application.Plan([]string{"cursor"}, "")
	if err != nil {
		t.Fatalf("Plan() error: %v", err)
	}
	sentinelPath := filepath.Join(workspace, "fixture-only-sensitive-sentinel")
	sentinel := []byte("fixture-only-sensitive-sentinel")
	if err := os.WriteFile(sentinelPath, sentinel, 0o600); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}
	if err := os.Remove(targetPath); err != nil {
		t.Fatalf("remove planned target: %v", err)
	}
	if err := os.Symlink(sentinelPath, targetPath); err != nil {
		t.Fatalf("replace planned target with symlink: %v", err)
	}

	if _, err := application.ApplyPlanned(preview); err == nil {
		t.Fatal("ApplyPlanned() error = nil, want symlink rejection")
	}
	got, err := os.ReadFile(sentinelPath)
	if err != nil {
		t.Fatalf("read sentinel: %v", err)
	}
	if !bytes.Equal(got, sentinel) {
		t.Errorf("sentinel = %q, want %q", got, sentinel)
	}
	for _, path := range []string{filepath.Join(root, "state.json"), filepath.Join(root, "journal", "cursor.json")} {
		if _, err := os.Lstat(path); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("unexpected MCM write %q: %v", path, err)
		}
	}
}

func TestApplyTightensPermissiveTargetModeAndPreservesRestrictiveMode(t *testing.T) {
	for _, test := range []struct {
		name string
		mode os.FileMode
		want os.FileMode
	}{
		{"permissive", 0o644, 0o600},
		{"restrictive", 0o400, 0o400},
	} {
		t.Run(test.name, func(t *testing.T) {
			workspace := t.TempDir()
			userHome := filepath.Join(workspace, "user-home")
			root := filepath.Join(workspace, "mcm-root")
			location := manifest.NewLocation(userHome, root, "")
			if err := location.Init(); err != nil {
				t.Fatalf("Init() error: %v", err)
			}
			if err := location.Save(manifest.Config{Version: 1, Servers: map[string]manifest.Server{
				"local": {Transport: manifest.TransportStdio, Command: "local-server"},
			}}); err != nil {
				t.Fatalf("Save() error: %v", err)
			}
			targetPath := filepath.Join(userHome, ".cursor", "mcp.json")
			if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
				t.Fatalf("create target parent: %v", err)
			}
			if err := os.WriteFile(targetPath, []byte(`{"mcpServers":{}}`), test.mode); err != nil {
				t.Fatalf("write target: %v", err)
			}

			if _, err := New(userHome, location).Apply([]string{"cursor"}, ""); err != nil {
				t.Fatalf("Apply() error: %v", err)
			}
			info, err := os.Stat(targetPath)
			if err != nil {
				t.Fatalf("stat target: %v", err)
			}
			if got := info.Mode().Perm(); got != test.want {
				t.Errorf("target mode = %o, want %o", got, test.want)
			}
		})
	}
}
