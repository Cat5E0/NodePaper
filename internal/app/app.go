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
	Clean(context.Context, CleanRequest) (CleanResult, error)
}

// Artifact and Diagnostic expose the shared result vocabulary at the
// application boundary while keeping their definitions in focused packages.
type Artifact = artifact.Artifact
type Diagnostic = diagnostic.Diagnostic

type InitRequest struct {
	ProjectDir string
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
	Success     bool         `json:"success"`
	ProjectRoot string       `json:"projectRoot"`
	Diagnostics []Diagnostic `json:"diagnostics"`
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

type CleanResult struct {
	Success     bool         `json:"success"`
	ProjectRoot string       `json:"projectRoot"`
	Artifacts   []Artifact   `json:"artifacts"`
	Diagnostics []Diagnostic `json:"diagnostics"`
}
