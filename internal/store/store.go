// Package store persists MCM ownership metadata without native configuration values.
package store

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/futuretea/mcm/internal/safeio"
)

const stateFileMode = 0o600

// State is the value-free ownership record for managed native targets.
type State struct {
	Version int                    `json:"version"`
	Targets map[string]TargetState `json:"targets"`
}

// TargetState records one target path and its MCM-managed server names.
type TargetState struct {
	Path   string   `json:"path"`
	Names  []string `json:"names"`
	Digest string   `json:"digest,omitempty"`
}

// Intent is a value-free record that permits state-only recovery after a native write.
type Intent struct {
	Target         string `json:"target"`
	Path           string `json:"path"`
	ExpectedDigest string `json:"expectedDigest"`
	DesiredDigest  string `json:"desiredDigest"`
	OldState       State  `json:"oldState"`
	NewState       State  `json:"newState"`
}

// Store owns state below one MCM private root.
type Store struct {
	root string
}

// New constructs a store for an initialized MCM root.
func New(root string) Store {
	return Store{root: root}
}

// Load returns an empty state when the state file has not yet been created.
func (store Store) Load() (State, error) {
	target, err := safeio.Open(filepath.Join(store.root, "state.json"), true)
	if err != nil {
		return State{}, err
	}
	defer target.Close()
	data, exists, _, err := target.Read()
	if err != nil {
		return State{}, err
	}
	if !exists {
		return emptyState(), nil
	}
	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return State{}, fmt.Errorf("decode state: %w", err)
	}
	if state.Version != 1 || state.Targets == nil {
		return State{}, fmt.Errorf("unsupported state")
	}
	return state, nil
}

// Save atomically persists value-free ownership metadata.
func (store Store) Save(state State) error {
	if state.Version == 0 {
		state.Version = 1
	}
	if state.Version != 1 || state.Targets == nil {
		return fmt.Errorf("invalid state")
	}
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	data = append(data, '\n')
	target, err := safeio.Open(filepath.Join(store.root, "state.json"), true)
	if err != nil {
		return err
	}
	defer target.Close()
	_, exists, mode, err := target.Read()
	if err != nil {
		return err
	}
	if !exists {
		mode = stateFileMode
	}
	return target.Replace(data, mode)
}

// WriteIntent persists a recovery intent before a native target replacement.
func (store Store) WriteIntent(intent Intent) error {
	if intent.Target == "" || intent.Path == "" || intent.DesiredDigest == "" || !validState(intent.OldState) || !validState(intent.NewState) {
		return fmt.Errorf("invalid journal intent")
	}
	if err := store.ensureJournal(); err != nil {
		return err
	}
	data, err := json.MarshalIndent(intent, "", "  ")
	if err != nil {
		return fmt.Errorf("encode journal intent: %w", err)
	}
	data = append(data, '\n')
	target, err := safeio.Open(store.intentPath(intent.Target), true)
	if err != nil {
		return err
	}
	defer target.Close()
	_, exists, mode, err := target.Read()
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("journal intent already exists for %q", intent.Target)
	}
	return target.Replace(data, mode)
}

// RemoveIntent removes an intent after its corresponding state commit.
func (store Store) RemoveIntent(target string) error {
	intent, err := safeio.Open(store.intentPath(target), true)
	if err != nil {
		return err
	}
	defer intent.Close()
	return intent.Remove()
}

// Recover reconciles journal intents by writing only MCM state; it never writes native targets.
func (store Store) Recover() error {
	entries, err := os.ReadDir(filepath.Join(store.root, "journal"))
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read journal: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			return fmt.Errorf("invalid journal entry %q", entry.Name())
		}
		intent, err := store.readIntent(filepath.Join(store.root, "journal", entry.Name()))
		if err != nil {
			return err
		}
		if err := store.recoverIntent(intent); err != nil {
			return err
		}
	}
	return nil
}

// RecoveryRequired reports whether an unfinished journal intent exists.
func (store Store) RecoveryRequired() (bool, error) {
	entries, err := os.ReadDir(filepath.Join(store.root, "journal"))
	if errors.Is(err, os.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("read journal: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			return false, fmt.Errorf("invalid journal entry %q", entry.Name())
		}
		return true, nil
	}
	return false, nil
}

func (store Store) recoverIntent(intent Intent) error {
	currentState, err := store.Load()
	if err != nil {
		return err
	}
	nativeDigest, err := readDigest(intent.Path)
	if err != nil {
		return fmt.Errorf("read recovery target: %w", err)
	}
	if nativeDigest == intent.DesiredDigest && statesEqual(currentState, intent.OldState) {
		if err := store.Save(intent.NewState); err != nil {
			return err
		}
		return store.RemoveIntent(intent.Target)
	}
	if nativeDigest == intent.ExpectedDigest && statesEqual(currentState, intent.OldState) {
		return store.RemoveIntent(intent.Target)
	}
	if nativeDigest == intent.DesiredDigest && statesEqual(currentState, intent.NewState) {
		return store.RemoveIntent(intent.Target)
	}
	return fmt.Errorf("recovery conflict for %q", intent.Target)
}

func (store Store) readIntent(path string) (Intent, error) {
	target, err := safeio.Open(path, false)
	if err != nil {
		return Intent{}, err
	}
	defer target.Close()
	data, exists, _, err := target.Read()
	if err != nil {
		return Intent{}, err
	}
	if !exists {
		return Intent{}, fmt.Errorf("journal intent disappeared")
	}
	var intent Intent
	if err := json.Unmarshal(data, &intent); err != nil {
		return Intent{}, fmt.Errorf("decode journal intent: %w", err)
	}
	if intent.Target == "" || intent.Path == "" || !validState(intent.OldState) || !validState(intent.NewState) {
		return Intent{}, fmt.Errorf("invalid journal intent")
	}
	return intent, nil
}

func (store Store) intentPath(target string) string {
	return filepath.Join(store.root, "journal", target+".json")
}

func (store Store) ensureJournal() error {
	path := filepath.Join(store.root, "journal")
	if err := os.Mkdir(path, 0o700); err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("journal is not an ordinary directory")
	}
	return os.Chmod(path, 0o700)
}

func validState(state State) bool {
	return state.Version == 1 && state.Targets != nil
}

func statesEqual(left, right State) bool {
	leftBytes, leftErr := json.Marshal(left)
	rightBytes, rightErr := json.Marshal(right)
	return leftErr == nil && rightErr == nil && string(leftBytes) == string(rightBytes)
}

func readDigest(path string) (string, error) {
	target, err := safeio.Open(path, true)
	if err != nil {
		return "", err
	}
	defer target.Close()
	data, exists, _, err := target.Read()
	if err != nil {
		return "", err
	}
	if !exists {
		return "", nil
	}
	sum := sha256.Sum256(data)
	return fmt.Sprintf("%x", sum), nil
}

func emptyState() State {
	return State{Version: 1, Targets: map[string]TargetState{}}
}
