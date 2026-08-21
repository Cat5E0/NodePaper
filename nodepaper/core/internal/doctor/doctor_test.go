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
	// Every check Run produces is filed under a capability. A check that
	// reached a renderer without one would still be printed, but under the
	// catch-all heading rather than where its reader would look for it.
	for _, check := range result.Checks {
		if check.Group == "" {
			t.Errorf("check %q carries no Group", check.Name)
		}
	}
}

func TestCheckXeLaTeXMissingIsWarningNotFailure(t *testing.T) {
	// An absent TeX distribution costs exactly one capability. Reported as a
	// failure it took the whole doctor run down with it and told a machine that
	// can still convert and export that its toolchain was broken.
	check := checkXeLaTeX(context.Background(), "")
	if check.Status != StatusWarning {
		t.Fatalf("Status = %v, want StatusWarning; check = %#v", check.Status, check)
	}
	if check.Name != "XeLaTeX" {
		t.Fatalf("Name = %q, want %q", check.Name, "XeLaTeX")
	}
	// The message has to say which capability is lost and which is not, or the
	// warning is just a quieter version of the same confusion.
	for _, want := range []string{"nodepaper build", "nodepaper export"} {
		if !strings.Contains(check.Message, want) {
			t.Errorf("Message does not name %q: %q", want, check.Message)
		}
	}
	if check.Suggestion != texDistributionHelp {
		t.Errorf("Suggestion should stay the full install guidance, got %q", check.Suggestion)
	}
}

func TestCheckXeLaTeXPresentButUnusableStillFails(t *testing.T) {
	// A xelatex that is on PATH and cannot report its version is a damaged
	// installation, not a missing one: install guidance does not help and the
	// build will fail. That must stay a hard failure.
	fake := filepath.Join(t.TempDir(), "xelatex.exe")
	if err := os.WriteFile(fake, []byte("not an executable"), 0o644); err != nil {
		t.Fatal(err)
	}
	check := checkXeLaTeX(context.Background(), fake)
	if check.Status != StatusFail {
		t.Fatalf("Status = %v, want StatusFail; check = %#v", check.Status, check)
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
		{Name: "pandoc", Status: StatusPass, Message: "ok", Group: GroupToolchain},
		{Name: "LaTeX driver", Status: StatusWarning, Message: "untested", Suggestion: "verify version", Group: GroupPDFOutput},
		{Name: "xelatex", Status: StatusFail, Message: "missing", Group: GroupPDFOutput},
		{Name: "probe", Status: StatusSkipped, Message: "blocked", Group: GroupPDFOutput},
	}
	output := FormatChecks(checks)
	for _, expected := range []string{"PASS", "WARN", "FAIL", "SKIP", "verify version"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("FormatChecks() output missing %q:\n%s", expected, output)
		}
	}
}

func TestFormatChecksSectionsFollowTheChecksOwnGroups(t *testing.T) {
	// The last check repeats the first group on purpose: sections are buckets,
	// not runs, so a group must get one heading no matter where its members
	// appear in the list.
	checks := []Check{
		{Name: "Pandoc", Status: StatusPass, Message: "ok", Group: GroupToolchain},
		{Name: "XeLaTeX", Status: StatusWarning, Message: "not found", Group: GroupPDFOutput},
		{Name: "Chinese TeX probe", Status: StatusSkipped, Message: "blocked", Group: GroupPDFOutput},
		{Name: "profile", Status: StatusPass, Message: "loaded", Group: GroupToolchain},
	}
	output := FormatChecks(checks)

	for _, group := range []Group{GroupToolchain, GroupPDFOutput} {
		if count := strings.Count(output, string(group)+"\n"); count != 1 {
			t.Fatalf("heading %q appears %d times, want once:\n%s", group, count, output)
		}
	}
	if strings.Index(output, string(GroupToolchain)) > strings.Index(output, string(GroupPDFOutput)) {
		t.Fatalf("sections are not in order of first appearance:\n%s", output)
	}
	// The out-of-order Toolchain check belongs to its own section, so it is
	// printed before the next heading rather than under it.
	if strings.Index(output, "profile") > strings.Index(output, string(GroupPDFOutput)) {
		t.Fatalf("profile was filed under the wrong section:\n%s", output)
	}
	for _, name := range []string{"Pandoc", "XeLaTeX", "Chinese TeX probe", "profile"} {
		if !strings.Contains(output, name) {
			t.Fatalf("output lost check %q:\n%s", name, output)
		}
	}
}

func TestFormatChecksKeepsACheckWithNoGroup(t *testing.T) {
	// The check below belongs to no known capability - it is what a future
	// check looks like before anyone remembers to group it. Grouping may
	// misplace it; it may never swallow it.
	checks := []Check{
		{Name: "Pandoc", Status: StatusPass, Message: "ok", Group: GroupToolchain},
		{Name: "unregistered check", Status: StatusFail, Message: "invented by a later change", Suggestion: "still shown"},
	}
	output := FormatChecks(checks)
	for _, expected := range []string{"unregistered check", "invented by a later change", "still shown", string(GroupOther)} {
		if !strings.Contains(output, expected) {
			t.Fatalf("output missing %q:\n%s", expected, output)
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
