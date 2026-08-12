// Package safeio provides descriptor-bound private file operations for macOS and Linux.
package safeio

import (
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/sys/unix"
)

const privateFileMode = 0o600

// Target owns a validated parent directory descriptor and one final entry name.
type Target struct {
	dirFD  int
	name   string
	closed bool
}

// AcquireLock obtains a non-blocking advisory lock without following the final entry.
func AcquireLock(path string) (io.Closer, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("lock path must be absolute")
	}
	dirFD, err := unix.Open(filepath.Dir(path), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open lock directory: %w", err)
	}
	name := filepath.Base(path)
	fd, err := unix.Openat(dirFD, name, unix.O_RDWR|unix.O_CREAT|unix.O_NOFOLLOW|unix.O_CLOEXEC, privateFileMode)
	unix.Close(dirFD)
	if err != nil {
		return nil, fmt.Errorf("open lock: %w", err)
	}
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("stat lock: %w", err)
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFREG {
		unix.Close(fd)
		return nil, fmt.Errorf("lock is not a regular file")
	}
	if err := unix.Flock(fd, unix.LOCK_EX|unix.LOCK_NB); err != nil {
		unix.Close(fd)
		return nil, fmt.Errorf("MCM lock is held: %w", err)
	}
	file := os.NewFile(uintptr(fd), name)
	if err := file.Chmod(privateFileMode); err != nil {
		file.Close()
		return nil, fmt.Errorf("set lock mode: %w", err)
	}
	return file, nil
}

// Open validates path's parent and final entry without following a final symlink.
func Open(path string, allowMissing bool) (*Target, error) {
	if !filepath.IsAbs(path) {
		return nil, fmt.Errorf("path must be absolute")
	}
	name := filepath.Base(path)
	if name == "." || name == string(filepath.Separator) {
		return nil, fmt.Errorf("path must name a file")
	}
	dirFD, err := unix.Open(filepath.Dir(path), unix.O_RDONLY|unix.O_DIRECTORY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if err != nil {
		return nil, fmt.Errorf("open parent directory: %w", err)
	}
	target := &Target{dirFD: dirFD, name: name}
	_, exists, _, err := target.Read()
	if err != nil {
		target.Close()
		return nil, err
	}
	if !exists && !allowMissing {
		target.Close()
		return nil, fmt.Errorf("target does not exist")
	}
	return target, nil
}

// Close releases the parent directory descriptor.
func (target *Target) Close() error {
	if target == nil || target.closed {
		return nil
	}
	target.closed = true
	return unix.Close(target.dirFD)
}

// Read reopens and validates the final entry through the held parent descriptor.
func (target *Target) Read() ([]byte, bool, os.FileMode, error) {
	if target == nil || target.closed {
		return nil, false, 0, fmt.Errorf("target is closed")
	}
	fd, err := unix.Openat(target.dirFD, target.name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if errors.Is(err, unix.ENOENT) {
		return nil, false, 0, nil
	}
	if err != nil {
		return nil, false, 0, fmt.Errorf("open target: %w", err)
	}

	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		unix.Close(fd)
		return nil, false, 0, fmt.Errorf("stat target: %w", err)
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFREG {
		unix.Close(fd)
		return nil, false, 0, fmt.Errorf("target is not a regular file")
	}

	file := os.NewFile(uintptr(fd), target.name)
	data, readErr := io.ReadAll(file)
	closeErr := file.Close()
	if readErr != nil {
		return nil, false, 0, fmt.Errorf("read target: %w", readErr)
	}
	if closeErr != nil {
		return nil, false, 0, fmt.Errorf("close target: %w", closeErr)
	}
	return data, true, os.FileMode(stat.Mode & 0o777), nil
}

// EntryExists reports whether another regular entry exists in the held parent directory.
func (target *Target) EntryExists(name string) (bool, error) {
	if target == nil || target.closed {
		return false, fmt.Errorf("target is closed")
	}
	if filepath.Base(name) != name || name == "." || name == string(filepath.Separator) {
		return false, fmt.Errorf("entry must name a file")
	}
	fd, err := unix.Openat(target.dirFD, name, unix.O_RDONLY|unix.O_NOFOLLOW|unix.O_CLOEXEC, 0)
	if errors.Is(err, unix.ENOENT) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("open entry: %w", err)
	}
	defer unix.Close(fd)
	var stat unix.Stat_t
	if err := unix.Fstat(fd, &stat); err != nil {
		return false, fmt.Errorf("stat entry: %w", err)
	}
	if stat.Mode&unix.S_IFMT != unix.S_IFREG {
		return false, fmt.Errorf("entry is not a regular file")
	}
	return true, nil
}

// Replace atomically swaps the final entry without following it.
func (target *Target) Replace(data []byte, existingMode os.FileMode) error {
	if target == nil || target.closed {
		return fmt.Errorf("target is closed")
	}
	tempName, fd, err := target.createTemp()
	if err != nil {
		return err
	}
	committed := false
	defer func() {
		if !committed {
			_ = unix.Unlinkat(target.dirFD, tempName, 0)
		}
	}()

	file := os.NewFile(uintptr(fd), tempName)
	if _, err := file.Write(data); err != nil {
		file.Close()
		return fmt.Errorf("write temporary target: %w", err)
	}
	if err := file.Chmod(replacementMode(existingMode)); err != nil {
		file.Close()
		return fmt.Errorf("set temporary target mode: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("sync temporary target: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary target: %w", err)
	}
	if err := unix.Renameat(target.dirFD, tempName, target.dirFD, target.name); err != nil {
		return fmt.Errorf("replace target: %w", err)
	}
	committed = true
	if err := unix.Fsync(target.dirFD); err != nil {
		return fmt.Errorf("sync target directory: %w", err)
	}
	return nil
}

// Create writes a new final entry without replacing an entry created concurrently.
func (target *Target) Create(data []byte) error {
	if target == nil || target.closed {
		return fmt.Errorf("target is closed")
	}
	tempName, fd, err := target.createTemp()
	if err != nil {
		return err
	}
	defer unix.Unlinkat(target.dirFD, tempName, 0)

	file := os.NewFile(uintptr(fd), tempName)
	if _, err := file.Write(data); err != nil {
		file.Close()
		return fmt.Errorf("write temporary target: %w", err)
	}
	if err := file.Chmod(privateFileMode); err != nil {
		file.Close()
		return fmt.Errorf("set temporary target mode: %w", err)
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("sync temporary target: %w", err)
	}
	if err := file.Close(); err != nil {
		return fmt.Errorf("close temporary target: %w", err)
	}
	if err := unix.Linkat(target.dirFD, tempName, target.dirFD, target.name, 0); err != nil {
		return fmt.Errorf("create target: %w", err)
	}
	if err := unix.Fsync(target.dirFD); err != nil {
		return fmt.Errorf("sync target directory: %w", err)
	}
	return nil
}

func (target *Target) createTemp() (string, int, error) {
	for range 16 {
		var suffix [8]byte
		if _, err := rand.Read(suffix[:]); err != nil {
			return "", -1, fmt.Errorf("generate temporary name: %w", err)
		}
		name := fmt.Sprintf(".%s.mcm-%x", target.name, suffix)
		fd, err := unix.Openat(target.dirFD, name, unix.O_WRONLY|unix.O_CREAT|unix.O_EXCL|unix.O_NOFOLLOW|unix.O_CLOEXEC, privateFileMode)
		if errors.Is(err, unix.EEXIST) {
			continue
		}
		if err != nil {
			return "", -1, fmt.Errorf("create temporary target: %w", err)
		}
		return name, fd, nil
	}
	return "", -1, fmt.Errorf("allocate unique temporary target")
}

func replacementMode(existing os.FileMode) os.FileMode {
	if existing == 0 || existing.Perm()&0o077 != 0 {
		return privateFileMode
	}
	return existing.Perm()
}
