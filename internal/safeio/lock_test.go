package safeio

import (
	"path/filepath"
	"testing"
)

func TestAcquireLock_WhenLockIsHeld_ReturnsError(t *testing.T) {
	lockPath := filepath.Join(t.TempDir(), "mcm.lock")

	firstLock, err := AcquireLock(lockPath)
	if err != nil {
		t.Fatalf("acquire first lock: %v", err)
	}
	t.Cleanup(func() {
		if err := firstLock.Close(); err != nil {
			t.Errorf("close first lock: %v", err)
		}
	})

	if _, err := AcquireLock(lockPath); err == nil {
		t.Fatal("acquire second lock: expected an error while the first lock is held")
	}
}
