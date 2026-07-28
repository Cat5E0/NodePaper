package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"nodepaper/internal/app"
	"nodepaper/internal/cli"
	"nodepaper/internal/output"
)

var version = "dev"

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	invocation, err := cli.Parse(args)
	if err != nil {
		fmt.Fprintf(stderr, "nodepaper: %v\n\n%s", err, cli.Usage)
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

	application := app.New()

	switch invocation.Format {
	case cli.FormatJSON:
		return runJSON(application, invocation, stdout, stderr)
	default:
		return runText(application, invocation, stdout, stderr)
	}
}

func runText(application app.App, inv cli.Invocation, stdout, stderr io.Writer) int {
	ctx := context.Background()
	tw := &output.TextWriter{W: stdout}

	switch inv.Command {
	case cli.CommandInit:
		result, err := application.Init(ctx, app.InitRequest{ProjectDir: inv.ProjectDir})
		if err != nil {
			fmt.Fprintf(stderr, "nodepaper: %v\n", err)
			return 1
		}
		tw.Init(result)
		if output.IsTerminalSuccess(result.Diagnostics, result.Success) {
			return 0
		}
		return 1

	case cli.CommandDoctor:
		result, err := application.Doctor(ctx, app.DoctorRequest{ProjectDir: inv.ProjectDir})
		if err != nil {
			fmt.Fprintf(stderr, "nodepaper: %v\n", err)
			return 1
		}
		tw.Doctor(result)
		if output.IsTerminalSuccess(result.Diagnostics, result.Success) {
			return 0
		}
		return 1

	case cli.CommandValidate:
		result, err := application.Validate(ctx, app.ValidateRequest{ProjectDir: inv.ProjectDir})
		if err != nil {
			fmt.Fprintf(stderr, "nodepaper: %v\n", err)
			return 1
		}
		tw.Validate(result)
		if output.IsTerminalSuccess(result.Diagnostics, result.Success) {
			return 0
		}
		return 1

	case cli.CommandBuild:
		result, err := application.Build(ctx, app.BuildRequest{ProjectDir: inv.ProjectDir})
		if err != nil {
			fmt.Fprintf(stderr, "nodepaper: %v\n", err)
			return 1
		}
		tw.Build(result)
		if output.IsTerminalSuccess(result.Diagnostics, result.Success) {
			return 0
		}
		return 1

	case cli.CommandClean:
		result, err := application.Clean(ctx, app.CleanRequest{ProjectDir: inv.ProjectDir, All: inv.CleanAll})
		if err != nil {
			fmt.Fprintf(stderr, "nodepaper: %v\n", err)
			return 1
		}
		tw.Clean(result)
		if output.IsTerminalSuccess(result.Diagnostics, result.Success) {
			return 0
		}
		return 1

	default:
		fmt.Fprintf(stderr, "nodepaper: unknown command %q\n", inv.Command)
		return 1
	}
}

func runJSON(application app.App, inv cli.Invocation, stdout, stderr io.Writer) int {
	ctx := context.Background()
	jw := &output.JSONWriter{W: stdout}

	var err error
	terminalSuccess := false
	switch inv.Command {
	case cli.CommandInit:
		result, appErr := application.Init(ctx, app.InitRequest{ProjectDir: inv.ProjectDir})
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
		return 1
	}

	if err != nil {
		fmt.Fprintf(stderr, "nodepaper: %v\n", err)
		return 1
	}
	if !terminalSuccess {
		return 1
	}
	return 0
}
