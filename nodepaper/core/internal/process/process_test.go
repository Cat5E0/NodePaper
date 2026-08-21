package process

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestHelperProcess(t *testing.T) {
	separator := -1
	for i, arg := range os.Args {
		if arg == "--" {
			separator = i
			break
		}
	}
	if separator < 0 || separator+1 >= len(os.Args) {
		return
	}

	switch os.Args[separator+1] {
	case "success":
		fmt.Println("hello")
	case "cwd":
		wd, err := os.Getwd()
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		fmt.Println(wd)
	case "exit42":
		os.Exit(42)
	case "sleep":
		time.Sleep(10 * time.Second)
	case "tree-parent":
		if separator+2 >= len(os.Args) {
			os.Exit(2)
		}
		child := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", "tree-child")
		if err := child.Start(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(2)
		}
		if err := os.WriteFile(os.Args[separator+2], []byte(strconv.Itoa(child.Process.Pid)), 0o644); err != nil {
			_ = child.Process.Kill()
			os.Exit(2)
		}
		_ = child.Wait()
	case "tree-child":
		time.Sleep(10 * time.Second)
	case "stderr":
		fmt.Fprintln(os.Stderr, "error")
	case "env":
		fmt.Println(os.Getenv("NODEPAPER_TEST_ENV"))
	default:
		fmt.Fprintln(os.Stderr, "unknown helper mode")
		os.Exit(2)
	}
	os.Exit(0)
}

func runHelper(ctx context.Context, runner *Runner, mode string, extra ...string) (Result, error) {
	args := []string{"-test.run=TestHelperProcess", "--", mode}
	args = append(args, extra...)
	return runner.Run(ctx, os.Args[0], args...)
}

func TestRunSuccess(t *testing.T) {
	result, err := runHelper(context.Background(), &Runner{}, "success")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if result.ExitCode != 0 {
		t.Fatalf("ExitCode = %d, want 0", result.ExitCode)
	}
	if !strings.Contains(result.Stdout, "hello") {
		t.Fatalf("Stdout = %q, want hello", result.Stdout)
	}
	if result.Duration <= 0 {
		t.Fatal("Duration is zero")
	}
}

func TestRunWithWorkingDirectory(t *testing.T) {
	dir := t.TempDir()
	result, err := runHelper(context.Background(), &Runner{Dir: dir}, "cwd")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.EqualFold(strings.TrimSpace(result.Stdout), dir) {
		t.Fatalf("Stdout = %q, want %s", result.Stdout, dir)
	}
}

func TestRunExitError(t *testing.T) {
	result, err := runHelper(context.Background(), &Runner{}, "exit42")
	if err == nil {
		t.Fatal("Run() error = nil")
	}
	if result.ExitCode != 42 {
		t.Fatalf("ExitCode = %d, want 42", result.ExitCode)
	}
}

func TestRunCancellation(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	_, err := runHelper(ctx, &Runner{}, "sleep")
	if err == nil {
		t.Fatal("Run() error = nil for cancelled command")
	}
	if !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("Run() error = %v, want cancelled", err)
	}
}

func TestRunCancellationTerminatesProcessTree(t *testing.T) {
	pidPath := filepath.Join(t.TempDir(), "child.pid")
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() {
		_, err := runHelper(ctx, &Runner{}, "tree-parent", pidPath)
		done <- err
	}()

	var childPID int
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		data, err := os.ReadFile(pidPath)
		if err == nil {
			childPID, err = strconv.Atoi(strings.TrimSpace(string(data)))
			if err == nil && childPID > 0 {
				break
			}
		}
		time.Sleep(20 * time.Millisecond)
	}
	if childPID == 0 {
		cancel()
		t.Fatal("helper did not report child PID")
	}
	t.Cleanup(func() {
		if processAlive(childPID) {
			if child, err := os.FindProcess(childPID); err == nil {
				_ = child.Kill()
			}
		}
	})

	cancel()
	select {
	case err := <-done:
		if err == nil || !strings.Contains(err.Error(), "cancelled") {
			t.Fatalf("Run() error = %v, want cancellation", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("cancelled process tree did not return")
	}

	deadline = time.Now().Add(2 * time.Second)
	for processAlive(childPID) && time.Now().Before(deadline) {
		time.Sleep(20 * time.Millisecond)
	}
	if processAlive(childPID) {
		t.Fatalf("child process %d survived cancellation", childPID)
	}
}

func TestRunEnvironmentExtendsParentEnvironment(t *testing.T) {
	runner := &Runner{Env: []string{"NODEPAPER_TEST_ENV=visible"}}
	result, err := runHelper(context.Background(), runner, "env")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if strings.TrimSpace(result.Stdout) != "visible" {
		t.Fatalf("Stdout = %q, want visible", result.Stdout)
	}
}

func TestRunCommandNotFound(t *testing.T) {
	_, err := (&Runner{}).Run(context.Background(), "nonexistent-command-xyz")
	if err == nil {
		t.Fatal("Run() error = nil for nonexistent command")
	}
}

func TestResultSummary(t *testing.T) {
	tests := []struct {
		exitCode int
		contains string
	}{
		{0, "[ok]"},
		{1, "[exit 1]"},
		{-1, "exit -1"},
	}

	for _, test := range tests {
		result := Result{ExitCode: test.exitCode, Command: "test"}
		summary := result.Summary()
		if !strings.Contains(summary, test.contains) {
			t.Fatalf("Summary() = %q, want substring %q", summary, test.contains)
		}
	}
}

func TestLogFriendlyCommand(t *testing.T) {
	got := LogFriendlyCommand("echo", []string{"hello", "world"})
	if got != "echo hello world" {
		t.Fatalf("LogFriendlyCommand() = %q, want 'echo hello world'", got)
	}
}

func TestExitCodeOK(t *testing.T) {
	if !ExitCodeOK(Result{ExitCode: 0}) {
		t.Fatal("ExitCodeOK(0) = false")
	}
	if ExitCodeOK(Result{ExitCode: 1}) {
		t.Fatal("ExitCodeOK(1) = true")
	}
}

func TestCaptureSizeLimit(t *testing.T) {
	var writer writerWithLimit
	writer.limit = 16

	writer.Write([]byte("hello"))
	if writer.String() != "hello" {
		t.Fatalf("before limit: %q", writer.String())
	}

	writer.Write([]byte("worldworldworldworld"))
	output := writer.String()
	if !strings.Contains(output, "hello") {
		t.Fatalf("missing prefix: %q", output)
	}
	if !strings.Contains(output, "[output truncated]") {
		t.Fatalf("missing truncation marker: %q", output)
	}
	if len(output) > 100 {
		t.Fatalf("output too large: %d bytes", len(output))
	}

	before := writer.String()
	writer.Write([]byte("zzzzzzzzzzzzzzzzzzzzz"))
	if writer.String() != before {
		t.Fatal("write after truncation should be ignored")
	}
}

func TestStderrCapture(t *testing.T) {
	result, err := runHelper(context.Background(), &Runner{}, "stderr")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if !strings.Contains(result.Stderr, "error") {
		t.Fatalf("Stderr = %q, want 'error'", result.Stderr)
	}
}
