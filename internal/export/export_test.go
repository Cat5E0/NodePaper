package export

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"nodepaper/internal/buildlock"
	"nodepaper/internal/diagnostic"
	"nodepaper/internal/process"
)

// ---------- fakes --------------------------------------------------------

type fakeCall struct {
	dir     string
	command string
	args    []string
}

type fakeExecutor struct {
	mu       sync.Mutex
	calls    []fakeCall
	behavior func(dir, command string, args []string) (process.Result, error)
}

func (f *fakeExecutor) Run(_ context.Context, dir, command string, args ...string) (process.Result, error) {
	f.mu.Lock()
	f.calls = append(f.calls, fakeCall{dir: dir, command: command, args: append([]string(nil), args...)})
	f.mu.Unlock()
	if f.behavior != nil {
		return f.behavior(dir, command, args)
	}
	return fakeConversion(dir, command, args)
}

// fakeConversion stands in for Build-Paper.ps1 -SkipPdf: it writes the .tex
// the real script would have produced and nothing else.
func fakeConversion(dir, command string, args []string) (process.Result, error) {
	result := process.Result{Command: command, Args: args, Dir: dir}
	if command != "powershell.exe" {
		// The verification chain reaches this fake too; treat it as success.
		return result, nil
	}
	output := argumentValue(args, "-Output")
	if output == "" {
		return process.Result{ExitCode: 1}, errors.New("fake conversion received no -Output")
	}
	if err := os.MkdirAll(filepath.Dir(output), 0o755); err != nil {
		return process.Result{ExitCode: 1}, err
	}
	body := "\\documentclass{ctexart}\\begin{document}exported\\end{document}\n"
	return result, os.WriteFile(output, []byte(body), 0o644)
}

func argumentValue(args []string, name string) string {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == name {
			return args[i+1]
		}
	}
	return ""
}

func hasArgument(args []string, name string) bool {
	for _, arg := range args {
		if arg == name {
			return true
		}
	}
	return false
}

func hasCode(diags []diagnostic.Diagnostic, code string) bool {
	for _, d := range diags {
		if d.Code == code {
			return true
		}
	}
	return false
}

func fixtureProject(t *testing.T) string {
	t.Helper()
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("NODEPAPER_PROFILE_DIR", filepath.Join(repoRoot, "profiles", "cumcm"))
	source := filepath.Join(repoRoot, "tests", "fixtures", "complete-single-file")
	destination := filepath.Join(t.TempDir(), "project")
	if err := copyTree(source, destination); err != nil {
		t.Fatalf("copy fixture: %v", err)
	}
	return destination
}

func profileDir(t *testing.T) string {
	t.Helper()
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(repoRoot, "profiles", "cumcm")
}

func runExport(t *testing.T, opts Options, executor *fakeExecutor) Result {
	t.Helper()
	return runWithExecutorAndResources(context.Background(), opts, executor,
		filepath.Join(t.TempDir(), "Build-Paper.ps1"), profileDir(t))
}

// ---------- bibliography modes -------------------------------------------

func TestParseBibModeAcceptsOnlyTheThreeModes(t *testing.T) {
	for _, value := range []string{"bibtex", "biblatex", "inline"} {
		if _, err := ParseBibMode(value); err != nil {
			t.Errorf("ParseBibMode(%q) = %v, want nil", value, err)
		}
	}
	if _, err := ParseBibMode("natbib"); err == nil {
		t.Error("ParseBibMode(\"natbib\") accepted a -CiteMethod value as a --bib value")
	}
}

func TestBibModeMapsToCiteMethod(t *testing.T) {
	want := map[BibMode]string{
		BibBibTeX:   "natbib",
		BibBibLaTeX: "biblatex",
		BibInline:   "citeproc",
	}
	for mode, expected := range want {
		if got := mode.citeMethod(); got != expected {
			t.Errorf("%s.citeMethod() = %q, want %q", mode, got, expected)
		}
	}
}

// ---------- happy paths ---------------------------------------------------

func TestExportWritesTheDeliverable(t *testing.T) {
	projectDir := fixtureProject(t)
	target := filepath.Join(t.TempDir(), "latex")
	executor := &fakeExecutor{}

	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target}, executor)
	if !result.Success {
		t.Fatalf("export failed: %#v", result.Diagnostics)
	}
	if result.Zipped || result.ExportPath != target || result.ExportDir != target {
		t.Fatalf("directory result = %#v", result)
	}
	for _, relative := range []string{
		"paper.tex",
		"references.bib",
		"README.txt",
		filepath.Join("images", "demand-trend.png"),
		filepath.Join("images", "station-map.png"),
	} {
		if _, err := os.Stat(filepath.Join(target, relative)); err != nil {
			t.Errorf("missing exported file %s: %v", relative, err)
		}
	}
	if _, err := os.Stat(buildlock.LockPath(projectDir)); !os.IsNotExist(err) {
		t.Errorf("build lock remains after export: %v", err)
	}
	// The export working directory is transient; nothing may survive it, and
	// .nodepaper/build must be left exactly as the last build left it.
	if _, err := os.Stat(filepath.Join(projectDir, ".nodepaper", "export")); !os.IsNotExist(err) {
		t.Errorf("export intermediates remain: %v", err)
	}
}

func TestExportInfersAnOverleafReadyZipCaseInsensitively(t *testing.T) {
	projectDir := fixtureProject(t)
	target := filepath.Join(t.TempDir(), "latex-project.ZIP")

	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target}, &fakeExecutor{})
	if !result.Success {
		t.Fatalf("export failed: %#v", result.Diagnostics)
	}
	if result.ExportPath != target || result.ExportDir != "" || !result.Zipped {
		t.Fatalf("zip result = %#v", result)
	}
	reader, err := zip.OpenReader(target)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer reader.Close()
	want := map[string]bool{
		"paper.tex":               false,
		"references.bib":          false,
		"README.txt":              false,
		"images/demand-trend.png": false,
	}
	for _, file := range reader.File {
		if _, ok := want[file.Name]; ok {
			want[file.Name] = true
		}
		if strings.HasPrefix(file.Name, "latex-project/") {
			t.Errorf("zip entry has a wrapper directory: %s", file.Name)
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("zip is missing %s", name)
		}
	}
}

func TestZipExportIsDeterministic(t *testing.T) {
	projectDir := fixtureProject(t)
	first := filepath.Join(t.TempDir(), "first.zip")
	second := filepath.Join(t.TempDir(), "second.zip")

	for _, target := range []string{first, second} {
		result := runExport(t, Options{ProjectDir: projectDir, ToPath: target}, &fakeExecutor{})
		if !result.Success {
			t.Fatalf("export %s failed: %#v", target, result.Diagnostics)
		}
	}
	firstBytes, err := os.ReadFile(first)
	if err != nil {
		t.Fatal(err)
	}
	secondBytes, err := os.ReadFile(second)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(firstBytes, secondBytes) {
		t.Fatal("ZIP exports of the same project differ")
	}
}

func TestZipExportRefusesAnExistingFileWithoutForce(t *testing.T) {
	projectDir := fixtureProject(t)
	target := filepath.Join(t.TempDir(), "latex-project.zip")
	if err := os.WriteFile(target, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target}, &fakeExecutor{})
	if result.Success || !hasCode(result.Diagnostics, CodeTargetNotEmpty) {
		t.Fatalf("result = %#v, want %s", result, CodeTargetNotEmpty)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "keep" {
		t.Fatalf("existing archive changed: data=%q err=%v", data, err)
	}
}

func TestZipExportWithForceReplacesTheExistingFile(t *testing.T) {
	projectDir := fixtureProject(t)
	target := filepath.Join(t.TempDir(), "latex-project.zip")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target, Force: true}, &fakeExecutor{})
	if !result.Success {
		t.Fatalf("export failed: %#v", result.Diagnostics)
	}
	reader, err := zip.OpenReader(target)
	if err != nil {
		t.Fatalf("replacement is not a zip: %v", err)
	}
	if err := reader.Close(); err != nil {
		t.Fatalf("close replacement zip: %v", err)
	}
}

func TestZipExportRejectsADirectoryTargetEvenWithForce(t *testing.T) {
	projectDir := fixtureProject(t)
	target := filepath.Join(t.TempDir(), "latex-project.zip")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}

	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target, Force: true}, &fakeExecutor{})
	if result.Success || !hasCode(result.Diagnostics, CodeTargetUnusable) {
		t.Fatalf("result = %#v, want %s", result, CodeTargetUnusable)
	}
}

func TestExportPassesSkipPdfAndTheMappedCiteMethod(t *testing.T) {
	for mode, citeMethod := range map[BibMode]string{
		BibBibTeX:   "natbib",
		BibBibLaTeX: "biblatex",
		BibInline:   "citeproc",
	} {
		t.Run(string(mode), func(t *testing.T) {
			projectDir := fixtureProject(t)
			target := filepath.Join(t.TempDir(), "latex")
			executor := &fakeExecutor{}

			result := runExport(t, Options{ProjectDir: projectDir, ToPath: target, Bib: mode}, executor)
			if !result.Success {
				t.Fatalf("export failed: %#v", result.Diagnostics)
			}
			if len(executor.calls) != 1 {
				t.Fatalf("calls = %#v, want exactly one conversion", executor.calls)
			}
			args := executor.calls[0].args
			if !hasArgument(args, "-SkipPdf") {
				t.Errorf("-SkipPdf missing; export must not run XeLaTeX: %#v", args)
			}
			if got := argumentValue(args, "-CiteMethod"); got != citeMethod {
				t.Errorf("-CiteMethod = %q, want %q", got, citeMethod)
			}
			// Every --bib mode needs it, including inline: inline uses the same
			// template.tex the build uses, so without this flag that route
			// would ship the platform-guessing preamble again.
			if !hasArgument(args, "-ExportMode") {
				t.Errorf("-ExportMode missing; the export would ship a preamble that guesses fonts: %#v", args)
			}
			// inline renders the references into the .tex, so shipping a .bib
			// would be a file nothing reads.
			_, err := os.Stat(filepath.Join(target, "references.bib"))
			if mode == BibInline && !os.IsNotExist(err) {
				t.Errorf("inline export contains references.bib: %v", err)
			}
			if mode != BibInline && err != nil {
				t.Errorf("%s export has no references.bib: %v", mode, err)
			}
		})
	}
}

func TestExportCopiesOnlyReferencedImages(t *testing.T) {
	projectDir := fixtureProject(t)
	unused := filepath.Join(projectDir, "images", "never-referenced.png")
	if err := os.WriteFile(unused, []byte("not a real png"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "latex")

	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target}, &fakeExecutor{})
	if !result.Success {
		t.Fatalf("export failed: %#v", result.Diagnostics)
	}
	if _, err := os.Stat(filepath.Join(target, "images", "never-referenced.png")); !os.IsNotExist(err) {
		t.Errorf("unreferenced image was exported: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "images", "demand-trend.png")); err != nil {
		t.Errorf("referenced image was not exported: %v", err)
	}
}

func TestExportCopiesDeclaredFragments(t *testing.T) {
	projectDir := fixtureProject(t)
	if err := os.MkdirAll(filepath.Join(projectDir, "tables"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "tables", "summary.tex"), []byte("\\textbf{ok}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(projectDir, "nodepaper.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, append(data, []byte("\nlatexFragments:\n  - tables/summary.tex\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	// The fragment has to be referenced or validation rejects it as unused.
	paperPath := filepath.Join(projectDir, "paper.md")
	paper, err := os.ReadFile(paperPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(paperPath, append(paper, []byte("\n\\input{tables/summary.tex}\n")...), 0o644); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(t.TempDir(), "latex")
	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target}, &fakeExecutor{})
	if !result.Success {
		t.Fatalf("export failed: %#v", result.Diagnostics)
	}
	if _, err := os.Stat(filepath.Join(target, "tables", "summary.tex")); err != nil {
		t.Errorf("declared fragment was not exported: %v", err)
	}
}

// ---------- target directory ---------------------------------------------

func TestExportRefusesANonEmptyTargetWithoutForce(t *testing.T) {
	projectDir := fixtureProject(t)
	target := t.TempDir()
	if err := os.WriteFile(filepath.Join(target, "notes.txt"), []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target}, &fakeExecutor{})
	if result.Success || !hasCode(result.Diagnostics, CodeTargetNotEmpty) {
		t.Fatalf("result = %#v, want %s", result, CodeTargetNotEmpty)
	}
}

func TestExportWithForceKeepsUnrelatedFiles(t *testing.T) {
	projectDir := fixtureProject(t)
	target := t.TempDir()
	unrelated := filepath.Join(target, "notes.txt")
	if err := os.WriteFile(unrelated, []byte("mine"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target, Force: true}, &fakeExecutor{})
	if !result.Success {
		t.Fatalf("export failed: %#v", result.Diagnostics)
	}
	// Files NodePaper did not put there are none of its business: --force means
	// "write into this directory", never "clear it out".
	if _, err := os.Stat(unrelated); err != nil {
		t.Errorf("--force removed an unrelated file: %v", err)
	}
	if _, err := os.Stat(filepath.Join(target, "paper.tex")); err != nil {
		t.Errorf("paper.tex was not written: %v", err)
	}
}

func TestDirectoryExportRejectsAnExistingFile(t *testing.T) {
	projectDir := fixtureProject(t)
	target := filepath.Join(t.TempDir(), "latex-project")
	if err := os.WriteFile(target, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target, Force: true}, &fakeExecutor{})
	if result.Success || !hasCode(result.Diagnostics, CodeTargetUnusable) {
		t.Fatalf("result = %#v, want %s", result, CodeTargetUnusable)
	}
	data, err := os.ReadFile(target)
	if err != nil || string(data) != "keep" {
		t.Fatalf("existing file changed: data=%q err=%v", data, err)
	}
}

func TestExportWarnsButProceedsInsideTheProject(t *testing.T) {
	projectDir := fixtureProject(t)
	target := filepath.Join(projectDir, "latex-export")

	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target}, &fakeExecutor{})
	if !result.Success {
		t.Fatalf("export inside the project failed: %#v", result.Diagnostics)
	}
	if !hasCode(result.Diagnostics, CodeTargetInsideProject) {
		t.Fatalf("diagnostics = %#v, want %s warning", result.Diagnostics, CodeTargetInsideProject)
	}
	for _, d := range result.Diagnostics {
		if d.Code == CodeTargetInsideProject && d.Severity != diagnostic.SeverityWarning {
			t.Fatalf("%s severity = %q, want warning", CodeTargetInsideProject, d.Severity)
		}
	}
}

func TestZipExportWarnsButProceedsInsideTheProject(t *testing.T) {
	projectDir := fixtureProject(t)
	target := filepath.Join(projectDir, "latex-export.zip")

	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target}, &fakeExecutor{})
	if !result.Success || !result.Zipped {
		t.Fatalf("zip export inside the project failed: %#v", result)
	}
	if !hasCode(result.Diagnostics, CodeTargetInsideProject) {
		t.Fatalf("diagnostics = %#v, want %s warning", result.Diagnostics, CodeTargetInsideProject)
	}
}

func TestExportRejectsTargetsInsideItsPrivateStagingDirectory(t *testing.T) {
	for _, relative := range []string{
		filepath.Join(".nodepaper", "export", "latex-project"),
		filepath.Join(".nodepaper", "export", "latex-project.zip"),
	} {
		t.Run(filepath.Base(relative), func(t *testing.T) {
			projectDir := fixtureProject(t)
			target := filepath.Join(projectDir, relative)

			result := runExport(t, Options{ProjectDir: projectDir, ToPath: target}, &fakeExecutor{})
			if result.Success || !hasCode(result.Diagnostics, CodeTargetUnusable) {
				t.Fatalf("result = %#v, want %s", result, CodeTargetUnusable)
			}
			if _, err := os.Stat(target); !os.IsNotExist(err) {
				t.Fatalf("unsafe target was created: %v", err)
			}
		})
	}
}

func TestExportRejectsTargetsAliasedIntoItsPrivateStagingDirectory(t *testing.T) {
	projectDir := fixtureProject(t)
	privateWorkDir := filepath.Join(projectDir, ".nodepaper", "export")
	if err := os.MkdirAll(privateWorkDir, 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(projectDir, "staging-alias")
	if err := createDirectoryAlias(privateWorkDir, alias); err != nil {
		if runtime.GOOS == "windows" {
			t.Fatalf("cannot create a junction for the Windows path-safety test: %v", err)
		}
		t.Skipf("cannot create a directory symlink in this environment: %v", err)
	}
	defer os.Remove(alias)

	for _, name := range []string{"latex-project", "latex-project.zip"} {
		t.Run(name, func(t *testing.T) {
			target := filepath.Join(alias, name)
			result := runExport(t, Options{ProjectDir: projectDir, ToPath: target, Force: true}, &fakeExecutor{})
			if result.Success || !hasCode(result.Diagnostics, CodeTargetUnusable) {
				t.Fatalf("result = %#v, want %s", result, CodeTargetUnusable)
			}
			if _, err := os.Stat(filepath.Join(privateWorkDir, name)); !os.IsNotExist(err) {
				t.Fatalf("unsafe target was created through alias: %v", err)
			}
		})
	}
}

func createDirectoryAlias(target, alias string) error {
	if runtime.GOOS != "windows" {
		return os.Symlink(target, alias)
	}
	output, err := osexec.Command("cmd.exe", "/d", "/c", "mklink", "/J", alias, target).CombinedOutput()
	if err != nil {
		return fmt.Errorf("mklink /J failed: %w: %s", err, strings.TrimSpace(string(output)))
	}
	return nil
}

func TestExportRejectsAMissingDestination(t *testing.T) {
	projectDir := fixtureProject(t)
	result := runExport(t, Options{ProjectDir: projectDir}, &fakeExecutor{})
	if result.Success || !hasCode(result.Diagnostics, CodeTargetUnusable) {
		t.Fatalf("result = %#v, want %s", result, CodeTargetUnusable)
	}
}

// ---------- failures ------------------------------------------------------

func TestExportStopsOnValidationFailure(t *testing.T) {
	projectDir := fixtureProject(t)
	if err := os.Remove(filepath.Join(projectDir, "references.bib")); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "latex")
	executor := &fakeExecutor{}

	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target}, executor)
	if result.Success || !hasCode(result.Diagnostics, CodeValidationFailed) {
		t.Fatalf("result = %#v, want %s", result, CodeValidationFailed)
	}
	if len(executor.calls) != 0 {
		t.Errorf("conversion ran despite invalid project: %#v", executor.calls)
	}
	if _, err := os.Stat(target); !os.IsNotExist(err) {
		t.Errorf("export directory was created for a failed export: %v", err)
	}
}

func TestExportReportsConversionFailure(t *testing.T) {
	projectDir := fixtureProject(t)
	target := filepath.Join(t.TempDir(), "latex")
	executor := &fakeExecutor{behavior: func(dir, command string, args []string) (process.Result, error) {
		return process.Result{Command: command, Args: args, Dir: dir, ExitCode: 1, Stderr: "pandoc: boom"},
			errors.New("exit status 1")
	}}

	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target}, executor)
	if result.Success || !hasCode(result.Diagnostics, CodeConversionFailed) {
		t.Fatalf("result = %#v, want %s", result, CodeConversionFailed)
	}
}

func TestExportReportsAMissingTex(t *testing.T) {
	projectDir := fixtureProject(t)
	target := filepath.Join(t.TempDir(), "latex")
	executor := &fakeExecutor{behavior: func(dir, command string, args []string) (process.Result, error) {
		return process.Result{Command: command, Args: args, Dir: dir}, nil // exit 0, writes nothing
	}}

	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target}, executor)
	if result.Success || !hasCode(result.Diagnostics, CodeTexMissing) {
		t.Fatalf("result = %#v, want %s", result, CodeTexMissing)
	}
}

// ---------- verification --------------------------------------------------

func TestVerifyLeavesNoIntermediatesInTheExport(t *testing.T) {
	projectDir := fixtureProject(t)
	target := filepath.Join(t.TempDir(), "latex")
	stubTools(t, "xelatex", "bibtex", "biber")

	var scratchDirs []string
	executor := &fakeExecutor{behavior: func(dir, command string, args []string) (process.Result, error) {
		if command == "powershell.exe" {
			return fakeConversion(dir, command, args)
		}
		// Behave like a real TeX tool: drop intermediates beside the source.
		scratchDirs = append(scratchDirs, dir)
		for _, name := range []string{"paper.aux", "paper.log", "paper.pdf", "paper.bbl"} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("intermediate"), 0o644); err != nil {
				return process.Result{ExitCode: 1}, err
			}
		}
		return process.Result{Command: command, Args: args, Dir: dir}, nil
	}}

	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target, Verify: true}, executor)
	if !result.Success {
		t.Fatalf("export failed: %#v", result.Diagnostics)
	}
	if !result.Verified {
		t.Fatal("Verified = false after a successful compile chain")
	}
	for _, name := range []string{"paper.aux", "paper.log", "paper.pdf", "paper.bbl"} {
		if _, err := os.Stat(filepath.Join(target, name)); !os.IsNotExist(err) {
			t.Errorf("%s was left in the delivered export: %v", name, err)
		}
	}
	for _, dir := range scratchDirs {
		if within(target, dir) {
			t.Errorf("verification compiled inside the export directory: %s", dir)
		}
		if _, err := os.Stat(dir); !os.IsNotExist(err) {
			t.Errorf("verification scratch directory %s survived: %v", dir, err)
		}
	}
}

func TestVerifyLeavesNoIntermediatesInTheZip(t *testing.T) {
	projectDir := fixtureProject(t)
	target := filepath.Join(t.TempDir(), "latex.zip")
	stubTools(t, "xelatex", "bibtex", "biber")

	executor := &fakeExecutor{behavior: func(dir, command string, args []string) (process.Result, error) {
		if command == "powershell.exe" {
			return fakeConversion(dir, command, args)
		}
		for _, name := range []string{"paper.aux", "paper.log", "paper.pdf", "paper.bbl"} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte("intermediate"), 0o644); err != nil {
				return process.Result{ExitCode: 1}, err
			}
		}
		return process.Result{Command: command, Args: args, Dir: dir}, nil
	}}

	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target, Verify: true}, executor)
	if !result.Success || !result.Verified || !result.Zipped {
		t.Fatalf("result = %#v", result)
	}
	reader, err := zip.OpenReader(target)
	if err != nil {
		t.Fatalf("open zip: %v", err)
	}
	defer reader.Close()
	for _, file := range reader.File {
		switch strings.ToLower(filepath.Ext(file.Name)) {
		case ".aux", ".log", ".pdf", ".bbl":
			t.Errorf("verification output was included in the ZIP: %s", file.Name)
		}
	}
}

func TestVerifyRunsTheChainTheReadmeDocuments(t *testing.T) {
	for mode, want := range map[BibMode][]string{
		BibBibTeX:   {"xelatex", "bibtex", "xelatex", "xelatex"},
		BibBibLaTeX: {"xelatex", "biber", "xelatex", "xelatex"},
		BibInline:   {"xelatex", "xelatex"},
	} {
		t.Run(string(mode), func(t *testing.T) {
			projectDir := fixtureProject(t)
			target := filepath.Join(t.TempDir(), "latex")
			stubTools(t, "xelatex", "bibtex", "biber")
			executor := &fakeExecutor{}

			result := runExport(t, Options{ProjectDir: projectDir, ToPath: target, Bib: mode, Verify: true}, executor)
			if !result.Success || !result.Verified {
				t.Fatalf("result = %#v", result)
			}
			var got []string
			for _, call := range executor.calls {
				if call.command == "powershell.exe" {
					continue
				}
				got = append(got, strings.TrimSuffix(filepath.Base(call.command), ".stub"))
			}
			if strings.Join(got, ",") != strings.Join(want, ",") {
				t.Fatalf("chain = %v, want %v", got, want)
			}
			readmeText := readme(mode)
			for _, tool := range want {
				if !strings.Contains(readmeText, tool) {
					t.Errorf("README.txt does not mention %q", tool)
				}
			}
		})
	}
}

func TestVerifyFailureIsReportedAndKeepsTheFiles(t *testing.T) {
	projectDir := fixtureProject(t)
	target := filepath.Join(t.TempDir(), "latex")
	stubTools(t, "xelatex", "bibtex", "biber")
	executor := &fakeExecutor{behavior: func(dir, command string, args []string) (process.Result, error) {
		if command == "powershell.exe" {
			return fakeConversion(dir, command, args)
		}
		return process.Result{Command: command, Args: args, Dir: dir, ExitCode: 1}, errors.New("exit status 1")
	}}

	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target, Verify: true}, executor)
	if result.Success || result.Verified {
		t.Fatalf("result = %#v, want a failed verification", result)
	}
	if !hasCode(result.Diagnostics, CodeVerifyFailed) {
		t.Fatalf("diagnostics = %#v, want %s", result.Diagnostics, CodeVerifyFailed)
	}
	// The export is on disk and usable; only the local compile check failed.
	if _, err := os.Stat(filepath.Join(target, "paper.tex")); err != nil {
		t.Errorf("paper.tex was removed after a failed verification: %v", err)
	}
}

func TestVerifyIsSkippedWithAWarningWhenAToolIsMissing(t *testing.T) {
	projectDir := fixtureProject(t)
	target := filepath.Join(t.TempDir(), "latex")
	restore := lookPath
	lookPath = func(name string) (string, error) { return "", fmt.Errorf("%s: not found", name) }
	t.Cleanup(func() { lookPath = restore })

	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target, Verify: true}, &fakeExecutor{})
	if !result.Success {
		t.Fatalf("a missing TeX tool must not fail the export: %#v", result.Diagnostics)
	}
	if result.Verified {
		t.Error("Verified = true although verification never ran")
	}
	if !hasCode(result.Diagnostics, CodeVerifySkipped) {
		t.Fatalf("diagnostics = %#v, want %s", result.Diagnostics, CodeVerifySkipped)
	}
}

// stubTools makes lookPath resolve the named tools to unique paths without
// requiring a TeX installation. The fake executor answers the calls.
func stubTools(t *testing.T, names ...string) {
	t.Helper()
	dir := t.TempDir()
	allowed := make(map[string]string, len(names))
	for _, name := range names {
		allowed[name] = filepath.Join(dir, name+".stub")
	}
	restore := lookPath
	lookPath = func(name string) (string, error) {
		if path, ok := allowed[name]; ok {
			return path, nil
		}
		return "", fmt.Errorf("%s: not found", name)
	}
	t.Cleanup(func() { lookPath = restore })
}

// ---------- README --------------------------------------------------------

func TestReadmeStatesTheOneWayBoundaryAndIgnoreAdvice(t *testing.T) {
	for _, mode := range []BibMode{BibBibTeX, BibBibLaTeX, BibInline} {
		text := readme(mode)
		for _, needle := range []string{
			"one-way",
			"never read back",
			"No .gitignore was created",
			"*.aux",
			"tlmgr install",
			"miktex packages install",
			"XeLaTeX",
		} {
			if !strings.Contains(text, needle) {
				t.Errorf("%s README.txt does not mention %q", mode, needle)
			}
		}
	}
}

// Overleaf is where an export is most likely to be compiled by someone who did
// not produce it, and it is the one target with a setting that must be changed
// by hand: it defaults to pdfLaTeX, whose failure names fontspec internals and
// never mentions the compiler. It is also the one target with a compile time
// limit the recipient cannot raise from inside the document, so the section
// states the free-plan cap before the upload steps rather than after them.
func TestReadmeExplainsOverleaf(t *testing.T) {
	for _, mode := range []BibMode{BibBibTeX, BibBibLaTeX, BibInline} {
		text := readme(mode)
		for _, needle := range []string{
			"Compiling on Overleaf",
			"after 10",
			"240 seconds",
			"7-day trial",
			"Upload Project",
			"Compiler > XeLaTeX",
			"defaults to pdfLaTeX",
			"Noto Serif CJK SC",
			"No usable Chinese font found",
			"fonts-noto-cjk",
		} {
			if !strings.Contains(text, needle) {
				t.Errorf("%s README.txt does not mention %q", mode, needle)
			}
		}
	}
	// The zip layout is the other thing people get wrong; Overleaf cannot find
	// paper.tex when the archive wraps it in a folder.
	if !strings.Contains(readme(BibInline), "not the enclosing folder") {
		t.Error("README.txt does not say to zip the contents rather than the folder")
	}
	// Order matters more than presence: a cap disclosed after the upload steps
	// is read only by someone who already spent the time it was meant to save.
	for _, mode := range []BibMode{BibBibTeX, BibBibLaTeX, BibInline} {
		text := readme(mode)
		if strings.Index(text, "after 10") > strings.Index(text, "Upload Project") {
			t.Errorf("%s README.txt states the Overleaf time limit after the upload steps", mode)
		}
	}
}

func TestReadmeExplainsTheGbt7714TitleCaseWorkaroundInBibtexModeOnly(t *testing.T) {
	bibtex := readme(BibBibTeX)
	if !strings.Contains(bibtex, "title = {{") || !strings.Contains(bibtex, "sentence case") {
		t.Errorf("bibtex README.txt does not explain the double-brace workaround:\n%s", bibtex)
	}
	for _, mode := range []BibMode{BibBibLaTeX, BibInline} {
		if strings.Contains(readme(mode), "title = {{") {
			t.Errorf("%s README.txt carries the gbt7714-only workaround", mode)
		}
	}
}

func TestReadmeNeverShipsAGitignore(t *testing.T) {
	projectDir := fixtureProject(t)
	target := filepath.Join(t.TempDir(), "latex")
	if result := runExport(t, Options{ProjectDir: projectDir, ToPath: target}, &fakeExecutor{}); !result.Success {
		t.Fatalf("export failed: %#v", result.Diagnostics)
	}
	if _, err := os.Stat(filepath.Join(target, ".gitignore")); !os.IsNotExist(err) {
		t.Errorf("export created a .gitignore: %v", err)
	}
}
