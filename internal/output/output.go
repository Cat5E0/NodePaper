// Package output renders application results as human-readable text or
// machine-readable JSON. Both renderers consume the same structured result
// data from the application service.
package output

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"nodepaper/internal/app"
	"nodepaper/internal/diagnostic"
)

const schemaVersion = 1

// ---------- Text renderer ------------------------------------------------

// TextWriter writes application results in a human-readable format.
type TextWriter struct {
	W io.Writer
}

// Init renders an InitResult.
func (tw *TextWriter) Init(result app.InitResult) {
	tw.writeProjectRoot(result.ProjectRoot)
	tw.writeSuccess(result.Success)
	tw.writeArtifacts(result.Artifacts)
	tw.writeDiagnostics(result.Diagnostics)
	if IsTerminalSuccess(result.Diagnostics, result.Success) {
		fmt.Fprintf(tw.W, "Next: cd \"%s\", then run nodepaper validate\n", result.ProjectRoot)
	}
}

// Doctor renders a DoctorResult, one section per capability.
//
// The sections exist because the capabilities are independent: a machine
// without TeX cannot render a PDF and can still convert and export, and a flat
// list gave the reader no way to see which of the two they were looking at.
// Every check is printed under the group it carries (app.DoctorCheck.Group,
// straight from doctor.Check.Group), never matched by name here - a renderer
// that recognised names would quietly misfile whatever check is added next.
func (tw *TextWriter) Doctor(result app.DoctorResult) {
	tw.writeProjectRoot(result.ProjectRoot)
	for i, section := range groupDoctorChecks(result.Checks) {
		if i > 0 {
			fmt.Fprintln(tw.W)
		}
		indent := ""
		if section.group != "" {
			fmt.Fprintln(tw.W, section.group)
			indent = "  "
		}
		for _, c := range section.checks {
			fmt.Fprintf(tw.W, "%s%-4s %-20s %s\n", indent, doctorStatusPrefix(c.Status), c.Name, c.Message)
			tw.writeDoctorSuggestion(indent, c.Suggestion)
		}
	}
	fmt.Fprintln(tw.W)
	tw.writeSuccess(result.Success)
	// Doctor can fail before any check runs - project discovery is the common
	// case - and then the checks above are empty and the reason lives only in
	// the diagnostics. Without this the text output said "Failed" and nothing
	// else while --format json carried NP1001, which breaks the rule that both
	// formats render the same diagnostic data.
	tw.writeDiagnostics(result.Diagnostics)
	if IsTerminalSuccess(result.Diagnostics, result.Success) {
		if result.ProjectRoot == "" {
			fmt.Fprintln(tw.W, "Next: run nodepaper init <project-directory>")
		} else {
			fmt.Fprintln(tw.W, "Next: run nodepaper validate")
		}
	}
}

// writeDoctorSuggestion prints a check's suggestion under it, indenting every
// line of a multi-line one. Without that, a suggestion as long as the TeX
// install guide spilled back to column zero and read as though it had left the
// section it belongs to. This package renders app-layer results and does not
// import doctor - the same convention doctorStatusPrefix follows - so
// doctor.FormatChecks keeps its own copy of this shaping.
func (tw *TextWriter) writeDoctorSuggestion(indent, suggestion string) {
	if suggestion == "" {
		return
	}
	for i, line := range strings.Split(strings.ReplaceAll(suggestion, "\r\n", "\n"), "\n") {
		marker := "  "
		if i == 0 {
			marker = "→ "
		}
		fmt.Fprintf(tw.W, "%s     %s%s\n", indent, marker, line)
	}
}

// doctorSection is a run of doctor checks that share a group.
type doctorSection struct {
	group  string
	checks []app.DoctorCheck
}

// groupDoctorChecks buckets checks by the group they carry, in order of first
// appearance, so the sections follow the order the checks were run instead of
// a separate ordered list that a new check could be left out of. Checks
// without a group form their own headingless section rather than being
// dropped: an unreported check is the one outcome doctor must never produce.
func groupDoctorChecks(checks []app.DoctorCheck) []doctorSection {
	var sections []doctorSection
	position := make(map[string]int, len(checks))
	for _, c := range checks {
		if index, ok := position[c.Group]; ok {
			sections[index].checks = append(sections[index].checks, c)
			continue
		}
		position[c.Group] = len(sections)
		sections = append(sections, doctorSection{group: c.Group, checks: []app.DoctorCheck{c}})
	}
	return sections
}

func doctorStatusPrefix(status string) string {
	switch status {
	case "pass":
		return "PASS"
	case "warning":
		return "WARN"
	case "fail":
		return "FAIL"
	case "skipped":
		return "SKIP"
	default:
		return "??? "
	}
}

// Validate renders a ValidateResult.
func (tw *TextWriter) Validate(result app.ValidateResult) {
	tw.writeProjectRoot(result.ProjectRoot)
	tw.writeSuccess(result.Success)
	tw.writeDiagnostics(result.Diagnostics)
	if IsTerminalSuccess(result.Diagnostics, result.Success) {
		fmt.Fprintln(tw.W, "Next: run nodepaper build")
	}
}

// Build renders a BuildResult.
func (tw *TextWriter) Build(result app.BuildResult) {
	tw.writeProjectRoot(result.ProjectRoot)
	if result.BuildID != "" {
		fmt.Fprintf(tw.W, "Build ID: %s\n", result.BuildID)
	}
	tw.writeSuccess(result.Success)
	tw.writeArtifacts(result.Artifacts)
	tw.writeDiagnostics(result.Diagnostics)
	if IsTerminalSuccess(result.Diagnostics, result.Success) {
		for _, artifact := range result.Artifacts {
			if artifact.Kind == "pdf" {
				fmt.Fprintf(tw.W, "Next: open the PDF at %s\n", artifact.Path)
				break
			}
		}
	}
}

// Export renders an ExportResult.
func (tw *TextWriter) Export(result app.ExportResult) {
	tw.writeProjectRoot(result.ProjectRoot)
	exportPath := result.ExportPath
	if exportPath == "" {
		exportPath = result.ExportDir
	}
	if exportPath != "" {
		fmt.Fprintf(tw.W, "Export: %s\n", exportPath)
	}
	if result.BibMode != "" {
		fmt.Fprintf(tw.W, "Bibliography: %s\n", result.BibMode)
	}
	tw.writeSuccess(result.Success)
	tw.writeArtifacts(result.Artifacts)
	tw.writeDiagnostics(result.Diagnostics)
	if !IsTerminalSuccess(result.Diagnostics, result.Success) {
		return
	}
	if result.Verified {
		fmt.Fprintln(tw.W, "Verified: the compile chain below succeeded on this machine.")
		// Said every time, because a green local compile is the exact result
		// people over-read: the recipient's TeX distribution and fonts are a
		// different environment and this proves nothing about it.
		fmt.Fprintln(tw.W, "This does not guarantee it compiles on the recipient's machine; their")
		fmt.Fprintln(tw.W, "TeX packages and fonts still decide that.")
	}
	fmt.Fprintln(tw.W, "Next:")
	if result.Zipped {
		// Local TeX is the primary route and Overleaf the secondary one, so the
		// local line comes first and the Overleaf line carries the free-plan
		// compile cap: a full paper of this kind does not finish inside it, and
		// a recipient who learns that only after uploading has already spent
		// the time the notice was meant to save.
		fmt.Fprintln(tw.W, "  extract the ZIP and open README.txt for the local compile commands")
		fmt.Fprintf(tw.W, "  or upload \"%s\" with Overleaf > New Project > Upload Project\n", exportPath)
		fmt.Fprintln(tw.W, "  (compiler must be XeLaTeX; Overleaf's free plan stops a compile after")
		fmt.Fprintln(tw.W, "  10 seconds, which a full paper does not finish inside)")
		return
	}
	fmt.Fprintf(tw.W, "  cd \"%s\"\n", result.ExportDir)
	// The chain comes from the result rather than from a copy kept here, so
	// the terminal, README.txt and --verify always name the same commands.
	for _, command := range result.CompileCommands {
		fmt.Fprintf(tw.W, "  %s\n", command)
	}
	fmt.Fprintln(tw.W, "  open README.txt for the required packages and notes")
}

// Clean renders a CleanResult.
func (tw *TextWriter) Clean(result app.CleanResult) {
	tw.writeProjectRoot(result.ProjectRoot)
	tw.writeSuccess(result.Success)
	tw.writeArtifacts(result.Artifacts)
	tw.writeDiagnostics(result.Diagnostics)
}

func (tw *TextWriter) writeProjectRoot(root string) {
	if root != "" {
		fmt.Fprintf(tw.W, "Project: %s\n", root)
	}
}

func (tw *TextWriter) writeSuccess(ok bool) {
	if ok {
		fmt.Fprintln(tw.W, "✓ Success")
	} else {
		fmt.Fprintln(tw.W, "✗ Failed")
	}
}

func (tw *TextWriter) writeArtifacts(artifacts []app.Artifact) {
	for _, a := range artifacts {
		fmt.Fprintf(tw.W, "  • %s: %s\n", a.Kind, a.Path)
	}
}

func (tw *TextWriter) writeDiagnostics(diags []diagnostic.Diagnostic) {
	for _, d := range diags {
		prefix := statusPrefix(d.Severity)
		fmt.Fprintf(tw.W, "  %s [%s] %s", prefix, d.Code, d.Message)
		if d.File != "" {
			fmt.Fprintf(tw.W, "  (%s", d.File)
			if d.Line > 0 {
				fmt.Fprintf(tw.W, ":%d", d.Line)
			}
			fmt.Fprint(tw.W, ")")
		}
		fmt.Fprintln(tw.W)
		// A multi-line suggestion has its continuation lines indented under the
		// first, so that guidance several lines long - install instructions, a
		// list of commands - still reads as part of the diagnostic instead of
		// as unrelated output flush against the left margin.
		if d.Suggestion != "" {
			for i, line := range strings.Split(strings.ReplaceAll(d.Suggestion, "\r\n", "\n"), "\n") {
				if i == 0 {
					fmt.Fprintf(tw.W, "      Suggestion: %s\n", line)
					continue
				}
				fmt.Fprintf(tw.W, "                  %s\n", line)
			}
		}
	}
}

func statusPrefix(s diagnostic.Severity) string {
	switch s {
	case diagnostic.SeverityError:
		return "✗"
	case diagnostic.SeverityWarning:
		return "⚠"
	case diagnostic.SeverityInfo:
		return "ℹ"
	default:
		return "•"
	}
}

// ---------- JSON renderer ------------------------------------------------

// JSONWriter writes application results as machine-readable JSON. Every
// output is a single JSON object with a schemaVersion envelope.
type JSONWriter struct {
	W io.Writer
}

// Init renders an InitResult as JSON.
func (jw *JSONWriter) Init(result app.InitResult) error {
	return jw.writeEnvelope(result)
}

// Doctor renders a DoctorResult as JSON.
func (jw *JSONWriter) Doctor(result app.DoctorResult) error {
	return jw.writeEnvelope(result)
}

// Validate renders a ValidateResult as JSON.
func (jw *JSONWriter) Validate(result app.ValidateResult) error {
	return jw.writeEnvelope(result)
}

// Build renders a BuildResult as JSON.
func (jw *JSONWriter) Build(result app.BuildResult) error {
	return jw.writeEnvelope(result)
}

// Export renders an ExportResult as JSON.
func (jw *JSONWriter) Export(result app.ExportResult) error {
	return jw.writeEnvelope(result)
}

// Clean renders a CleanResult as JSON.
func (jw *JSONWriter) Clean(result app.CleanResult) error {
	return jw.writeEnvelope(result)
}

// writeEnvelope serialises result to JSON, injects schemaVersion, and writes
// exactly one JSON object followed by a newline.
func (jw *JSONWriter) writeEnvelope(result any) error {
	raw, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("json marshal: %w", err)
	}

	var payload map[string]any
	if err := json.Unmarshal(raw, &payload); err != nil {
		return fmt.Errorf("json re-read: %w", err)
	}

	payload["schemaVersion"] = schemaVersion

	out, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return fmt.Errorf("json envelope marshal: %w", err)
	}

	_, err = fmt.Fprintf(jw.W, "%s\n", out)
	return err
}

// ---------- helpers -------------------------------------------------------

// IsTerminalSuccess returns true when the result signals success and contains
// no error-level diagnostics. Consumers that need to map CLI exit codes can
// use this without inspecting diagnostic severities themselves.
func IsTerminalSuccess(diags []diagnostic.Diagnostic, success bool) bool {
	if !success {
		return false
	}
	for _, d := range diags {
		if d.Severity == diagnostic.SeverityError {
			return false
		}
	}
	return true
}

// FormatDiagnostic returns a single-line representation of a diagnostic
// suitable for embedding in log output.
func FormatDiagnostic(d diagnostic.Diagnostic) string {
	var sb strings.Builder
	sb.WriteString(string(d.Severity))
	sb.WriteString(" ")
	sb.WriteString(d.Code)
	sb.WriteString(": ")
	sb.WriteString(d.Message)
	if d.File != "" {
		sb.WriteString(" (")
		sb.WriteString(d.File)
		if d.Line > 0 {
			sb.WriteString(fmt.Sprintf(":%d", d.Line))
		}
		sb.WriteString(")")
	}
	return sb.String()
}
