// Package cli parses NodePaper command-line invocations without implementing
// application behavior.
package cli

import (
	"fmt"
	"strings"
)

const Usage = `Usage:
  nodepaper
  nodepaper init [project-directory] [--ai-guide] [--non-interactive] [--format text|json]
  nodepaper doctor [project-directory] [--format text|json]
  nodepaper validate [project-directory] [--format text|json]
  nodepaper build [project-directory] [--format text|json]
  nodepaper clean [project-directory] [--all] [--format text|json]
  nodepaper --version
  nodepaper --help
`

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

type Command string

const (
	CommandInit     Command = "init"
	CommandDoctor   Command = "doctor"
	CommandValidate Command = "validate"
	CommandBuild    Command = "build"
	CommandClean    Command = "clean"
)

type Invocation struct {
	Command        Command
	ProjectDir     string
	Format         Format
	CleanAll       bool
	AIGuide        bool
	NonInteractive bool
	Onboarding     bool
	Version        bool
	Help           bool
}

// UsageError is a syntax error with a focused next step. The CLI prints the
// suggestion instead of burying the problem under the complete usage text.
type UsageError struct {
	Message    string
	Suggestion string
}

func (e *UsageError) Error() string { return e.Message }

// Parse validates CLI syntax. It performs no filesystem access and does not
// invoke application services.
func Parse(args []string) (Invocation, error) {
	invocation := Invocation{Format: FormatText}
	if len(args) == 0 {
		invocation.Onboarding = true
		return invocation, nil
	}

	for index := 0; index < len(args); index++ {
		argument := args[index]
		switch {
		case argument == "--version":
			invocation.Version = true
		case argument == "--help" || argument == "-h":
			invocation.Help = true
		case argument == "--all":
			invocation.CleanAll = true
		case argument == "--ai-guide":
			invocation.AIGuide = true
		case argument == "--non-interactive":
			invocation.NonInteractive = true
		case argument == "--format":
			index++
			if index >= len(args) {
				return Invocation{}, usageError("--format requires text or json", "Try: nodepaper <command> --format json")
			}
			format, err := parseFormat(args[index])
			if err != nil {
				return Invocation{}, err
			}
			invocation.Format = format
		case strings.HasPrefix(argument, "--format="):
			format, err := parseFormat(strings.TrimPrefix(argument, "--format="))
			if err != nil {
				return Invocation{}, err
			}
			invocation.Format = format
		case strings.HasPrefix(argument, "-"):
			return Invocation{}, usageError(fmt.Sprintf("unknown option %q", argument), "Try: nodepaper --help")
		case invocation.Command == "":
			command, err := parseCommand(argument)
			if err != nil {
				return Invocation{}, err
			}
			invocation.Command = command
		case invocation.ProjectDir == "":
			invocation.ProjectDir = argument
		default:
			return Invocation{}, usageError(fmt.Sprintf("unexpected argument %q", argument), fmt.Sprintf("Try: nodepaper %s [project-directory]", invocation.Command))
		}
	}

	if invocation.Version {
		if invocation.Command != "" || invocation.ProjectDir != "" || invocation.CleanAll || invocation.AIGuide || invocation.NonInteractive || invocation.Help || invocation.Format != FormatText {
			return Invocation{}, usageError("--version cannot be combined with a command or other options", "Try: nodepaper --version")
		}
		return invocation, nil
	}
	if invocation.Command == "" {
		if invocation.Help {
			return invocation, nil
		}
		return Invocation{}, usageError("a command is required", "Try: nodepaper --help")
	}
	if invocation.CleanAll && invocation.Command != CommandClean {
		return Invocation{}, usageError("--all is only valid with clean", "Try: nodepaper clean [project-directory] --all")
	}
	if invocation.AIGuide && invocation.Command != CommandInit {
		return Invocation{}, usageError("--ai-guide is only valid with init", "Try: nodepaper init <project-directory> --ai-guide")
	}
	if invocation.NonInteractive && invocation.Command != CommandInit {
		return Invocation{}, usageError("--non-interactive is only valid with init", "Try: nodepaper init <project-directory> --non-interactive")
	}

	return invocation, nil
}

// InitDirectoryRequiredError is used after TTY detection when an init command
// without a path cannot enter the interactive flow.
func InitDirectoryRequiredError() error {
	return usageError("init requires a project directory in non-interactive mode", "Try: nodepaper init <project-directory>")
}

func parseCommand(value string) (Command, error) {
	command := Command(value)
	switch command {
	case CommandInit, CommandDoctor, CommandValidate, CommandBuild, CommandClean:
		return command, nil
	default:
		suggestion := "Try: nodepaper --help"
		if nearest := nearestCommand(value); nearest != "" {
			suggestion = fmt.Sprintf("Did you mean: nodepaper %s [project-directory]", nearest)
		}
		return "", usageError(fmt.Sprintf("unknown command %q", value), suggestion)
	}
}

func nearestCommand(value string) Command {
	for _, command := range []Command{CommandInit, CommandDoctor, CommandValidate, CommandBuild, CommandClean} {
		if editDistance(strings.ToLower(value), string(command)) <= 2 {
			return command
		}
	}
	return ""
}

func editDistance(a, b string) int {
	previous := make([]int, len(b)+1)
	for j := range previous {
		previous[j] = j
	}
	for i := 1; i <= len(a); i++ {
		current := make([]int, len(b)+1)
		current[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			current[j] = min(current[j-1]+1, previous[j]+1, previous[j-1]+cost)
		}
		previous = current
	}
	return previous[len(b)]
}

func parseFormat(value string) (Format, error) {
	format := Format(value)
	switch format {
	case FormatText, FormatJSON:
		return format, nil
	default:
		return "", usageError(fmt.Sprintf("unsupported format %q; use text or json", value), "Try: nodepaper <command> --format json")
	}
}

func usageError(message, suggestion string) error {
	return &UsageError{Message: message, Suggestion: suggestion}
}
