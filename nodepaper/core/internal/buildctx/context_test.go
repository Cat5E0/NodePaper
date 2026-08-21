package buildctx

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestNewCreatesDirectories(t *testing.T) {
	root := t.TempDir()
	ctx, err := New(root, []string{"paper.md"}, "cumcm")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	if ctx.BuildID == "" {
		t.Fatal("BuildID is empty")
	}
	if !strings.HasPrefix(ctx.BuildID, "build-") {
		t.Fatalf("BuildID = %q, want prefix build-", ctx.BuildID)
	}
	if ctx.ProjectRoot != root {
		t.Fatalf("ProjectRoot = %q, want %q", ctx.ProjectRoot, root)
	}

	// Verify directories exist.
	for _, dir := range []string{ctx.WorkDir, ctx.LogDir} {
		if info, err := os.Stat(dir); err != nil || !info.IsDir() {
			t.Fatalf("directory not created: %s", dir)
		}
	}

	if ctx.LogPath != filepath.Join(ctx.LogDir, ctx.BuildID+".log") {
		t.Fatalf("LogPath = %q", ctx.LogPath)
	}
}

func TestNewCreatesDistinctBuildIDs(t *testing.T) {
	root := t.TempDir()
	first, err := New(root, nil, "cumcm")
	if err != nil {
		t.Fatal(err)
	}
	second, err := New(root, nil, "cumcm")
	if err != nil {
		t.Fatal(err)
	}
	if first.BuildID == second.BuildID {
		t.Fatalf("consecutive Build IDs collide: %q", first.BuildID)
	}
}

func TestNewCreatesDistinctBuildIDsConcurrently(t *testing.T) {
	root := t.TempDir()
	const count = 32
	ids := make(chan string, count)
	errs := make(chan error, count)
	var wg sync.WaitGroup
	for range count {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ctx, err := New(root, nil, "cumcm")
			if err != nil {
				errs <- err
				return
			}
			ids <- ctx.BuildID
		}()
	}
	wg.Wait()
	close(ids)
	close(errs)
	for err := range errs {
		t.Fatalf("New() error = %v", err)
	}
	seen := make(map[string]struct{}, count)
	for id := range ids {
		if _, exists := seen[id]; exists {
			t.Fatalf("concurrent Build IDs collide: %q", id)
		}
		seen[id] = struct{}{}
	}
	if len(seen) != count {
		t.Fatalf("got %d unique Build IDs, want %d", len(seen), count)
	}
}

func TestResolveInWork(t *testing.T) {
	root := t.TempDir()
	ctx, err := New(root, nil, "cumcm")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	got := ctx.ResolveInWork("paper.tex")
	expected := filepath.Join(ctx.WorkDir, "paper.tex")
	if got != expected {
		t.Fatalf("ResolveInWork() = %q, want %q", got, expected)
	}
}

func TestResolveOutput(t *testing.T) {
	root := t.TempDir()
	ctx, err := New(root, nil, "cumcm")
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}

	got := ctx.ResolveOutput("paper.pdf")
	expected := filepath.Join(root, "dist", "paper.pdf")
	if got != expected {
		t.Fatalf("ResolveOutput() = %q, want %q", got, expected)
	}
}
