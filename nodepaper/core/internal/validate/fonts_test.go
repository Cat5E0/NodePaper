package validate

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"nodepaper/internal/diagnostic"
)

// pointFontDirsAt redirects the probe at a directory the test controls, so the
// machine running the suite cannot decide the outcome.
func pointFontDirsAt(t *testing.T, dir string) {
	t.Helper()
	t.Setenv("WINDIR", dir)
	t.Setenv("LOCALAPPDATA", filepath.Join(dir, "local"))
}

func writeFontFile(t *testing.T, dir, name string) {
	t.Helper()
	fontsDir := filepath.Join(dir, "Fonts")
	if err := os.MkdirAll(fontsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(fontsDir, name), []byte("stub"), 0o644); err != nil {
		t.Fatalf("write %s: %v", name, err)
	}
}

func requireWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "windows" {
		t.Skip("the supplemental font probe only applies to Windows")
	}
}

func TestChineseFontsPresentProducesNoDiagnostic(t *testing.T) {
	requireWindows(t)
	dir := t.TempDir()
	writeFontFile(t, dir, "simhei.ttf")
	writeFontFile(t, dir, "simkai.ttf")
	pointFontDirsAt(t, dir)

	if diags := checkChineseFonts(); diags != nil {
		t.Fatalf("diagnostics = %#v, want none when both fonts are installed", diags)
	}
}

func TestChineseFontsMissingWarnsWithoutFailing(t *testing.T) {
	requireWindows(t)
	dir := t.TempDir()
	writeFontFile(t, dir, "simkai.ttf") // KaiTi present, SimHei absent
	pointFontDirsAt(t, dir)

	diags := checkChineseFonts()
	if len(diags) != 1 {
		t.Fatalf("diagnostics = %#v, want exactly one", diags)
	}
	got := diags[0]
	if got.Code != "NP2403" {
		t.Fatalf("Code = %q, want NP2403", got.Code)
	}
	// A missing optional font must never block a build: NodePaper synthesises
	// the weight and the author still gets a paper.
	if got.Severity != diagnostic.SeverityWarning {
		t.Fatalf("Severity = %v, want warning", got.Severity)
	}
	if hasErrors(diags) {
		t.Fatal("NP2403 must not make validate fail")
	}
	if !strings.Contains(got.Message, "SimHei") {
		t.Fatalf("Message does not name the missing font: %q", got.Message)
	}
	if strings.Contains(got.Message, "KaiTi") {
		t.Fatalf("Message names an installed font: %q", got.Message)
	}
	// The author needs an action, not just a fact.
	if !strings.Contains(got.Suggestion, "Optional features") {
		t.Fatalf("Suggestion lacks the install route: %q", got.Suggestion)
	}
}

// A probe that cannot see any font directory proves nothing. Warning anyway
// would train users to ignore the message, so silence is the correct output.
func TestChineseFontsUndeterminedStaysSilent(t *testing.T) {
	requireWindows(t)
	t.Setenv("WINDIR", "")
	t.Setenv("LOCALAPPDATA", "")

	if diags := checkChineseFonts(); diags != nil {
		t.Fatalf("diagnostics = %#v, want none when the probe cannot resolve a font directory", diags)
	}
}

// The classification of a single font file now lives in internal/fonts and is
// covered by that package's own tests; what validate still owns is the
// diagnostic built on top of it, above.
