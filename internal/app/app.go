// Package app coordinates manifest, adapter, and local ownership operations.
package app

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/futuretea/mcm/internal/adapter"
	"github.com/futuretea/mcm/internal/manifest"
	"github.com/futuretea/mcm/internal/safeio"
	"github.com/futuretea/mcm/internal/store"
)

var targetOrder = []string{
	"cursor", "claude-code", "codex", "vs-code", "qoder-cli", "qoder-ide", "opencode", "mcp-cli", "mcpc",
}

// App coordinates one process home with one MCM manifest location.
type App struct {
	userHome string
	location manifest.Location
	store    store.Store
}

// PlanItem is a value-free preview for one native target.
type PlanItem struct {
	Target   string
	Path     string
	Changes  []adapter.Change
	desired  []byte
	mode     uint32
	source   string
	manifest string
}

// StatusItem summarizes the file-level state of one selected target.
type StatusItem struct {
	Target string
	Path   string
	State  string
}

// New constructs the local MCM application service.
func New(userHome string, location manifest.Location) App {
	return App{userHome: userHome, location: location, store: store.New(location.Root)}
}

// Recover reconciles MCM state journals without reading or writing native targets.
func (application App) Recover() error {
	lock, err := safeio.AcquireLock(filepath.Join(application.location.Root, "lock"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	defer lock.Close()
	return application.store.Recover()
}

// Status reports file synchronization without rendering or writing targets.
func (application App) Status(targets []string, override string) ([]StatusItem, error) {
	selected, err := normalizeTargets(targets)
	if err != nil {
		return nil, err
	}
	if override != "" && len(selected) != 1 {
		return nil, fmt.Errorf("target path requires exactly one target")
	}
	if required, err := application.store.RecoveryRequired(); err != nil {
		return nil, err
	} else if required {
		return nil, fmt.Errorf("recovery required")
	}
	config, err := application.location.Load()
	if err != nil {
		return nil, err
	}
	state, err := application.store.Load()
	if err != nil {
		return nil, err
	}
	items := make([]StatusItem, 0, len(selected))
	for _, targetName := range selected {
		configured := ""
		if target, ok := config.Targets[targetName]; ok {
			configured = target.Path
		}
		path, err := adapter.ResolvePath(targetName, application.userHome, application.location.Root, override, configured)
		if err != nil {
			return nil, err
		}
		if targetName == "opencode" && override == "" && configured == "" {
			path, err = ResolveOpenCodePath(application.userHome)
			if err != nil {
				return nil, err
			}
		}
		item := StatusItem{Target: targetName, Path: path, State: "missing"}
		previous, managed := state.Targets[targetName]
		if !managed || previous.Path != path {
			items = append(items, item)
			continue
		}
		target, err := safeio.Open(path, true)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				items = append(items, item)
				continue
			}
			return nil, fmt.Errorf("open %s status target: %w", targetName, err)
		}
		data, exists, _, readErr := target.Read()
		closeErr := target.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read %s status target: %w", targetName, readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close %s status target: %w", targetName, closeErr)
		}
		if exists {
			item.State = "modified"
			if digest(data) == previous.Digest {
				item.State = "synchronized"
			}
		}
		items = append(items, item)
	}
	return items, nil
}

// Plan validates target selection and renders a read-only native configuration preview.
func (application App) Plan(targets []string, override string) ([]PlanItem, error) {
	selected, err := normalizeTargets(targets)
	if err != nil {
		return nil, err
	}
	if override != "" && len(selected) != 1 {
		return nil, fmt.Errorf("target path requires exactly one target")
	}
	if required, err := application.store.RecoveryRequired(); err != nil {
		return nil, err
	} else if required {
		return nil, fmt.Errorf("recovery required")
	}
	config, manifestDigest, err := application.location.Snapshot()
	if err != nil {
		return nil, err
	}
	state, err := application.store.Load()
	if err != nil {
		return nil, err
	}
	items := make([]PlanItem, 0, len(selected))
	for _, targetName := range selected {
		configured := ""
		if target, ok := config.Targets[targetName]; ok {
			configured = target.Path
		}
		path, err := adapter.ResolvePath(targetName, application.userHome, application.location.Root, override, configured)
		if err != nil {
			return nil, err
		}
		openCodeDefault := targetName == "opencode" && override == "" && configured == ""
		if openCodeDefault {
			path, err = ResolveOpenCodePath(application.userHome)
			if err != nil {
				return nil, err
			}
			if err := afterOpenCodeResolution(path); err != nil {
				return nil, err
			}
		}
		target, err := safeio.Open(path, true)
		if err != nil {
			if errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("open %s target: create parent directory %s, then retry: %w", targetName, filepath.Dir(path), err)
			}
			return nil, fmt.Errorf("open %s target: %w", targetName, err)
		}
		if openCodeDefault {
			if err := validateOpenCodeSibling(target, path); err != nil {
				target.Close()
				return nil, err
			}
		}
		data, exists, mode, readErr := target.Read()
		closeErr := target.Close()
		if readErr != nil {
			return nil, fmt.Errorf("read %s target: %w", targetName, readErr)
		}
		if closeErr != nil {
			return nil, fmt.Errorf("close %s target: %w", targetName, closeErr)
		}
		if !exists {
			data = nil
		}
		managed := map[string]struct{}{}
		if previous, ok := state.Targets[targetName]; ok && previous.Path == path {
			for _, name := range previous.Names {
				managed[name] = struct{}{}
			}
		}
		rendered, err := adapter.Render(targetName, config, data, managed)
		if err != nil {
			return nil, fmt.Errorf("render %s target: %w", targetName, err)
		}
		source := ""
		if exists {
			source = digest(data)
		}
		items = append(items, PlanItem{Target: targetName, Path: path, Changes: rendered.Changes, desired: rendered.Bytes, mode: uint32(mode.Perm()), source: source, manifest: manifestDigest})
	}
	return items, nil
}

// Apply renders and atomically writes each selected target, then records ownership state.
func (application App) Apply(targets []string, override string) ([]PlanItem, error) {
	lock, err := safeio.AcquireLock(filepath.Join(application.location.Root, "lock"))
	if err != nil {
		return nil, err
	}
	defer lock.Close()
	if err := application.store.Recover(); err != nil {
		return nil, err
	}
	items, err := application.Plan(targets, override)
	if err != nil {
		return nil, err
	}
	return application.applyPlanned(items)
}

// ApplyPlanned commits a preview only if its manifest and target snapshots still match.
func (application App) ApplyPlanned(items []PlanItem) ([]PlanItem, error) {
	lock, err := safeio.AcquireLock(filepath.Join(application.location.Root, "lock"))
	if err != nil {
		return nil, err
	}
	defer lock.Close()
	if err := application.store.Recover(); err != nil {
		return nil, err
	}
	return application.applyPlanned(items)
}

func (application App) applyPlanned(items []PlanItem) ([]PlanItem, error) {
	if len(items) == 0 {
		return nil, fmt.Errorf("at least one target is required")
	}
	_, currentManifest, err := application.location.Snapshot()
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		if item.manifest != currentManifest {
			return nil, fmt.Errorf("manifest changed after plan")
		}
	}
	state, err := application.store.Load()
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		target, err := safeio.Open(item.Path, true)
		if err != nil {
			return nil, fmt.Errorf("open %s target for write: %w", item.Target, err)
		}
		if item.Target == "opencode" && isDefaultOpenCodePath(application.userHome, item.Path) {
			if err := validateOpenCodeSibling(target, item.Path); err != nil {
				target.Close()
				return nil, err
			}
		}
		current, exists, mode, readErr := target.Read()
		if readErr != nil {
			target.Close()
			return nil, fmt.Errorf("recheck %s target: %w", item.Target, readErr)
		}
		if !exists {
			mode = 0
		}
		currentDigest := ""
		if exists {
			currentDigest = digest(current)
		}
		if currentDigest != item.source {
			target.Close()
			return nil, fmt.Errorf("target %s changed after plan", item.Target)
		}
		names := managedNames(item.Changes)
		newState := cloneState(state)
		newState.Targets[item.Target] = store.TargetState{Path: item.Path, Names: names, Digest: digest(item.desired)}
		intent := store.Intent{
			Target:         item.Target,
			Path:           item.Path,
			ExpectedDigest: currentDigest,
			DesiredDigest:  digest(item.desired),
			OldState:       state,
			NewState:       newState,
		}
		if err := application.store.WriteIntent(intent); err != nil {
			target.Close()
			return nil, err
		}
		if err := target.Replace(item.desired, mode); err != nil {
			target.Close()
			return nil, fmt.Errorf("write %s target: %w", item.Target, err)
		}
		if err := target.Close(); err != nil {
			return nil, fmt.Errorf("close %s target: %w", item.Target, err)
		}
		if err := application.store.Save(newState); err != nil {
			return nil, err
		}
		if err := application.store.RemoveIntent(item.Target); err != nil {
			return nil, err
		}
		state = newState
	}
	return items, nil
}

func managedNames(changes []adapter.Change) []string {
	names := make([]string, 0, len(changes))
	for _, change := range changes {
		if change.Action != "remove" {
			names = append(names, change.Name)
		}
	}
	return names
}

func cloneState(state store.State) store.State {
	clone := store.State{Version: state.Version, Targets: make(map[string]store.TargetState, len(state.Targets))}
	for name, target := range state.Targets {
		target.Names = append([]string(nil), target.Names...)
		clone.Targets[name] = target
	}
	return clone
}

func digest(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

func normalizeTargets(targets []string) ([]string, error) {
	if len(targets) == 0 {
		return nil, fmt.Errorf("at least one target is required")
	}
	requested := make(map[string]struct{}, len(targets))
	for _, target := range targets {
		if !adapter.IsTarget(target) {
			return nil, fmt.Errorf("unsupported target %q", target)
		}
		requested[target] = struct{}{}
	}
	selected := make([]string, 0, len(requested))
	for _, target := range targetOrder {
		if _, ok := requested[target]; ok {
			selected = append(selected, target)
		}
	}
	return selected, nil
}
