package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRunVersion(t *testing.T) {
	oldVersion := version
	version = "0.1.0-test"
	t.Cleanup(func() { version = oldVersion })

	var stdout, stderr bytes.Buffer
	if code := run([]string{"--version"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run() exit code = %d, want 0", code)
	}
	if got, want := stdout.String(), "nodepaper 0.1.0-test\n"; got != want {
		t.Fatalf("stdout = %q, want %q", got, want)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunNoArgsShowsReadOnlyOnboardingOutsideProject(t *testing.T) {
	workingDir := t.TempDir()
	before, err := os.ReadDir(workingDir)
	if err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := runWithIO(context.Background(), nil, strings.NewReader(""), &stdout, &stderr, false, workingDir); code != 0 {
		t.Fatalf("run() exit code = %d, want 0", code)
	}
	for _, want := range []string{
		"No NodePaper Project", "nodepaper.yaml", "nodepaper init", "nodepaper doctor",
		// Someone unsure whether the install took effect needs this here, not
		// buried one level down in --help.
		"nodepaper --version", "nodepaper --help",
		// First-run users need to learn about the TeX prerequisite here rather
		// than partway through their first build.
		"TeX Live or MiKTeX", "xelatex",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("stdout missing %q: %s", want, stdout.String())
		}
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
	after, err := os.ReadDir(workingDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(before) != len(after) {
		t.Fatalf("onboarding modified the directory: before=%d after=%d", len(before), len(after))
	}
}

func TestRunNoArgsShowsProjectCommandsFromSubdirectory(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "nodepaper.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	subdir := filepath.Join(root, "sections")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := runWithIO(context.Background(), nil, strings.NewReader(""), &stdout, &stderr, false, subdir); code != 0 {
		t.Fatalf("run() exit code = %d, want 0", code)
	}
	for _, want := range []string{"Project found", root, "nodepaper validate", "nodepaper build", "nodepaper clean"} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("stdout missing %q: %s", want, stdout.String())
		}
	}
	// Someone with a Project already got past setup, so repeating the TeX
	// prerequisite on every bare invocation would just be noise.
	if strings.Contains(stdout.String(), "TeX Live or MiKTeX") {
		t.Errorf("prerequisite notice leaked into the in-project view: %s", stdout.String())
	}
}

func TestRunHelp(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run() exit code = %d, want 0", code)
	}
	if !strings.HasPrefix(stdout.String(), "Usage:\n") {
		t.Fatalf("stdout does not contain usage: %q", stdout.String())
	}
}

func TestRunUsageErrorIncludesFocusedSuggestion(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"buid"}, &stdout, &stderr); code != 2 {
		t.Fatalf("run() exit code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "unknown command") || !strings.Contains(stderr.String(), "nodepaper build") {
		t.Fatalf("stderr = %q, want error and correction", stderr.String())
	}
	if strings.Contains(stderr.String(), "Usage:\n") {
		t.Fatalf("stderr should not bury the correction under generic usage: %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

func TestRunInit(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "test-project")

	var stdout, stderr bytes.Buffer
	if code := run([]string{"init", projectDir}, &stdout, &stderr); code != 0 {
		t.Fatalf("run() exit code = %d, want 0\nstderr: %s", code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, "✓ Success") {
		t.Fatalf("stdout missing success: %s", out)
	}
	if !strings.Contains(out, "Project:") {
		t.Fatalf("stdout missing Project: %s", out)
	}
	if !strings.Contains(out, "markdown:") {
		t.Fatalf("stdout missing artifact: %s", out)
	}

	// Verify files were actually created.
	for _, name := range []string{"nodepaper.yaml", "paper.md", "references.bib", ".gitignore"} {
		path := filepath.Join(projectDir, name)
		if _, err := os.Stat(path); os.IsNotExist(err) {
			t.Fatalf("expected file not created: %s", path)
		}
	}
	imagesDir := filepath.Join(projectDir, "images")
	if info, err := os.Stat(imagesDir); err != nil || !info.IsDir() {
		t.Fatalf("images/ directory not created: %v", err)
	}
}

func TestRunInitMissingPathNeverPromptsWhenNonInteractive(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runWithIO(context.Background(), []string{"init"}, strings.NewReader("should-not-be-read\n"), &stdout, &stderr, false, "")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 || !strings.Contains(stderr.String(), "non-interactive") {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestRunInitJSONMissingPathNeverPrompts(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := runWithIO(context.Background(), []string{"init", "--format", "json"}, strings.NewReader("should-not-be-read\n"), &stdout, &stderr, true, "")
	if code != 2 {
		t.Fatalf("exit code = %d, want 2", code)
	}
	if stdout.Len() != 0 || strings.Contains(stderr.String(), "Continue? [Y/n]") {
		t.Fatalf("stdout=%q stderr=%q", stdout.String(), stderr.String())
	}
}

func TestRunInteractiveInitDefaultsToCurrentDirectoryAndAIGuide(t *testing.T) {
	projectDir := t.TempDir()
	// Enter on the directory confirmation (default: initialize here), then
	// Enter on the AI-guide question (default: generate AGENTS.md).
	input := strings.NewReader("\n\n")
	var stdout, stderr bytes.Buffer
	code := runWithIO(context.Background(), []string{"init"}, input, &stdout, &stderr, true, projectDir)
	if code != 0 {
		t.Fatalf("exit code = %d; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(projectDir, "nodepaper.yaml")); err != nil {
		t.Fatalf("default interactive init did not create nodepaper.yaml: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectDir, "AGENTS.md")); err != nil {
		t.Fatalf("default interactive init did not create AGENTS.md: %v", err)
	}
	if !strings.Contains(stdout.String(), projectDir) {
		t.Fatalf("confirmation does not show the current directory: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Continue? [Y/n]") {
		t.Fatalf("confirmation prompt missing: %s", stdout.String())
	}
	if !strings.Contains(stdout.String(), "Generate AI writing guide") {
		t.Fatalf("stdout lacks AI guide prompt: %s", stdout.String())
	}
}

func TestRunInteractiveInitDeclinesAIGuide(t *testing.T) {
	projectDir := t.TempDir()
	// "y" confirms the current directory, "n" declines the AI guide.
	input := strings.NewReader("y\nn\n")
	var stdout, stderr bytes.Buffer
	code := runWithIO(context.Background(), []string{"init"}, input, &stdout, &stderr, true, projectDir)
	if code != 0 {
		t.Fatalf("exit code = %d; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(projectDir, "nodepaper.yaml")); err != nil {
		t.Fatalf("interactive init did not create nodepaper.yaml: %v", err)
	}
	if _, err := os.Stat(filepath.Join(projectDir, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("AGENTS.md should not exist after declining; err=%v", err)
	}
}

func TestRunInteractiveInitDecliningDirectoryCancels(t *testing.T) {
	projectDir := t.TempDir()
	// "n" on the directory confirmation cancels before anything is created.
	input := strings.NewReader("n\n")
	var stdout, stderr bytes.Buffer
	code := runWithIO(context.Background(), []string{"init"}, input, &stdout, &stderr, true, projectDir)
	if code != 130 {
		t.Fatalf("exit code = %d, want 130; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stderr.String(), "canceled") {
		t.Fatalf("stderr does not mention cancelation: %s", stderr.String())
	}
	if _, err := os.Stat(filepath.Join(projectDir, "nodepaper.yaml")); !os.IsNotExist(err) {
		t.Fatalf("canceled init left project files; err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(projectDir, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("canceled init left AGENTS.md; err=%v", err)
	}
}

func TestRunInteractiveInitRejectsInvalidAnswersThenContinues(t *testing.T) {
	projectDir := t.TempDir()
	// "maybe" is invalid, then Enter continues with defaults.
	input := strings.NewReader("maybe\n\n\n")
	var stdout, stderr bytes.Buffer
	code := runWithIO(context.Background(), []string{"init"}, input, &stdout, &stderr, true, projectDir)
	if code != 0 {
		t.Fatalf("exit code = %d; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Please enter Y or N") {
		t.Fatalf("invalid answer was not rejected: %s", stdout.String())
	}
	if _, err := os.Stat(filepath.Join(projectDir, "nodepaper.yaml")); err != nil {
		t.Fatalf("init did not complete after the invalid answer: %v", err)
	}
}

func TestRunInteractiveInitEOFCancelsWithoutFiles(t *testing.T) {
	projectDir := t.TempDir()
	// EOF on the directory confirmation cancels atomically.
	input := strings.NewReader("")
	var stdout, stderr bytes.Buffer
	code := runWithIO(context.Background(), []string{"init"}, input, &stdout, &stderr, true, projectDir)
	if code != 130 {
		t.Fatalf("exit code = %d, want 130; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(projectDir, "nodepaper.yaml")); !os.IsNotExist(err) {
		t.Fatalf("canceled init left project files; err=%v", err)
	}
}

func TestRunInteractiveInitEOFAfterConfirmBeforeGuideCancels(t *testing.T) {
	projectDir := t.TempDir()
	// Confirm the directory, then hit EOF before the AI-guide answer.
	input := strings.NewReader("y\n")
	var stdout, stderr bytes.Buffer
	code := runWithIO(context.Background(), []string{"init"}, input, &stdout, &stderr, true, projectDir)
	if code != 130 {
		t.Fatalf("exit code = %d, want 130; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(filepath.Join(projectDir, "nodepaper.yaml")); !os.IsNotExist(err) {
		t.Fatalf("canceled init left project files; err=%v", err)
	}
}

func TestRunCanceledExplicitInitUsesExit130AndLeavesNoProject(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "project")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var stdout, stderr bytes.Buffer
	code := runWithIO(ctx, []string{"init", projectDir}, strings.NewReader(""), &stdout, &stderr, false, "")
	if code != 130 {
		t.Fatalf("exit code = %d, want 130; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if _, err := os.Stat(projectDir); !os.IsNotExist(err) {
		t.Fatalf("canceled init left project files; err=%v", err)
	}
}

func TestRunExplicitInitDoesNotConsumeInput(t *testing.T) {
	projectDir := filepath.Join(t.TempDir(), "project")
	input := &countingReader{Reader: strings.NewReader("unexpected\n")}
	var stdout, stderr bytes.Buffer
	code := runWithIO(context.Background(), []string{"init", projectDir}, input, &stdout, &stderr, true, "")
	if code != 0 {
		t.Fatalf("exit code = %d; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if input.reads != 0 {
		t.Fatalf("explicit init read stdin %d times", input.reads)
	}
	if _, err := os.Stat(filepath.Join(projectDir, "AGENTS.md")); !os.IsNotExist(err) {
		t.Fatalf("explicit init should not generate AGENTS.md by default; err=%v", err)
	}
}

type countingReader struct {
	io.Reader
	reads int
}

func (r *countingReader) Read(p []byte) (int, error) {
	r.reads++
	return r.Reader.Read(p)
}

func TestRunInitJSON(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "test-project")

	var stdout, stderr bytes.Buffer
	if code := run([]string{"init", projectDir, "--format", "json"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run() exit code = %d, want 0\nstderr: %s", code, stderr.String())
	}

	out := stdout.String()
	if !strings.Contains(out, `"schemaVersion"`) {
		t.Fatalf("stdout missing schemaVersion: %s", out)
	}
	if !strings.Contains(out, `"success"`) {
		t.Fatalf("stdout missing success: %s", out)
	}
	if !strings.Contains(out, `"markdown"`) {
		t.Fatalf("stdout missing artifact: %s", out)
	}
}

func TestRunValidateJSONFailureUsesNonzeroExitAndPureJSON(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "invalid-project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "nodepaper.yaml"), []byte("version: [\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var stdout, stderr bytes.Buffer
	code := run([]string{"validate", projectDir, "--format", "json"}, &stdout, &stderr)
	if code != 1 {
		t.Fatalf("run() exit code = %d, want 1; stdout=%s stderr=%s", code, stdout.String(), stderr.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty for a rendered business failure", stderr.String())
	}

	var payload struct {
		SchemaVersion int  `json:"schemaVersion"`
		Success       bool `json:"success"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &payload); err != nil {
		t.Fatalf("stdout is not one valid JSON object: %v\n%s", err, stdout.String())
	}
	if payload.SchemaVersion != 1 || payload.Success {
		t.Fatalf("payload = %+v, want schemaVersion=1 success=false", payload)
	}
}

func TestRunDoctor(t *testing.T) {
	var stdout, stderr bytes.Buffer
	// Doctor without project dir checks global toolchain only.
	code := run([]string{"doctor"}, &stdout, &stderr)
	// Doctor may succeed or fail depending on available tools, but should
	// not crash or return a "not yet implemented" error.
	if stderr.Len() > 0 {
		t.Logf("stderr (may contain warnings): %s", stderr.String())
	}
	if !strings.Contains(stdout.String(), "PASS") && !strings.Contains(stdout.String(), "FAIL") && !strings.Contains(stdout.String(), "WARN") {
		t.Fatalf("stdout should contain doctor checks: %s", stdout.String())
	}
	_ = code
}

func TestRunValidate(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "test-project")

	// Init a project first.
	var stdout, stderr bytes.Buffer
	code := run([]string{"init", projectDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("init exit code = %d, stderr: %s", code, stderr.String())
	}

	// Validate the project.
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"validate", projectDir}, &stdout, &stderr)
	// Should succeed since the init creates a valid project skeleton.
	if stderr.Len() > 0 {
		t.Logf("stderr: %s", stderr.String())
	}
	if code != 0 {
		t.Fatalf("validate exit code = %d, stdout: %s, stderr: %s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "✓ Success") {
		t.Fatalf("stdout missing success: %s", stdout.String())
	}
}

func TestRunBuild(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "test-project")

	// Init a project first.
	var stdout, stderr bytes.Buffer
	code := run([]string{"init", projectDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("init exit code = %d, stderr: %s", code, stderr.String())
	}

	// Build - may fail if pandoc/latexmk are not available, but should not crash.
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"build", projectDir}, &stdout, &stderr)
	// Build may succeed or fail depending on toolchain; just verify it doesn't panic.
	t.Logf("build exit code: %d, stdout: %s, stderr: %s", code, stdout.String(), stderr.String())
}

func TestRunClean(t *testing.T) {
	dir := t.TempDir()
	projectDir := filepath.Join(dir, "test-project")

	// Init a project first.
	var stdout, stderr bytes.Buffer
	code := run([]string{"init", projectDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("init exit code = %d, stderr: %s", code, stderr.String())
	}

	// Clean.
	stdout.Reset()
	stderr.Reset()
	code = run([]string{"clean", projectDir}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("clean exit code = %d, stderr: %s", code, stderr.String())
	}
}

func TestConsoleHoldNoticeGuidesToStartMenuAndTerminal(t *testing.T) {
	var stdout bytes.Buffer
	writeConsoleHoldNotice(&stdout)
	for _, want := range []string{
		"would close immediately",
		"nothing was installed or changed",
		"NodePaper-Setup-<version>-windows-x64.exe",
		"Start menu > NodePaper",
		"nodepaper --help",
		"Press Enter to close this window.",
	} {
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("console hold notice missing %q: %s", want, stdout.String())
		}
	}
}

func TestHoldConsoleReturnsOnEnterAndOnEndOfInput(t *testing.T) {
	for name, input := range map[string]string{"enter": "\n", "eof": ""} {
		t.Run(name, func(t *testing.T) {
			var stdout bytes.Buffer
			done := make(chan struct{})
			go func() {
				holdConsole(strings.NewReader(input), &stdout)
				close(done)
			}()
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("holdConsole did not return")
			}
			if stdout.Len() == 0 {
				t.Fatal("holdConsole printed nothing")
			}
		})
	}
}

func TestShouldHoldConsoleNeverBlocksAutomation(t *testing.T) {
	// Non-interactive input (pipes, redirection, CI) must never wait, and JSON
	// output must never wait even on an interactive console.
	if shouldHoldConsole(nil, false) {
		t.Error("shouldHoldConsole(non-interactive) = true, want false")
	}
	if shouldHoldConsole([]string{"build", "--format", "json"}, true) {
		t.Error("shouldHoldConsole(json) = true, want false")
	}
	if shouldHoldConsole([]string{"build", "--format=json"}, true) {
		t.Error("shouldHoldConsole(json=) = true, want false")
	}
	// The remaining interactive case depends on the real console layout: a test
	// process is never the only process attached to its console.
	if ownsConsoleWindow() && shouldHoldConsole(nil, true) != true {
		t.Error("shouldHoldConsole(owned console) = false, want true")
	}
}
