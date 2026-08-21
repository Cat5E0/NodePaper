// Package config reads, parses and validates nodepaper.yaml project
// configuration. All paths in the configuration are relative to the
// project root and are not resolved here.
package config

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ProjectConfig is the deserialised nodepaper.yaml.
type ProjectConfig struct {
	Version            int             `yaml:"version"`
	Profile            string          `yaml:"profile"`
	Source             string          `yaml:"source,omitempty"`
	Sources            []string        `yaml:"sources,omitempty"`
	LatexFragments     []string        `yaml:"latexFragments,omitempty"`
	Appendix           AppendixConfig  `yaml:"appendix,omitempty"`
	Highlight          HighlightConfig `yaml:"highlight,omitempty"`
	LineSpread         float64         `yaml:"linespread,omitempty"`
	AbstractLineSpread float64         `yaml:"abstractLinespread,omitempty"`
	// TitleAbstractSkip is the vertical gap between the abstract heading and
	// the abstract body; AbstractKeywordsSkip the one between the abstract body
	// and the keywords line. Both are in em. M4-00 D2 confirmed them together
	// with abstractLinespread on 2026-08-06 at 0.5em and 0.8em, but only
	// abstractLinespread was wired up: the two defaults shipped as literal
	// \vspace lengths in template.tex, so the effect was there and the knob was
	// not. Pointers rather than plain floats because 0 is a meaningful value
	// here - no gap at all - and the zero-means-unset shortcut the other
	// numeric fields use would make it unreachable.
	TitleAbstractSkip    *float64     `yaml:"titleAbstractSkip,omitempty"`
	AbstractKeywordsSkip *float64     `yaml:"abstractKeywordsSkip,omitempty"`
	MathFont             string       `yaml:"mathFont,omitempty"`
	Output               OutputConfig `yaml:"output,omitempty"`
}

// AppendixConfig controls numbering after the retained level-one appendix
// heading. The default is alpha. NewPage starts the appendix on a fresh page
// (default true).
type AppendixConfig struct {
	Numbering string `yaml:"numbering,omitempty"`
	NewPage   *bool  `yaml:"newPage,omitempty"`
}

// NewPageEnabled reports whether the appendix must start on a fresh page.
// The default is true when the field is not set.
func (a AppendixConfig) NewPageEnabled() bool {
	return a.NewPage == nil || *a.NewPage
}

// DefaultTitleAbstractSkipEm and DefaultAbstractKeywordsSkipEm are the M4-00 D2
// values that template.tex carried as literals before the fields existed, so an
// unset project renders byte-identically to before.
const (
	DefaultTitleAbstractSkipEm    = 0.5
	DefaultAbstractKeywordsSkipEm = 0.8

	// maxAbstractSkipEm bounds both gaps. The abstract shares page one with the
	// title and keywords, and the point of these knobs is to keep a long
	// abstract on that page; anything past a few em works against that, so the
	// range is deliberately narrow rather than "any length LaTeX accepts".
	maxAbstractSkipEm = 5.0
)

// TitleAbstractSkipEm returns the configured gap in em, or the default.
func (c ProjectConfig) TitleAbstractSkipEm() float64 {
	if c.TitleAbstractSkip == nil {
		return DefaultTitleAbstractSkipEm
	}
	return *c.TitleAbstractSkip
}

// AbstractKeywordsSkipEm returns the configured gap in em, or the default.
func (c ProjectConfig) AbstractKeywordsSkipEm() float64 {
	if c.AbstractKeywordsSkip == nil {
		return DefaultAbstractKeywordsSkipEm
	}
	return *c.AbstractKeywordsSkip
}

// HighlightConfig selects one of the finite, reviewed built-in Pandoc styles.
// Tango is the default.
type HighlightConfig struct {
	Style string `yaml:"style,omitempty"`
}

// OutputConfig controls the destination of built artifacts.
type OutputConfig struct {
	File string `yaml:"file,omitempty"`
}

// Load reads and validates a project configuration file. The returned
// configuration has been parsed but paths have not been resolved against the
// project root.
func Load(path string) (ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ProjectConfig{}, fmt.Errorf("cannot read config file %s: %w", path, err)
	}
	return Parse(data)
}

// Parse unmarshals and validates raw YAML bytes into a ProjectConfig.
func Parse(data []byte) (ProjectConfig, error) {
	var cfg ProjectConfig
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return ProjectConfig{}, fmt.Errorf("cannot parse YAML: %w", err)
	}
	if cfg.Appendix.Numbering == "" {
		cfg.Appendix.Numbering = "alpha"
	}
	if cfg.Highlight.Style == "" {
		cfg.Highlight.Style = "tango"
	}
	if cfg.LineSpread == 0 {
		cfg.LineSpread = 1.25
	}
	if cfg.AbstractLineSpread == 0 {
		cfg.AbstractLineSpread = 0.95
	}
	if cfg.MathFont == "" {
		cfg.MathFont = "cm"
	}

	if err := validate(cfg); err != nil {
		return ProjectConfig{}, err
	}

	return cfg, nil
}

func validate(cfg ProjectConfig) error {
	if cfg.Version != 1 {
		return fmt.Errorf("unsupported config version %d; only version 1 is supported", cfg.Version)
	}
	if cfg.Profile == "" {
		return fmt.Errorf("profile is required")
	}
	if cfg.Profile != "cumcm" {
		return fmt.Errorf("unsupported profile %q; v0.1 only supports cumcm", cfg.Profile)
	}

	hasSource := cfg.Source != ""
	hasSources := len(cfg.Sources) > 0

	if hasSource && hasSources {
		return fmt.Errorf("source and sources are mutually exclusive")
	}
	if !hasSource && !hasSources {
		return fmt.Errorf("source or sources is required")
	}
	if hasSources && len(cfg.Sources) == 0 {
		return fmt.Errorf("sources must not be empty when present")
	}

	for i, s := range cfg.Sources {
		if s == "" {
			return fmt.Errorf("sources[%d] is empty", i)
		}
	}
	for i, fragment := range cfg.LatexFragments {
		if fragment == "" {
			return fmt.Errorf("latexFragments[%d] is empty", i)
		}
	}
	switch cfg.Appendix.Numbering {
	case "alpha", "continuous", "none":
	default:
		return fmt.Errorf("appendix.numbering must be alpha, continuous, or none")
	}
	switch cfg.Highlight.Style {
	case "tango", "pygments", "kate":
	default:
		return fmt.Errorf("highlight.style must be tango, pygments, or kate")
	}
	if cfg.LineSpread < 1.0 || cfg.LineSpread > 1.3 {
		return fmt.Errorf("linespread must be between 1.0 and 1.3")
	}
	if cfg.AbstractLineSpread < 0.85 || cfg.AbstractLineSpread > cfg.LineSpread {
		return fmt.Errorf("abstractLinespread must be between 0.85 and linespread (%v)", cfg.LineSpread)
	}
	for _, skip := range []struct {
		name  string
		value *float64
	}{
		{"titleAbstractSkip", cfg.TitleAbstractSkip},
		{"abstractKeywordsSkip", cfg.AbstractKeywordsSkip},
	} {
		if skip.value == nil {
			continue
		}
		if *skip.value < 0 || *skip.value > maxAbstractSkipEm {
			return fmt.Errorf("%s must be between 0 and %v (em)", skip.name, maxAbstractSkipEm)
		}
	}
	switch cfg.MathFont {
	case "cm", "newtx":
	default:
		return fmt.Errorf("mathFont must be cm or newtx")
	}

	return nil
}

// SourceFiles returns the ordered list of Markdown source files. In
// single-source mode the result is a slice containing the single entry.
func (cfg ProjectConfig) SourceFiles() []string {
	if cfg.Source != "" {
		return []string{cfg.Source}
	}
	return cfg.Sources
}
