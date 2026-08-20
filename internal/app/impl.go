package app

import (
	"context"
	"os"
	"path/filepath"

	"nodepaper/internal/build"
	"nodepaper/internal/diagnostic"
	"nodepaper/internal/doctor"
	"nodepaper/internal/export"
	"nodepaper/internal/project"
	"nodepaper/internal/validate"
)

// New returns the default App implementation that orchestrates all NodePaper
// operations without rendering output or depending on CLI globals.
func New() App {
	resourceRoot, profileDir := findResourceDirs()
	return &appImpl{
		resourceRoot: resourceRoot,
		profileDir:   profileDir,
	}
}

type appImpl struct {
	resourceRoot string
	profileDir   string
}

func (a *appImpl) Doctor(ctx context.Context, req DoctorRequest) (DoctorResult, error) {
	projectRoot := ""
	if req.ProjectDir != "" {
		p, err := project.Discover(req.ProjectDir)
		if err != nil {
			if discovery, ok := err.(*project.DiscoveryError); ok {
				return DoctorResult{Diagnostics: []diagnostic.Diagnostic{discovery.Diagnostic}}, nil
			}
			return DoctorResult{}, err
		}
		projectRoot = p.Root
	} else if p, err := project.Discover(""); err == nil {
		projectRoot = p.Root
	} else if discovery, ok := err.(*project.DiscoveryError); !ok || discovery.Diagnostic.Code != project.CodeProjectNotFound {
		if ok {
			return DoctorResult{Diagnostics: []diagnostic.Diagnostic{discovery.Diagnostic}}, nil
		}
		return DoctorResult{}, err
	}

	dr := doctor.Run(ctx, projectRoot, a.resourceRoot)

	var checks []DoctorCheck
	for _, c := range dr.Checks {
		checks = append(checks, DoctorCheck{
			Name:       c.Name,
			Status:     string(c.Status),
			Message:    c.Message,
			Suggestion: c.Suggestion,
			Group:      string(c.Group),
		})
	}

	return DoctorResult{
		Success:     dr.Success,
		ProjectRoot: dr.ProjectRoot,
		Diagnostics: dr.Diagnostics,
		Checks:      checks,
	}, nil
}

func (a *appImpl) Validate(ctx context.Context, req ValidateRequest) (ValidateResult, error) {
	vr := validate.Run(ctx, req.ProjectDir)
	return ValidateResult{
		Success:     vr.Success,
		ProjectRoot: vr.ProjectRoot,
		Diagnostics: vr.Diagnostics,
	}, nil
}

func (a *appImpl) Build(ctx context.Context, req BuildRequest) (BuildResult, error) {
	br := build.Run(ctx, req.ProjectDir)
	var artifacts []Artifact
	for _, art := range br.Artifacts {
		artifacts = append(artifacts, Artifact{Kind: art.Kind, Path: art.Path})
	}
	return BuildResult{
		Success:     br.Success,
		BuildID:     br.BuildID,
		ProjectRoot: br.ProjectRoot,
		Artifacts:   artifacts,
		Diagnostics: br.Diagnostics,
	}, nil
}

func (a *appImpl) Export(ctx context.Context, req ExportRequest) (ExportResult, error) {
	mode := export.BibMode(req.Bib)
	if req.Bib != "" {
		parsed, err := export.ParseBibMode(req.Bib)
		if err != nil {
			return ExportResult{}, err
		}
		mode = parsed
	}
	er := export.Run(ctx, export.Options{
		ProjectDir: req.ProjectDir,
		ToPath:     req.ToPath,
		Bib:        mode,
		Verify:     req.Verify,
		Force:      req.Force,
	})
	var artifacts []Artifact
	for _, art := range er.Artifacts {
		artifacts = append(artifacts, Artifact{Kind: art.Kind, Path: art.Path})
	}
	return ExportResult{
		Success:         er.Success,
		ProjectRoot:     er.ProjectRoot,
		ExportPath:      er.ExportPath,
		ExportDir:       er.ExportDir,
		Zipped:          er.Zipped,
		BibMode:         er.BibMode,
		Verified:        er.Verified,
		CompileCommands: er.CompileCommands,
		Artifacts:       artifacts,
		Diagnostics:     er.Diagnostics,
	}, nil
}

func (a *appImpl) Clean(ctx context.Context, req CleanRequest) (CleanResult, error) {
	diags, err := build.Clean(req.ProjectDir, req.All)
	if err != nil {
		return CleanResult{}, err
	}
	return CleanResult{
		Success:     !hasErrorDiags(diags),
		ProjectRoot: req.ProjectDir,
		Diagnostics: diags,
	}, nil
}

func hasErrorDiags(diags []diagnostic.Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == diagnostic.SeverityError {
			return true
		}
	}
	return false
}

// findResourceDirs resolves the packaged resource tree. The returned
// resourceRoot contains profiles/ and tools/ and is empty when the packaged
// resources cannot be located; profileDir points at profiles/cumcm.
func findResourceDirs() (resourceRoot, profileDir string) {
	if override := os.Getenv("NODEPAPER_PROFILE_DIR"); override != "" {
		return filepath.Dir(filepath.Dir(override)), override
	}
	// Try profiles/cumcm relative to the executable first.
	if exe, err := os.Executable(); err == nil {
		root := filepath.Dir(exe)
		dir := filepath.Join(root, "profiles", "cumcm")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return root, dir
		}
	}
	// Fallback: look relative to the current working directory (dev mode).
	if wd, err := os.Getwd(); err == nil {
		dir := filepath.Join(wd, "profiles", "cumcm")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return wd, dir
		}
	}
	return "", ""
}
