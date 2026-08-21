// Package process runs external commands with controlled working directory,
// output capture, context cancellation and structured logging.
package process

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"time"
)

// Result captures the outcome of a completed or cancelled external command.
type Result struct {
	Command  string
	Args     []string
	Dir      string
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
}

// Runner holds configuration for external command execution.
type Runner struct {
	// Dir is the working directory for child processes. When empty the
	// process inherits the Go process working directory.
	Dir string

	// Env appends to the child environment. By default the child inherits
	// the parent environment.
	Env []string

	// CaptureSize is the maximum combined stdout+stderr size per command.
	// Zero means unlimited. When the buffer exceeds this limit output is
	// truncated and a warning is included in the captured text.
	CaptureSize int64
}

// Run executes a command and waits for completion. When ctx is cancelled the
// command process is terminated and a non-zero exit code is returned.
func (r *Runner) Run(ctx context.Context, command string, args ...string) (Result, error) {
	cmd := newCommand(ctx, command, args...)
	cmd.Dir = r.Dir
	if len(r.Env) > 0 {
		cmd.Env = r.buildEnv()
	}

	var stdout, stderr writerWithLimit
	if r.CaptureSize > 0 {
		stdout.limit = r.CaptureSize
		stderr.limit = r.CaptureSize
	}

	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	result := Result{
		Command:  command,
		Args:     args,
		Dir:      r.Dir,
		Duration: duration,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}

	if err != nil {
		if ctx.Err() != nil {
			result.ExitCode = -1
			return result, fmt.Errorf("command %q cancelled: %w", command, ctx.Err())
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		return result, fmt.Errorf("command %q failed: %w", command, err)
	}

	result.ExitCode = 0
	return result, nil
}

func newCommand(ctx context.Context, command string, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, command, args...)
	configureProcessTreeCancellation(cmd)
	return cmd
}

func (r *Runner) buildEnv() []string {
	// Start with os.Environ() instead of nil so child always inherits PATH.
	env := append([]string{}, execEnv()...)
	env = append(env, r.Env...)
	return env
}

// execEnv is a testable indirection for os.Environ.
var execEnv = os.Environ

// writerWithLimit wraps bytes.Buffer with an optional size cap.
type writerWithLimit struct {
	bytes.Buffer
	limit   int64
	written int64
}

func (w *writerWithLimit) Write(p []byte) (int, error) {
	if w.limit > 0 && w.written >= w.limit {
		return len(p), nil
	}

	n := len(p)
	remaining := w.limit - w.written
	if w.limit > 0 && int64(n) > remaining {
		p = p[:remaining]
	}

	written, err := w.Buffer.Write(p)
	w.written += int64(written)
	if w.limit > 0 && w.written >= w.limit {
		w.Buffer.WriteString("\n[output truncated]")
	}
	return n, err
}

// LogFriendlyCommand returns a human-readable representation of a command
// suitable for embedding in log output or diagnostic messages.
func LogFriendlyCommand(command string, args []string) string {
	s := command
	for _, a := range args {
		s += " " + a
	}
	return s
}

// ExitCodeOK returns true when the result has exit code 0.
func ExitCodeOK(r Result) bool {
	return r.ExitCode == 0
}

// Summary returns a one-line description of the command outcome.
func (r Result) Summary() string {
	status := "ok"
	if r.ExitCode != 0 {
		status = fmt.Sprintf("exit %d", r.ExitCode)
	}
	return fmt.Sprintf("[%s] %s (%s)", status, LogFriendlyCommand(r.Command, r.Args), r.Duration.Round(time.Millisecond))
}

// Go is a convenience that runs a Go tool command (e.g. "vet", "./...") using
// the same Runner config.
func (r *Runner) Go(ctx context.Context, args ...string) (Result, error) {
	return r.Run(ctx, "go", args...)
}

// Powershell runs a PowerShell script with -NoProfile.
func (r *Runner) Powershell(ctx context.Context, scriptPath string, args ...string) (Result, error) {
	all := append([]string{"-NoProfile", "-File", scriptPath}, args...)
	return r.Run(ctx, "powershell.exe", all...)
}

// Tee runs the command and additionally writes stdout to the supplied writer
// while still capturing it in the result.
func (r *Runner) Tee(ctx context.Context, w io.Writer, command string, args ...string) (Result, error) {
	cmd := newCommand(ctx, command, args...)
	cmd.Dir = r.Dir
	if len(r.Env) > 0 {
		cmd.Env = r.buildEnv()
	}

	var stdout, stderr writerWithLimit
	if r.CaptureSize > 0 {
		stdout.limit = r.CaptureSize
		stderr.limit = r.CaptureSize
	}

	cmd.Stdout = io.MultiWriter(&stdout, w)
	cmd.Stderr = &stderr

	start := time.Now()
	err := cmd.Run()
	duration := time.Since(start)

	result := Result{
		Command:  command,
		Args:     args,
		Dir:      r.Dir,
		Duration: duration,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
	}

	if err != nil {
		if ctx.Err() != nil {
			result.ExitCode = -1
			return result, fmt.Errorf("command %q cancelled: %w", command, ctx.Err())
		}
		if exitErr, ok := err.(*exec.ExitError); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = -1
		}
		return result, fmt.Errorf("command %q failed: %w", command, err)
	}

	result.ExitCode = 0
	return result, nil
}
