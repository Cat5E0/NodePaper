package main

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"

	"nodepaper/internal/app"
	"nodepaper/internal/cli"
	"nodepaper/internal/output"
	"nodepaper/internal/project"
)

var version = "dev"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	interactive := isTerminal(os.Stdin) && isTerminal(os.Stdout) && os.Getenv("CI") == ""
	os.Exit(runWithIO(ctx, os.Args[1:], os.Stdin, os.Stdout, os.Stderr, interactive, ""))
}

// run is the deterministic non-interactive entry used by command tests.
func run(args []string, stdout, stderr io.Writer) int {
	return runWithIO(context.Background(), args, strings.NewReader(""), stdout, stderr, false, "")
}

func runWithIO(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer, interactive bool, workingDir string) int {
	invocation, err := cli.Parse(args)
	if err != nil {
		writeUsageError(stderr, err)
		return 2
	}

	if invocation.Version {
		fmt.Fprintf(stdout, "nodepaper %s\n", version)
		return 0
	}
	if invocation.Help {
		fmt.Fprint(stdout, cli.Usage)
		return 0
	}
	if invocation.Onboarding {
		writeOnboarding(stdout, workingDir)
		return 0
	}

	if invocation.Command == cli.CommandInit && invocation.ProjectDir == "" {
		if invocation.Format == cli.FormatJSON || invocation.NonInteractive || !interactive {
			writeUsageError(stderr, cli.InitDirectoryRequiredError())
			return 2
		}
		projectDir, generateGuide, ok := promptInit(ctx, stdin, stdout, invocation.AIGuide)
		if !ok {
			fmt.Fprintln(stderr, "nodepaper: initialization canceled; no Project files were created.")
			return 130
		}
		invocation.ProjectDir = projectDir
		invocation.AIGuide = generateGuide
	}

	application := app.New()
	switch invocation.Format {
	case cli.FormatJSON:
		return runJSON(ctx, application, invocation, stdout, stderr)
	default:
		return runText(ctx, application, invocation, stdout, stderr)
	}
}

func writeUsageError(w io.Writer, err error) {
	fmt.Fprintf(w, "nodepaper: %v\n", err)
	var usageErr *cli.UsageError
	if errors.As(err, &usageErr) && usageErr.Suggestion != "" {
		fmt.Fprintln(w, usageErr.Suggestion)
	}
}

func writeOnboarding(w io.Writer, workingDir string) {
	if workingDir == "" {
		var err error
		workingDir, err = os.Getwd()
		if err != nil {
			fmt.Fprintln(w, "NodePaper could not inspect the current directory.")
			fmt.Fprintln(w, "Next: nodepaper doctor")
			fmt.Fprintln(w, "Help: nodepaper --help")
			return
		}
	}
	p, err := project.DiscoverFrom("", workingDir)
	if err == nil {
		fmt.Fprintln(w, "NodePaper Project found:")
		fmt.Fprintf(w, "  %s\n\n", p.Root)
		fmt.Fprintln(w, "Next:")
		fmt.Fprintln(w, "  nodepaper validate")
		fmt.Fprintln(w, "  nodepaper build")
		fmt.Fprintln(w, "  nodepaper clean")
		fmt.Fprintln(w, "  nodepaper --help")
		return
	}

	fmt.Fprintln(w, "No NodePaper Project was found (nodepaper.yaml is missing).")
	fmt.Fprintln(w, "Next:")
	fmt.Fprintln(w, `  nodepaper init <project-directory>`)
	fmt.Fprintln(w, "  nodepaper doctor")
	fmt.Fprintln(w, "  nodepaper --help")
}

func promptInit(ctx context.Context, input io.Reader, output io.Writer, guideAlreadySelected bool) (string, bool, bool) {
	reader := bufio.NewReader(input)
	fmt.Fprint(output, "Project directory (leave empty to cancel): ")
	projectDir, ok := readPromptLine(ctx, reader)
	projectDir = strings.TrimSpace(projectDir)
	if !ok || projectDir == "" {
		return "", false, false
	}
	projectDir = trimMatchingQuotes(projectDir)

	if guideAlreadySelected {
		return projectDir, true, true
	}
	for {
		fmt.Fprint(output, "Generate AI writing guide AGENTS.md? (Y/n): ")
		answer, ok := readPromptLine(ctx, reader)
		if !ok {
			return "", false, false
		}
		switch strings.ToLower(strings.TrimSpace(answer)) {
		case "", "y", "yes":
			return projectDir, true, true
		case "n", "no":
			return projectDir, false, true
		default:
			fmt.Fprintln(output, "Please enter Y or N.")
		}
	}
}

func readPromptLine(ctx context.Context, reader *bufio.Reader) (string, bool) {
	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := reader.ReadString('\n')
		ch <- result{line: line, err: err}
	}()
	select {
	case <-ctx.Done():
		return "", false
	case got := <-ch:
		if got.err != nil && !(errors.Is(got.err, io.EOF) && got.line != "") {
			return "", false
		}
		return strings.TrimRight(got.line, "\r\n"), true
	}
}

func trimMatchingQuotes(value string) string {
	if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
		return value[1 : len(value)-1]
	}
	return value
}

func isTerminal(file *os.File) bool {
	info, err := file.Stat()
	return err == nil && info.Mode()&os.ModeCharDevice != 0
}

func runText(ctx context.Context, application app.App, inv cli.Invocation, stdout, stderr io.Writer) int {
	tw := &output.TextWriter{W: stdout}

	switch inv.Command {
	case cli.CommandInit:
		result, err := application.Init(ctx, app.InitRequest{ProjectDir: inv.ProjectDir, GenerateAIGuide: inv.AIGuide})
		if err != nil {
			fmt.Fprintf(stderr, "nodepaper: %v\n", err)
			return 1
		}
		tw.Init(result)
		return resultExitCode(ctx, result.Diagnostics, result.Success)

	case cli.CommandDoctor:
		result, err := application.Doctor(ctx, app.DoctorRequest{ProjectDir: inv.ProjectDir})
		if err != nil {
			fmt.Fprintf(stderr, "nodepaper: %v\n", err)
			return 1
		}
		tw.Doctor(result)
		return resultExitCode(ctx, result.Diagnostics, result.Success)

	case cli.CommandValidate:
		result, err := application.Validate(ctx, app.ValidateRequest{ProjectDir: inv.ProjectDir})
		if err != nil {
			fmt.Fprintf(stderr, "nodepaper: %v\n", err)
			return 1
		}
		tw.Validate(result)
		return resultExitCode(ctx, result.Diagnostics, result.Success)

	case cli.CommandBuild:
		result, err := application.Build(ctx, app.BuildRequest{ProjectDir: inv.ProjectDir})
		if err != nil {
			fmt.Fprintf(stderr, "nodepaper: %v\n", err)
			return 1
		}
		tw.Build(result)
		return resultExitCode(ctx, result.Diagnostics, result.Success)

	case cli.CommandClean:
		result, err := application.Clean(ctx, app.CleanRequest{ProjectDir: inv.ProjectDir, All: inv.CleanAll})
		if err != nil {
			fmt.Fprintf(stderr, "nodepaper: %v\n", err)
			return 1
		}
		tw.Clean(result)
		return resultExitCode(ctx, result.Diagnostics, result.Success)

	default:
		fmt.Fprintf(stderr, "nodepaper: unknown command %q\n", inv.Command)
		return 2
	}
}

func runJSON(ctx context.Context, application app.App, inv cli.Invocation, stdout, stderr io.Writer) int {
	jw := &output.JSONWriter{W: stdout}

	var err error
	terminalSuccess := false
	switch inv.Command {
	case cli.CommandInit:
		result, appErr := application.Init(ctx, app.InitRequest{ProjectDir: inv.ProjectDir, GenerateAIGuide: inv.AIGuide})
		if appErr != nil {
			err = appErr
		} else {
			err = jw.Init(result)
			terminalSuccess = output.IsTerminalSuccess(result.Diagnostics, result.Success)
		}
	case cli.CommandDoctor:
		result, appErr := application.Doctor(ctx, app.DoctorRequest{ProjectDir: inv.ProjectDir})
		if appErr != nil {
			err = appErr
		} else {
			err = jw.Doctor(result)
			terminalSuccess = output.IsTerminalSuccess(result.Diagnostics, result.Success)
		}
	case cli.CommandValidate:
		result, appErr := application.Validate(ctx, app.ValidateRequest{ProjectDir: inv.ProjectDir})
		if appErr != nil {
			err = appErr
		} else {
			err = jw.Validate(result)
			terminalSuccess = output.IsTerminalSuccess(result.Diagnostics, result.Success)
		}
	case cli.CommandBuild:
		result, appErr := application.Build(ctx, app.BuildRequest{ProjectDir: inv.ProjectDir})
		if appErr != nil {
			err = appErr
		} else {
			err = jw.Build(result)
			terminalSuccess = output.IsTerminalSuccess(result.Diagnostics, result.Success)
		}
	case cli.CommandClean:
		result, appErr := application.Clean(ctx, app.CleanRequest{ProjectDir: inv.ProjectDir, All: inv.CleanAll})
		if appErr != nil {
			err = appErr
		} else {
			err = jw.Clean(result)
			terminalSuccess = output.IsTerminalSuccess(result.Diagnostics, result.Success)
		}
	default:
		fmt.Fprintf(stderr, "nodepaper: unknown command %q\n", inv.Command)
		return 2
	}

	if err != nil {
		fmt.Fprintf(stderr, "nodepaper: %v\n", err)
		if ctx.Err() != nil {
			return 130
		}
		return 1
	}
	if ctx.Err() != nil {
		return 130
	}
	if !terminalSuccess {
		return 1
	}
	return 0
}

func resultExitCode(ctx context.Context, diagnostics []app.Diagnostic, success bool) int {
	if ctx.Err() != nil {
		return 130
	}
	if output.IsTerminalSuccess(diagnostics, success) {
		return 0
	}
	return 1
}
