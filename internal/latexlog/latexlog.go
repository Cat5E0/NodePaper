// Package latexlog classifies build-critical LaTeX log messages.
package latexlog

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"
)

type Category string

const (
	CategoryFatal          Category = "fatal"
	CategoryOverflow       Category = "overflow"
	CategoryMissingFont    Category = "missing-character-font"
	CategoryUnresolved     Category = "unresolved-citation-reference"
	CategoryRerun          Category = "rerun-not-converged"
	CategoryUnknownWarning Category = "unknown-warning"
	CategoryAllowedWarning Category = "allowed-warning"
)

// Finding is one classified log line.
type Finding struct {
	Category Category
	Line     int
	Text     string
	Reason   string
}

// Allowlist is the reviewed warning contract stored in a Profile.
type Allowlist struct {
	SchemaVersion int              `json:"schemaVersion"`
	Entries       []AllowlistEntry `json:"entries"`
}

type AllowlistEntry struct {
	Pattern      string   `json:"pattern"`
	Reason       string   `json:"reason"`
	Source       string   `json:"source"`
	ToolVersions []string `json:"toolVersions"`
	re           *regexp.Regexp
}

var classifiers = []struct {
	category Category
	patterns []*regexp.Regexp
}{
	{CategoryFatal, compilePatterns(`^! `, `LaTeX Error:`, `Package .+ Error:`, `Undefined control sequence\.`, `File .+ not found`, `Missing .+ inserted`, `Extra alignment tab`, `Runaway argument`, `TeX capacity exceeded`, `Emergency stop`, `Fatal error occurred`)},
	{CategoryOverflow, compilePatterns(`Overfull \\[hv]box`)},
	{CategoryMissingFont, compilePatterns(`Missing character:`, `LaTeX Font Warning:`, `Package fontspec Warning:`)},
	{CategoryUnresolved, compilePatterns(`Citation .+ undefined`, `Reference .+ undefined`, `There were undefined references`, `There were undefined citations`)},
	{CategoryRerun, compilePatterns(`Label\(s\) may have changed`, `Rerun to get cross-references right`, `Please .*rerun`, `rerunfilecheck Warning:`)},
	{CategoryUnknownWarning, compilePatterns(`LaTeX Warning:`, `Package .+ Warning:`)},
}

// LoadAllowlist strictly reads and compiles a reviewed warning allowlist.
func LoadAllowlist(path string) (Allowlist, error) {
	file, err := os.Open(path)
	if err != nil {
		return Allowlist{}, err
	}
	defer file.Close()
	var allowlist Allowlist
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&allowlist); err != nil {
		return Allowlist{}, fmt.Errorf("parse warning allowlist: %w", err)
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			return Allowlist{}, fmt.Errorf("warning allowlist contains multiple JSON values")
		}
		return Allowlist{}, fmt.Errorf("parse warning allowlist: %w", err)
	}
	if allowlist.SchemaVersion != 1 {
		return Allowlist{}, fmt.Errorf("unsupported warning allowlist schema version %d", allowlist.SchemaVersion)
	}
	for index := range allowlist.Entries {
		entry := &allowlist.Entries[index]
		if entry.Pattern == "" || entry.Reason == "" || entry.Source == "" || len(entry.ToolVersions) == 0 {
			return Allowlist{}, fmt.Errorf("warning allowlist entry %d requires pattern, reason, source and toolVersions", index)
		}
		entry.re, err = regexp.Compile(entry.Pattern)
		if err != nil {
			return Allowlist{}, fmt.Errorf("compile warning allowlist entry %d: %w", index, err)
		}
	}
	return allowlist, nil
}

// Classify returns all build-critical and explicitly allowed warning lines.
func Classify(data []byte, allowlist Allowlist) []Finding {
	var findings []Finding
	scanner := bufio.NewScanner(bytes.NewReader(data))
	lineNumber := 0
	for scanner.Scan() {
		lineNumber++
		line := scanner.Text()
		category := classifyLine(line)
		if category == "" {
			continue
		}
		finding := Finding{Category: category, Line: lineNumber, Text: strings.TrimSpace(line)}
		if category == CategoryUnknownWarning {
			for _, entry := range allowlist.Entries {
				if entry.re.MatchString(line) {
					finding.Category = CategoryAllowedWarning
					finding.Reason = entry.Reason
					break
				}
			}
		}
		findings = append(findings, finding)
	}
	return findings
}

func classifyLine(line string) Category {
	for _, classifier := range classifiers {
		for _, pattern := range classifier.patterns {
			if pattern.MatchString(line) {
				return classifier.category
			}
		}
	}
	return ""
}

func compilePatterns(patterns ...string) []*regexp.Regexp {
	result := make([]*regexp.Regexp, 0, len(patterns))
	for _, pattern := range patterns {
		result = append(result, regexp.MustCompile(pattern))
	}
	return result
}
