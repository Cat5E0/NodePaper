package main

import (
	"bytes"
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

func TestRunCommandIsNotYetAvailable(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"validate"}, &stdout, &stderr); code != 1 {
		t.Fatalf("run() exit code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "not available in this development build") {
		t.Fatalf("stderr = %q, want development-build diagnostic", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}
