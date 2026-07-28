// Package build orchestrates the full build pipeline for a NodePaper project.
package build

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"nodepaper/internal/buildctx"
	"nodepaper/internal/buildlock"
	"nodepaper/internal/config"
	"nodepaper/internal/diagnostic"
	"nodepaper/internal/process"
	"nodepaper/internal/project"
	"nodepaper/internal/validate"
)

// Result holds the outcome of a build.
type Result struct {
	Success     bool
	BuildID     string
	ProjectRoot string
	Artifacts   []Artifact
	Diagnostics []diagnostic.Diagnostic
}

// Artifact describes a produced file.
type Artifact struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
}

type commandExecutor interface {
	Run(ctx context.Context, dir, command string, args ...string) (process.Result, error)
}

type processExecutor struct{}

func (processExecutor) Run(ctx context.Context, dir, command string, args ...string) (process.Result, error) {
	return (&process.Runner{Dir: dir}).Run(ctx, command, args...)
}

type buildLogger struct {
	file *os.File
}

func newBuildLogger(path string) (*buildLogger, error) {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &buildLogger{file: file}, nil
}

func (l *buildLogger) Printf(format string, args ...any) {
	if l == nil || l.file == nil {
		return
	}
	fmt.Fprintf(l.file, "%s ", time.Now().Format(time.RFC3339Nano))
	fmt.Fprintf(l.file, format, args...)
	if !strings.HasSuffix(format, "\n") {
		fmt.Fprintln(l.file)
	}
}

func (l *buildLogger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}

// Run orchestrates the complete build for a project. It discovers the project,
// validates it, acquires the build lock, delegates conversion and compilation
// to the v0.1 PowerShell transition script, and atomically publishes the PDF.
func Run(ctx context.Context, projectDir string) Result {
	return runWithExecutorAndScript(ctx, projectDir, processExecutor{}, defaultBuildScriptPath())
}

func runWithExecutor(ctx context.Context, projectDir string, executor commandExecutor) Result {
	return runWithExecutorAndScript(ctx, projectDir, executor, defaultBuildScriptPath())
}

func runWithExecutorAndScript(ctx context.Context, projectDir string, executor commandExecutor, scriptPath string) Result {
	var result Result

	// 1. Discover project.
	p, err := project.Discover(projectDir)
	if err != nil {
		if de, ok := err.(*project.DiscoveryError); ok {
			result.Diagnostics = append(result.Diagnostics, de.Diagnostic)
		} else {
			result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
				Severity: diagnostic.SeverityError,
				Code:     "NP1001",
				Message:  fmt.Sprintf("cannot discover project: %v", err),
				Source:   "build",
			})
		}
		return result
	}
	result.ProjectRoot = p.Root

	// 2. Load config.
	cfg, err := config.Load(p.ConfigPath)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, configDiag(err))
		return result
	}

	// 3. Run validation.
	vr := validate.Run(ctx, projectDir)
	if !vr.Success {
		result.Diagnostics = vr.Diagnostics
		return result
	}

	// The existing v0.1 PowerShell transition script accepts one Markdown
	// source. Reject multiple sources instead of silently dropping content;
	// M3 will extend the transition layer for ordered multi-file projects.
	if len(cfg.SourceFiles()) != 1 {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Severity:   diagnostic.SeverityError,
			Code:       "NP5002",
			Message:    "the v0.1 PowerShell transition build supports exactly one Markdown source",
			Suggestion: "Use a single source for the M2 baseline; multi-file build support is scheduled for M3.",
			Source:     "build",
		})
		return result
	}

	// 4. Create the build context first so the project lock records the real,
	// externally visible Build ID rather than a placeholder. Directory creation
	// is limited to idempotent project-local MkdirAll calls.
	bctx, err := buildctx.New(p.Root, cfg.SourceFiles(), cfg.Profile)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Severity: diagnostic.SeverityError,
			Code:     "NP1301",
			Message:  fmt.Sprintf("cannot create build context: %v", err),
			Source:   "build",
		})
		return result
	}
	result.BuildID = bctx.BuildID

	// 5. Acquire the project build lock with the real Build ID.
	held, err := buildlock.Acquire(p.Root, bctx.BuildID)
	if err != nil {
		if he, ok := err.(*buildlock.ErrHeld); ok {
			result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
				Severity:   diagnostic.SeverityError,
				Code:       "NP1201",
				Message:    he.Error(),
				Suggestion: "Wait for the current build to finish.",
				Source:     "build",
			})
			return result
		}
		if se, ok := err.(*buildlock.ErrStale); ok {
			result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
				Severity:   diagnostic.SeverityError,
				Code:       "NP1202",
				Message:    se.Error(),
				Suggestion: "Run 'nodepaper clean' to remove the stale lock.",
				Source:     "build",
			})
			return result
		}
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Severity: diagnostic.SeverityError,
			Code:     "NP1299",
			Message:  fmt.Sprintf("cannot acquire build lock: %v", err),
			Source:   "build",
		})
		return result
	}
	defer held.Release()

	logger, err := newBuildLogger(bctx.LogPath)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Severity: diagnostic.SeverityError,
			Code:     "NP1304",
			Message:  fmt.Sprintf("cannot create build log: %v", err),
			Source:   "build",
		})
		return result
	}
	result.Artifacts = []Artifact{{Kind: "log", Path: bctx.LogPath}}
	logger.Printf("Build ID: %s", bctx.BuildID)
	logger.Printf("Project Root: %s", bctx.ProjectRoot)
	logger.Printf("Profile: %s", bctx.Profile)
	logger.Printf("Sources: %s", strings.Join(bctx.Sources, ", "))
	defer func() {
		logger.Printf("Success: %t", result.Success)
		_ = logger.Close()
	}()

	// 6. Prepare dist/ directory.
	if err := os.MkdirAll(bctx.OutputDir, 0755); err != nil {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Severity: diagnostic.SeverityError,
			Code:     "NP1302",
			Message:  fmt.Sprintf("cannot create output directory: %v", err),
			Source:   "build",
		})
		return result
	}

	// 7. Remove previous intermediates so a skipped or incomplete transition
	// build cannot be mistaken for a newly generated artifact.
	texPath := bctx.ResolveInWork("paper.tex")
	tmpPDF := bctx.ResolveInWork("paper.pdf")
	for _, path := range []string{texPath, tmpPDF} {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
				Severity: diagnostic.SeverityError,
				Code:     "NP1305",
				Message:  fmt.Sprintf("cannot remove previous build intermediate %s: %v", path, err),
				Source:   "build",
			})
			return result
		}
	}

	// 8. Delegate Markdown conversion and LaTeX compilation to the existing
	// PowerShell transition layer. Go retains orchestration and publication.
	if diags := runPowerShellBuild(ctx, executor, logger, bctx, cfg, scriptPath, texPath); len(diags) > 0 {
		result.Diagnostics = append(result.Diagnostics, diags...)
		return result
	}

	// 9. The legacy script may exit successfully when LaTeX is unavailable,
	// so PDF existence is an explicit Go-side postcondition.
	if _, err := os.Stat(tmpPDF); err != nil {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Severity:   diagnostic.SeverityError,
			Code:       "NP6002",
			Message:    "PDF was not generated by the PowerShell build",
			Suggestion: "Check the PowerShell and LaTeX logs and verify latexmk/XeLaTeX are installed.",
			Source:     "build",
		})
		return result
	}

	// 10. Validate the temporary PDF before publishing it.
	if err := validateGeneratedPDF(tmpPDF); err != nil {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Severity:   diagnostic.SeverityError,
			Code:       "NP7001",
			Message:    fmt.Sprintf("generated PDF is invalid: %v", err),
			Suggestion: "Inspect the LaTeX log and rebuild after fixing the error.",
			Source:     "build",
		})
		return result
	}

	// 11. Atomic publish to the configured project-relative output path.
	outputFile := cfg.Output.File
	if outputFile == "" {
		outputFile = filepath.Join("dist", "paper.pdf")
	}
	finalPDF, err := p.Resolve(outputFile)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Severity: diagnostic.SeverityError,
			Code:     "NP1503",
			Message:  fmt.Sprintf("output path outside project: %s", outputFile),
			Source:   "build",
		})
		return result
	}
	if err := atomicPublish(tmpPDF, finalPDF); err != nil {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Severity:   diagnostic.SeverityError,
			Code:       "NP1303",
			Message:    fmt.Sprintf("cannot publish PDF: %v", err),
			Suggestion: "Check disk space and permissions.",
			Source:     "build",
		})
		return result
	}

	result.Success = true
	result.Artifacts = []Artifact{
		{Kind: "pdf", Path: finalPDF},
		{Kind: "log", Path: bctx.LogPath},
	}
	return result
}

// ---------- PowerShell transition build ---------------------------------

func defaultBuildScriptPath() string {
	if override := os.Getenv("NODEPAPER_BUILD_SCRIPT"); override != "" {
		return override
	}
	executable, err := os.Executable()
	if err != nil {
		return "Build-Paper.ps1"
	}
	return filepath.Join(filepath.Dir(executable), "Build-Paper.ps1")
}

func runPowerShellBuild(ctx context.Context, executor commandExecutor, logger *buildLogger, bctx *buildctx.Context, cfg config.ProjectConfig, scriptPath, texPath string) []diagnostic.Diagnostic {
	sourcePath := filepath.Join(bctx.ProjectRoot, cfg.SourceFiles()[0])
	powerShellLogDir := filepath.Join(filepath.Dir(bctx.LogPath), bctx.BuildID+"-powershell")
	args := []string{
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-File", scriptPath,
		"-MarkdownPath", sourcePath,
		"-Output", texPath,
		"-TemplateName", "assignment",
		"-BuildDirectory", bctx.WorkDir,
		"-LogDirectory", powerShellLogDir,
	}

	logger.Printf("Command: %s", process.LogFriendlyCommand("powershell.exe", args))
	logger.Printf("Working Directory: %s", bctx.ProjectRoot)
	processResult, err := executor.Run(ctx, bctx.ProjectRoot, "powershell.exe", args...)
	logProcessResult(logger, processResult, err)
	if err != nil || processResult.ExitCode != 0 {
		message := fmt.Sprintf("PowerShell build exited with code %d", processResult.ExitCode)
		if err != nil {
			message = fmt.Sprintf("PowerShell build failed: %v", err)
		}
		return []diagnostic.Diagnostic{{
			Severity:   diagnostic.SeverityError,
			Code:       "NP5001",
			Message:    message,
			Suggestion: "Inspect the NodePaper build log and the PowerShell transition log.",
			Source:     "build",
		}}
	}
	return nil
}

func logProcessResult(logger *buildLogger, result process.Result, runErr error) {
	logger.Printf("Exit Code: %d", result.ExitCode)
	if result.Stdout != "" {
		logger.Printf("stdout:\n%s", result.Stdout)
	}
	if result.Stderr != "" {
		logger.Printf("stderr:\n%s", result.Stderr)
	}
	if runErr != nil {
		logger.Printf("Process Error: %v", runErr)
	}
}

// ---------- PDF validation and atomic publish ----------------------------

func validateGeneratedPDF(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	header := make([]byte, 5)
	if _, err := file.Read(header); err != nil {
		return fmt.Errorf("read PDF header: %w", err)
	}
	if string(header) != "%PDF-" {
		return fmt.Errorf("missing %%PDF- header")
	}
	return nil
}

func atomicPublish(src, dst string) error {
	if _, err := os.Stat(src); os.IsNotExist(err) {
		return fmt.Errorf("source PDF not found: %s", src)
	}

	// Ensure destination directory exists.
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	// Read source and write to temp file, then rename.
	data, err := os.ReadFile(src)
	if err != nil {
		return fmt.Errorf("read source: %w", err)
	}

	tmpPath := dst + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("write temp: %w", err)
	}

	if err := os.Rename(tmpPath, dst); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("rename to final: %w", err)
	}

	return nil
}

// Clean removes build artifacts from a project.
// When all is true, dist/ is also removed.
func Clean(projectDir string, all bool) ([]diagnostic.Diagnostic, error) {
	p, err := project.Discover(projectDir)
	if err != nil {
		return nil, err
	}

	var diags []diagnostic.Diagnostic

	// Check for active build lock before cleaning.
	lockPath := buildlock.LockPath(p.Root)
	if info, err := os.Stat(lockPath); err == nil && info.Mode().IsRegular() {
		return []diagnostic.Diagnostic{{
			Severity:   diagnostic.SeverityError,
			Code:       "NP1203",
			Message:    "Cannot clean while a build is in progress or a lock file exists.",
			Suggestion: "Wait for the build to finish, or remove the stale lock manually.",
			Source:     "clean",
		}}, nil
	}

	buildDir := filepath.Join(p.Root, ".nodepaper", "build")
	if err := os.RemoveAll(buildDir); err != nil {
		diags = append(diags, diagnostic.Diagnostic{
			Severity: diagnostic.SeverityWarning,
			Code:     "NP1401",
			Message:  fmt.Sprintf("cannot clean build directory: %v", err),
			Source:   "clean",
		})
	}

	if all {
		distDir := filepath.Join(p.Root, "dist")
		if err := os.RemoveAll(distDir); err != nil {
			diags = append(diags, diagnostic.Diagnostic{
				Severity: diagnostic.SeverityWarning,
				Code:     "NP1401",
				Message:  fmt.Sprintf("cannot clean dist directory: %v", err),
				Source:   "clean",
			})
		}
	}

	return diags, nil
}

// ---------- helpers ------------------------------------------------------

func hasErrors(diags []diagnostic.Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == diagnostic.SeverityError {
			return true
		}
	}
	return false
}

func configDiag(err error) diagnostic.Diagnostic {
	code := "NP1502"
	if strings.Contains(err.Error(), "cannot parse YAML") {
		code = "NP1501"
	}
	return diagnostic.Diagnostic{
		Severity:   diagnostic.SeverityError,
		Code:       code,
		Message:    fmt.Sprintf("invalid configuration: %v", err),
		File:       "nodepaper.yaml",
		Suggestion: "Fix nodepaper.yaml and try again.",
		Source:     "build",
	}
}
