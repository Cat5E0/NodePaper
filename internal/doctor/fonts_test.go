package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeProbeLog(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "probe.log")
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatalf("write probe log: %v", err)
	}
	return path
}

func TestChineseProbeReportsResolvedFonts(t *testing.T) {
	path := writeProbeLog(t, `Package fontspec Info:
(fontspec)             Font family 'SimSun(0)' created for font 'SimSun' with
(fontspec)             options [Script={CJK},BoldFont={SimHei},ItalicFont={KaiTi}].
Package fontspec Info:
(fontspec)             Font family 'SimHei(0)' created for font 'SimHei' with
(fontspec)             options [Script={CJK}].
`)
	got := chineseProbeResult(path)
	if got.Status != StatusPass {
		t.Fatalf("Status = %v, want pass", got.Status)
	}
	// Naming the fonts is the point: two machines whose PDFs differ can be
	// compared on this line instead of on guesswork.
	for _, want := range []string{"SimSun", "SimHei"} {
		if !strings.Contains(got.Message, want) {
			t.Fatalf("Message does not name %s: %q", want, got.Message)
		}
	}
}

func TestChineseProbeWarnsWhenStylesAreSynthesised(t *testing.T) {
	path := writeProbeLog(t, `Package fontspec Info:
(fontspec)             Font family 'SimSun(0)' created for font 'SimSun' with
(fontspec)             options [Script={CJK},AutoFakeBold={4},AutoFakeSlant={0.167}].
Package fontspec Info:
(fontspec)             Font family 'SimSun(1)' created for font 'SimSun' with
(fontspec)             options [Script={CJK},AutoFakeBold={4}].
`)
	got := chineseProbeResult(path)
	// Warning, not fail: builds succeed and nothing is lost, so failing here
	// would misrepresent a cosmetic compromise as a broken environment.
	if got.Status != StatusWarning {
		t.Fatalf("Status = %v, want warning", got.Status)
	}
	if strings.Count(got.Message, "SimSun") != 1 {
		t.Fatalf("Message repeats the font: %q", got.Message)
	}
	if !strings.Contains(got.Suggestion, "Optional features") {
		t.Fatalf("Suggestion lacks the install route: %q", got.Suggestion)
	}
}

// An unreadable or fontspec-free log still means the probe compiled; the check
// falls back to the plain success message rather than inventing a problem.
func TestChineseProbeFallsBackWithoutFontRecords(t *testing.T) {
	for name, path := range map[string]string{
		"missing log":  filepath.Join(t.TempDir(), "absent.log"),
		"no fontspec":  writeProbeLog(t, "This is XeTeX, Version 3.141592653\nOutput written on probe.pdf.\n"),
	} {
		got := chineseProbeResult(path)
		if got.Status != StatusPass {
			t.Fatalf("%s: Status = %v, want pass", name, got.Status)
		}
	}
}

func TestXeLaTeXHelpIsActionable(t *testing.T) {
	got := checkXeLaTeX(t.Context(), "")
	if got.Status != StatusFail {
		t.Fatalf("Status = %v, want fail", got.Status)
	}
	// The one-line version of this message told users to install TeX without
	// saying it is a multi-gigabyte, multi-hour step. Each of these is the
	// piece that was missing.
	for _, want := range []string{
		"miktex.org/download",
		"tug.org/texlive/windows.html",
		"~140 MB",
		"~6.3 GB",
		"NEW terminal",
		"does not require latexmk or Perl",
	} {
		if !strings.Contains(got.Suggestion, want) {
			t.Fatalf("Suggestion is missing %q:\n%s", want, got.Suggestion)
		}
	}
}
