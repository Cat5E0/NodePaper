// Package buildlock provides a filesystem-based mutual-exclusion lock that
// prevents concurrent builds on the same project. The lock records enough
// metadata to diagnose stale locks left behind by crashed processes.
package buildlock

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const lockFileName = "build.lock"

// Info is recorded inside the lock file for diagnostic purposes.
type Info struct {
	BuildID     string    `json:"buildId"`
	PID         int       `json:"pid"`
	StartedAt   time.Time `json:"startedAt"`
	ProjectRoot string    `json:"projectRoot"`
}

// Held represents an acquired project lock. Call Release when the build
// completes, fails, or is cancelled.
type Held struct {
	path string
}

// Acquire attempts to create the project lock. It returns ErrHeld when another
// build is in progress, and ErrStale when a lock exists but the owning process
// is no longer running (the caller may delete the lock and retry after
// informing the user).
func Acquire(projectRoot, buildID string) (*Held, error) {
	dotDir := filepath.Join(projectRoot, ".nodepaper")
	if err := os.MkdirAll(dotDir, 0755); err != nil {
		return nil, fmt.Errorf("create .nodepaper: %w", err)
	}

	lockPath := filepath.Join(dotDir, lockFileName)
	info := Info{
		BuildID:     buildID,
		PID:         os.Getpid(),
		StartedAt:   time.Now(),
		ProjectRoot: projectRoot,
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("marshal lock info: %w", err)
	}

	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
	if err != nil {
		if os.IsExist(err) {
			existing, readErr := readLock(lockPath)
			if readErr != nil {
				return nil, fmt.Errorf("cannot read existing lock file %s: %w", lockPath, readErr)
			}
			if !processExists(existing.PID) {
				return nil, &ErrStale{Existing: existing, Path: lockPath}
			}
			return nil, &ErrHeld{Existing: existing}
		}
		return nil, fmt.Errorf("create lock file %s: %w", lockPath, err)
	}
	defer f.Close()

	if _, err := f.Write(data); err != nil {
		os.Remove(lockPath)
		return nil, fmt.Errorf("write lock file: %w", err)
	}

	return &Held{path: lockPath}, nil
}

// Release removes the lock file. Safe to call multiple times.
func (h *Held) Release() error {
	if h == nil || h.path == "" {
		return nil
	}
	err := os.Remove(h.path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("release lock %s: %w", h.path, err)
	}
	h.path = ""
	return nil
}

// LockPath returns the expected lock file path for a project.
func LockPath(projectRoot string) string {
	return filepath.Join(projectRoot, ".nodepaper", lockFileName)
}

// ---------- error types --------------------------------------------------

// ErrHeld is returned when a valid, live lock already exists.
type ErrHeld struct {
	Existing Info
}

func (e *ErrHeld) Error() string {
	return fmt.Sprintf("NP1201: project build is already in progress (build %s, pid %d, started %s)",
		e.Existing.BuildID, e.Existing.PID,
		e.Existing.StartedAt.Format(time.RFC3339))
}

// ErrStale is returned when the lock file exists but the owning process is
// no longer running.
type ErrStale struct {
	Existing Info
	Path     string
}

func (e *ErrStale) Error() string {
	return fmt.Sprintf("stale lock from build %s (pid %d, started %s); suggest running 'nodepaper clean' to remove it",
		e.Existing.BuildID, e.Existing.PID,
		e.Existing.StartedAt.Format(time.RFC3339))
}

// ---------- helpers ------------------------------------------------------

func readLock(path string) (Info, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Info{}, err
	}
	var info Info
	if err := json.Unmarshal(data, &info); err != nil {
		return Info{}, fmt.Errorf("corrupt lock file at %s: %w", path, err)
	}
	return info, nil
}
