package project

import (
	"errors"
	"os"
	"path/filepath"
	"testing"

	"nodepaper/internal/diagnostic"
)

func TestDiscoverFromCurrentProjectDirectory(t *testing.T) {
	root := newProject(t)

	got, err := DiscoverFrom("", root)
	if err != nil {
		t.Fatalf("DiscoverFrom() error = %v", err)
	}
	assertProject(t, got, root)
}

func TestDiscoverFromProjectSubdirectory(t *testing.T) {
	root := newProject(t)
	subdir := filepath.Join(root, "sections", "model")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := DiscoverFrom("", subdir)
	if err != nil {
		t.Fatalf("DiscoverFrom() error = %v", err)
	}
	assertProject(t, got, root)
}

func TestDiscoverFromExplicitAbsoluteDirectory(t *testing.T) {
	root := newProject(t)
	elsewhere := t.TempDir()

	got, err := DiscoverFrom(root, elsewhere)
	if err != nil {
		t.Fatalf("DiscoverFrom() error = %v", err)
	}
	assertProject(t, got, root)
}

func TestDiscoverFromExplicitRelativeDirectory(t *testing.T) {
	workspace := t.TempDir()
	root := filepath.Join(workspace, "papers", "demo")
	createProject(t, root)

	got, err := DiscoverFrom(filepath.Join("papers", "demo"), workspace)
	if err != nil {
		t.Fatalf("DiscoverFrom() error = %v", err)
	}
	assertProject(t, got, root)
}

func TestExplicitDirectoryTakesPriority(t *testing.T) {
	workspace := t.TempDir()
	current := filepath.Join(workspace, "current")
	explicit := filepath.Join(workspace, "explicit")
	createProject(t, current)
	createProject(t, explicit)

	got, err := DiscoverFrom(explicit, current)
	if err != nil {
		t.Fatalf("DiscoverFrom() error = %v", err)
	}
	assertProject(t, got, explicit)
}

func TestDiscoverFromMissingProject(t *testing.T) {
	start := t.TempDir()

	_, err := DiscoverFrom("", start)
	assertDiscoveryError(t, err, CodeProjectNotFound)
}

func TestDiscoverFromNonexistentExplicitDirectory(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")

	_, err := DiscoverFrom(missing, t.TempDir())
	assertDiscoveryError(t, err, CodeProjectNotFound)
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("error does not preserve os.ErrNotExist: %v", err)
	}
}

func TestDiscoverFromExplicitDirectoryWithoutMarker(t *testing.T) {
	root := t.TempDir()

	_, err := DiscoverFrom(root, t.TempDir())
	assertDiscoveryError(t, err, CodeProjectNotFound)
}

func TestDiscoverFromExplicitFile(t *testing.T) {
	root := t.TempDir()
	file := filepath.Join(root, "paper.md")
	if err := os.WriteFile(file, []byte("# paper\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := DiscoverFrom(file, root)
	assertDiscoveryError(t, err, CodeProjectNotDirectory)
}

func TestDiscoverRejectsDirectoryMarker(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, MarkerFile), 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := DiscoverFrom(root, t.TempDir())
	assertDiscoveryError(t, err, CodeProjectNotDirectory)
}

func TestDiscoverFromRejectsEmptyWorkingDirectory(t *testing.T) {
	_, err := DiscoverFrom("", "")
	assertDiscoveryError(t, err, CodeProjectPathUnreadable)
}

func TestDiscoverFromDoesNotChangeWorkingDirectory(t *testing.T) {
	before, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}

	root := newProject(t)
	if _, err := DiscoverFrom("", root); err != nil {
		t.Fatalf("DiscoverFrom() error = %v", err)
	}

	after, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if after != before {
		t.Fatalf("working directory changed from %q to %q", before, after)
	}
}

func TestProjectResolve(t *testing.T) {
	root := newProject(t)
	project := Project{Root: root, ConfigPath: filepath.Join(root, MarkerFile)}

	relative := filepath.Join("images", "plot.png")
	got, err := project.Resolve(relative)
	if err != nil {
		t.Fatalf("Resolve(relative) error = %v", err)
	}
	if want := filepath.Join(root, relative); got != want {
		t.Fatalf("Resolve(relative) = %q, want %q", got, want)
	}

	absolute := filepath.Join(root, "dist", "paper.pdf")
	got, err = project.Resolve(absolute)
	if err != nil {
		t.Fatalf("Resolve(absolute) error = %v", err)
	}
	if want := filepath.Clean(absolute); got != want {
		t.Fatalf("Resolve(absolute) = %q, want %q", got, want)
	}
}

func TestProjectResolveRejectsPathOutsideRoot(t *testing.T) {
	root := newProject(t)
	project := Project{Root: root, ConfigPath: filepath.Join(root, MarkerFile)}

	for _, path := range []string{
		filepath.Join("..", "outside.md"),
		filepath.Join(t.TempDir(), "outside.md"),
	} {
		_, err := project.Resolve(path)
		if !errors.Is(err, ErrPathOutsideProject) {
			t.Fatalf("Resolve(%q) error = %v, want ErrPathOutsideProject", path, err)
		}
	}
}

func newProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	createProject(t, root)
	return root
}

func createProject(t *testing.T, root string) {
	t.Helper()
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, MarkerFile), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertProject(t *testing.T, got Project, root string) {
	t.Helper()
	wantRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatal(err)
	}
	if got.Root != wantRoot {
		t.Fatalf("Project.Root = %q, want %q", got.Root, wantRoot)
	}
	wantConfig := filepath.Join(wantRoot, MarkerFile)
	if got.ConfigPath != wantConfig {
		t.Fatalf("Project.ConfigPath = %q, want %q", got.ConfigPath, wantConfig)
	}
}

func assertDiscoveryError(t *testing.T, err error, code string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}
	var discoveryErr *DiscoveryError
	if !errors.As(err, &discoveryErr) {
		t.Fatalf("error type = %T, want *DiscoveryError", err)
	}
	if discoveryErr.Diagnostic.Code != code {
		t.Fatalf("diagnostic code = %q, want %q", discoveryErr.Diagnostic.Code, code)
	}
	if discoveryErr.Diagnostic.Severity != diagnostic.SeverityError {
		t.Fatalf("severity = %q, want %q", discoveryErr.Diagnostic.Severity, diagnostic.SeverityError)
	}
	if discoveryErr.Diagnostic.Source != "project" {
		t.Fatalf("source = %q, want project", discoveryErr.Diagnostic.Source)
	}
	if discoveryErr.Diagnostic.Suggestion == "" {
		t.Fatal("suggestion is empty")
	}
}
