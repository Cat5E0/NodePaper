package build

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"nodepaper/internal/buildlock"
	"nodepaper/internal/diagnostic"
	"nodepaper/internal/process"
)

type fakeExecutor struct {
	mu       sync.Mutex
	calls    []fakeCall
	behavior func(context.Context, string, string, []string) (process.Result, error)
}

type fakeCall struct {
	dir     string
	command string
	args    []string
}

func (f *fakeExecutor) Run(ctx context.Context, dir, command string, args ...string) (process.Result, error) {
	f.mu.Lock()
	f.calls = append(f.calls, fakeCall{dir: dir, command: command, args: append([]string(nil), args...)})
	f.mu.Unlock()
	if f.behavior != nil {
		return f.behavior(ctx, dir, command, args)
	}
	return successfulFakeTool(ctx, dir, command, args)
}

func successfulFakeTool(_ context.Context, dir, command string, args []string) (process.Result, error) {
	result := process.Result{Command: command, Args: args, Dir: dir, ExitCode: 0}
	if command != "powershell.exe" {
		return result, errors.New("fake tool expected powershell.exe")
	}
	output := argumentValue(args, "-Output")
	buildDir := argumentValue(args, "-BuildDirectory")
	if output == "" || buildDir == "" {
		return result, errors.New("fake PowerShell did not receive output and build directory")
	}
	if err := os.WriteFile(output, []byte("\\documentclass{article}\\begin{document}ok\\end{document}"), 0o644); err != nil {
		return result, err
	}
	if err := os.WriteFile(filepath.Join(buildDir, "paper.pdf"), []byte("%PDF-1.4\nfake\n%%EOF\n"), 0o644); err != nil {
		return result, err
	}
	if err := os.WriteFile(filepath.Join(buildDir, "paper.log"), nil, 0o644); err != nil {
		return result, err
	}
	return result, nil
}

func TestDefaultBuildScriptPathHonorsTestOverride(t *testing.T) {
	want := filepath.Join(t.TempDir(), "Build-Paper.ps1")
	t.Setenv("NODEPAPER_BUILD_SCRIPT", want)
	if got := defaultBuildScriptPath(); got != want {
		t.Fatalf("defaultBuildScriptPath() = %q, want %q", got, want)
	}
}

func TestDefaultProfileDirHonorsTestOverride(t *testing.T) {
	want := filepath.Join(t.TempDir(), "profiles", "cumcm")
	t.Setenv("NODEPAPER_PROFILE_DIR", want)
	if got := defaultProfileDir(); got != want {
		t.Fatalf("defaultProfileDir() = %q, want %q", got, want)
	}
}

func TestBuildRejectsMissingProfileBeforePowerShell(t *testing.T) {
	projectDir := copyBuildFixture(t, "minimal-valid")
	executor := &fakeExecutor{}
	result := runWithExecutorAndResources(
		context.Background(),
		projectDir,
		executor,
		filepath.Join(t.TempDir(), "Build-Paper.ps1"),
		filepath.Join(t.TempDir(), "missing-profile"),
	)
	if result.Success || !hasDiagnosticCode(result, "NP1601") {
		t.Fatalf("result = %#v, want NP1601", result)
	}
	if len(executor.calls) != 0 {
		t.Fatalf("PowerShell was called with a missing Profile: %#v", executor.calls)
	}
}

func TestBuildWithFakeToolsPublishesPDF(t *testing.T) {
	projectDir := copyBuildFixture(t, "minimal-valid")
	executor := &fakeExecutor{}

	result := runWithExecutor(context.Background(), projectDir, executor)
	if !result.Success {
		t.Fatalf("build failed: %#v", result.Diagnostics)
	}
	if result.BuildID == "" {
		t.Fatal("BuildID is empty")
	}
	pdf := filepath.Join(projectDir, "dist", "paper.pdf")
	data, err := os.ReadFile(pdf)
	if err != nil {
		t.Fatalf("read published PDF: %v", err)
	}
	if !strings.HasPrefix(string(data), "%PDF-") {
		t.Fatalf("published artifact is not a PDF: %q", data)
	}
	if _, err := os.Stat(buildlock.LockPath(projectDir)); !os.IsNotExist(err) {
		t.Fatalf("build lock remains after success: %v", err)
	}
	logPath := artifactPath(result.Artifacts, "log")
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("read build log: %v", err)
	}
	for _, expected := range []string{"Build ID:", "Project Root:", "Command: powershell.exe", "Build-Paper.ps1", "Exit Code: 0", "Success: true"} {
		if !strings.Contains(string(logData), expected) {
			t.Fatalf("build log missing %q:\n%s", expected, logData)
		}
	}
	if len(executor.calls) != 1 || executor.calls[0].command != "powershell.exe" {
		t.Fatalf("tool calls = %#v, want one PowerShell call", executor.calls)
	}
	call := executor.calls[0]
	if !samePath(call.dir, projectDir) {
		t.Fatalf("PowerShell dir = %q, want project root %q", call.dir, projectDir)
	}
	assertArgumentSequence(t, call.args, "-NoProfile", "-ExecutionPolicy", "Bypass", "-File")
	if script := argumentValue(call.args, "-File"); filepath.Base(script) != "Build-Paper.ps1" {
		t.Fatalf("PowerShell script = %q, want Build-Paper.ps1", script)
	}
	manifestPath := argumentValue(call.args, "-SourceManifest")
	if !samePath(manifestPath, filepath.Join(projectDir, ".nodepaper", "build", "sources.json")) {
		t.Fatalf("SourceManifest = %q", manifestPath)
	}
	manifestData, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatalf("read source manifest: %v", err)
	}
	var manifest struct {
		Sources            []string `json:"sources"`
		LatexFragments     []string `json:"latexFragments"`
		AppendixNumbering  string   `json:"appendixNumbering"`
		HighlightStyle     string   `json:"highlightStyle"`
		LineSpread         float64  `json:"linespread"`
		AbstractLineSpread float64  `json:"abstractLinespread"`
	}
	if err := json.Unmarshal(manifestData, &manifest); err != nil {
		t.Fatalf("decode source manifest: %v", err)
	}
	if len(manifest.Sources) != 1 || !samePath(manifest.Sources[0], filepath.Join(projectDir, "paper.md")) {
		t.Fatalf("manifest sources = %#v", manifest.Sources)
	}
	if len(manifest.LatexFragments) != 0 || manifest.AppendixNumbering != "alpha" || manifest.HighlightStyle != "tango" || manifest.LineSpread != 1.25 || manifest.AbstractLineSpread != 0.95 {
		t.Fatalf("manifest fragments=%#v appendix=%q highlight=%q linespread=%v abstractLinespread=%v", manifest.LatexFragments, manifest.AppendixNumbering, manifest.HighlightStyle, manifest.LineSpread, manifest.AbstractLineSpread)
	}
	if root := argumentValue(call.args, "-ProjectRoot"); !samePath(root, projectDir) {
		t.Fatalf("ProjectRoot = %q", root)
	}
	if profile := argumentValue(call.args, "-ProfileDirectory"); filepath.Base(profile) != "cumcm" {
		t.Fatalf("ProfileDirectory = %q", profile)
	}
	if output := argumentValue(call.args, "-Output"); !samePath(output, filepath.Join(projectDir, ".nodepaper", "build", "paper.tex")) {
		t.Fatalf("Output = %q", output)
	}
}

func TestBuildRejectsProfileMutation(t *testing.T) {
	projectDir := copyBuildFixture(t, "minimal-valid")
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	profileDir := filepath.Join(t.TempDir(), "cumcm")
	if err := copyBuildTree(filepath.Join(repoRoot, "profiles", "cumcm"), profileDir); err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{behavior: func(ctx context.Context, dir, command string, args []string) (process.Result, error) {
		result, runErr := successfulFakeTool(ctx, dir, command, args)
		if runErr == nil {
			runErr = os.WriteFile(filepath.Join(profileDir, "template.tex"), []byte("changed during build"), 0o644)
		}
		return result, runErr
	}}
	result := runWithExecutorAndResources(context.Background(), projectDir, executor, defaultBuildScriptPath(), profileDir)
	if result.Success || !hasDiagnosticCode(result, "NP1602") {
		t.Fatalf("result = %#v, want NP1602", result)
	}
}

func TestBuildRejectsFragmentMutation(t *testing.T) {
	projectDir := copyBuildFixture(t, "minimal-valid")
	fragmentPath := filepath.Join(projectDir, "tables", "result.tex")
	if err := os.MkdirAll(filepath.Dir(fragmentPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(fragmentPath, []byte("safe"), 0o644); err != nil {
		t.Fatal(err)
	}
	configPath := filepath.Join(projectDir, "nodepaper.yaml")
	configData := "version: 1\nprofile: cumcm\nsource: paper.md\nlatexFragments:\n  - tables/result.tex\n"
	if err := os.WriteFile(configPath, []byte(configData), 0o644); err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{behavior: func(ctx context.Context, dir, command string, args []string) (process.Result, error) {
		result, runErr := successfulFakeTool(ctx, dir, command, args)
		if runErr == nil {
			runErr = os.WriteFile(fragmentPath, []byte("changed"), 0o644)
		}
		return result, runErr
	}}
	result := runWithExecutor(context.Background(), projectDir, executor)
	if result.Success || !hasDiagnosticCode(result, "NP2510") {
		t.Fatalf("result = %#v, want NP2510", result)
	}
}

func TestBuildUsesConfiguredOutputPath(t *testing.T) {
	projectDir := copyBuildFixture(t, "minimal-valid")
	configPath := filepath.Join(projectDir, "nodepaper.yaml")
	configData := "version: 1\nprofile: cumcm\nsource: paper.md\noutput:\n  file: exports/result.pdf\n"
	if err := os.WriteFile(configPath, []byte(configData), 0o644); err != nil {
		t.Fatal(err)
	}

	result := runWithExecutor(context.Background(), projectDir, &fakeExecutor{})
	if !result.Success {
		t.Fatalf("build failed: %#v", result.Diagnostics)
	}
	want := filepath.Join(projectDir, "exports", "result.pdf")
	if got := artifactPath(result.Artifacts, "pdf"); !samePath(got, want) {
		t.Fatalf("PDF artifact = %q, want %q", got, want)
	}
	if _, err := os.Stat(want); err != nil {
		t.Fatalf("configured PDF not published: %v", err)
	}
}

func TestBuildFailurePreservesOldPDFAndReleasesLock(t *testing.T) {
	projectDir := copyBuildFixture(t, "minimal-valid")
	distDir := filepath.Join(projectDir, "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldPDF := []byte("%PDF-1.4\nold-success\n")
	pdfPath := filepath.Join(distDir, "paper.pdf")
	if err := os.WriteFile(pdfPath, oldPDF, 0o644); err != nil {
		t.Fatal(err)
	}

	staleLog := filepath.Join(projectDir, ".nodepaper", "build", "paper.log")
	if err := os.MkdirAll(filepath.Dir(staleLog), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(staleLog, []byte("Missing character: stale diagnostic\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{behavior: func(_ context.Context, dir, command string, args []string) (process.Result, error) {
		return process.Result{Command: command, Args: args, Dir: dir, ExitCode: 7, Stderr: "controlled pandoc failure"}, nil
	}}
	result := runWithExecutor(context.Background(), projectDir, executor)
	if result.Success || !hasDiagnosticCode(result, "NP5001") {
		t.Fatalf("result = %#v, want NP5001 failure", result)
	}
	if hasDiagnosticCode(result, "NP6102") {
		t.Fatalf("stale LaTeX log was classified: %#v", result.Diagnostics)
	}
	if _, err := os.Stat(staleLog); !os.IsNotExist(err) {
		t.Fatalf("stale LaTeX log was retained: %v", err)
	}
	got, err := os.ReadFile(pdfPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(oldPDF) {
		t.Fatalf("old PDF was overwritten: %q", got)
	}
	if _, err := os.Stat(buildlock.LockPath(projectDir)); !os.IsNotExist(err) {
		t.Fatalf("build lock remains after failure: %v", err)
	}
	logPath := artifactPath(result.Artifacts, "log")
	logData, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatalf("failure log not preserved: %v", err)
	}
	if !strings.Contains(string(logData), "controlled pandoc failure") || !strings.Contains(string(logData), "Success: false") {
		t.Fatalf("failure log lacks process diagnostics:\n%s", logData)
	}
}

func TestBuildClassifiesFatalLatexLogAfterPowerShellFailure(t *testing.T) {
	projectDir := copyBuildFixture(t, "minimal-valid")
	executor := &fakeExecutor{behavior: func(_ context.Context, dir, command string, args []string) (process.Result, error) {
		buildDir := argumentValue(args, "-BuildDirectory")
		if err := os.WriteFile(filepath.Join(buildDir, "paper.log"), []byte("Undefined control sequence.\n"), 0o644); err != nil {
			return process.Result{}, err
		}
		return process.Result{Command: command, Args: args, Dir: dir, ExitCode: 12}, nil
	}}
	result := runWithExecutor(context.Background(), projectDir, executor)
	if result.Success || !hasDiagnosticCode(result, "NP5001") || !hasDiagnosticCode(result, "NP6106") {
		t.Fatalf("result = %#v, want NP5001 and classified fatal NP6106", result)
	}
}

func TestBuildReportsMissingPDF(t *testing.T) {
	projectDir := copyBuildFixture(t, "minimal-valid")
	stalePDF := filepath.Join(projectDir, ".nodepaper", "build", "paper.pdf")
	if err := os.MkdirAll(filepath.Dir(stalePDF), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(stalePDF, []byte("%PDF-1.4\nstale\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{behavior: func(_ context.Context, dir, command string, args []string) (process.Result, error) {
		return process.Result{Command: command, Args: args, Dir: dir, ExitCode: 0}, nil
	}}

	result := runWithExecutor(context.Background(), projectDir, executor)
	if result.Success || !hasDiagnosticCode(result, "NP6002") {
		t.Fatalf("result = %#v, want NP6002 failure", result)
	}
	if _, err := os.Stat(stalePDF); !os.IsNotExist(err) {
		t.Fatalf("stale intermediate PDF was accepted or retained: %v", err)
	}
}

func TestBuildRejectsDamagedPDFAndPreservesOldPDF(t *testing.T) {
	projectDir := copyBuildFixture(t, "minimal-valid")
	distDir := filepath.Join(projectDir, "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldPDF := []byte("%PDF-1.4\nold-success\n")
	finalPDF := filepath.Join(distDir, "paper.pdf")
	if err := os.WriteFile(finalPDF, oldPDF, 0o644); err != nil {
		t.Fatal(err)
	}

	executor := &fakeExecutor{behavior: func(_ context.Context, dir, command string, args []string) (process.Result, error) {
		outputDir := argumentValue(args, "-BuildDirectory")
		if err := os.WriteFile(filepath.Join(outputDir, "paper.pdf"), []byte("not a PDF"), 0o644); err != nil {
			return process.Result{}, err
		}
		return process.Result{Command: command, Args: args, Dir: dir, ExitCode: 0}, nil
	}}
	result := runWithExecutor(context.Background(), projectDir, executor)
	if result.Success || !hasDiagnosticCode(result, "NP7001") {
		t.Fatalf("result = %#v, want NP7001", result)
	}
	got, err := os.ReadFile(finalPDF)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(oldPDF) {
		t.Fatalf("old PDF was overwritten: %q", got)
	}
}

func TestBuildRejectsPDFWithoutEOFMarker(t *testing.T) {
	projectDir := copyBuildFixture(t, "minimal-valid")
	executor := &fakeExecutor{behavior: func(_ context.Context, dir, command string, args []string) (process.Result, error) {
		outputDir := argumentValue(args, "-BuildDirectory")
		if err := os.WriteFile(filepath.Join(outputDir, "paper.pdf"), []byte("%PDF-1.4\nincomplete\n"), 0o644); err != nil {
			return process.Result{}, err
		}
		return process.Result{Command: command, Args: args, Dir: dir, ExitCode: 0}, nil
	}}
	result := runWithExecutor(context.Background(), projectDir, executor)
	if result.Success || !hasDiagnosticCode(result, "NP7001") {
		t.Fatalf("result = %#v, want NP7001", result)
	}
}

func TestBuildRejectsCriticalLatexLogAndPreservesOldPDF(t *testing.T) {
	projectDir := copyBuildFixture(t, "minimal-valid")
	distDir := filepath.Join(projectDir, "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatal(err)
	}
	oldPDF := []byte("%PDF-1.4\nold\n%%EOF\n")
	finalPDF := filepath.Join(distDir, "paper.pdf")
	if err := os.WriteFile(finalPDF, oldPDF, 0o644); err != nil {
		t.Fatal(err)
	}
	executor := &fakeExecutor{behavior: func(ctx context.Context, dir, command string, args []string) (process.Result, error) {
		result, err := successfulFakeTool(ctx, dir, command, args)
		if err == nil {
			err = os.WriteFile(filepath.Join(argumentValue(args, "-BuildDirectory"), "paper.log"), []byte("Overfull \\hbox (8.0pt too wide)\n"), 0o644)
		}
		return result, err
	}}
	result := runWithExecutor(context.Background(), projectDir, executor)
	if result.Success || !hasDiagnosticCode(result, "NP6101") {
		t.Fatalf("result = %#v, want NP6101", result)
	}
	got, err := os.ReadFile(finalPDF)
	if err != nil || string(got) != string(oldPDF) {
		t.Fatalf("old PDF changed: %q, %v", got, err)
	}
}

func TestBuildCancellationReleasesLock(t *testing.T) {
	projectDir := copyBuildFixture(t, "minimal-valid")
	started := make(chan struct{})
	executor := &fakeExecutor{behavior: func(ctx context.Context, dir, command string, args []string) (process.Result, error) {
		close(started)
		<-ctx.Done()
		return process.Result{Command: command, Args: args, Dir: dir, ExitCode: -1}, ctx.Err()
	}}
	ctx, cancel := context.WithCancel(context.Background())
	resultCh := make(chan Result, 1)
	go func() { resultCh <- runWithExecutor(ctx, projectDir, executor) }()
	select {
	case <-started:
		cancel()
	case <-time.After(5 * time.Second):
		t.Fatal("fake tool did not start")
	}
	result := <-resultCh
	if result.Success {
		t.Fatal("cancelled build succeeded")
	}
	if _, err := os.Stat(buildlock.LockPath(projectDir)); !os.IsNotExist(err) {
		t.Fatalf("build lock remains after cancellation: %v", err)
	}
}

func TestDifferentProjectsBuildIndependently(t *testing.T) {
	projectA := copyBuildFixture(t, "minimal-valid")
	projectB := copyBuildFixture(t, "minimal-valid")
	results := make(chan Result, 2)
	go func() { results <- runWithExecutor(context.Background(), projectA, &fakeExecutor{}) }()
	go func() { results <- runWithExecutor(context.Background(), projectB, &fakeExecutor{}) }()

	first := <-results
	second := <-results
	for _, result := range []Result{first, second} {
		if !result.Success {
			t.Fatalf("independent build failed: %#v", result.Diagnostics)
		}
	}
	if first.ProjectRoot == second.ProjectRoot || first.BuildID == second.BuildID {
		t.Fatalf("builds were not isolated: first=%#v second=%#v", first, second)
	}
	for _, projectDir := range []string{projectA, projectB} {
		if _, err := os.Stat(filepath.Join(projectDir, "dist", "paper.pdf")); err != nil {
			t.Fatalf("project %s did not receive its PDF: %v", projectDir, err)
		}
	}
}

func TestSecondBuildOfSameProjectIsRejected(t *testing.T) {
	projectDir := copyBuildFixture(t, "minimal-valid")
	started := make(chan struct{})
	release := make(chan struct{})
	firstExecutor := &fakeExecutor{behavior: func(ctx context.Context, dir, command string, args []string) (process.Result, error) {
		if command == "powershell.exe" {
			close(started)
		}
		select {
		case <-release:
			return successfulFakeTool(ctx, dir, command, args)
		case <-ctx.Done():
			return process.Result{Command: command, Args: args, Dir: dir, ExitCode: -1}, ctx.Err()
		}
	}}
	firstResult := make(chan Result, 1)
	go func() { firstResult <- runWithExecutor(context.Background(), projectDir, firstExecutor) }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("first build did not reach fake tool")
	}

	lockData, err := os.ReadFile(buildlock.LockPath(projectDir))
	if err != nil {
		t.Fatalf("read active lock: %v", err)
	}
	var lockInfo buildlock.Info
	if err := json.Unmarshal(lockData, &lockInfo); err != nil {
		t.Fatalf("decode active lock: %v", err)
	}
	if lockInfo.BuildID == "" || lockInfo.BuildID == "pending" {
		t.Fatalf("active lock does not contain the real Build ID: %#v", lockInfo)
	}

	second := runWithExecutor(context.Background(), projectDir, &fakeExecutor{})
	if second.Success || !hasDiagnosticCode(second, "NP1201") {
		t.Fatalf("second build = %#v, want NP1201", second)
	}
	close(release)
	if result := <-firstResult; !result.Success {
		t.Fatalf("first build failed after release: %#v", result.Diagnostics)
	} else if result.BuildID != lockInfo.BuildID {
		t.Fatalf("lock Build ID = %q, result Build ID = %q", lockInfo.BuildID, result.BuildID)
	}
}

func TestBuildUsesConfigurationDiagnosticCodes(t *testing.T) {
	tests := []struct {
		fixture string
		code    string
	}{
		{fixture: "invalid-yaml", code: "NP1501"},
		{fixture: "source-and-sources", code: "NP1502"},
	}
	for _, test := range tests {
		t.Run(test.fixture, func(t *testing.T) {
			projectDir := copyBuildFixture(t, test.fixture)
			executor := &fakeExecutor{}
			result := runWithExecutor(context.Background(), projectDir, executor)
			if result.Success || !hasDiagnosticCode(result, test.code) {
				t.Fatalf("result = %#v, want %s", result, test.code)
			}
			if len(executor.calls) != 0 {
				t.Fatalf("PowerShell was called for invalid configuration: %#v", executor.calls)
			}
		})
	}
}

func TestBuildPreservesConfiguredMultiSourceOrderForPowerShell(t *testing.T) {
	projectDir := copyBuildFixture(t, "complete-multi-file")
	executor := &fakeExecutor{}
	result := runWithExecutor(context.Background(), projectDir, executor)
	if !result.Success {
		t.Fatalf("multi-file build failed: %#v", result.Diagnostics)
	}
	if len(executor.calls) != 1 {
		t.Fatalf("PowerShell calls = %#v, want one", executor.calls)
	}
	manifestPath := argumentValue(executor.calls[0].args, "-SourceManifest")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	var manifest struct {
		Sources []string `json:"sources"`
	}
	if err := json.Unmarshal(data, &manifest); err != nil {
		t.Fatal(err)
	}
	want := []string{
		"sections/01-frontmatter-abstract.md",
		"sections/02-problem.md",
		"sections/03-assumptions-symbols.md",
		"sections/04-data.md",
		"sections/05-model.md",
		"sections/06-solution-results.md",
		"sections/07-evaluation-appendix.md",
	}
	if len(manifest.Sources) != len(want) {
		t.Fatalf("manifest sources = %#v, want %d", manifest.Sources, len(want))
	}
	for index := range want {
		if !samePath(manifest.Sources[index], filepath.Join(projectDir, want[index])) {
			t.Fatalf("source[%d] = %q, want %q", index, manifest.Sources[index], want[index])
		}
	}
}

func TestBuildLockFixtures(t *testing.T) {
	tests := []struct {
		fixture string
		code    string
	}{
		{"damaged-lock", "NP1299"},
		{"stale-lock", "NP1202"},
	}
	for _, test := range tests {
		t.Run(test.fixture, func(t *testing.T) {
			projectDir := copyBuildFixture(t, test.fixture)
			result := runWithExecutor(context.Background(), projectDir, &fakeExecutor{})
			if result.Success || !hasDiagnosticCode(result, test.code) {
				t.Fatalf("result = %#v, want %s failure", result, test.code)
			}
		})
	}
}

func TestCleanBoundariesAndActiveLock(t *testing.T) {
	projectDir := copyBuildFixture(t, "minimal-valid")
	buildDir := filepath.Join(projectDir, ".nodepaper", "build")
	distDir := filepath.Join(projectDir, "dist")
	for _, dir := range []string{buildDir, distDir} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "keep-test.txt"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	if diags, err := Clean(projectDir, false); err != nil || hasErrorDiagnostics(diags) {
		t.Fatalf("Clean() = %#v, %v", diags, err)
	}
	if _, err := os.Stat(buildDir); !os.IsNotExist(err) {
		t.Fatalf("build directory remains: %v", err)
	}
	if _, err := os.Stat(distDir); err != nil {
		t.Fatalf("dist should remain without --all: %v", err)
	}

	held, err := buildlock.Acquire(projectDir, "clean-test")
	if err != nil {
		t.Fatal(err)
	}
	defer held.Release()
	diags, err := Clean(projectDir, true)
	if err != nil || !containsDiagnosticCode(diags, "NP1203") {
		t.Fatalf("Clean() with lock = %#v, %v; want NP1203", diags, err)
	}
	if _, err := os.Stat(distDir); err != nil {
		t.Fatalf("dist was removed while lock held: %v", err)
	}
}

func artifactPath(artifacts []Artifact, kind string) string {
	for _, artifact := range artifacts {
		if artifact.Kind == kind {
			return artifact.Path
		}
	}
	return ""
}

func assertArgumentSequence(t *testing.T, args []string, want ...string) {
	t.Helper()
	for start := 0; start+len(want) <= len(args); start++ {
		matched := true
		for i := range want {
			if args[start+i] != want[i] {
				matched = false
				break
			}
		}
		if matched {
			return
		}
	}
	t.Fatalf("args %#v do not contain sequence %#v", args, want)
}

func argumentValue(args []string, name string) string {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == name {
			return args[i+1]
		}
	}
	return ""
}

func hasDiagnosticCode(result Result, code string) bool {
	return containsDiagnosticCode(result.Diagnostics, code)
}

func hasErrorDiagnostics(diags []diagnostic.Diagnostic) bool {
	for _, diag := range diags {
		if diag.Severity == diagnostic.SeverityError {
			return true
		}
	}
	return false
}

func containsDiagnosticCode(diags []diagnostic.Diagnostic, code string) bool {
	for _, diag := range diags {
		if diag.Code == code {
			return true
		}
	}
	return false
}

func samePath(a, b string) bool {
	return strings.EqualFold(filepath.Clean(a), filepath.Clean(b))
}

func copyBuildFixture(t *testing.T, name string) string {
	t.Helper()
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("NODEPAPER_PROFILE_DIR", filepath.Join(repoRoot, "profiles", "cumcm"))
	source := filepath.Join(repoRoot, "nodepaper-test-fixtures", "tests", "fixtures", name)
	destination := filepath.Join(t.TempDir(), "project")
	if err := copyBuildTree(source, destination); err != nil {
		t.Fatalf("copy fixture %s: %v", name, err)
	}
	return destination
}

func copyBuildTree(source, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, info.Mode().Perm())
		if err != nil {
			return err
		}
		_, copyErr := io.Copy(out, in)
		closeErr := out.Close()
		if copyErr != nil {
			return copyErr
		}
		return closeErr
	})
}
