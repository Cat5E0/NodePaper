// Package app defines the application-service boundary shared by presentation
// adapters such as the CLI and a future GUI.
package app

import (
	"context"

	"nodepaper/internal/artifact"
	"nodepaper/internal/diagnostic"
)

// App is the complete application-service boundary. Implementations own
// operation orchestration and return structured data without rendering output.
type App interface {
	Init(context.Context, InitRequest) (InitResult, error)
	Doctor(context.Context, DoctorRequest) (DoctorResult, error)
	Validate(context.Context, ValidateRequest) (ValidateResult, error)
	Build(context.Context, BuildRequest) (BuildResult, error)
	Export(context.Context, ExportRequest) (ExportResult, error)
	Clean(context.Context, CleanRequest) (CleanResult, error)
}

// Artifact and Diagnostic expose the shared result vocabulary at the
// application boundary while keeping their definitions in focused packages.
type Artifact = artifact.Artifact
type Diagnostic = diagnostic.Diagnostic

type InitRequest struct {
	ProjectDir      string
	GenerateAIGuide bool
}

type DoctorRequest struct {
	ProjectDir string
}

type ValidateRequest struct {
	ProjectDir string
}

type BuildRequest struct {
	ProjectDir string
}

// ExportRequest asks for an editable LaTeX project. Bib is "bibtex",
// "biblatex" or "inline"; an empty value means the default.
type ExportRequest struct {
	ProjectDir string
	ToDir      string
	Bib        string
	Verify     bool
	Force      bool
}

type CleanRequest struct {
	ProjectDir string
	All        bool
}

type InitResult struct {
	Success     bool         `json:"success"`
	ProjectRoot string       `json:"projectRoot"`
	Artifacts   []Artifact   `json:"artifacts"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type DoctorResult struct {
	Success     bool          `json:"success"`
	ProjectRoot string        `json:"projectRoot"`
	Diagnostics []Diagnostic  `json:"diagnostics"`
	Checks      []DoctorCheck `json:"checks,omitempty"`
}

// DoctorCheck is a structured pass/warn/fail item from the doctor package.
type DoctorCheck struct {
	Name       string `json:"name"`
	Status     string `json:"status"`
	Message    string `json:"message"`
	Suggestion string `json:"suggestion,omitempty"`
}

type ValidateResult struct {
	Success     bool         `json:"success"`
	ProjectRoot string       `json:"projectRoot"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type BuildResult struct {
	Success     bool         `json:"success"`
	BuildID     string       `json:"buildId"`
	ProjectRoot string       `json:"projectRoot"`
	Artifacts   []Artifact   `json:"artifacts"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}

type ExportResult struct {
	Success     bool   `json:"success"`
	ProjectRoot string `json:"projectRoot"`
	ExportDir   string `json:"exportDir"`
	BibMode     string `json:"bibMode"`
	// Verified is true only when --verify ran the compile chain and it
	// succeeded. It is false both when verification was not requested and
	// when it could not run, so it never implies more than it proves.
	Verified bool `json:"verified"`
	// CompileCommands is the command chain the recipient of the export has to
	// run, in order.
	CompileCommands []string     `json:"compileCommands"`
	Artifacts       []Artifact   `json:"artifacts"`
	Diagnostics     []Diagnostic `json:"diagnostics"`
}

type CleanResult struct {
	Success     bool         `json:"success"`
	ProjectRoot string       `json:"projectRoot"`
	Artifacts   []Artifact   `json:"artifacts"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}
