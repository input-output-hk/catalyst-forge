package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"time"
)

// Lock represents a filesystem-based lock.
type Lock struct {
	path string
	file *os.File
}

// AcquireLock attempts to acquire an exclusive lock on the given path.
// This uses advisory locking via flock on Unix systems.
func AcquireLock(path string) (*Lock, error) {
	// Ensure lock directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}

	// Open or create the lock file
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	// Try to acquire exclusive lock with retries
	maxRetries := 100
	retryDelay := 10 * time.Millisecond
	
	for i := 0; i < maxRetries; i++ {
		err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
		if err == nil {
			// Lock acquired successfully
			return &Lock{
				path: path,
				file: file,
			}, nil
		}

		if err != syscall.EWOULDBLOCK {
			// Unexpected error
			_ = file.Close()
			return nil, fmt.Errorf("failed to acquire lock: %w", err)
		}

		// Lock is held by another process, wait and retry
		if i < maxRetries-1 {
			time.Sleep(retryDelay)
			// Exponential backoff with max delay
			if retryDelay < 100*time.Millisecond {
				retryDelay = retryDelay * 2
			}
		}
	}

	_ = file.Close()
	return nil, fmt.Errorf("failed to acquire lock after %d retries", maxRetries)
}

// TryAcquireLock attempts to acquire a lock without blocking.
func TryAcquireLock(path string) (*Lock, error) {
	// Ensure lock directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create lock directory: %w", err)
	}

	// Open or create the lock file
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open lock file: %w", err)
	}

	// Try to acquire exclusive lock without blocking
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
	if err != nil {
		_ = file.Close()
		if err == syscall.EWOULDBLOCK {
			return nil, fmt.Errorf("lock is already held")
		}
		return nil, fmt.Errorf("failed to acquire lock: %w", err)
	}

	return &Lock{
		path: path,
		file: file,
	}, nil
}

// Release releases the lock and removes the lock file.
func (l *Lock) Release() error {
	if l.file == nil {
		return nil
	}

	// Release the lock
	err := syscall.Flock(int(l.file.Fd()), syscall.LOCK_UN)
	if err != nil {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	// Close the file
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("failed to close lock file: %w", err)
	}

	// Remove the lock file
	if err := os.Remove(l.path); err != nil && !os.IsNotExist(err) {
		// It's okay if the file doesn't exist
		return fmt.Errorf("failed to remove lock file: %w", err)
	}

	l.file = nil
	return nil
}

// IsLocked checks if a lock file exists and is locked.
func IsLocked(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer func() { _ = file.Close() }()

	// Try to acquire a shared lock without blocking
	err = syscall.Flock(int(file.Fd()), syscall.LOCK_SH|syscall.LOCK_NB)
	if err == syscall.EWOULDBLOCK {
		return true
	}

	// If we got the lock, release it
	if err == nil {
		_ = syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
	}

	return false
}

// CleanStaleLocks removes lock files that are not currently held.
// This is useful for cleanup after crashes.
func CleanStaleLocks(dir string) error {
	files, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		lockPath := filepath.Join(dir, file.Name())
		if !IsLocked(lockPath) {
			// Lock is not held, safe to remove
			_ = os.Remove(lockPath)
		}
	}

	return nil
}