package safeio

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenRejectsFinalSymlinkWithoutReadingTarget(t *testing.T) {
	directory := t.TempDir()
	sentinel := filepath.Join(directory, "sentinel")
	if err := os.WriteFile(sentinel, []byte("must not be read"), 0o600); err != nil {
		t.Fatalf("write sentinel: %v", err)
	}

	target := filepath.Join(directory, "target")
	if err := os.Symlink(sentinel, target); err != nil {
		t.Fatalf("create target symlink: %v", err)
	}

	if _, err := Open(target, false); err == nil {
		t.Fatal("Open accepted a final symlink")
	}
}

func TestRemoveDeletesRegularEntry(t *testing.T) {
	directory := t.TempDir()
	targetPath := filepath.Join(directory, "target")
	if err := os.WriteFile(targetPath, []byte("remove me"), 0o600); err != nil {
		t.Fatalf("write target: %v", err)
	}
	target, err := Open(targetPath, true)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	defer target.Close()
	if err := target.Remove(); err != nil {
		t.Fatalf("Remove() error: %v", err)
	}
	if _, err := os.Lstat(targetPath); !os.IsNotExist(err) {
		t.Errorf("target after Remove() error = %v, want not exist", err)
	}
}
