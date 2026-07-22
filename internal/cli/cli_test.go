package cli

import (
	"strings"
	"testing"
)

func TestParseCommands(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want Invocation
	}{
		{"init", []string{"init", `D:\papers\new`}, Invocation{Command: CommandInit, ProjectDir: `D:\papers\new`, Format: FormatText}},
		{"doctor global", []string{"doctor"}, Invocation{Command: CommandDoctor, Format: FormatText}},
		{"validate project", []string{"validate", `D:\papers\a`}, Invocation{Command: CommandValidate, ProjectDir: `D:\papers\a`, Format: FormatText}},
		{"build json", []string{"build", `D:\papers\a`, "--format", "json"}, Invocation{Command: CommandBuild, ProjectDir: `D:\papers\a`, Format: FormatJSON}},
		{"format before command", []string{"--format=json", "build", `D:\papers\a`}, Invocation{Command: CommandBuild, ProjectDir: `D:\papers\a`, Format: FormatJSON}},
		{"clean all", []string{"clean", `D:\papers\a`, "--all"}, Invocation{Command: CommandClean, ProjectDir: `D:\papers\a`, Format: FormatText, CleanAll: true}},
		{"version", []string{"--version"}, Invocation{Format: FormatText, Version: true}},
		{"root help", nil, Invocation{Format: FormatText, Help: true}},
		{"command help", []string{"build", "--help"}, Invocation{Command: CommandBuild, Format: FormatText, Help: true}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			got, err := Parse(test.args)
			if err != nil {
				t.Fatalf("Parse() error = %v", err)
			}
			if got != test.want {
				t.Fatalf("Parse() = %#v, want %#v", got, test.want)
			}
		})
	}
}

func TestParseRejectsInvalidInvocations(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		message string
	}{
		{"unknown command", []string{"publish"}, "unknown command"},
		{"unknown option", []string{"build", "--quiet"}, "unknown option"},
		{"missing format", []string{"build", "--format"}, "requires text or json"},
		{"invalid format", []string{"build", "--format", "yaml"}, "unsupported format"},
		{"extra project", []string{"build", "one", "two"}, "unexpected argument"},
		{"init without project", []string{"init"}, "requires a project directory"},
		{"all on build", []string{"build", "--all"}, "only valid with clean"},
		{"version with command", []string{"build", "--version"}, "cannot be combined"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, err := Parse(test.args)
			if err == nil {
				t.Fatal("Parse() error = nil")
			}
			if !strings.Contains(err.Error(), test.message) {
				t.Fatalf("Parse() error = %q, want substring %q", err, test.message)
			}
		})
	}
}
