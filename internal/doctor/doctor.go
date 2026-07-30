// Package doctor inspects the local environment and reports whether the
// NodePaper toolchain is ready to build.
package doctor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"nodepaper/internal/diagnostic"
	"nodepaper/internal/process"
	"nodepaper/internal/profile"
)

// Status summarises a single tool or resource check.
type Status string

const (
	StatusPass    Status = "pass"
	StatusWarning Status = "warning"
	StatusFail    Status = "fail"
	StatusSkipped Status = "skipped"
)

// Check is the outcome of inspecting a single tool or resource.
type Check struct {
	Name       string `json:"name"`
	Status     Status `json:"status"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

// Result collects the findings from a doctor run.
type Result struct {
	Success     bool                    `json:"success"`
	ProjectRoot string                  `json:"projectRoot,omitempty"`
	Checks      []Check                 `json:"checks"`
	Diagnostics []diagnostic.Diagnostic `json:"diagnostics"`
}

// Toolchain represents the local tool binaries. Paths may be empty when a tool
// is not installed.
type Toolchain struct {
	Powershell     string
	Pandoc         string
	PandocCrossref string
	Latexmk        string
	XeLaTeX        string
}

// Run inspects the local environment. When projectRoot is non-empty it also
// verifies project resources.
func Run(ctx context.Context, projectRoot string, builtinProfileDir string) Result {
	var checks []Check

	checks = append(checks, checkPowershell())
	tc := findToolchain()
	checks = append(checks,
		checkPandoc(tc.Pandoc),
		checkPandocCrossref(tc.PandocCrossref),
		checkLatexmk(tc.Latexmk),
		checkXeLaTeX(tc.XeLaTeX),
	)
	checks = append(checks, checkChineseProbe(ctx, tc))

	if projectRoot != "" {
		checks = append(checks, checkProjectResources(projectRoot, builtinProfileDir)...)
	}

	result := Result{Success: true, Checks: checks}
	for _, c := range checks {
		if c.Status == StatusFail {
			result.Success = false
			result.Diagnostics = append(result.Diagnostics, checkToDiag(c))
		}
		if c.Status == StatusWarning {
			result.Diagnostics = append(result.Diagnostics, checkToDiag(c))
		}
	}
	if projectRoot != "" {
		result.ProjectRoot = projectRoot
	}
	return result
}

// ---------- individual checks -------------------------------------------

func checkPowershell() Check {
	path, err := exec.LookPath("powershell.exe")
	if err != nil {
		path, err = exec.LookPath("pwsh")
	}
	if err != nil {
		return Check{
			Name:       "PowerShell",
			Status:     StatusFail,
			Message:    "PowerShell not found",
			Suggestion: "Install PowerShell 7+ or ensure powershell.exe is in PATH.",
		}
	}
	return Check{
		Name:    "PowerShell",
		Status:  StatusPass,
		Message: path,
	}
}

func findToolchain() Toolchain {
	var tc Toolchain
	tc.Pandoc, _ = exec.LookPath("pandoc")
	tc.PandocCrossref, _ = exec.LookPath("pandoc-crossref")
	tc.Latexmk, _ = exec.LookPath("latexmk")
	tc.XeLaTeX, _ = exec.LookPath("xelatex")
	return tc
}

func checkPandoc(path string) Check {
	if path == "" {
		return Check{
			Name:       "Pandoc",
			Status:     StatusFail,
			Message:    "Pandoc not found",
			Suggestion: "Install Pandoc 3.x and ensure pandoc is in PATH.",
		}
	}
	return Check{
		Name:    "Pandoc",
		Status:  StatusPass,
		Message: path,
	}
}

func checkPandocCrossref(path string) Check {
	if path == "" {
		return Check{
			Name:       "pandoc-crossref",
			Status:     StatusFail,
			Message:    "pandoc-crossref not found",
			Suggestion: "Install pandoc-crossref and ensure it is in PATH.",
		}
	}
	return Check{
		Name:    "pandoc-crossref",
		Status:  StatusPass,
		Message: path,
	}
}

func checkLatexmk(path string) Check {
	if path == "" {
		return Check{
			Name:       "latexmk",
			Status:     StatusFail,
			Message:    "latexmk not found",
			Suggestion: "Install TeX Live or MiKTeX and ensure latexmk is in PATH.",
		}
	}
	return Check{
		Name:    "latexmk",
		Status:  StatusPass,
		Message: path,
	}
}

func checkXeLaTeX(path string) Check {
	if path == "" {
		return Check{
			Name:       "XeLaTeX",
			Status:     StatusFail,
			Message:    "XeLaTeX not found",
			Suggestion: "Install TeX Live or MiKTeX and ensure xelatex is in PATH.",
		}
	}
	return Check{
		Name:    "XeLaTeX",
		Status:  StatusPass,
		Message: path,
	}
}

func checkChineseProbe(ctx context.Context, tc Toolchain) Check {
	if tc.Latexmk == "" || tc.XeLaTeX == "" {
		return Check{
			Name:       "Chinese TeX probe",
			Status:     StatusSkipped,
			Message:    "latexmk or XeLaTeX prerequisite is missing",
			Suggestion: "Install the missing TeX tools, then run doctor again.",
		}
	}

	dir, err := os.MkdirTemp("", "nodepaper-doctor-probe-*")
	if err != nil {
		return Check{Name: "Chinese TeX probe", Status: StatusFail, Message: fmt.Sprintf("cannot create probe directory: %v", err)}
	}
	defer os.RemoveAll(dir)

	const document = `\documentclass[UTF8]{ctexart}
\begin{document}
NodePaper 中文环境探针
\end{document}
`
	texPath := filepath.Join(dir, "probe.tex")
	if err := os.WriteFile(texPath, []byte(document), 0o644); err != nil {
		return Check{Name: "Chinese TeX probe", Status: StatusFail, Message: fmt.Sprintf("cannot write probe: %v", err)}
	}

	runner := &process.Runner{Dir: dir, CaptureSize: 256 * 1024}
	processResult, runErr := runner.Run(ctx, tc.Latexmk,
		"-xelatex",
		"-interaction=nonstopmode",
		"-halt-on-error",
		"-file-line-error",
		"-outdir="+dir,
		texPath,
	)
	if runErr != nil || processResult.ExitCode != 0 {
		message := fmt.Sprintf("minimal Chinese document failed with exit code %d", processResult.ExitCode)
		if runErr != nil {
			message = fmt.Sprintf("minimal Chinese document failed: %v", runErr)
		}
		return Check{
			Name:       "Chinese TeX probe",
			Status:     StatusFail,
			Message:    message,
			Suggestion: "Inspect the TeX installation, ctex package and configured Chinese fonts.",
		}
	}

	pdfPath := filepath.Join(dir, "probe.pdf")
	data, err := os.ReadFile(pdfPath)
	if err != nil || len(data) < 5 || string(data[:5]) != "%PDF-" {
		return Check{
			Name:       "Chinese TeX probe",
			Status:     StatusFail,
			Message:    "minimal Chinese compile did not produce a readable PDF",
			Suggestion: "Inspect the TeX log and reinstall the ctex/XeLaTeX components.",
		}
	}
	return Check{
		Name:    "Chinese TeX probe",
		Status:  StatusPass,
		Message: "minimal ctex document compiled successfully",
	}
}

func checkProjectResources(projectRoot, builtinProfileDir string) []Check {
	var checks []Check

	// Check nodepaper.yaml exists.
	configPath := filepath.Join(projectRoot, "nodepaper.yaml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		checks = append(checks, Check{
			Name:       "nodepaper.yaml",
			Status:     StatusFail,
			Message:    fmt.Sprintf("Project config not found: %s", configPath),
			Suggestion: "Run 'nodepaper init' to create a project.",
		})
	} else {
		checks = append(checks, Check{
			Name:    "nodepaper.yaml",
			Status:  StatusPass,
			Message: configPath,
		})
	}

	// Strictly load every declared resource in the immutable built-in Profile.
	if builtinProfileDir != "" {
		loaded, err := profile.Load(builtinProfileDir)
		if err != nil {
			checks = append(checks, Check{
				Name:       "profile",
				Status:     StatusFail,
				Message:    fmt.Sprintf("CUMCM Profile is invalid: %v", err),
				Suggestion: "Reinstall NodePaper or restore profiles/cumcm.",
			})
		} else {
			checks = append(checks, Check{
				Name:    "profile",
				Status:  StatusPass,
				Message: fmt.Sprintf("%s (rules %s, version %s)", loaded.Dir, loaded.Definition.RulesVersion, loaded.Definition.Version),
			})
		}
	}

	return checks
}

// ---------- helpers ------------------------------------------------------

func checkToDiag(c Check) diagnostic.Diagnostic {
	severity := diagnostic.SeverityError
	if c.Status == StatusWarning || c.Status == StatusSkipped {
		severity = diagnostic.SeverityWarning
	}
	return diagnostic.Diagnostic{
		Severity:   severity,
		Code:       fmt.Sprintf("NP4%02d", severityCode(c)),
		Message:    fmt.Sprintf("[%s] %s: %s", c.Status, c.Name, c.Message),
		Suggestion: c.Suggestion,
		Source:     "doctor",
	}
}

func severityCode(c Check) int {
	switch c.Status {
	case StatusPass:
		return 0
	case StatusWarning:
		return 1
	case StatusFail:
		return 2
	default:
		return 9
	}
}

// FormatChecks returns a multi-line human-readable summary of checks.
func FormatChecks(checks []Check) string {
	var sb strings.Builder
	for _, c := range checks {
		prefix := "PASS"
		switch c.Status {
		case StatusWarning:
			prefix = "WARN"
		case StatusFail:
			prefix = "FAIL"
		case StatusSkipped:
			prefix = "SKIP"
		}
		fmt.Fprintf(&sb, "%-4s %-20s %s\n", prefix, c.Name, c.Message)
		if c.Suggestion != "" {
			fmt.Fprintf(&sb, "     → %s\n", c.Suggestion)
		}
	}
	return sb.String()
}
