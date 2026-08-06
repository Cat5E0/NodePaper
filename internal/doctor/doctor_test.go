package doctor

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFindToolchainPrefersBundledTools(t *testing.T) {
	root := t.TempDir()
	pandocExe := filepath.Join(root, "tools", "windows-x64", "pandoc", "pandoc.exe")
	crossrefExe := filepath.Join(root, "tools", "windows-x64", "pandoc-crossref", "pandoc-crossref.exe")
	if err := os.MkdirAll(filepath.Dir(pandocExe), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(crossrefExe), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pandocExe, []byte("bundled"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(crossrefExe, []byte("bundled"), 0o644); err != nil {
		t.Fatal(err)
	}

	tc := findToolchain(root)
	if tc.Pandoc != pandocExe {
		t.Fatalf("Pandoc = %q, want bundled %q", tc.Pandoc, pandocExe)
	}
	if tc.PandocCrossref != crossrefExe {
		t.Fatalf("PandocCrossref = %q, want bundled %q", tc.PandocCrossref, crossrefExe)
	}

	// A resource root without bundled tools must not fabricate a bundled path.
	tc = findToolchain(t.TempDir())
	if strings.Contains(tc.Pandoc, "tools"+string(filepath.Separator)+"windows-x64") {
		t.Fatalf("Pandoc unexpectedly resolved to a bundled path: %q", tc.Pandoc)
	}
	if strings.Contains(tc.PandocCrossref, "tools"+string(filepath.Separator)+"windows-x64") {
		t.Fatalf("PandocCrossref unexpectedly resolved to a bundled path: %q", tc.PandocCrossref)
	}
}

func TestRunSuccessMatchesFailingChecks(t *testing.T) {
	result := Run(context.Background(), "", "")
	hasFailure := false
	for _, check := range result.Checks {
		if check.Status == StatusFail {
			hasFailure = true
		}
	}
	if result.Success == hasFailure {
		t.Fatalf("Success = %v with hasFailure = %v; checks = %#v", result.Success, hasFailure, result.Checks)
	}
}

func TestFormatChecks(t *testing.T) {
	checks := []Check{
		{Name: "pandoc", Status: StatusPass, Message: "ok"},
		{Name: "latexmk", Status: StatusWarning, Message: "untested", Suggestion: "verify version"},
		{Name: "xelatex", Status: StatusFail, Message: "missing"},
		{Name: "probe", Status: StatusSkipped, Message: "blocked"},
	}
	output := FormatChecks(checks)
	for _, expected := range []string{"PASS", "WARN", "FAIL", "SKIP", "verify version"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("FormatChecks() output missing %q:\n%s", expected, output)
		}
	}
}
