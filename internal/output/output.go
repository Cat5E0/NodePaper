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

// Doctor renders a DoctorResult.
func (tw *TextWriter) Doctor(result app.DoctorResult) {
	tw.writeProjectRoot(result.ProjectRoot)
	for _, c := range result.Checks {
		prefix := doctorStatusPrefix(c.Status)
		fmt.Fprintf(tw.W, "%-4s %-20s %s\n", prefix, c.Name, c.Message)
		if c.Suggestion != "" {
			fmt.Fprintf(tw.W, "     → %s\n", c.Suggestion)
		}
	}
	fmt.Fprintln(tw.W)
	tw.writeSuccess(result.Success)
	if IsTerminalSuccess(result.Diagnostics, result.Success) {
		if result.ProjectRoot == "" {
			fmt.Fprintln(tw.W, "Next: run nodepaper init <project-directory>")
		} else {
			fmt.Fprintln(tw.W, "Next: run nodepaper validate")
		}
	}
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
	if result.ExportDir != "" {
		fmt.Fprintf(tw.W, "Export: %s\n", result.ExportDir)
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
		if d.Suggestion != "" {
			fmt.Fprintf(tw.W, "      Suggestion: %s\n", d.Suggestion)
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
