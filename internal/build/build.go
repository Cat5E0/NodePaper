// Package build orchestrates the full build pipeline for a NodePaper project.
package build

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"
	"time"

	"nodepaper/internal/buildctx"
	"nodepaper/internal/buildlock"
	"nodepaper/internal/config"
	"nodepaper/internal/diagnostic"
	"nodepaper/internal/fragment"
	"nodepaper/internal/latexlog"
	"nodepaper/internal/process"
	"nodepaper/internal/profile"
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
	// FontsUsed names the font families the document resolved to. Names only,
	// never paths, so it carries no machine-specific detail and is safe to
	// record alongside the build.
	FontsUsed []latexlog.FontFamily
	// Engine is the TeX engine that produced the PDF, for example
	// "XeTeX, Version 3.141592653-2.6-0.999997 (TeX Live 2025)". NodePaper
	// neither bundles nor pins TeX, so this is the part of the toolchain that
	// legitimately differs between users; recording it makes that difference
	// visible when two builds disagree.
	Engine string
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
	return runWithExecutorAndResources(ctx, projectDir, processExecutor{}, defaultBuildScriptPath(), defaultProfileDir())
}

func runWithExecutor(ctx context.Context, projectDir string, executor commandExecutor) Result {
	return runWithExecutorAndResources(ctx, projectDir, executor, defaultBuildScriptPath(), defaultProfileDir())
}

func runWithExecutorAndScript(ctx context.Context, projectDir string, executor commandExecutor, scriptPath string) Result {
	return runWithExecutorAndResources(ctx, projectDir, executor, scriptPath, defaultProfileDir())
}

func runWithExecutorAndResources(ctx context.Context, projectDir string, executor commandExecutor, scriptPath, profileDir string) Result {
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

	// 3. Load and validate the immutable built-in Profile before acquiring a
	// project lock or invoking any external tool.
	loadedProfile, err := profile.Load(profileDir)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Severity:   diagnostic.SeverityError,
			Code:       "NP1601",
			Message:    fmt.Sprintf("cannot load CUMCM Profile: %v", err),
			Suggestion: "Reinstall NodePaper or restore the profiles/cumcm resources.",
			Source:     "build",
		})
		return result
	}

	// 4. Run validation.
	vr := validate.Run(ctx, projectDir)
	if !vr.Success {
		result.Diagnostics = vr.Diagnostics
		return result
	}

	// 5. Create the build context first so the project lock records the real,
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

	// 6. Acquire the project build lock with the real Build ID.
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
	logger.Printf("NodePaper Version: %s", nodepaperVersion())
	logger.Printf("Project Root: %s", bctx.ProjectRoot)
	logger.Printf("Profile: %s", bctx.Profile)
	logger.Printf("Profile Version: %s", loadedProfile.Definition.Version)
	logger.Printf("Profile Rules Version: %s", loadedProfile.Definition.RulesVersion)
	logger.Printf("Profile SHA-256: %s", loadedProfile.SHA256)
	for _, resource := range loadedProfile.Resources {
		logger.Printf("Profile Resource: %s sha256=%s", resource.Relative, resource.SHA256)
	}
	logger.Printf("Sources: %s", strings.Join(bctx.Sources, ", "))

	fragmentFiles, fragmentIssues := fragment.Inspect(bctx.ProjectRoot, cfg.LatexFragments)
	if len(fragmentIssues) > 0 {
		result.Diagnostics = append(result.Diagnostics, fragmentDiagnostics(fragmentIssues)...)
		return result
	}
	for _, file := range fragmentFiles {
		logger.Printf("LaTeX Fragment: %s sha256=%s", file.Relative, file.SHA256)
	}
	logger.Printf("Appendix Numbering: %s", cfg.Appendix.Numbering)
	logger.Printf("Highlight Style: %s", cfg.Highlight.Style)
	allowlist, err := latexlog.LoadAllowlist(loadedProfile.WarningAllowlist)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Severity:   diagnostic.SeverityError,
			Code:       "NP1601",
			Message:    fmt.Sprintf("cannot load Profile Warning allowlist: %v", err),
			Suggestion: "Restore the reviewed Profile warning-allowlist.json.",
			Source:     "build",
		})
		return result
	}
	defer func() {
		logger.Printf("Success: %t", result.Success)
		_ = logger.Close()
	}()

	// 7. Prepare dist/ directory.
	if err := os.MkdirAll(bctx.OutputDir, 0755); err != nil {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Severity: diagnostic.SeverityError,
			Code:     "NP1302",
			Message:  fmt.Sprintf("cannot create output directory: %v", err),
			Source:   "build",
		})
		return result
	}

	// 8. Recreate the managed WorkDir so stale build intermediates, logs, or
	// outputs cannot be mistaken for artifacts from this build.
	if err := os.RemoveAll(bctx.WorkDir); err != nil {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Severity: diagnostic.SeverityError,
			Code:     "NP1305",
			Message:  fmt.Sprintf("cannot remove previous build intermediates: %v", err),
			Source:   "build",
		})
		return result
	}
	if err := os.MkdirAll(bctx.WorkDir, 0o755); err != nil {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Severity: diagnostic.SeverityError,
			Code:     "NP1305",
			Message:  fmt.Sprintf("cannot recreate build directory: %v", err),
			Source:   "build",
		})
		return result
	}
	texPath := bctx.ResolveInWork("paper.tex")
	tmpPDF := bctx.ResolveInWork("paper.pdf")
	latexLogPath := bctx.ResolveInWork("paper.log")
	sourceManifestPath := bctx.ResolveInWork("sources.json")

	// The source manifest preserves configured order without relying on shell
	// string joining. It is project-local, written only while the lock is held,
	// and consumed by the PowerShell transition layer as UTF-8 JSON.
	absoluteSources := make([]string, 0, len(cfg.SourceFiles()))
	for _, source := range cfg.SourceFiles() {
		absoluteSources = append(absoluteSources, filepath.Join(bctx.ProjectRoot, source))
	}
	absoluteFragments := make([]string, 0, len(fragmentFiles))
	for _, file := range fragmentFiles {
		absoluteFragments = append(absoluteFragments, file.Path)
	}
	manifestData, err := json.MarshalIndent(struct {
		Sources            []string `json:"sources"`
		LatexFragments     []string `json:"latexFragments"`
		AppendixNumbering  string   `json:"appendixNumbering"`
		HighlightStyle     string   `json:"highlightStyle"`
		LineSpread         float64  `json:"linespread"`
		AbstractLineSpread float64  `json:"abstractLinespread"`
		MathFont           string   `json:"mathFont"`
		AppendixNewPage    bool     `json:"appendixNewPage"`
	}{
		Sources:            absoluteSources,
		LatexFragments:     absoluteFragments,
		AppendixNumbering:  cfg.Appendix.Numbering,
		HighlightStyle:     cfg.Highlight.Style,
		LineSpread:         cfg.LineSpread,
		AbstractLineSpread: cfg.AbstractLineSpread,
		MathFont:           cfg.MathFont,
		AppendixNewPage:    cfg.Appendix.NewPageEnabled(),
	}, "", "  ")
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Severity: diagnostic.SeverityError,
			Code:     "NP1306",
			Message:  fmt.Sprintf("cannot encode source manifest: %v", err),
			Source:   "build",
		})
		return result
	}
	if err := os.WriteFile(sourceManifestPath, append(manifestData, '\n'), 0o644); err != nil {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Severity: diagnostic.SeverityError,
			Code:     "NP1306",
			Message:  fmt.Sprintf("cannot write source manifest: %v", err),
			Source:   "build",
		})
		return result
	}

	// 9. Delegate ordered Markdown conversion, Citeproc/CSL processing and
	// LaTeX compilation to the PowerShell transition layer. Go retains
	// orchestration, locking, validation and publication.
	processDiags := runPowerShellBuild(ctx, executor, logger, bctx, scriptPath, loadedProfile.Dir, sourceManifestPath, texPath)
	if len(processDiags) > 0 {
		result.Diagnostics = append(result.Diagnostics, processDiags...)
		// The transition script may return non-zero after producing a useful
		// final log. Preserve the transition failure while also returning
		// classified Fatal/Warning diagnostics instead of forcing the user to
		// parse an opaque NP5001.
		if _, statErr := os.Stat(latexLogPath); statErr == nil {
			result.Diagnostics = append(result.Diagnostics, inspectLatexLog(logger, latexLogPath, allowlist)...)
		}
	}

	currentProfileHash, err := profile.Snapshot(loadedProfile.Dir)
	if err != nil || currentProfileHash != loadedProfile.SHA256 {
		message := "Profile resources changed during build"
		if err != nil {
			message = fmt.Sprintf("cannot verify Profile resources after build: %v", err)
		}
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Severity:   diagnostic.SeverityError,
			Code:       "NP1602",
			Message:    message,
			Suggestion: "Restore the installed Profile and rebuild; do not modify Profile files during a build.",
			Source:     "build",
		})
	}
	if issues := fragment.Verify(fragmentFiles); len(issues) > 0 {
		result.Diagnostics = append(result.Diagnostics, fragmentDiagnostics(issues)...)
	}
	if len(result.Diagnostics) > 0 {
		return result
	}

	// 10. The transition script must generate a PDF; existence is an explicit
	// so PDF existence is an explicit Go-side postcondition.
	if _, err := os.Stat(tmpPDF); err != nil {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Severity:   diagnostic.SeverityError,
			Code:       "NP6002",
			Message:    "PDF was not generated by the PowerShell build",
			Suggestion: "Check the PowerShell and LaTeX logs and verify XeLaTeX is installed.",
			Source:     "build",
		})
		return result
	}

	// 11. Validate the temporary PDF before publishing it.
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
	// Overfull boxes are reported as warnings: the PDF itself is valid and the
	// cause is almost always the source content (a display formula written on
	// one line, a long unbreakable string, an oversized table) rather than a
	// NodePaper defect. The author is told where to fix it and still gets the
	// PDF. Defects that make the PDF wrong - missing glyphs, unresolved
	// references, fatal LaTeX errors - keep blocking publication.
	logDiags := inspectLatexLog(logger, latexLogPath, allowlist)
	result.Diagnostics = append(result.Diagnostics, logDiags...)
	if hasError(logDiags) {
		return result
	}

	result.FontsUsed = inspectFonts(logger, latexLogPath)
	result.Engine = inspectEngine(logger, latexLogPath)
	result.Diagnostics = append(result.Diagnostics, synthesisedFontDiagnostics(result.FontsUsed, latexLogPath)...)

	// 12. Atomic publish to the configured project-relative output path.
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

func nodepaperVersion() string {
	if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "" {
		return info.Main.Version
	}
	return "devel"
}

// DefaultBuildScriptPath and DefaultProfileDir expose the resource lookup the
// build uses. `nodepaper export` has to reach the same Build-Paper.ps1 and the
// same installed Profile as `nodepaper build`; going through these keeps a
// single definition of "which script" and "which Profile" instead of a second
// copy that could drift.
func DefaultBuildScriptPath() string { return defaultBuildScriptPath() }

// DefaultProfileDir returns the installed CUMCM Profile directory.
func DefaultProfileDir() string { return defaultProfileDir() }

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

func defaultProfileDir() string {
	if override := os.Getenv("NODEPAPER_PROFILE_DIR"); override != "" {
		return override
	}
	executable, err := os.Executable()
	if err != nil {
		return filepath.Join("profiles", "cumcm")
	}
	return filepath.Join(filepath.Dir(executable), "profiles", "cumcm")
}

func runPowerShellBuild(ctx context.Context, executor commandExecutor, logger *buildLogger, bctx *buildctx.Context, scriptPath, profileDir, sourceManifestPath, texPath string) []diagnostic.Diagnostic {
	powerShellLogDir := filepath.Join(filepath.Dir(bctx.LogPath), bctx.BuildID+"-powershell")
	args := []string{
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-File", scriptPath,
		"-SourceManifest", sourceManifestPath,
		"-ProjectRoot", bctx.ProjectRoot,
		"-ProfileDirectory", profileDir,
		"-Output", texPath,
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

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat PDF: %w", err)
	}
	const maxElectronicPaperSize = 20 * 1024 * 1024
	if info.Size() == 0 {
		return fmt.Errorf("PDF is empty")
	}
	if info.Size() > maxElectronicPaperSize {
		return fmt.Errorf("PDF size %d exceeds the 20 MB CUMCM electronic-paper limit", info.Size())
	}

	header := make([]byte, 5)
	if _, err := file.Read(header); err != nil {
		return fmt.Errorf("read PDF header: %w", err)
	}
	if string(header) != "%PDF-" {
		return fmt.Errorf("missing %%PDF- header")
	}

	tailSize := int64(1024)
	if info.Size() < tailSize {
		tailSize = info.Size()
	}
	tail := make([]byte, tailSize)
	if _, err := file.ReadAt(tail, info.Size()-tailSize); err != nil {
		return fmt.Errorf("read PDF trailer: %w", err)
	}
	if !strings.Contains(string(tail), "%%EOF") {
		return fmt.Errorf("missing PDF EOF marker")
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

// hasError reports whether any diagnostic blocks publication.
func hasError(diags []diagnostic.Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == diagnostic.SeverityError {
			return true
		}
	}
	return false
}

// inspectFonts records which font families the document actually resolved to.
// Two machines that disagree about a PDF can be compared on this instead of on
// guesswork; a missing log is not a build failure, only a missing record.
func inspectFonts(logger *buildLogger, path string) []latexlog.FontFamily {
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Printf("Font record skipped: %v", err)
		return nil
	}
	families := latexlog.Fonts(data)
	for _, family := range families {
		logger.Printf("Font family: %s -> %s [%s]", family.Family, family.Font, family.Options)
	}
	return families
}

func inspectEngine(logger *buildLogger, path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Printf("Engine record skipped: %v", err)
		return ""
	}
	engine := latexlog.Engine(data)
	if engine == "" {
		logger.Printf("Engine record skipped: the LaTeX log does not name an engine")
		return ""
	}
	logger.Printf("Engine: %s", engine)
	return engine
}

// synthesisedFontDiagnostics reports weights that fontspec had to fake. This is
// a warning, not an error: the PDF is correct and complete, and the author
// usually cannot do anything about it - the fonts are an optional Windows
// feature. Blocking the build would leave them with no paper at all, which is
// strictly worse than a paper whose emphasis is slightly softer.
func synthesisedFontDiagnostics(families []latexlog.FontFamily, path string) []diagnostic.Diagnostic {
	var synthesised []string
	for _, family := range families {
		if strings.Contains(family.Options, "AutoFakeBold") {
			synthesised = append(synthesised, family.Font)
		}
	}
	if len(synthesised) == 0 {
		return nil
	}
	return []diagnostic.Diagnostic{{
		Severity: diagnostic.SeverityWarning,
		Code:     "NP6107",
		Message: fmt.Sprintf("bold Chinese text was synthesised from %s because SimHei is not installed",
			strings.Join(dedupe(synthesised), ", ")),
		File: path,
		Suggestion: "The PDF was published and no characters were lost. To get the real bold face, " +
			"add \"Chinese (Simplified) Supplemental Fonts\" under Settings > System > Optional features.",
		Source: "latex",
	}}
}

func dedupe(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}

func inspectLatexLog(logger *buildLogger, path string, allowlist latexlog.Allowlist) []diagnostic.Diagnostic {
	data, err := os.ReadFile(path)
	if err != nil {
		return []diagnostic.Diagnostic{{
			Severity:   diagnostic.SeverityError,
			Code:       "NP6199",
			Message:    fmt.Sprintf("cannot read final LaTeX log: %v", err),
			File:       path,
			Suggestion: "Inspect the PowerShell transition log and ensure XeLaTeX produced paper.log.",
			Source:     "latex",
		}}
	}
	codes := map[latexlog.Category]string{
		latexlog.CategoryFatal:          "NP6106",
		latexlog.CategoryOverflow:       "NP6101",
		latexlog.CategoryMissingFont:    "NP6102",
		latexlog.CategoryUnresolved:     "NP6103",
		latexlog.CategoryRerun:          "NP6104",
		latexlog.CategoryUnknownWarning: "NP6105",
	}
	var diags []diagnostic.Diagnostic
	for _, finding := range latexlog.Classify(data, allowlist) {
		logger.Printf("LaTeX Log: category=%s line=%d text=%s", finding.Category, finding.Line, finding.Text)
		if finding.Category == latexlog.CategoryAllowedWarning {
			logger.Printf("Allowed LaTeX Warning: reason=%s", finding.Reason)
			continue
		}
		severity := diagnostic.SeverityError
		suggestion := "Fix the LaTeX source or Profile; do not allowlist a warning without maintainer review."
		if finding.Category == latexlog.CategoryOverflow {
			severity = diagnostic.SeverityWarning
			suggestion = "Content overflows the text area. Split a long display formula across lines (aligned/split), break a long unbreakable string, or narrow an oversized table or image. The PDF was still published."
		}
		diags = append(diags, diagnostic.Diagnostic{
			Severity:   severity,
			Code:       codes[finding.Category],
			Message:    fmt.Sprintf("%s: %s", finding.Category, finding.Text),
			File:       path,
			Line:       finding.Line,
			Suggestion: suggestion,
			Source:     "latex",
		})
	}
	return diags
}

func fragmentDiagnostics(issues []fragment.Issue) []diagnostic.Diagnostic {
	diags := make([]diagnostic.Diagnostic, 0, len(issues))
	for _, issue := range issues {
		diags = append(diags, diagnostic.Diagnostic{
			Severity:   diagnostic.SeverityError,
			Code:       issue.Code,
			Message:    issue.Message,
			File:       issue.Path,
			Line:       issue.Line,
			Suggestion: "Keep declared LaTeX Fragments unchanged, UTF-8, and inside the Project Root during a build.",
			Source:     "build",
		})
	}
	return diags
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
