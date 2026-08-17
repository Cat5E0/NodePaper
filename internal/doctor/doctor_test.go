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

func TestLatexExportPackagesResultAlwaysPasses(t *testing.T) {
	cases := []struct {
		name                       string
		gbt7714, biblatexGB, biber bool
	}{
		{name: "all present", gbt7714: true, biblatexGB: true, biber: true},
		{name: "all absent", gbt7714: false, biblatexGB: false, biber: false},
		{name: "partially present", gbt7714: true, biblatexGB: false, biber: true},
	}

	forbidden := []string{"missing", "required", "not found", "unsupported", "invalid", "must install"}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			check := latexExportPackagesResult(tc.gbt7714, tc.biblatexGB, tc.biber)
			if check.Status != StatusPass {
				t.Fatalf("Status = %v, want StatusPass", check.Status)
			}
			if check.Name != latexExportPackagesCheckName {
				t.Fatalf("Name = %q, want %q", check.Name, latexExportPackagesCheckName)
			}
			for _, word := range forbidden {
				if strings.Contains(check.Message, word) {
					t.Fatalf("Message contains judgmental word %q: %q", word, check.Message)
				}
				if strings.Contains(check.Suggestion, word) {
					t.Fatalf("Suggestion contains judgmental word %q: %q", word, check.Suggestion)
				}
			}
			allPresent := tc.gbt7714 && tc.biblatexGB && tc.biber
			if allPresent && check.Suggestion != "" {
				t.Fatalf("Suggestion should be empty when everything is present, got %q", check.Suggestion)
			}
			if !allPresent {
				if check.Suggestion == "" {
					t.Fatalf("Suggestion should be non-empty when something is absent")
				}
				// The wording must stay conditional ("if you would like to
				// ... you can optionally"), never an instruction. Most users
				// never export, so their absence is a normal state.
				if !strings.Contains(check.Suggestion, "If you would like to export") ||
					!strings.Contains(check.Suggestion, "optionally install") {
					t.Fatalf("Suggestion is not phrased as conditional: %q", check.Suggestion)
				}
				if !strings.Contains(check.Suggestion, "tlmgr install") || !strings.Contains(check.Suggestion, "miktex packages install") {
					t.Fatalf("Suggestion missing install commands: %q", check.Suggestion)
				}
			}
		})
	}
}

func TestCheckLaTeXExportPackagesAlwaysPasses(t *testing.T) {
	// Regardless of what is actually installed on the machine running the
	// test, this check must never be anything other than StatusPass.
	check := checkLaTeXExportPackages(context.Background())
	if check.Status != StatusPass {
		t.Fatalf("Status = %v, want StatusPass; check = %#v", check.Status, check)
	}
	if check.Name != latexExportPackagesCheckName {
		t.Fatalf("Name = %q, want %q", check.Name, latexExportPackagesCheckName)
	}
	if check.Message == "" {
		t.Fatalf("Message should not be empty")
	}
}

func TestCheckLaTeXExportPackagesWithoutKpsewhich(t *testing.T) {
	// With PATH pointing only at an empty directory, kpsewhich cannot be
	// found at all. The check must still pass and must not blame the
	// missing TeX installation - that is the XeLaTeX check's job.
	t.Setenv("PATH", t.TempDir())

	check := checkLaTeXExportPackages(context.Background())
	if check.Status != StatusPass {
		t.Fatalf("Status = %v, want StatusPass; check = %#v", check.Status, check)
	}
	for _, word := range []string{"missing", "required", "not found", "unsupported", "invalid", "must install"} {
		if strings.Contains(check.Message, word) {
			t.Fatalf("Message contains judgmental word %q: %q", word, check.Message)
		}
	}
}

func TestFormatChecks(t *testing.T) {
	checks := []Check{
		{Name: "pandoc", Status: StatusPass, Message: "ok"},
		{Name: "LaTeX driver", Status: StatusWarning, Message: "untested", Suggestion: "verify version"},
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

func TestFirstLaTeXError(t *testing.T) {
	cases := map[string]struct {
		sources []string
		want    string
	}{
		"bang form": {
			sources: []string{"This is XeTeX\n! LaTeX Error: File `xeCJK.sty' not found.\nmore\n"},
			want:    "LaTeX Error: File `xeCJK.sty' not found.",
		},
		"file-line-error form": {
			sources: []string{"This is XeTeX\n./probe.tex:29: ! Undefined control sequence.\n"},
			want:    "Undefined control sequence.",
		},
		"no error anywhere": {
			sources: []string{"This is XeTeX\nOutput written on probe.pdf.\n", ""},
			want:    "",
		},
		// The captured streams are passed before the log precisely because the
		// log may be absent; a later source must still be consulted.
		"second source carries it": {
			sources: []string{"", "! Emergency stop.\n"},
			want:    "Emergency stop.",
		},
		"earlier source wins": {
			sources: []string{"! first.\n", "! second.\n"},
			want:    "first.",
		},
		"no sources": {sources: nil, want: ""},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			if got := firstLaTeXError(tc.sources...); got != tc.want {
				t.Fatalf("firstLaTeXError() = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestReadFileString(t *testing.T) {
	path := filepath.Join(t.TempDir(), "probe.log")
	if err := os.WriteFile(path, []byte("content"), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := readFileString(path); got != "content" {
		t.Fatalf("readFileString() = %q, want %q", got, "content")
	}
	if got := readFileString(filepath.Join(t.TempDir(), "absent.log")); got != "" {
		t.Fatalf("unreadable file should yield %q, got %q", "", got)
	}
}
