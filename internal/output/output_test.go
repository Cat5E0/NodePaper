package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"nodepaper/internal/app"
	"nodepaper/internal/diagnostic"
)

func makeDiag(severity diagnostic.Severity, code, msg, file string, line int, suggestion string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity:   severity,
		Code:       code,
		Message:    msg,
		File:       file,
		Line:       line,
		Suggestion: suggestion,
	}
}

// ---------- TextWriter tests ---------------------------------------------

func TestTextWriterInitSuccess(t *testing.T) {
	var buf bytes.Buffer
	tw := &TextWriter{W: &buf}

	tw.Init(app.InitResult{
		Success:     true,
		ProjectRoot: `D:\papers\a`,
		Artifacts: []app.Artifact{
			{Kind: "markdown", Path: `D:\papers\a\paper.md`},
		},
	})

	out := buf.String()
	if !strings.Contains(out, "Project: D:\\papers\\a") {
		t.Fatalf("missing project root: %s", out)
	}
	if !strings.Contains(out, "✓ Success") {
		t.Fatalf("missing success indicator: %s", out)
	}
	if !strings.Contains(out, "markdown: D:\\papers\\a\\paper.md") {
		t.Fatalf("missing artifact: %s", out)
	}
	if !strings.Contains(out, "nodepaper validate") {
		t.Fatalf("missing next step: %s", out)
	}
}

func TestTextWriterInitFailure(t *testing.T) {
	var buf bytes.Buffer
	tw := &TextWriter{W: &buf}

	tw.Init(app.InitResult{
		Success:     false,
		ProjectRoot: `D:\papers\a`,
		Diagnostics: []diagnostic.Diagnostic{
			makeDiag(diagnostic.SeverityError, "NP1001", "not found", `D:\papers\a`, 0, "create nodepaper.yaml"),
		},
	})

	out := buf.String()
	if !strings.Contains(out, "✗ Failed") {
		t.Fatalf("missing failure indicator: %s", out)
	}
	if !strings.Contains(out, "[NP1001]") {
		t.Fatalf("missing diagnostic code: %s", out)
	}
	if !strings.Contains(out, "not found") {
		t.Fatalf("missing diagnostic message: %s", out)
	}
	if !strings.Contains(out, "Suggestion: create nodepaper.yaml") {
		t.Fatalf("missing suggestion: %s", out)
	}
}

func TestTextWriterDoctorSuccess(t *testing.T) {
	var buf bytes.Buffer
	tw := &TextWriter{W: &buf}

	tw.Doctor(app.DoctorResult{
		Success:     true,
		ProjectRoot: `D:\papers\a`,
	})

	out := buf.String()
	if !strings.Contains(out, "✓ Success") {
		t.Fatalf("missing success indicator: %s", out)
	}
}

func TestTextWriterValidateWithWarnings(t *testing.T) {
	var buf bytes.Buffer
	tw := &TextWriter{W: &buf}

	tw.Validate(app.ValidateResult{
		Success:     true,
		ProjectRoot: `D:\papers\a`,
		Diagnostics: []diagnostic.Diagnostic{
			makeDiag(diagnostic.SeverityWarning, "NP2001", "raw LaTeX found", "paper.md", 42, ""),
			makeDiag(diagnostic.SeverityInfo, "NP2002", "using default profile", "", 0, ""),
		},
	})

	out := buf.String()
	if !strings.Contains(out, "✓ Success") {
		t.Fatalf("missing success indicator (warnings should not flip success): %s", out)
	}
	if !strings.Contains(out, "⚠") {
		t.Fatalf("missing warning prefix: %s", out)
	}
	if !strings.Contains(out, "ℹ") {
		t.Fatalf("missing info prefix: %s", out)
	}
	if !strings.Contains(out, "paper.md:42") {
		t.Fatalf("missing file:line: %s", out)
	}
}

func TestTextWriterBuild(t *testing.T) {
	var buf bytes.Buffer
	tw := &TextWriter{W: &buf}

	tw.Build(app.BuildResult{
		Success:     true,
		BuildID:     "build-001",
		ProjectRoot: `D:\papers\a`,
		Artifacts: []app.Artifact{
			{Kind: "pdf", Path: `D:\papers\a\dist\paper.pdf`},
			{Kind: "log", Path: `D:\papers\a\.nodepaper\logs\build-001.log`},
		},
	})

	out := buf.String()
	if !strings.Contains(out, "Build ID: build-001") {
		t.Fatalf("missing build id: %s", out)
	}
	if !strings.Contains(out, "pdf: D:\\papers\\a\\dist\\paper.pdf") {
		t.Fatalf("missing pdf artifact: %s", out)
	}
	if !strings.Contains(out, "open the PDF") {
		t.Fatalf("missing next step: %s", out)
	}
}

func TestTextWriterClean(t *testing.T) {
	var buf bytes.Buffer
	tw := &TextWriter{W: &buf}

	tw.Clean(app.CleanResult{
		Success:     true,
		ProjectRoot: `D:\papers\a`,
	})

	out := buf.String()
	if !strings.Contains(out, "✓ Success") {
		t.Fatalf("missing success indicator: %s", out)
	}
}

func TestTextWriterWithoutProjectRoot(t *testing.T) {
	var buf bytes.Buffer
	tw := &TextWriter{W: &buf}

	tw.Init(app.InitResult{Success: false})
	out := buf.String()
	if strings.Contains(out, "Project:") {
		t.Fatalf("should not print empty project root: %s", out)
	}
}

// ---------- JSONWriter tests ---------------------------------------------

func TestJSONWriterInitSuccess(t *testing.T) {
	var buf bytes.Buffer
	jw := &JSONWriter{W: &buf}

	err := jw.Init(app.InitResult{
		Success:     true,
		ProjectRoot: `D:\papers\a`,
		Artifacts: []app.Artifact{
			{Kind: "markdown", Path: `D:\papers\a\paper.md`},
		},
	})
	if err != nil {
		t.Fatalf("Init() error = %v", err)
	}

	raw := buf.String()
	if !strings.HasSuffix(raw, "\n") {
		t.Fatalf("JSON output should end with newline: %q", raw)
	}

	var envelope map[string]any
	if err := json.Unmarshal([]byte(raw), &envelope); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, raw)
	}

	if v, ok := envelope["schemaVersion"].(float64); !ok || v != 1 {
		t.Fatalf("schemaVersion = %v, want 1", envelope["schemaVersion"])
	}
	if v, ok := envelope["success"].(bool); !ok || !v {
		t.Fatalf("success = %v, want true", envelope["success"])
	}
	if v, ok := envelope["projectRoot"].(string); !ok || v != `D:\papers\a` {
		t.Fatalf("projectRoot = %v, want D:\\papers\\a", envelope["projectRoot"])
	}

	artifacts, ok := envelope["artifacts"].([]any)
	if !ok || len(artifacts) != 1 {
		t.Fatalf("artifacts = %v", envelope["artifacts"])
	}
}

func TestJSONWriterDoctorFailure(t *testing.T) {
	var buf bytes.Buffer
	jw := &JSONWriter{W: &buf}

	err := jw.Doctor(app.DoctorResult{
		Success:     false,
		ProjectRoot: `D:\papers\a`,
		Diagnostics: []diagnostic.Diagnostic{
			makeDiag(diagnostic.SeverityError, "NP4001", "Pandoc not found", "", 0, "Install Pandoc"),
		},
	})
	if err != nil {
		t.Fatalf("Doctor() error = %v", err)
	}

	var envelope map[string]any
	if err := json.Unmarshal([]byte(buf.String()), &envelope); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if v, ok := envelope["success"].(bool); ok && v {
		t.Fatal("success should be false")
	}
	diags, ok := envelope["diagnostics"].([]any)
	if !ok || len(diags) != 1 {
		t.Fatalf("diagnostics missing or empty: %v", envelope["diagnostics"])
	}
}

func TestJSONWriterBuild(t *testing.T) {
	var buf bytes.Buffer
	jw := &JSONWriter{W: &buf}

	err := jw.Build(app.BuildResult{
		Success:     true,
		BuildID:     "build-001",
		ProjectRoot: `D:\papers\a`,
		Artifacts: []app.Artifact{
			{Kind: "pdf", Path: `D:\papers\a\dist\paper.pdf`},
		},
	})
	if err != nil {
		t.Fatalf("Build() error = %v", err)
	}

	var envelope map[string]any
	if err := json.Unmarshal([]byte(buf.String()), &envelope); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if v, ok := envelope["buildId"].(string); !ok || v != "build-001" {
		t.Fatalf("buildId = %v, want build-001", envelope["buildId"])
	}
}

func TestJSONWriterValidate(t *testing.T) {
	var buf bytes.Buffer
	jw := &JSONWriter{W: &buf}

	err := jw.Validate(app.ValidateResult{
		Success:     true,
		ProjectRoot: `D:\papers\a`,
		Diagnostics: []diagnostic.Diagnostic{
			makeDiag(diagnostic.SeverityWarning, "NP2101", "duplicate ref", "paper.md", 10, ""),
		},
	})
	if err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	var envelope map[string]any
	if err := json.Unmarshal([]byte(buf.String()), &envelope); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestJSONWriterClean(t *testing.T) {
	var buf bytes.Buffer
	jw := &JSONWriter{W: &buf}

	err := jw.Clean(app.CleanResult{
		Success:     true,
		ProjectRoot: `D:\papers\a`,
	})
	if err != nil {
		t.Fatalf("Clean() error = %v", err)
	}

	var envelope map[string]any
	if err := json.Unmarshal([]byte(buf.String()), &envelope); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
}

func TestJSONWriterSingleValidObject(t *testing.T) {
	// stdout must contain exactly one valid JSON object and no extra text.
	var buf bytes.Buffer
	jw := &JSONWriter{W: &buf}

	jw.Init(app.InitResult{Success: true, ProjectRoot: `D:\papers\a`})

	out := strings.TrimSpace(buf.String())

	// Must be a single valid JSON object.
	var parsed any
	if err := json.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid JSON: %v\n%s", err, out)
	}
	if _, ok := parsed.(map[string]any); !ok {
		t.Fatalf("output is not a JSON object: %s", out)
	}

	// Verify first character is '{'
	if !strings.HasPrefix(out, "{") {
		t.Fatalf("JSON output should start with '{': %s", out)
	}
}

func TestJSONAndTextDiagnosticsConsistent(t *testing.T) {
	diags := []diagnostic.Diagnostic{
		makeDiag(diagnostic.SeverityError, "NP1001", "not found", "paper.md", 5, "check path"),
	}

	var textBuf bytes.Buffer
	tw := &TextWriter{W: &textBuf}
	tw.Validate(app.ValidateResult{
		Success:     false,
		ProjectRoot: `D:\papers\a`,
		Diagnostics: diags,
	})

	var jsonBuf bytes.Buffer
	jw := &JSONWriter{W: &jsonBuf}
	jw.Validate(app.ValidateResult{
		Success:     false,
		ProjectRoot: `D:\papers\a`,
		Diagnostics: diags,
	})

	textOut := textBuf.String()
	jsonOut := jsonBuf.String()

	if !strings.Contains(textOut, "NP1001") {
		t.Fatalf("text output missing diagnostic code: %s", textOut)
	}
	if !strings.Contains(jsonOut, "NP1001") {
		t.Fatalf("JSON output missing diagnostic code: %s", jsonOut)
	}
}

// ---------- IsTerminalSuccess --------------------------------------------

func TestIsTerminalSuccess(t *testing.T) {
	tests := []struct {
		name    string
		diags   []diagnostic.Diagnostic
		success bool
		want    bool
	}{
		{"success no diags", nil, true, true},
		{"success with warning", []diagnostic.Diagnostic{
			makeDiag(diagnostic.SeverityWarning, "X", "warn", "", 0, ""),
		}, true, true},
		{"success with info", []diagnostic.Diagnostic{
			makeDiag(diagnostic.SeverityInfo, "X", "info", "", 0, ""),
		}, true, true},
		{"fail flag", nil, false, false},
		{"success with error", []diagnostic.Diagnostic{
			makeDiag(diagnostic.SeverityError, "X", "err", "", 0, ""),
		}, true, false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsTerminalSuccess(test.diags, test.success); got != test.want {
				t.Fatalf("IsTerminalSuccess() = %v, want %v", got, test.want)
			}
		})
	}
}

// ---------- FormatDiagnostic ---------------------------------------------

func TestFormatDiagnostic(t *testing.T) {
	d := makeDiag(diagnostic.SeverityError, "NP1001", "not found", "paper.md", 5, "check path")
	got := FormatDiagnostic(d)
	if !strings.Contains(got, "error NP1001: not found") {
		t.Fatalf("FormatDiagnostic() = %q, missing core info", got)
	}
	if !strings.Contains(got, "paper.md:5") {
		t.Fatalf("FormatDiagnostic() = %q, missing file:line", got)
	}
}
