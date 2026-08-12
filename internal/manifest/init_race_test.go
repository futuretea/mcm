package manifest

import (
	"errors"
	"os"
	"testing"
)

func TestLocationInitRejectsManifestCreatedAfterAbsenceCheck(t *testing.T) {
	home := t.TempDir()
	location := NewLocation(home, "", "")
	external := []byte("external: manifest\n")

	previousHook := afterManifestAbsenceCheck
	afterManifestAbsenceCheck = func(path string) error {
		return os.WriteFile(path, external, 0o600)
	}
	t.Cleanup(func() {
		afterManifestAbsenceCheck = previousHook
	})

	err := location.Init()
	if err == nil {
		t.Fatal("Init() error = nil, want failure when manifest appears during creation")
	}
	if !errors.Is(err, os.ErrExist) {
		t.Errorf("Init() error = %v, want an already-exists error", err)
	}

	got, readErr := os.ReadFile(location.ConfigPath)
	if readErr != nil {
		t.Fatalf("read externally created manifest: %v", readErr)
	}
	if string(got) != string(external) {
		t.Errorf("manifest content = %q, want external bytes %q", got, external)
	}
}
