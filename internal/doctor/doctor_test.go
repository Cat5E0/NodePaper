package doctor

import (
	"context"
	"strings"
	"testing"
)

func TestRunSuccessMatchesFailingChecks(t *testing.T) {
	result := Run(context.Background(), "", "")
	hasFailure := false
	for _, check := range result.Checks {
		if check.Status == StatusFail {
			hasFailure = true
		}
	}
	if result.Success == hasFailure {
		t.Fatalf("Success = %v with hasFailure = %v; checks = %#v", result.Success, hasFailure, result.Checks)
	}
}

func TestFormatChecks(t *testing.T) {
	checks := []Check{
		{Name: "pandoc", Status: StatusPass, Message: "ok"},
		{Name: "latexmk", Status: StatusWarning, Message: "untested", Suggestion: "verify version"},
		{Name: "xelatex", Status: StatusFail, Message: "missing"},
		{Name: "probe", Status: StatusSkipped, Message: "blocked"},
	}
	output := FormatChecks(checks)
	for _, expected := range []string{"PASS", "WARN", "FAIL", "SKIP", "verify version"} {
		if !strings.Contains(output, expected) {
			t.Fatalf("FormatChecks() output missing %q:\n%s", expected, output)
		}
	}
}
