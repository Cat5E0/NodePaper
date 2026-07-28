package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunVersion(t *testing.T) {
	oldVersion := version
	version = "0.1.0-test"
	t.Cleanup(func() { version = oldVersion })

	var stdout, stderr bytes.Buffer
	if code := run([]string{"--version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run() exit code = %d, want 0", code)
	}
	if got, want := stdout.String(), "nodepaper 0.1.0-test\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr); code != 0 {
		t.Fatalf("run() exit code = %d, want 0", code)
	}
	if !strings.HasPrefix(stdout.String(), "Usage:\n") {
		t.Fatalf("stdout does not contain usage: %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunUsageError(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"build", "--format", "yaml"}, &stdout, &stderr); code != 2 {
		t.Fatalf("run() exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "unsupported format") {
		t.Fatalf("stderr = %q, want format error", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

func TestRunInit(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "test-project")

	var stdout, stderr bytes.Buffer
	if code := run([]string{"init", projectDir}, &stdout, &stderr); code != 0 {
		t.Fatalf("run() exit code = %d, want 0\nstderr: %s", code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "✓ Success") {
		t.Fatalf("stdout missing success: %s", out)
	}
	if !strings.Contains(out, "Project:") {
		t.Fatalf("stdout missing Project: %s", out)
	}
	if !strings.Contains(out, "markdown:") {
		t.Fatalf("stdout missing artifact: %s", out)
	}

	// Verify files were actually created.
	for _, name := range []string{"nodepaper.yaml", "paper.md", "references.bib", ".gitignore"} {
		path := filepath.Join(projectDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Fatalf("expected file not created: %s", path)
		}
	}
	imagesDir := filepath.Join(projectDir, "images")
	if info, err := os.Stat(imagesDir); err != nil || !info.IsDir() {
		t.Fatalf("images/ directory not created: %v", err)
	}
}

func TestRunInitJSON(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "test-project")

	var stdout, stderr bytes.Buffer
	if code := run([]string{"init", projectDir, "--format", "json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run() exit code = %d, want 0\nstderr: %s", code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, `"schemaVersion"`) {
		t.Fatalf("stdout missing schemaVersion: %s", out)
	}
	if !strings.Contains(out, `"success"`) {
		t.Fatalf("stdout missing success: %s", out)
	}
	if !strings.Contains(out, `"markdown"`) {
		t.Fatalf("stdout missing artifact: %s", out)
	}
}

func TestRunValidateJSONFailureUsesNonzeroExitAndPureJSON(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "invalid-project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "nodepaper.yaml"), []byte("version: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"validate", projectDir, "--format", "json"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("run() exit code = %d, want 1; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty for a rendered business failure", stderr.String())
	}

	var payload struct {
		SchemaVersion int  `json:"schemaVersion"`
		Success       bool `json:"success"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("stdout is not one valid JSON object: %v\n%s", err, stdout.String())
	}
	if payload.SchemaVersion != 1 || payload.Success {
		t.Fatalf("payload = %+v, want schemaVersion=1 success=false", payload)
	}
}

func TestRunDoctor(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// Doctor without project dir checks global toolchain only.
	code := run([]string{"doctor"}, &stdout, &stderr)
	// Doctor may succeed or fail depending on available tools, but should
	// not crash or return a "not yet implemented" error.
	if stderr.Len() > 0 {
		t.Logf("stderr (may contain warnings): %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "PASS") && !strings.Contains(stdout.String(), "FAIL") && !strings.Contains(stdout.String(), "WARN") {
		t.Fatalf("stdout should contain doctor checks: %s", stdout.String())
	}
	_ = code
}

func TestRunValidate(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "test-project")

	// Init a project first.
	var stdout, stderr bytes.Buffer
	code := run([]string{"init", projectDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("init exit code = %d, stderr: %s", code, stderr.String())
	}

	// Validate the project.
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"validate", projectDir}, &stdout, &stderr)
	// Should succeed since the init creates a valid project skeleton.
	if stderr.Len() > 0 {
		t.Logf("stderr: %s", stderr.String())
	}
	if code != 0 {
		t.Fatalf("validate exit code = %d, stdout: %s, stderr: %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "✓ Success") {
		t.Fatalf("stdout missing success: %s", stdout.String())
	}
}

func TestRunBuild(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "test-project")

	// Init a project first.
	var stdout, stderr bytes.Buffer
	code := run([]string{"init", projectDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("init exit code = %d, stderr: %s", code, stderr.String())
	}

	// Build - may fail if pandoc/latexmk are not available, but should not crash.
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"build", projectDir}, &stdout, &stderr)
	// Build may succeed or fail depending on toolchain; just verify it doesn't panic.
	t.Logf("build exit code: %d, stdout: %s, stderr: %s", code, stdout.String(), stderr.String())
}

func TestRunClean(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "test-project")

	// Init a project first.
	var stdout, stderr bytes.Buffer
	code := run([]string{"init", projectDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("init exit code = %d, stderr: %s", code, stderr.String())
	}

	// Clean.
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"clean", projectDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("clean exit code = %d, stderr: %s", code, stderr.String())
	}
}
