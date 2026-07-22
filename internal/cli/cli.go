// Package cli parses NodePaper command-line invocations without implementing
// application behavior.
package cli

import (
	"fmt"
	"strings"
)

const Usage = `Usage:
  nodepaper init <project-directory> [--format text|json]
  nodepaper doctor [project-directory] [--format text|json]
  nodepaper validate [project-directory] [--format text|json]
  nodepaper build [project-directory] [--format text|json]
  nodepaper clean [project-directory] [--all] [--format text|json]
  nodepaper --version
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
	Command    Command
	ProjectDir string
	Format     Format
	CleanAll   bool
	Version    bool
	Help       bool
}

// Parse validates CLI syntax. It performs no filesystem access and does not
// invoke application services.
func Parse(args []string) (Invocation, error) {
	invocation := Invocation{Format: FormatText}

	for index := 0; index < len(args); index++ {
		argument := args[index]
		switch {
		case argument == "--version":
			invocation.Version = true
		case argument == "--help" || argument == "-h":
			invocation.Help = true
		case argument == "--all":
			invocation.CleanAll = true
		case argument == "--format":
			index++
			if index >= len(args) {
				return Invocation{}, fmt.Errorf("--format requires text or json")
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
			return Invocation{}, fmt.Errorf("unknown option %q", argument)
		case invocation.Command == "":
			command, err := parseCommand(argument)
			if err != nil {
				return Invocation{}, err
			}
			invocation.Command = command
		case invocation.ProjectDir == "":
			invocation.ProjectDir = argument
		default:
			return Invocation{}, fmt.Errorf("unexpected argument %q", argument)
		}
	}

	if len(args) == 0 {
		invocation.Help = true
	}
	if invocation.Version {
		if invocation.Command != "" || invocation.ProjectDir != "" || invocation.CleanAll || invocation.Help || invocation.Format != FormatText {
			return Invocation{}, fmt.Errorf("--version cannot be combined with a command or other options")
		}
		return invocation, nil
	}
	if invocation.Command == "" {
		if invocation.Help {
			return invocation, nil
		}
		return Invocation{}, fmt.Errorf("a command is required")
	}
	if invocation.Command == CommandInit && invocation.ProjectDir == "" && !invocation.Help {
		return Invocation{}, fmt.Errorf("init requires a project directory")
	}
	if invocation.CleanAll && invocation.Command != CommandClean {
		return Invocation{}, fmt.Errorf("--all is only valid with clean")
	}

	return invocation, nil
}

func parseCommand(value string) (Command, error) {
	command := Command(value)
	switch command {
	case CommandInit, CommandDoctor, CommandValidate, CommandBuild, CommandClean:
		return command, nil
	default:
		return "", fmt.Errorf("unknown command %q", value)
	}
}

func parseFormat(value string) (Format, error) {
	format := Format(value)
	switch format {
	case FormatText, FormatJSON:
		return format, nil
	default:
		return "", fmt.Errorf("unsupported format %q; use text or json", value)
	}
}
