package doctor

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"nodepaper/internal/fonts"
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
		"missing log": filepath.Join(t.TempDir(), "absent.log"),
		"no fontspec": writeProbeLog(t, "This is XeTeX, Version 3.141592653\nOutput written on probe.pdf.\n"),
	} {
		got := chineseProbeResult(path)
		if got.Status != StatusPass {
			t.Fatalf("%s: Status = %v, want pass", name, got.Status)
		}
	}
}

// installFonts points the supplemental font probe at a directory the test
// controls and installs the named font files in it, so the machine running the
// suite cannot decide which probe document is chosen.
func installFonts(t *testing.T, names ...string) {
	t.Helper()
	dir := t.TempDir()
	fontsDir := filepath.Join(dir, "Fonts")
	if err := os.MkdirAll(fontsDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(fontsDir, name), []byte("stub"), 0o644); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}
	t.Setenv("WINDIR", dir)
	t.Setenv("LOCALAPPDATA", filepath.Join(dir, "local"))
}

// The probe has to compile what the build compiles. Before this, doctor always
// compiled a bare ctexart, so a machine with SimSun but without SimHei got a
// failing Chinese probe from a build that succeeds.
func TestChineseProbeDocumentFollowsTheBuild(t *testing.T) {
	cases := []struct {
		name         string
		installed    []string
		wantFallback bool
	}{
		{name: "all fonts installed", installed: []string{"simhei.ttf", "simkai.ttf"}, wantFallback: false},
		{name: "SimHei missing", installed: []string{"simkai.ttf"}, wantFallback: true},
		{name: "KaiTi missing", installed: []string{"simhei.ttf"}, wantFallback: true},
		{name: "both missing", installed: nil, wantFallback: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			installFonts(t, tc.installed...)

			document, usedFallback := chineseProbeDocument(fonts.ProbeSupplemental())
			if usedFallback != tc.wantFallback {
				t.Fatalf("usedFallback = %v, want %v", usedFallback, tc.wantFallback)
			}
			if tc.wantFallback {
				if document != chineseProbeFallback {
					t.Fatalf("document is not the fallback:\n%s", document)
				}
			} else if document != chineseProbeStandard {
				t.Fatalf("document is not the standard probe:\n%s", document)
			}
			// Whichever branch was taken, the probe must still be a compilable
			// Chinese document rather than an empty preamble.
			if !strings.Contains(document, "ctexart") || !strings.Contains(document, "中文环境探针") {
				t.Fatalf("document is not a Chinese ctexart probe:\n%s", document)
			}
		})
	}
}

// An unreadable font directory proves nothing, and the fallback compiles on
// machines that do have the fonts while the bare ctexart does not compile on
// machines that lack them. So the undetermined case takes the fallback.
func TestChineseProbeDocumentFallsBackWhenUndetermined(t *testing.T) {
	t.Setenv("WINDIR", "")
	t.Setenv("LOCALAPPDATA", "")

	document, usedFallback := chineseProbeDocument(fonts.ProbeSupplemental())
	if !usedFallback || document != chineseProbeFallback {
		t.Fatalf("usedFallback = %v, want the fallback document", usedFallback)
	}
}

// The fallback probe is only meaningful while it matches the preamble the build
// really uses, so it is compared against the profile itself rather than against
// a copy of it in the test.
func TestChineseProbeFallbackMatchesTheProfile(t *testing.T) {
	templatePath := filepath.Join("..", "..", "profiles", "cumcm", "template.tex")
	data, err := os.ReadFile(templatePath)
	if err != nil {
		t.Skipf("profile template not readable from here: %v", err)
	}

	template := string(data)
	start := strings.Index(template, "$if(nodepaper-font-fallback)$")
	end := strings.Index(template, "$else$")
	if start < 0 || end < 0 || end < start {
		t.Fatalf("%s no longer has a $if(nodepaper-font-fallback)$ ... $else$ arm", templatePath)
	}

	for _, line := range strings.Split(template[start:end], "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if !strings.HasPrefix(line, `\setCJK`) && !strings.HasPrefix(line, `\providecommand`) {
			continue
		}
		if !strings.Contains(chineseProbeFallback, line) {
			t.Fatalf("fallback probe is missing the profile's line %q", line)
		}
	}
	// fontset=none is the part that makes the rest legal: without it ctex binds
	// SimHei during \documentclass and the probe fails before any of the lines
	// above run.
	if !strings.Contains(chineseProbeFallback, "fontset=none") {
		t.Fatal("fallback probe does not disable ctex's own font binding")
	}
}

func TestChineseProbeFailureSuggestionMatchesTheDocument(t *testing.T) {
	standard := chineseProbeFailureSuggestion(false)
	if !strings.Contains(standard, "configured Chinese fonts") {
		t.Fatalf("standard suggestion changed unexpectedly: %q", standard)
	}

	fallback := chineseProbeFailureSuggestion(true)
	// Under the fallback the missing optional fonts are already known and were
	// not used, so blaming them would send the user after the wrong thing.
	if !strings.Contains(fallback, "not the cause") {
		t.Fatalf("fallback suggestion still blames the optional fonts: %q", fallback)
	}
	for _, want := range []string{"SimSun", "ctex", "fallback"} {
		if !strings.Contains(fallback, want) {
			t.Fatalf("fallback suggestion does not mention %q: %q", want, fallback)
		}
	}
}

// doctor must recognise both distributions it tells users to install. The
// MiKTeX lines below are NOT from a machine we ran: they are the forms reported
// publicly, kept here so the intent is testable before the MiKTeX E2E runner
// prints its real `xelatex --version`. Replace them with the observed line once
// that output is available.
func TestXeLaTeXVersionLineAcceptsBothDistributions(t *testing.T) {
	accepted := map[string]string{
		// Observed locally, TeX Live 2025.
		"TeX Live":                           "XeTeX 3.141592653-2.6-0.999997 (TeX Live 2025)",
		"MiKTeX (confirmed run 31993318613)": "MiKTeX-XeTeX 4.18 (MiKTeX 26.5)",
		"older MiKTeX (unverified)":          "This is XeTeX, Version 3.141592653-2.6-0.999993 (MiKTeX 21.6.28)",
		"MiKTeX portable (unverified)":       "MiKTeX-XeTeX 4.10 (MiKTeX 23.5 Portable)",
	}
	for name, line := range accepted {
		if !xelatexVersionLine.MatchString(line) {
			t.Fatalf("%s version line not recognised: %q", name, line)
		}
	}

	// Prose lines from TeX Live's own --version output that also contain the
	// word XeTeX. They must not be mistaken for the version line.
	for _, line := range []string{
		"covered by the terms of both the XeTeX copyright and",
		"named COPYING and the XeTeX source.",
		"Primary author of XeTeX: Jonathan Kew.",
		"kpathsea version 6.4.1",
	} {
		if xelatexVersionLine.MatchString(line) {
			t.Fatalf("non-version line matched: %q", line)
		}
	}
}

func TestXeLaTeXHelpIsActionable(t *testing.T) {
	got := checkXeLaTeX(t.Context(), "")
	// The severity of this check is asserted in
	// TestCheckXeLaTeXMissingIsWarningNotFailure; this test is about the
	// guidance being usable whatever the severity is.
	if got.Status != StatusWarning {
		t.Fatalf("Status = %v, want warning", got.Status)
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
		// Without this line the failure reads as "nothing works here"; export
		// needs no TeX and is the one thing that still does work.
		"nodepaper export",
	} {
		if !strings.Contains(got.Suggestion, want) {
			t.Fatalf("Suggestion is missing %q:\n%s", want, got.Suggestion)
		}
	}
}
