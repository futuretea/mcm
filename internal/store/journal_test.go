package store

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestRecoverCommitsDesiredStateWithoutWritingTarget(t *testing.T) {
	root := t.TempDir()
	targetPath := filepath.Join(root, "target.json")
	targetBytes := []byte("{\"mcpServers\":{\"local\":{}}}\n")
	if err := os.WriteFile(targetPath, targetBytes, 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}

	oldState := State{Version: 1, Targets: map[string]TargetState{}}
	store := New(root)
	if err := store.Save(oldState); err != nil {
		t.Fatalf("Save() error = %v", err)
	}

	digest := sha256.Sum256(targetBytes)
	intent := Intent{
		Target:        "cursor",
		Path:          targetPath,
		DesiredDigest: hex.EncodeToString(digest[:]),
		OldState:      oldState,
		NewState: State{
			Version: 1,
			Targets: map[string]TargetState{
				"cursor": {Path: targetPath},
			},
		},
	}
	if err := store.WriteIntent(intent); err != nil {
		t.Fatalf("WriteIntent() error = %v", err)
	}

	if err := store.Recover(); err != nil {
		t.Fatalf("Recover() error = %v", err)
	}

	state, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	target, ok := state.Targets["cursor"]
	if !ok {
		t.Fatal("recovered state does not contain cursor target")
	}
	if got := target.Path; got != targetPath {
		t.Errorf("recovered target path = %q, want %q", got, targetPath)
	}

	gotBytes, err := os.ReadFile(targetPath)
	if err != nil {
		t.Fatalf("read target after recovery: %v", err)
	}
	if string(gotBytes) != string(targetBytes) {
		t.Errorf("target contents after recovery = %q, want %q", gotBytes, targetBytes)
	}

	_, err = os.Stat(filepath.Join(root, "journal", "cursor.json"))
	if !errors.Is(err, os.ErrNotExist) {
		t.Errorf("journal/cursor.json after recovery error = %v, want not exist", err)
	}
}
