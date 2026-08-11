package cli

import (
	"errors"
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
		{"interactive init candidate", []string{"init"}, Invocation{Command: CommandInit, Format: FormatText}},
		{"init ai guide", []string{"init", `D:\papers\new`, "--ai-guide"}, Invocation{Command: CommandInit, ProjectDir: `D:\papers\new`, Format: FormatText, AIGuide: true}},
		{"init non-interactive", []string{"init", "--non-interactive"}, Invocation{Command: CommandInit, Format: FormatText, NonInteractive: true}},
		{"doctor global", []string{"doctor"}, Invocation{Command: CommandDoctor, Format: FormatText}},
		{"validate project", []string{"validate", `D:\papers\a`}, Invocation{Command: CommandValidate, ProjectDir: `D:\papers\a`, Format: FormatText}},
		{"build json", []string{"build", `D:\papers\a`, "--format", "json"}, Invocation{Command: CommandBuild, ProjectDir: `D:\papers\a`, Format: FormatJSON}},
		{"format before command", []string{"--format=json", "build", `D:\papers\a`}, Invocation{Command: CommandBuild, ProjectDir: `D:\papers\a`, Format: FormatJSON}},
		{"clean all", []string{"clean", `D:\papers\a`, "--all"}, Invocation{Command: CommandClean, ProjectDir: `D:\papers\a`, Format: FormatText, CleanAll: true}},
		{"version", []string{"--version"}, Invocation{Format: FormatText, Version: true}},
		{"onboarding", nil, Invocation{Format: FormatText, Onboarding: true}},
		{"root help", []string{"--help"}, Invocation{Format: FormatText, Help: true}},
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
		name       string
		args       []string
		message    string
		suggestion string
	}{
		{"unknown command", []string{"publish"}, "unknown command", "nodepaper --help"},
		{"misspelled command", []string{"buid"}, "unknown command", "nodepaper build"},
		{"unknown option", []string{"build", "--quiet"}, "unknown option", "nodepaper --help"},
		{"missing format", []string{"build", "--format"}, "requires text or json", "--format json"},
		{"invalid format", []string{"build", "--format", "yaml"}, "unsupported format", "--format json"},
		{"extra project", []string{"build", "one", "two"}, "unexpected argument", "nodepaper build"},
		{"all on build", []string{"build", "--all"}, "only valid with clean", "nodepaper clean"},
		{"ai guide on build", []string{"build", "--ai-guide"}, "only valid with init", "nodepaper init"},
		{"non interactive on build", []string{"build", "--non-interactive"}, "only valid with init", "nodepaper init"},
		{"version with command", []string{"build", "--version"}, "cannot be combined", "nodepaper --version"},
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
			var usageErr *UsageError
			if !errors.As(err, &usageErr) {
				t.Fatalf("Parse() error type = %T, want *UsageError", err)
			}
			if !strings.Contains(usageErr.Suggestion, test.suggestion) {
				t.Fatalf("suggestion = %q, want substring %q", usageErr.Suggestion, test.suggestion)
			}
		})
	}
}

func TestInitDirectoryRequiredError(t *testing.T) {
	err := InitDirectoryRequiredError()
	if !strings.Contains(err.Error(), "non-interactive") {
		t.Fatalf("error = %q", err)
	}
}
