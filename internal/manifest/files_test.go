package manifest

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLocationInitCreatesPrivateManifestFiles(t *testing.T) {
	home := t.TempDir()
	location := NewLocation(home, "", "")

	if err := location.Init(); err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	assertDirectoryMode(t, location.Root, 0o700)
	assertDirectoryMode(t, filepath.Join(location.Root, "journal"), 0o700)
	assertDirectoryMode(t, filepath.Join(location.Root, "exports"), 0o700)

	if filepath.Base(location.ConfigPath) != "config.yaml" {
		t.Errorf("ConfigPath base = %q, want config.yaml", filepath.Base(location.ConfigPath))
	}
	info, err := os.Stat(location.ConfigPath)
	if err != nil {
		t.Fatalf("stat config file %q: %v", location.ConfigPath, err)
	}
	if !info.Mode().IsRegular() {
		t.Errorf("config file mode = %v, want regular file", info.Mode())
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("config file permissions = %o, want %o", got, 0o600)
	}
}

func assertDirectoryMode(t *testing.T, path string, want os.FileMode) {
	t.Helper()

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat directory %q: %v", path, err)
	}
	if !info.IsDir() {
		t.Errorf("path %q mode = %v, want directory", path, info.Mode())
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("directory %q permissions = %o, want %o", path, got, want)
	}
}
