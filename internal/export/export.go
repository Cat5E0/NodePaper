// Package export produces a standalone, editable LaTeX project from a
// NodePaper project.
//
// The exported project is a one-way copy: it is generated from the Markdown
// sources and never read back. Nothing here changes the `nodepaper build`
// route - export re-runs the same conversion script with -SkipPdf and, for the
// BibTeX and biblatex modes, a different -CiteMethod, then copies the produced
// .tex together with the resources it references into the target directory or
// an Overleaf-ready ZIP archive selected by the destination suffix.
package export

import (
	"archive/zip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"nodepaper/internal/build"
	"nodepaper/internal/buildctx"
	"nodepaper/internal/buildlock"
	"nodepaper/internal/config"
	"nodepaper/internal/diagnostic"
	"nodepaper/internal/fragment"
	"nodepaper/internal/process"
	"nodepaper/internal/profile"
	"nodepaper/internal/project"
	"nodepaper/internal/validate"
)

// Diagnostic codes owned by export. NP8xxx is reserved for this command.
const (
	// CodeTargetNotEmpty - the target directory is non-empty or the target
	// archive already exists and --force was not given.
	CodeTargetNotEmpty = "NP8001"
	// CodeTargetUnusable - the target path cannot be used as an export
	// destination (missing --to, wrong kind, unreadable, uncreatable).
	CodeTargetUnusable = "NP8002"
	// CodeTargetInsideProject - the target directory lies inside the Project
	// Root. Exporting still proceeds; this is a warning.
	CodeTargetInsideProject = "NP8003"
	// CodeValidationFailed - validation failed, so nothing was exported.
	CodeValidationFailed = "NP8004"
	// CodeConversionFailed - the PowerShell conversion did not succeed.
	CodeConversionFailed = "NP8005"
	// CodeTexMissing - the conversion succeeded but produced no paper.tex.
	CodeTexMissing = "NP8006"
	// CodeCopyFailed - an export artifact could not be written.
	CodeCopyFailed = "NP8007"
	// CodeResourceSkipped - a referenced resource could not be copied and was
	// left out of the export. Warning.
	CodeResourceSkipped = "NP8008"
	// CodeVerifySkipped - --verify was requested but a required LaTeX tool is
	// not available. Warning; the export itself is unaffected.
	CodeVerifySkipped = "NP8009"
	// CodeVerifyFailed - --verify ran the compile chain and it failed.
	CodeVerifyFailed = "NP8010"
	// CodeVerifyWorkspace - --verify could not prepare or clean its temporary
	// directory. Warning.
	CodeVerifyWorkspace = "NP8011"
)

const source = "export"

// BibMode selects how the exported project carries its bibliography.
type BibMode string

const (
	// BibBibTeX exports \cite commands plus references.bib, compiled by
	// bibtex with the gbt7714 style. This is the default: bibtex exists in
	// every TeX distribution and the resulting citation marks match the PDF
	// `nodepaper build` produces.
	BibBibTeX BibMode = "bibtex"
	// BibBibLaTeX exports \autocite commands plus references.bib, compiled by
	// biber with biblatex-gb7714-2015.
	BibBibLaTeX BibMode = "biblatex"
	// BibInline renders the reference list into the .tex as plain text, so the
	// export needs neither a .bib file nor a bibliography processor.
	BibInline BibMode = "inline"
)

// ParseBibMode validates a --bib value.
func ParseBibMode(value string) (BibMode, error) {
	switch BibMode(value) {
	case BibBibTeX, BibBibLaTeX, BibInline:
		return BibMode(value), nil
	default:
		return "", fmt.Errorf("unsupported bibliography mode %q; use bibtex, biblatex or inline", value)
	}
}

// citeMethod maps a mode onto the -CiteMethod value Build-Paper.ps1 accepts.
// inline deliberately maps to the citeproc default so that route stays exactly
// the command line the main build has always used.
func (m BibMode) citeMethod() string {
	switch m {
	case BibBibTeX:
		return "natbib"
	case BibBibLaTeX:
		return "biblatex"
	default:
		return "citeproc"
	}
}

// needsBibFile reports whether the exported project has to ship references.bib.
func (m BibMode) needsBibFile() bool { return m != BibInline }

// Options describes one export request.
type Options struct {
	ProjectDir string
	// ToPath is the export destination. A case-insensitive .zip suffix selects
	// an Overleaf-ready archive; every other path selects a directory.
	ToPath string
	Bib    BibMode
	// Verify compiles the exported project in a temporary directory and
	// reports the outcome. No intermediate file is left in ToPath.
	Verify bool
	// Force allows writing into a non-empty directory or replacing an archive.
	Force bool
}

// Artifact is a file written into the exported project.
type Artifact struct {
	Kind string `json:"kind"`
	Path string `json:"path"`
}

// Result holds the outcome of an export.
type Result struct {
	Success     bool
	ProjectRoot string
	ExportPath  string
	ExportDir   string
	Zipped      bool
	BibMode     string
	// Verified is true only when --verify actually ran the compile chain and
	// it succeeded.
	Verified bool
	// CompileCommands is the chain the recipient has to run, the same list
	// README.txt prints and --verify executes.
	CompileCommands []string
	Artifacts       []Artifact
	Diagnostics     []diagnostic.Diagnostic
}

type commandExecutor interface {
	Run(ctx context.Context, dir, command string, args ...string) (process.Result, error)
}

type processExecutor struct{}

func (processExecutor) Run(ctx context.Context, dir, command string, args ...string) (process.Result, error) {
	return (&process.Runner{Dir: dir}).Run(ctx, command, args...)
}

// lookPath is a testable indirection for locating the LaTeX tools --verify
// needs. Nothing else in export resolves executables.
var lookPath = exec.LookPath

// Run exports the project at opts.ProjectDir into opts.ToPath.
func Run(ctx context.Context, opts Options) Result {
	return runWithExecutorAndResources(ctx, opts, processExecutor{}, build.DefaultBuildScriptPath(), build.DefaultProfileDir())
}

func runWithExecutorAndResources(ctx context.Context, opts Options, executor commandExecutor, scriptPath, profileDir string) Result {
	mode := opts.Bib
	if mode == "" {
		mode = BibBibTeX
	}
	result := Result{BibMode: string(mode), CompileCommands: CompileCommands(mode)}

	// 1. Discover the project, read its configuration and load the Profile,
	// exactly as a build does, before touching anything on disk.
	p, err := project.Discover(opts.ProjectDir)
	if err != nil {
		if de, ok := err.(*project.DiscoveryError); ok {
			result.Diagnostics = append(result.Diagnostics, de.Diagnostic)
		} else {
			result.Diagnostics = append(result.Diagnostics, errorDiag("NP1001", fmt.Sprintf("cannot discover project: %v", err), ""))
		}
		return result
	}
	result.ProjectRoot = p.Root

	cfg, err := config.Load(p.ConfigPath)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Severity:   diagnostic.SeverityError,
			Code:       configCode(err),
			Message:    fmt.Sprintf("invalid configuration: %v", err),
			File:       "nodepaper.yaml",
			Suggestion: "Fix nodepaper.yaml and try again.",
			Source:     source,
		})
		return result
	}

	loadedProfile, err := profile.Load(profileDir)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, errorDiag("NP1601",
			fmt.Sprintf("cannot load CUMCM Profile: %v", err),
			"Reinstall NodePaper or restore the profiles/cumcm resources."))
		return result
	}

	// 2. Validation gates the export. An invalid project would export a
	// LaTeX tree with the same defect and hand the recipient a broken build.
	vr := validate.Run(ctx, opts.ProjectDir)
	if !vr.Success {
		result.Diagnostics = append(result.Diagnostics, vr.Diagnostics...)
		result.Diagnostics = append(result.Diagnostics, errorDiag(CodeValidationFailed,
			"project validation failed; nothing was exported",
			"Fix the reported problems, then run 'nodepaper export' again."))
		return result
	}

	// 3. Resolve and inspect the destination before taking the build lock or
	// clearing export's private staging directory.
	target, diags := resolveTarget(p.Root, opts)
	result.Diagnostics = append(result.Diagnostics, diags...)
	if hasError(result.Diagnostics) {
		return result
	}
	result.ExportPath = target
	zipTarget := isZipTarget(target)
	result.Zipped = zipTarget
	if !zipTarget {
		result.ExportDir = target
	}

	// 4. Take the project build lock so an export and a build cannot write
	// .nodepaper/ at the same time.
	bctx, err := buildctx.New(p.Root, cfg.SourceFiles(), cfg.Profile)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, errorDiag("NP1301", fmt.Sprintf("cannot create build context: %v", err), ""))
		return result
	}
	held, err := buildlock.Acquire(p.Root, bctx.BuildID)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, lockDiag(err))
		return result
	}
	defer held.Release()

	// 5. Export runs in its own working directory so a previous build's
	// intermediates under .nodepaper/build stay untouched. It is removed on
	// the way out, while the lock is still held.
	workDir := filepath.Join(p.Root, ".nodepaper", "export")
	if err := os.RemoveAll(workDir); err != nil {
		result.Diagnostics = append(result.Diagnostics, errorDiag("NP1305", fmt.Sprintf("cannot remove previous export intermediates: %v", err), ""))
		return result
	}
	if err := os.MkdirAll(workDir, 0o755); err != nil {
		result.Diagnostics = append(result.Diagnostics, errorDiag("NP1305", fmt.Sprintf("cannot create export directory: %v", err), ""))
		return result
	}
	defer os.RemoveAll(workDir)

	logger, err := newLogger(bctx.LogPath)
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, errorDiag("NP1304", fmt.Sprintf("cannot create export log: %v", err), ""))
		return result
	}
	defer func() {
		logger.Printf("Success: %t", result.Success)
		_ = logger.Close()
	}()
	logger.Printf("Export ID: %s", bctx.BuildID)
	logger.Printf("Project Root: %s", p.Root)
	logger.Printf("Export Destination: %s", target)
	logger.Printf("Bibliography Mode: %s (-CiteMethod %s)", mode, mode.citeMethod())
	logger.Printf("Profile Version: %s", loadedProfile.Definition.Version)
	logger.Printf("Profile SHA-256: %s", loadedProfile.SHA256)

	fragmentFiles, fragmentIssues := fragment.Inspect(p.Root, cfg.LatexFragments)
	if len(fragmentIssues) > 0 {
		for _, issue := range fragmentIssues {
			result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
				Severity:   diagnostic.SeverityError,
				Code:       issue.Code,
				Message:    issue.Message,
				File:       issue.Path,
				Line:       issue.Line,
				Suggestion: "Keep declared LaTeX Fragments unchanged, UTF-8, and inside the Project Root.",
				Source:     source,
			})
		}
		return result
	}

	// 6. Write the source manifest the conversion script consumes, in the
	// same shape the build writes it.
	manifestPath := filepath.Join(workDir, "sources.json")
	if err := writeSourceManifest(manifestPath, p.Root, cfg, fragmentFiles); err != nil {
		result.Diagnostics = append(result.Diagnostics, errorDiag("NP1306", fmt.Sprintf("cannot write source manifest: %v", err), ""))
		return result
	}

	// 7. Convert. -SkipPdf stops the transition script right after Pandoc, so
	// no XeLaTeX runs unless the user asked for --verify.
	texPath := filepath.Join(workDir, "paper.tex")
	if diags := runConversion(ctx, executor, logger, conversionRequest{
		scriptPath:   scriptPath,
		profileDir:   loadedProfile.Dir,
		projectRoot:  p.Root,
		manifestPath: manifestPath,
		texPath:      texPath,
		workDir:      workDir,
		logDir:       filepath.Join(bctx.LogDir, bctx.BuildID+"-export-powershell"),
		citeMethod:   mode.citeMethod(),
	}); len(diags) > 0 {
		result.Diagnostics = append(result.Diagnostics, diags...)
		return result
	}
	if info, err := os.Stat(texPath); err != nil || !info.Mode().IsRegular() {
		result.Diagnostics = append(result.Diagnostics, errorDiag(CodeTexMissing,
			"the conversion reported success but produced no paper.tex",
			"Inspect the NodePaper export log and the PowerShell transition log."))
		return result
	}

	// 8. Assemble the deliverable. ZIP exports are assembled in the private
	// work directory first, so --verify sees the exact tree that is archived.
	deliverableDir := target
	if zipTarget {
		deliverableDir = filepath.Join(workDir, "deliverable")
	}
	artifacts, copyDiags := assemble(p, cfg, mode, texPath, deliverableDir, fragmentFiles)
	result.Artifacts = artifacts
	result.Diagnostics = append(result.Diagnostics, copyDiags...)
	if hasError(result.Diagnostics) {
		return result
	}

	// 9. Optional verification, always in a scratch directory so the
	// delivered project keeps no .aux, .log, .bbl or .pdf behind.
	if opts.Verify {
		verified, verifyDiags := verify(ctx, executor, logger, deliverableDir, mode)
		result.Verified = verified
		result.Diagnostics = append(result.Diagnostics, verifyDiags...)
		if hasError(result.Diagnostics) {
			return result
		}
	}

	if zipTarget {
		if err := writeZip(deliverableDir, target); err != nil {
			result.Diagnostics = append(result.Diagnostics, errorDiag(CodeCopyFailed,
				fmt.Sprintf("cannot write export archive %s: %v", target, err),
				"Check the destination path, free space and permissions."))
			return result
		}
		result.Artifacts = []Artifact{{Kind: "zip", Path: target}}
	}

	result.Success = true
	return result
}

// ---------- target destination ------------------------------------------

func resolveTarget(projectRoot string, opts Options) (string, []diagnostic.Diagnostic) {
	if strings.TrimSpace(opts.ToPath) == "" {
		return "", []diagnostic.Diagnostic{errorDiag(CodeTargetUnusable,
			"no export destination was given",
			"Pass a destination: nodepaper export --to <directory-or-zip>")}
	}
	target, err := filepath.Abs(opts.ToPath)
	if err != nil {
		return "", []diagnostic.Diagnostic{errorDiag(CodeTargetUnusable,
			fmt.Sprintf("cannot resolve export destination %q: %v", opts.ToPath, err),
			"Pass a writable destination path.")}
	}
	privateWorkDir := filepath.Join(projectRoot, ".nodepaper", "export")
	realPrivateWorkDir, err := canonicalPathWithMissingTail(privateWorkDir)
	if err != nil {
		return "", []diagnostic.Diagnostic{errorDiag(CodeTargetUnusable,
			fmt.Sprintf("cannot resolve NodePaper's private staging directory %s: %v", privateWorkDir, err),
			"Check that the Project Root and .nodepaper directory are accessible.")}
	}
	realTarget, err := canonicalPathWithMissingTail(target)
	if err != nil {
		return "", []diagnostic.Diagnostic{errorDiag(CodeTargetUnusable,
			fmt.Sprintf("cannot resolve export destination links for %s: %v", target, err),
			"Choose a writable destination whose parent directories are accessible.")}
	}
	if within(realPrivateWorkDir, realTarget) {
		return "", []diagnostic.Diagnostic{errorDiag(CodeTargetUnusable,
			fmt.Sprintf("export destination is inside NodePaper's private staging directory: %s", target),
			"Choose a destination outside .nodepaper/export; that directory is deleted after every export.")}
	}
	zipTarget := isZipTarget(target)

	var diags []diagnostic.Diagnostic
	info, err := os.Stat(target)
	switch {
	case zipTarget && err == nil && info.IsDir():
		return "", []diagnostic.Diagnostic{errorDiag(CodeTargetUnusable,
			fmt.Sprintf("ZIP export destination is a directory: %s", target),
			"Pass a file path such as paper.zip.")}
	case zipTarget && err == nil && !opts.Force:
		return "", []diagnostic.Diagnostic{errorDiag(CodeTargetNotEmpty,
			fmt.Sprintf("export archive already exists: %s", target),
			"Choose another path, or re-run with --force to replace this archive.")}
	case zipTarget && err == nil:
		// --force permits replacing this exact file after the new archive has
		// been written successfully to a sibling temporary file.
	case !zipTarget && err == nil && !info.IsDir():
		return "", []diagnostic.Diagnostic{errorDiag(CodeTargetUnusable,
			fmt.Sprintf("export destination is not a directory: %s", target),
			"Pass a directory, not a file.")}
	case !zipTarget && err == nil:
		entries, readErr := os.ReadDir(target)
		if readErr != nil {
			return "", []diagnostic.Diagnostic{errorDiag(CodeTargetUnusable,
				fmt.Sprintf("cannot read export directory %s: %v", target, readErr),
				"Check that the directory is accessible.")}
		}
		// Only the count matters. NodePaper does not inventory the directory
		// and never refuses to export because of files it did not expect.
		if len(entries) > 0 && !opts.Force {
			return "", []diagnostic.Diagnostic{errorDiag(CodeTargetNotEmpty,
				fmt.Sprintf("export directory is not empty: %s (%d entries)", target, len(entries)),
				"Pass an empty directory, or re-run with --force to write into this one.")}
		}
	case !os.IsNotExist(err):
		return "", []diagnostic.Diagnostic{errorDiag(CodeTargetUnusable,
			fmt.Sprintf("cannot inspect export directory %s: %v", target, err),
			"Check that the path is accessible.")}
	}

	// Exporting inside the project works, but the copy then looks like part of
	// the Markdown project while being a dead-end snapshot of it, and it will
	// be picked up by the project's own version control. Worth saying once.
	realProjectRoot, err := canonicalPathWithMissingTail(projectRoot)
	if err != nil {
		return "", []diagnostic.Diagnostic{errorDiag(CodeTargetUnusable,
			fmt.Sprintf("cannot resolve Project Root links for %s: %v", projectRoot, err),
			"Check that the Project Root is accessible.")}
	}
	if within(realProjectRoot, realTarget) {
		diags = append(diags, diagnostic.Diagnostic{
			Severity: diagnostic.SeverityWarning,
			Code:     CodeTargetInsideProject,
			Message:  fmt.Sprintf("the export destination is inside the Project Root: %s", target),
			Suggestion: "Exporting continues. The exported LaTeX project is a one-way copy that never " +
				"flows back into the Markdown project; keeping it outside the project avoids committing " +
				"it by accident.",
			Source: source,
		})
	}
	return target, diags
}

func isZipTarget(path string) bool {
	return strings.EqualFold(filepath.Ext(path), ".zip")
}

// writeZip creates a directly uploadable Overleaf archive: paper.tex and its
// resources are at the ZIP root, with no enclosing directory. A fixed
// timestamp and lexical traversal keep identical exports reproducible.
func writeZip(sourceDir, target string) (returnErr error) {
	parent := filepath.Dir(target)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		return err
	}
	temporary, err := os.CreateTemp(parent, ".nodepaper-export-*.zip")
	if err != nil {
		return err
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = temporary.Close()
		if returnErr != nil {
			_ = os.Remove(temporaryPath)
		}
	}()

	archive := zip.NewWriter(temporary)
	fixedTime := time.Date(2000, time.January, 1, 0, 0, 0, 0, time.UTC)
	err = filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		relative, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		header := &zip.FileHeader{Name: filepath.ToSlash(relative), Method: zip.Deflate}
		header.SetModTime(fixedTime)
		writer, err := archive.CreateHeader(header)
		if err != nil {
			return err
		}
		file, err := os.Open(path)
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(writer, file)
		closeErr := file.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
	if err != nil {
		_ = archive.Close()
		return err
	}
	if err := archive.Close(); err != nil {
		return err
	}
	if err := temporary.Sync(); err != nil {
		return err
	}
	if err := temporary.Close(); err != nil {
		return err
	}
	if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Rename(temporaryPath, target); err != nil {
		return err
	}
	return nil
}

func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// canonicalPathWithMissingTail resolves every symlink or Windows junction in
// the nearest existing ancestor, then reattaches any path elements that do not
// exist yet. filepath.EvalSymlinks alone cannot inspect a new export target;
// resolving its ancestor prevents an alias from hiding that the target really
// lies inside .nodepaper/export.
func canonicalPathWithMissingTail(path string) (string, error) {
	current := filepath.Clean(path)
	var missing []string
	for {
		_, err := os.Lstat(current)
		switch {
		case err == nil:
			real, evalErr := canonicalExistingPath(current)
			if evalErr != nil {
				return "", evalErr
			}
			for i := len(missing) - 1; i >= 0; i-- {
				real = filepath.Join(real, missing[i])
			}
			return filepath.Clean(real), nil
		case !os.IsNotExist(err):
			return "", err
		}

		parent := filepath.Dir(current)
		if parent == current {
			return "", err
		}
		missing = append(missing, filepath.Base(current))
		current = parent
	}
}

// ---------- conversion ---------------------------------------------------

type conversionRequest struct {
	scriptPath   string
	profileDir   string
	projectRoot  string
	manifestPath string
	texPath      string
	workDir      string
	logDir       string
	citeMethod   string
}

// runConversion drives Build-Paper.ps1. It is deliberately separate from the
// build's own invocation: the flags differ (-SkipPdf, -CiteMethod) and sharing
// one call site would make it possible to change the build's command line by
// editing the export.
func runConversion(ctx context.Context, executor commandExecutor, logger *logWriter, req conversionRequest) []diagnostic.Diagnostic {
	args := []string{
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-File", req.scriptPath,
		"-SourceManifest", req.manifestPath,
		"-ProjectRoot", req.projectRoot,
		"-ProfileDirectory", req.profileDir,
		"-Output", req.texPath,
		"-BuildDirectory", req.workDir,
		"-LogDirectory", req.logDir,
		"-CiteMethod", req.citeMethod,
		// The export is compiled somewhere NodePaper cannot see. -ExportMode
		// makes the Profile emit a preamble that picks its Chinese fonts with
		// \IfFontExistsTF at compile time and sets \tracinglostchars=3, so the
		// same .tex produces the right glyphs on a Windows machine, on a Linux
		// machine and on Overleaf, and refuses to hide the ones it cannot.
		// `nodepaper build` never passes this: its .tex is compiled here.
		"-ExportMode",
		"-SkipPdf",
	}
	logger.Printf("Command: %s", process.LogFriendlyCommand("powershell.exe", args))
	processResult, err := executor.Run(ctx, req.projectRoot, "powershell.exe", args...)
	logger.Printf("Exit Code: %d", processResult.ExitCode)
	if processResult.Stdout != "" {
		logger.Printf("stdout:\n%s", processResult.Stdout)
	}
	if processResult.Stderr != "" {
		logger.Printf("stderr:\n%s", processResult.Stderr)
	}
	if err == nil && processResult.ExitCode == 0 {
		return nil
	}
	message := fmt.Sprintf("LaTeX conversion exited with code %d", processResult.ExitCode)
	if err != nil {
		message = fmt.Sprintf("LaTeX conversion failed: %v", err)
	}
	return []diagnostic.Diagnostic{errorDiag(CodeConversionFailed, message,
		"Inspect the NodePaper export log and the PowerShell transition log.")}
}

func writeSourceManifest(path, projectRoot string, cfg config.ProjectConfig, fragments []fragment.File) error {
	absoluteSources := make([]string, 0, len(cfg.SourceFiles()))
	for _, relative := range cfg.SourceFiles() {
		absoluteSources = append(absoluteSources, filepath.Join(projectRoot, relative))
	}
	absoluteFragments := make([]string, 0, len(fragments))
	for _, file := range fragments {
		absoluteFragments = append(absoluteFragments, file.Path)
	}
	data, err := json.MarshalIndent(struct {
		Sources              []string `json:"sources"`
		LatexFragments       []string `json:"latexFragments"`
		AppendixNumbering    string   `json:"appendixNumbering"`
		HighlightStyle       string   `json:"highlightStyle"`
		LineSpread           float64  `json:"linespread"`
		AbstractLineSpread   float64  `json:"abstractLinespread"`
		TitleAbstractSkip    float64  `json:"titleAbstractSkip"`
		AbstractKeywordsSkip float64  `json:"abstractKeywordsSkip"`
		MathFont             string   `json:"mathFont"`
		AppendixNewPage      bool     `json:"appendixNewPage"`
	}{
		Sources:              absoluteSources,
		LatexFragments:       absoluteFragments,
		AppendixNumbering:    cfg.Appendix.Numbering,
		HighlightStyle:       cfg.Highlight.Style,
		LineSpread:           cfg.LineSpread,
		AbstractLineSpread:   cfg.AbstractLineSpread,
		TitleAbstractSkip:    cfg.TitleAbstractSkipEm(),
		AbstractKeywordsSkip: cfg.AbstractKeywordsSkipEm(),
		MathFont:             cfg.MathFont,
		AppendixNewPage:      cfg.Appendix.NewPageEnabled(),
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}

// ---------- assembling the deliverable -----------------------------------

// assemble writes paper.tex, the bibliography, every referenced image, the
// declared LaTeX Fragments and README.txt into the target directory.
func assemble(p project.Project, cfg config.ProjectConfig, mode BibMode, texPath, target string, fragments []fragment.File) ([]Artifact, []diagnostic.Diagnostic) {
	var artifacts []Artifact
	var diags []diagnostic.Diagnostic

	if err := os.MkdirAll(target, 0o755); err != nil {
		return nil, []diagnostic.Diagnostic{errorDiag(CodeCopyFailed,
			fmt.Sprintf("cannot create export directory %s: %v", target, err),
			"Check the path and the available permissions.")}
	}

	place := func(kind, sourcePath, relative string) bool {
		destination := filepath.Join(target, filepath.FromSlash(relative))
		if !within(target, destination) {
			diags = append(diags, errorDiag(CodeCopyFailed,
				fmt.Sprintf("refusing to write outside the export directory: %s", relative), ""))
			return false
		}
		if err := copyFile(sourcePath, destination); err != nil {
			diags = append(diags, errorDiag(CodeCopyFailed,
				fmt.Sprintf("cannot write %s: %v", relative, err),
				"Check the available disk space and permissions."))
			return false
		}
		artifacts = append(artifacts, Artifact{Kind: kind, Path: destination})
		return true
	}

	place("tex", texPath, "paper.tex")

	if mode.needsBibFile() {
		if bibPath, err := p.Resolve("references.bib"); err == nil {
			place("bibliography", bibPath, "references.bib")
		}
	}

	for _, relative := range referencedImages(p, cfg) {
		resolved, err := p.Resolve(filepath.FromSlash(relative))
		if err != nil {
			diags = append(diags, warningDiag(CodeResourceSkipped,
				fmt.Sprintf("image path outside the project was not exported: %s", relative),
				"Move the image inside the project and export again."))
			continue
		}
		if info, statErr := os.Stat(resolved); statErr != nil || !info.Mode().IsRegular() {
			diags = append(diags, warningDiag(CodeResourceSkipped,
				fmt.Sprintf("referenced image could not be exported: %s", relative),
				"Check that the image exists, then export again."))
			continue
		}
		place("image", resolved, relative)
	}

	for _, file := range fragments {
		place("fragment", file.Path, file.Relative)
	}

	readmePath := filepath.Join(target, "README.txt")
	if err := os.WriteFile(readmePath, []byte(readme(mode)), 0o644); err != nil {
		diags = append(diags, errorDiag(CodeCopyFailed,
			fmt.Sprintf("cannot write README.txt: %v", err),
			"Check the available disk space and permissions."))
	} else {
		artifacts = append(artifacts, Artifact{Kind: "readme", Path: readmePath})
	}

	return artifacts, diags
}

// referencedImages returns the deduplicated, ordered set of image paths the
// Markdown sources use. Images that are present in the project but never
// referenced are deliberately left out: the export is the document, not the
// working directory it grew in.
func referencedImages(p project.Project, cfg config.ProjectConfig) []string {
	seen := make(map[string]struct{})
	var references []string
	for _, relative := range cfg.SourceFiles() {
		resolved, err := p.Resolve(relative)
		if err != nil {
			continue
		}
		data, err := os.ReadFile(resolved)
		if err != nil {
			continue
		}
		for _, reference := range validate.ImageReferences(data) {
			if _, ok := seen[reference]; ok {
				continue
			}
			seen[reference] = struct{}{}
			references = append(references, reference)
		}
	}
	return references
}

func copyFile(sourcePath, destination string) error {
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}
	return os.WriteFile(destination, data, 0o644)
}

func copyTree(sourceDir, destination string) error {
	return filepath.WalkDir(sourceDir, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		if relative == "." {
			return os.MkdirAll(destination, 0o755)
		}
		if entry.IsDir() {
			return os.MkdirAll(filepath.Join(destination, relative), 0o755)
		}
		if !entry.Type().IsRegular() {
			return nil
		}
		return copyFile(path, filepath.Join(destination, relative))
	})
}

// ---------- diagnostics helpers ------------------------------------------

func errorDiag(code, message, suggestion string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity:   diagnostic.SeverityError,
		Code:       code,
		Message:    message,
		Suggestion: suggestion,
		Source:     source,
	}
}

func warningDiag(code, message, suggestion string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity:   diagnostic.SeverityWarning,
		Code:       code,
		Message:    message,
		Suggestion: suggestion,
		Source:     source,
	}
}

func hasError(diags []diagnostic.Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == diagnostic.SeverityError {
			return true
		}
	}
	return false
}

func configCode(err error) string {
	if strings.Contains(err.Error(), "cannot parse YAML") {
		return "NP1501"
	}
	return "NP1502"
}

func lockDiag(err error) diagnostic.Diagnostic {
	if he, ok := err.(*buildlock.ErrHeld); ok {
		return errorDiag("NP1201", he.Error(), "Wait for the current build to finish.")
	}
	if se, ok := err.(*buildlock.ErrStale); ok {
		return errorDiag("NP1202", se.Error(), "Run 'nodepaper clean' to remove the stale lock.")
	}
	return errorDiag("NP1299", fmt.Sprintf("cannot acquire build lock: %v", err), "")
}

// ---------- log ----------------------------------------------------------

type logWriter struct {
	file *os.File
}

func newLogger(path string) (*logWriter, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, err
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o644)
	if err != nil {
		return nil, err
	}
	return &logWriter{file: file}, nil
}

func (l *logWriter) Printf(format string, args ...any) {
	if l == nil || l.file == nil {
		return
	}
	fmt.Fprintf(l.file, "%s ", time.Now().Format(time.RFC3339Nano))
	fmt.Fprintf(l.file, format, args...)
	if !strings.HasSuffix(format, "\n") {
		fmt.Fprintln(l.file)
	}
}

func (l *logWriter) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	err := l.file.Close()
	l.file = nil
	return err
}
