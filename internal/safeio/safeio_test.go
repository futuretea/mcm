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
