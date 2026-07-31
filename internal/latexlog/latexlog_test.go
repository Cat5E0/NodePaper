package latexlog

import (
	"os"
	"path/filepath"
	"regexp"
	"testing"
)

func TestClassifyCriticalLogLines(t *testing.T) {
	data := []byte(`Overfull \hbox (12.0pt too wide)
Missing character: There is no 字 in font test
LaTeX Warning: Reference 'x' on page 1 undefined on input line 2.
Package rerunfilecheck Warning: File changed. Rerun to get outlines right
Package demo Warning: review me
Undefined control sequence.
! LaTeX Error: broken
`)
	findings := Classify(data, Allowlist{SchemaVersion: 1})
	want := []Category{CategoryOverflow, CategoryMissingFont, CategoryUnresolved, CategoryRerun, CategoryUnknownWarning, CategoryFatal, CategoryFatal}
	if len(findings) != len(want) {
		t.Fatalf("findings = %#v", findings)
	}
	for index := range want {
		if findings[index].Category != want[index] {
			t.Fatalf("finding[%d] = %#v, want %s", index, findings[index], want[index])
		}
	}
}

func TestExactReviewedWarningCanBeAllowed(t *testing.T) {
	entry := AllowlistEntry{
		Pattern:      `^Package demo Warning: fixed harmless warning$`,
		Reason:       "upstream emits a harmless marker",
		Source:       "demo package issue 1",
		ToolVersions: []string{"demo 1.0"},
	}
	entry.re = mustCompile(t, entry.Pattern)
	findings := Classify([]byte("Package demo Warning: fixed harmless warning\nPackage demo Warning: another warning\n"), Allowlist{SchemaVersion: 1, Entries: []AllowlistEntry{entry}})
	if findings[0].Category != CategoryAllowedWarning || findings[1].Category != CategoryUnknownWarning {
		t.Fatalf("findings = %#v", findings)
	}
}

func TestLoadAllowlistStrictValidation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "allowlist.json")
	valid := `{"schemaVersion":1,"entries":[{"pattern":"^Package demo Warning:","reason":"known","source":"issue","toolVersions":["1.0"]}]}`
	if err := os.WriteFile(path, []byte(valid), 0o644); err != nil {
		t.Fatal(err)
	}
	allowlist, err := LoadAllowlist(path)
	if err != nil || len(allowlist.Entries) != 1 {
		t.Fatalf("LoadAllowlist() = %#v, %v", allowlist, err)
	}
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1,"entries":[{"pattern":".*"}]}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadAllowlist(path); err == nil {
		t.Fatal("incomplete allowlist entry accepted")
	}
}

func mustCompile(t *testing.T, pattern string) *regexp.Regexp {
	t.Helper()
	re, err := regexp.Compile(pattern)
	if err != nil {
		t.Fatal(err)
	}
	return re
}
