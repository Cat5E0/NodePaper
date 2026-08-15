package build

import (
	"strings"
	"testing"

	"nodepaper/internal/diagnostic"
	"nodepaper/internal/latexlog"
)

func TestSynthesisedFontDiagnosticsSilentOnRealFonts(t *testing.T) {
	families := []latexlog.FontFamily{
		{Family: "SimSun(0)", Font: "SimSun", Options: "Script={CJK},BoldFont={SimHei},ItalicFont={KaiTi}"},
		{Family: "SimHei(0)", Font: "SimHei", Options: "Script={CJK}"},
	}
	if diags := synthesisedFontDiagnostics(families, "paper.log"); diags != nil {
		t.Fatalf("diagnostics = %#v, want none when the real bold face was used", diags)
	}
	if diags := synthesisedFontDiagnostics(nil, "paper.log"); diags != nil {
		t.Fatalf("diagnostics = %#v, want none for a log without fontspec records", diags)
	}
}

func TestSynthesisedFontDiagnosticsWarnsButDoesNotBlock(t *testing.T) {
	// What a real fallback build reports: every family bound to SimSun with the
	// weight faked, because SimHei was not installed.
	families := []latexlog.FontFamily{
		{Family: "SimSun(0)", Font: "SimSun", Options: "Script={CJK},AutoFakeBold={4},AutoFakeSlant={0.167}"},
		{Family: "SimSun(1)", Font: "SimSun", Options: "Script={CJK},AutoFakeBold={4}"},
	}
	diags := synthesisedFontDiagnostics(families, "paper.log")
	if len(diags) != 1 {
		t.Fatalf("diagnostics = %#v, want exactly one", diags)
	}
	got := diags[0]
	if got.Code != "NP6107" {
		t.Fatalf("Code = %q, want NP6107", got.Code)
	}
	// The PDF is complete and correct; only the weight is synthesised. Failing
	// here would leave the author with no paper over an optional Windows font.
	if got.Severity != diagnostic.SeverityWarning {
		t.Fatalf("Severity = %v, want warning", got.Severity)
	}
	if hasError(diags) {
		t.Fatal("NP6107 must not block publication")
	}
	// Both families are SimSun; the message should say so once.
	if strings.Count(got.Message, "SimSun") != 1 {
		t.Fatalf("Message repeats the font: %q", got.Message)
	}
	if !strings.Contains(got.Suggestion, "Optional features") {
		t.Fatalf("Suggestion lacks the install route: %q", got.Suggestion)
	}
}

func TestDedupePreservesOrder(t *testing.T) {
	got := dedupe([]string{"SimSun", "SimSun", "FandolSong", "SimSun"})
	if len(got) != 2 || got[0] != "SimSun" || got[1] != "FandolSong" {
		t.Fatalf("dedupe = %#v", got)
	}
}
