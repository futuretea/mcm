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

func TestRecoverHandlesEveryJournalStateWithoutWritingNativeTarget(t *testing.T) {
	for _, test := range []struct {
		name          string
		native        []byte
		state         string
		wantNewState  bool
		wantRecovered bool
	}{
		{"desired_old", []byte("desired"), "old", true, true},
		{"expected_old", []byte("expected"), "old", false, true},
		{"desired_new", []byte("desired"), "new", true, true},
		{"mismatch", []byte("different"), "old", false, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			targetPath := filepath.Join(root, "target.json")
			if err := os.WriteFile(targetPath, test.native, 0o600); err != nil {
				t.Fatalf("write target: %v", err)
			}
			oldState := State{Version: 1, Targets: map[string]TargetState{}}
			newState := State{Version: 1, Targets: map[string]TargetState{"cursor": {Path: targetPath}}}
			journal := New(root)
			if test.state == "new" {
				if err := journal.Save(newState); err != nil {
					t.Fatalf("save new state: %v", err)
				}
			} else if err := journal.Save(oldState); err != nil {
				t.Fatalf("save old state: %v", err)
			}
			expected := sha256.Sum256([]byte("expected"))
			desired := sha256.Sum256([]byte("desired"))
			intent := Intent{
				Target:         "cursor",
				Path:           targetPath,
				ExpectedDigest: hex.EncodeToString(expected[:]),
				DesiredDigest:  hex.EncodeToString(desired[:]),
				OldState:       oldState,
				NewState:       newState,
			}
			if err := journal.WriteIntent(intent); err != nil {
				t.Fatalf("WriteIntent() error: %v", err)
			}

			err := journal.Recover()
			if test.wantRecovered && err != nil {
				t.Fatalf("Recover() error: %v", err)
			}
			if !test.wantRecovered && err == nil {
				t.Fatal("Recover() error = nil, want conflict")
			}
			gotNative, readErr := os.ReadFile(targetPath)
			if readErr != nil {
				t.Fatalf("read target: %v", readErr)
			}
			if string(gotNative) != string(test.native) {
				t.Errorf("target changed to %q, want %q", gotNative, test.native)
			}
			state, err := journal.Load()
			if err != nil {
				t.Fatalf("Load() error: %v", err)
			}
			wantState := oldState
			if test.wantNewState {
				wantState = newState
			}
			if !statesEqual(state, wantState) {
				t.Errorf("state = %#v, want %#v", state, wantState)
			}
			_, statErr := os.Stat(filepath.Join(root, "journal", "cursor.json"))
			if test.wantRecovered && !errors.Is(statErr, os.ErrNotExist) {
				t.Errorf("intent after recovery error = %v, want not exist", statErr)
			}
			if !test.wantRecovered && statErr != nil {
				t.Errorf("intent after conflict error = %v, want present", statErr)
			}
		})
	}
}
