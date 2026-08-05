package config

import (
	"strings"
	"testing"
)

func TestParseMinimalSingleSource(t *testing.T) {
	cfg, err := Parse([]byte(`
version: 1
profile: cumcm
source: paper.md
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Version != 1 {
		t.Fatalf("Version = %d, want 1", cfg.Version)
	}
	if cfg.Profile != "cumcm" {
		t.Fatalf("Profile = %q, want cumcm", cfg.Profile)
	}
	if cfg.Source != "paper.md" {
		t.Fatalf("Source = %q, want paper.md", cfg.Source)
	}
	if cfg.Output.File != "" {
		t.Fatalf("Output.File = %q, want empty (use default)", cfg.Output.File)
	}
}

func TestParseMinimalMultiSource(t *testing.T) {
	cfg, err := Parse([]byte(`
version: 1
profile: cumcm
sources:
  - sections/01-abstract.md
  - sections/02-problem.md
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if len(cfg.Sources) != 2 {
		t.Fatalf("len(Sources) = %d, want 2", len(cfg.Sources))
	}
}

func TestParseWithFragmentsAndAppendix(t *testing.T) {
	cfg, err := Parse([]byte(`
version: 1
profile: cumcm
source: paper.md
latexFragments:
  - tables/result.tex
  - equations/objective.tex
appendix:
  numbering: continuous
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if got := cfg.LatexFragments; len(got) != 2 || got[0] != "tables/result.tex" || got[1] != "equations/objective.tex" {
		t.Fatalf("LatexFragments = %#v", got)
	}
	if cfg.Appendix.Numbering != "continuous" {
		t.Fatalf("Appendix.Numbering = %q", cfg.Appendix.Numbering)
	}
}

func TestAppendixNumberingDefaultsToAlpha(t *testing.T) {
	cfg, err := Parse([]byte("version: 1\nprofile: cumcm\nsource: paper.md\n"))
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Appendix.Numbering != "alpha" {
		t.Fatalf("Appendix.Numbering = %q, want alpha", cfg.Appendix.Numbering)
	}
	if cfg.Highlight.Style != "tango" {
		t.Fatalf("Highlight.Style = %q, want tango", cfg.Highlight.Style)
	}
	if cfg.LineSpread != 1.25 {
		t.Fatalf("LineSpread = %v, want default 1.25", cfg.LineSpread)
	}
	if cfg.AbstractLineSpread != 0.95 {
		t.Fatalf("AbstractLineSpread = %v, want default 0.95", cfg.AbstractLineSpread)
	}
	if cfg.MathFont != "cm" {
		t.Fatalf("MathFont = %q, want default cm", cfg.MathFont)
	}
}

func TestLineSpreadSelection(t *testing.T) {
	cfg, err := Parse([]byte("version: 1\nprofile: cumcm\nsource: paper.md\nlinespread: 1.1\nabstractLinespread: 0.9\nmathFont: newtx\n"))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.LineSpread != 1.1 {
		t.Fatalf("LineSpread = %v, want 1.1", cfg.LineSpread)
	}
	if cfg.AbstractLineSpread != 0.9 {
		t.Fatalf("AbstractLineSpread = %v, want 0.9", cfg.AbstractLineSpread)
	}
	if cfg.MathFont != "newtx" {
		t.Fatalf("MathFont = %q, want newtx", cfg.MathFont)
	}
}

func TestHighlightStyleSelection(t *testing.T) {
	for _, style := range []string{"tango", "pygments", "kate"} {
		cfg, err := Parse([]byte("version: 1\nprofile: cumcm\nsource: paper.md\nhighlight:\n  style: " + style + "\n"))
		if err != nil {
			t.Fatalf("Parse(%s): %v", style, err)
		}
		if cfg.Highlight.Style != style {
			t.Fatalf("Highlight.Style = %q, want %q", cfg.Highlight.Style, style)
		}
	}
}

func TestParseWithOutput(t *testing.T) {
	cfg, err := Parse([]byte(`
version: 1
profile: cumcm
source: paper.md
output:
  file: dist/paper.pdf
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Output.File != "dist/paper.pdf" {
		t.Fatalf("Output.File = %q, want dist/paper.pdf", cfg.Output.File)
	}
}

func TestSourceFilesSingle(t *testing.T) {
	cfg, err := Parse([]byte(`
version: 1
profile: cumcm
source: paper.md
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	files := cfg.SourceFiles()
	if len(files) != 1 || files[0] != "paper.md" {
		t.Fatalf("SourceFiles() = %v, want [paper.md]", files)
	}
}

func TestSourceFilesMulti(t *testing.T) {
	cfg, err := Parse([]byte(`
version: 1
profile: cumcm
sources:
  - a.md
  - b.md
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	files := cfg.SourceFiles()
	if len(files) != 2 || files[0] != "a.md" || files[1] != "b.md" {
		t.Fatalf("SourceFiles() = %v, want [a.md b.md]", files)
	}
}

func TestParseRejects(t *testing.T) {
	tests := []struct {
		name    string
		yaml    string
		message string
	}{
		{"invalid yaml", "version: [", "cannot parse YAML"},
		{"wrong version", "version: 2\nprofile: cumcm\nsource: a.md", "unsupported config version 2"},
		{"missing profile", "version: 1\nsource: a.md", "profile is required"},
		{"wrong profile", "version: 1\nprofile: acm\nsource: a.md", "unsupported profile"},
		{"both source and sources", "version: 1\nprofile: cumcm\nsource: a.md\nsources:\n  - b.md", "mutually exclusive"},
		{"neither source nor sources", "version: 1\nprofile: cumcm", "source or sources is required"},
		{"empty source string", "version: 1\nprofile: cumcm\nsource:", "source or sources is required"},
		{"empty sources item", "version: 1\nprofile: cumcm\nsources:\n  - a.md\n  - \"\"", "empty"},
		{"empty fragment", "version: 1\nprofile: cumcm\nsource: a.md\nlatexFragments:\n  - \"\"", "latexFragments[0] is empty"},
		{"invalid appendix numbering", "version: 1\nprofile: cumcm\nsource: a.md\nappendix:\n  numbering: roman", "appendix.numbering must be"},
		{"invalid highlight style", "version: 1\nprofile: cumcm\nsource: a.md\nhighlight:\n  style: minted", "highlight.style must be"},
		{"linespread too small", "version: 1\nprofile: cumcm\nsource: a.md\nlinespread: 0.9", "linespread must be between 1.0 and 1.3"},
		{"linespread too large", "version: 1\nprofile: cumcm\nsource: a.md\nlinespread: 1.4", "linespread must be between 1.0 and 1.3"},
		{"abstractLinespread too small", "version: 1\nprofile: cumcm\nsource: a.md\nabstractLinespread: 0.8", "abstractLinespread must be between 0.85"},
		{"abstractLinespread above linespread", "version: 1\nprofile: cumcm\nsource: a.md\nlinespread: 1.1\nabstractLinespread: 1.2", "abstractLinespread must be between 0.85"},
		{"invalid mathFont", "version: 1\nprofile: cumcm\nsource: a.md\nmathFont: xits", "mathFont must be cm or newtx"},
		{"unknown highlight field", "version: 1\nprofile: cumcm\nsource: a.md\nhighlight:\n  unexpected: true", "field unexpected not found"},
		{"unknown top-level field", "version: 1\nprofile: cumcm\nsource: a.md\nunexpected: true", "field unexpected not found"},
		{"unknown appendix field", "version: 1\nprofile: cumcm\nsource: a.md\nappendix:\n  unexpected: true", "field unexpected not found"},
		{"unknown output field", "version: 1\nprofile: cumcm\nsource: a.md\noutput:\n  file: dist/a.pdf\n  unexpected: true", "field unexpected not found"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse([]byte(test.yaml))
			if err == nil {
				t.Fatal("Parse() error = nil")
			}
			if !strings.Contains(err.Error(), test.message) {
				t.Fatalf("Parse() error = %q, want substring %q", err, test.message)
			}
		})
	}
}

func TestOutputFieldDefault(t *testing.T) {
	cfg, err := Parse([]byte(`
version: 1
profile: cumcm
source: paper.md
`))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	// Output.File should default to empty; the app layer decides the
	// actual default path.
	if cfg.Output.File != "" {
		t.Fatalf("Output.File = %q, want empty", cfg.Output.File)
	}
}

func TestConfigRoundTrip(t *testing.T) {
	input := `
version: 1
profile: cumcm
source: paper.md
output:
  file: dist/out.pdf
`
	cfg, err := Parse([]byte(input))
	if err != nil {
		t.Fatalf("Parse() error = %v", err)
	}
	if cfg.Version != 1 {
		t.Fatalf("Version = %d", cfg.Version)
	}
	if cfg.Source != "paper.md" {
		t.Fatalf("Source = %q", cfg.Source)
	}
	if cfg.Output.File != "dist/out.pdf" {
		t.Fatalf("Output.File = %q", cfg.Output.File)
	}
}
