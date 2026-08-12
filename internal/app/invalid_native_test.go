package app

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/futuretea/mcm/internal/manifest"
)

func TestApplyRejectsTrailingNativeJSONWithoutWriting(t *testing.T) {
	workspace := t.TempDir()
	userHome := filepath.Join(workspace, "user-home")
	root := filepath.Join(workspace, "mcm-root")
	location := manifest.NewLocation(userHome, root, "")
	if err := location.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}
	if err := location.Save(manifest.Config{Version: 1, Servers: map[string]manifest.Server{
		"local": {Transport: manifest.TransportStdio, Command: "node"},
	}}); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	targetPath := filepath.Join(userHome, ".cursor", "mcp.json")
	if err := os.MkdirAll(filepath.Dir(targetPath), 0o700); err != nil {
		t.Fatalf("create target directory: %v", err)
	}
	existing := []byte(`{"mcpServers": {}} trailing`)
	if err := os.WriteFile(targetPath, existing, 0o600); err != nil {
		t.Fatalf("write malformed target: %v", err)
	}

	if _, err := New(userHome, location).Apply([]string{"cursor"}, ""); err == nil {
		t.Fatal("Apply([cursor], \"\") error = nil, want malformed JSON error")
	}
	got, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read target after rejected Apply: %v", err)
	}
	if !bytes.Equal(got, existing) {
		t.Errorf("target bytes after rejected Apply = %q, want unchanged %q", got, existing)
	}
	for _, path := range []string{filepath.Join(root, "state.json"), filepath.Join(root, "journal", "cursor.json")} {
		if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
			t.Errorf("%s exists after rejected Apply: stat error = %v, want not exist", path, err)
		}
	}
}
