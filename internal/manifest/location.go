package manifest

import (
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/futuretea/mcm/internal/safeio"
	"github.com/goccy/go-yaml"
)

const (
	privateDirMode  = 0o700
	privateFileMode = 0o600
)

var initialConfig = []byte("version: 1\nservers: {}\ntargets: {}\n")

var afterManifestAbsenceCheck = func(string) error { return nil }

// Location identifies the MCM-owned root and the selected manifest path.
type Location struct {
	Root       string
	ConfigPath string
}

// NewLocation derives MCM paths without changing the native client home.
func NewLocation(userHome, homeOverride, configOverride string) Location {
	root := homeOverride
	if root == "" {
		root = filepath.Join(userHome, ".mcm")
	}
	configPath := configOverride
	if configPath == "" {
		configPath = filepath.Join(root, "config.yaml")
	}
	return Location{Root: root, ConfigPath: configPath}
}

// Init creates only MCM-owned private directories and a new manifest.
func (location Location) Init() error {
	if err := ensurePrivateDirectory(location.Root, true); err != nil {
		return fmt.Errorf("initialize mcm root: %w", err)
	}
	for _, name := range []string{"journal", "exports"} {
		if err := ensurePrivateDirectory(filepath.Join(location.Root, name), true); err != nil {
			return fmt.Errorf("initialize %s: %w", name, err)
		}
	}

	parent := filepath.Dir(location.ConfigPath)
	if parent != location.Root {
		if err := validateDirectory(parent); err != nil {
			return fmt.Errorf("validate manifest parent: %w", err)
		}
	}
	return createManifest(location.ConfigPath)
}

// Load reads and validates the selected manifest through a no-follow file descriptor.
func (location Location) Load() (Config, error) {
	config, _, err := location.Snapshot()
	return config, err
}

// Snapshot reads the selected manifest and returns its validated content and digest.
func (location Location) Snapshot() (Config, string, error) {
	target, err := safeio.Open(location.ConfigPath, false)
	if err != nil {
		return Config{}, "", err
	}
	defer target.Close()
	data, exists, _, err := target.Read()
	if err != nil {
		return Config{}, "", err
	}
	if !exists {
		return Config{}, "", fmt.Errorf("manifest does not exist")
	}
	config, err := Decode(data)
	if err != nil {
		return Config{}, "", err
	}
	sum := sha256.Sum256(data)
	return config, hex.EncodeToString(sum[:]), nil
}

// Save validates and atomically replaces the selected manifest.
func (location Location) Save(config Config) error {
	if err := Validate(config); err != nil {
		return err
	}
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("encode manifest: %w", err)
	}
	target, err := safeio.Open(location.ConfigPath, false)
	if err != nil {
		return err
	}
	defer target.Close()
	_, exists, mode, err := target.Read()
	if err != nil {
		return err
	}
	if !exists {
		return fmt.Errorf("manifest does not exist")
	}
	return target.Replace(data, mode)
}

func ensurePrivateDirectory(path string, create bool) error {
	info, err := os.Lstat(path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) || !create {
			return err
		}
		if err := os.Mkdir(path, privateDirMode); err != nil && !errors.Is(err, os.ErrExist) {
			return err
		}
		info, err = os.Lstat(path)
		if err != nil {
			return err
		}
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("%s is not an ordinary directory", path)
	}
	return os.Chmod(path, privateDirMode)
}

func validateDirectory(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("%s is not an ordinary directory", path)
	}
	return nil
}

func createManifest(path string) error {
	target, err := safeio.Open(path, true)
	if err != nil {
		return err
	}
	defer target.Close()

	_, exists, _, err := target.Read()
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("manifest already exists")
	}
	if err := afterManifestAbsenceCheck(path); err != nil {
		return err
	}
	return target.Create(initialConfig)
}
