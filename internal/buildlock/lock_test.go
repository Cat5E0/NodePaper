package buildlock

import (
	"encoding/json"
	"os"
	"testing"
	"time"
)

func TestAcquireRelease(t *testing.T) {
	root := t.TempDir()
	held, err := Acquire(root, "build-001")
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if held == nil {
		t.Fatal("Acquire() held = nil")
	}

	// Verify lock file exists.
	lockPath := LockPath(root)
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		t.Fatal("lock file was not created")
	}

	// Release should remove the lock.
	if err := held.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatal("lock file still exists after release")
	}
}

func TestDoubleAcquire(t *testing.T) {
	root := t.TempDir()

	held, err := Acquire(root, "build-001")
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	defer held.Release()

	_, err = Acquire(root, "build-002")
	if err == nil {
		t.Fatal("second Acquire() error = nil")
	}
	if _, ok := err.(*ErrHeld); !ok {
		t.Fatalf("second Acquire() error type = %T, want *ErrHeld", err)
	}
}

func TestDifferentProjectsDoNotConflict(t *testing.T) {
	dirA := t.TempDir()
	dirB := t.TempDir()

	heldA, err := Acquire(dirA, "build-a")
	if err != nil {
		t.Fatalf("Acquire(A) error = %v", err)
	}
	defer heldA.Release()

	heldB, err := Acquire(dirB, "build-b")
	if err != nil {
		t.Fatalf("Acquire(B) error = %v", err)
	}
	heldB.Release()
}

func TestReleaseAfterFailure(t *testing.T) {
	root := t.TempDir()

	held, err := Acquire(root, "build-001")
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if err := held.Release(); err != nil {
		t.Fatalf("Release() error = %v", err)
	}

	// Second acquire should succeed.
	held2, err := Acquire(root, "build-002")
	if err != nil {
		t.Fatalf("Acquire() after release error = %v", err)
	}
	held2.Release()
}

func TestReleaseNil(t *testing.T) {
	var held *Held
	if err := held.Release(); err != nil {
		t.Fatalf("Release() on nil = %v", err)
	}
}

func TestDoubleRelease(t *testing.T) {
	root := t.TempDir()

	held, err := Acquire(root, "build-001")
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if err := held.Release(); err != nil {
		t.Fatalf("first Release() error = %v", err)
	}
	if err := held.Release(); err != nil {
		t.Fatalf("second Release() error = %v", err)
	}
}

func TestStaleLockDetection(t *testing.T) {
	root := t.TempDir()
	dotDir := root + "/.nodepaper"
	os.MkdirAll(dotDir, 0755)

	// Write a lock file referencing a PID that does not exist.
	info := Info{
		BuildID:     "build-old",
		PID:         99999, // unlikely to exist
		StartedAt:   testTime(),
		ProjectRoot: root,
	}
	writeLockFile(t, LockPath(root), info)

	_, err := Acquire(root, "build-new")
	if err == nil {
		t.Fatal("Acquire() on stale lock error = nil")
	}
	if _, ok := err.(*ErrStale); !ok {
		t.Fatalf("Acquire() error type = %T, want *ErrStale: %v", err, err)
	}
}

// helpers

func testTime() time.Time {
	return time.Date(2026, 7, 21, 10, 0, 0, 0, time.FixedZone("CST", 8*3600))
}

func jsonMarshal(v any) ([]byte, error) {
	return json.MarshalIndent(v, "", "  ")
}

func writeLockFile(t *testing.T, path string, info Info) {
	t.Helper()
	data, err := jsonMarshal(info)
	if err != nil {
		t.Fatalf("writeLockFile marshal: %v", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatalf("writeLockFile: %v", err)
	}
}
