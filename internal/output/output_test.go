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

func TestTextWriterIndentsMultiLineDiagnosticSuggestion(t *testing.T) {
	var buf bytes.Buffer
	tw := &TextWriter{W: &buf}

	tw.Build(app.BuildResult{
		ProjectRoot: `D:\papers\a`,
		Diagnostics: []diagnostic.Diagnostic{
			makeDiag(diagnostic.SeverityError, "NP6002", "no PDF was produced", "", 0, "Install a TeX distribution:\n  MiKTeX     https://miktex.org/download"),
		},
	})

	out := buf.String()
	if !strings.Contains(out, "      Suggestion: Install a TeX distribution:\n") {
		t.Fatalf("first suggestion line changed shape:\n%s", out)
	}
	if !strings.Contains(out, "\n                    MiKTeX     https://miktex.org/download\n") {
		t.Fatalf("continuation line is not indented under the suggestion:\n%s", out)
	}
}

func TestTextWriterDoctorGroupsChecksByCapability(t *testing.T) {
	var buf bytes.Buffer
	tw := &TextWriter{W: &buf}

	// The second Toolchain check comes last on purpose: the renderer buckets by
	// the group each check carries, so one heading has to cover both.
	tw.Doctor(app.DoctorResult{
		Success: true,
		Checks: []app.DoctorCheck{
			{Name: "Pandoc", Status: "pass", Message: "pandoc 3.9", Group: "Toolchain"},
			{Name: "XeLaTeX", Status: "warning", Message: "XeLaTeX not found", Suggestion: "install a TeX distribution\nsecond guidance line", Group: "PDF output (nodepaper build)"},
			{Name: "profile", Status: "pass", Message: "loaded", Group: "Toolchain"},
		},
	})

	out := buf.String()
	for _, group := range []string{"Toolchain", "PDF output (nodepaper build)"} {
		if count := strings.Count(out, group+"\n"); count != 1 {
			t.Fatalf("heading %q appears %d times, want once:\n%s", group, count, out)
		}
	}
	if strings.Index(out, "profile") > strings.Index(out, "PDF output (nodepaper build)") {
		t.Fatalf("profile was filed under the wrong heading:\n%s", out)
	}
	if !strings.Contains(out, "  WARN") {
		t.Fatalf("grouped checks should be indented under their heading:\n%s", out)
	}
	if !strings.Contains(out, "install a TeX distribution") {
		t.Fatalf("suggestion was dropped:\n%s", out)
	}
	// A long suggestion is the one thing that can visually escape its section,
	// so its continuation lines are indented with it.
	if !strings.Contains(out, "\n         second guidance line\n") {
		t.Fatalf("continuation line of a multi-line suggestion is not indented:\n%s", out)
	}
}

func TestTextWriterDoctorKeepsACheckWithNoGroup(t *testing.T) {
	// A check that arrives without a group is what a future check looks like
	// before anyone groups it. It may lose its heading; it may not disappear.
	var buf bytes.Buffer
	tw := &TextWriter{W: &buf}

	tw.Doctor(app.DoctorResult{
		Success: true,
		Checks: []app.DoctorCheck{
			{Name: "Pandoc", Status: "pass", Message: "pandoc 3.9", Group: "Toolchain"},
			{Name: "unregistered check", Status: "fail", Message: "invented by a later change", Suggestion: "still shown"},
		},
	})

	out := buf.String()
	for _, expected := range []string{"unregistered check", "invented by a later change", "still shown"} {
		if !strings.Contains(out, expected) {
			t.Fatalf("output missing %q:\n%s", expected, out)
		}
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

func TestTextWriterExportPrintsTheChainForTheChosenMode(t *testing.T) {
	for bibMode, want := range map[string][]string{
		"bibtex":   {"xelatex paper.tex", "bibtex paper", "xelatex paper.tex"},
		"biblatex": {"xelatex paper.tex", "biber paper", "xelatex paper.tex"},
		"inline":   {"xelatex paper.tex", "xelatex paper.tex"},
	} {
		var buf bytes.Buffer
		tw := &TextWriter{W: &buf}
		tw.Export(app.ExportResult{
			Success:         true,
			ProjectRoot:     `D:\papers\a`,
			ExportDir:       `D:\out\latex`,
			BibMode:         bibMode,
			CompileCommands: want,
			Artifacts:       []app.Artifact{{Kind: "tex", Path: `D:\out\latex\paper.tex`}},
		})
		out := buf.String()
		if !strings.Contains(out, "Export: D:\\out\\latex") {
			t.Errorf("%s: missing export directory: %s", bibMode, out)
		}
		if !strings.Contains(out, "Next:\n") {
			t.Errorf("%s: missing indented next-step list: %s", bibMode, out)
		}
		for _, command := range want {
			if !strings.Contains(out, "  "+command+"\n") {
				t.Errorf("%s: missing command %q: %s", bibMode, command, out)
			}
		}
		if bibMode != "bibtex" && strings.Contains(out, "bibtex paper") {
			t.Errorf("%s: printed the bibtex step: %s", bibMode, out)
		}
		if bibMode != "biblatex" && strings.Contains(out, "biber paper") {
			t.Errorf("%s: printed the biber step: %s", bibMode, out)
		}
	}
}

func TestTextWriterExportQualifiesVerification(t *testing.T) {
	var buf bytes.Buffer
	tw := &TextWriter{W: &buf}
	tw.Export(app.ExportResult{
		Success:         true,
		ProjectRoot:     `D:\papers\a`,
		ExportDir:       `D:\out\latex`,
		BibMode:         "bibtex",
		Verified:        true,
		CompileCommands: []string{"xelatex paper.tex"},
	})
	out := buf.String()
	if !strings.Contains(out, "Verified:") {
		t.Fatalf("missing verification line: %s", out)
	}
	// A local compile says nothing about the recipient's machine, and the
	// output has to say so or people will read it as a guarantee.
	if !strings.Contains(out, "does not guarantee") {
		t.Fatalf("verification is not qualified: %s", out)
	}

	buf.Reset()
	tw.Export(app.ExportResult{Success: true, ExportDir: `D:\out\latex`, BibMode: "bibtex"})
	if strings.Contains(buf.String(), "Verified:") {
		t.Fatalf("unverified export claims verification: %s", buf.String())
	}
}

func TestTextWriterExportZipIsReadyForOverleaf(t *testing.T) {
	var buf bytes.Buffer
	tw := &TextWriter{W: &buf}
	tw.Export(app.ExportResult{
		Success:     true,
		ProjectRoot: `D:\papers\a`,
		ExportPath:  `D:\out\paper-latex.zip`,
		Zipped:      true,
		BibMode:     "bibtex",
	})

	out := buf.String()
	if !strings.Contains(out, `Export: D:\out\paper-latex.zip`) {
		t.Fatalf("missing ZIP path: %s", out)
	}
	if !strings.Contains(out, "Upload Project") {
		t.Fatalf("missing direct-upload guidance: %s", out)
	}
	if strings.Contains(out, "xelatex paper.tex") {
		t.Fatalf("ZIP output should not tell the user to cd into it and compile: %s", out)
	}
	// Overleaf is the secondary route, so the local one is named first and the
	// free-plan cap travels with the Overleaf line rather than being omitted.
	if !strings.Contains(out, "10 seconds") {
		t.Fatalf("ZIP output does not state the Overleaf free-plan compile cap: %s", out)
	}
	if strings.Index(out, "README.txt") > strings.Index(out, "Upload Project") {
		t.Fatalf("ZIP output puts Overleaf ahead of the local compile route: %s", out)
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

func TestJSONWriterExport(t *testing.T) {
	var buf bytes.Buffer
	jw := &JSONWriter{W: &buf}

	err := jw.Export(app.ExportResult{
		Success:         true,
		ProjectRoot:     `D:\papers\a`,
		ExportPath:      `D:\out\latex`,
		ExportDir:       `D:\out\latex`,
		Zipped:          false,
		BibMode:         "bibtex",
		Verified:        true,
		CompileCommands: []string{"xelatex paper.tex", "bibtex paper", "xelatex paper.tex", "xelatex paper.tex"},
		Artifacts:       []app.Artifact{{Kind: "tex", Path: `D:\out\latex\paper.tex`}},
	})
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	var envelope map[string]any
	if err := json.Unmarshal(buf.Bytes(), &envelope); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if v, ok := envelope["schemaVersion"].(float64); !ok || int(v) != schemaVersion {
		t.Fatalf("schemaVersion = %v, want %d", envelope["schemaVersion"], schemaVersion)
	}
	for key, want := range map[string]any{
		"exportPath": `D:\out\latex`,
		"exportDir":  `D:\out\latex`,
		"bibMode":    "bibtex",
		"zipped":     false,
		"verified":   true,
	} {
		if envelope[key] != want {
			t.Errorf("%s = %v, want %v", key, envelope[key], want)
		}
	}
	commands, ok := envelope["compileCommands"].([]any)
	if !ok || len(commands) != 4 || commands[1] != "bibtex paper" {
		t.Fatalf("compileCommands = %v, want the four-step bibtex chain", envelope["compileCommands"])
	}
}

func TestJSONWriterZipExportUsesExportPathWithoutExportDir(t *testing.T) {
	var buf bytes.Buffer
	jw := &JSONWriter{W: &buf}
	target := `D:\out\paper.zip`

	err := jw.Export(app.ExportResult{
		Success:     true,
		ProjectRoot: `D:\papers\a`,
		ExportPath:  target,
		Zipped:      true,
		BibMode:     "bibtex",
		Artifacts:   []app.Artifact{{Kind: "zip", Path: target}},
	})
	if err != nil {
		t.Fatalf("Export() error = %v", err)
	}
	var envelope map[string]any
	if err := json.Unmarshal(buf.Bytes(), &envelope); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if envelope["exportPath"] != target || envelope["zipped"] != true {
		t.Fatalf("ZIP fields = %#v", envelope)
	}
	if _, exists := envelope["exportDir"]; exists {
		t.Fatalf("ZIP JSON unexpectedly contains exportDir: %#v", envelope)
	}
	artifacts, ok := envelope["artifacts"].([]any)
	if !ok || len(artifacts) != 1 {
		t.Fatalf("artifacts = %#v", envelope["artifacts"])
	}
	artifact, ok := artifacts[0].(map[string]any)
	if !ok || artifact["kind"] != "zip" || artifact["path"] != target {
		t.Fatalf("ZIP artifact = %#v", artifacts[0])
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
