package validate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateAcceptsDeclaredFragmentInput(t *testing.T) {
	projectDir := fragmentProject(t, `latexFragments:
  - tables/result.tex
`)
	if err := os.MkdirAll(filepath.Join(projectDir, "tables"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "tables", "result.tex"), []byte("\\begin{longtable}{ll}\na & b\\\\\n\\end{longtable}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	appendPaper(t, projectDir, "\n\\input{tables/result.tex}\n")
	result := Run(context.Background(), projectDir)
	if !result.Success {
		t.Fatalf("Run() diagnostics = %#v", result.Diagnostics)
	}
}

func TestValidateRejectsUndeclaredFragmentInput(t *testing.T) {
	projectDir := fragmentProject(t, "")
	appendPaper(t, projectDir, "\n\\input{tables/result.tex}\n")
	result := Run(context.Background(), projectDir)
	if !diagnosticsContain(result, "NP2509") {
		t.Fatalf("Run() diagnostics = %#v, want NP2509", result.Diagnostics)
	}
}

func TestValidateRejectsUnsafeRawTexCommand(t *testing.T) {
	for _, source := range []string{
		"\n\\csname write18\\endcsname{unsafe}\n",
		"\n\\in% hidden\nclude{outside.tex}\n",
		"\nEscaped percent \\% does not hide \\write18{unsafe}\n",
	} {
		projectDir := fragmentProject(t, "")
		appendPaper(t, projectDir, source)
		result := Run(context.Background(), projectDir)
		if !diagnosticsContain(result, "NP2508") {
			t.Fatalf("Run() diagnostics = %#v, want NP2508 for %q", result.Diagnostics, source)
		}
	}
}

func TestValidateReportsUnsafeFragment(t *testing.T) {
	projectDir := fragmentProject(t, `latexFragments:
  - tables/result.tex
`)
	if err := os.MkdirAll(filepath.Join(projectDir, "tables"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projectDir, "tables", "result.tex"), []byte("\\input{hidden.tex}\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	result := Run(context.Background(), projectDir)
	if !diagnosticsContain(result, "NP2507") {
		t.Fatalf("Run() diagnostics = %#v, want NP2507", result.Diagnostics)
	}
}

func fragmentProject(t *testing.T, configSuffix string) string {
	t.Helper()
	repo := repositoryRoot(t)
	projectDir := filepath.Join(t.TempDir(), "project")
	copyTree(t, filepath.Join(repo, "nodepaper-test-fixtures", "tests", "fixtures", "minimal-valid"), projectDir)
	config := "version: 1\nprofile: cumcm\nsource: paper.md\n" + configSuffix
	if err := os.WriteFile(filepath.Join(projectDir, "nodepaper.yaml"), []byte(config), 0o644); err != nil {
		t.Fatal(err)
	}
	return projectDir
}

func appendPaper(t *testing.T, projectDir, content string) {
	t.Helper()
	path := filepath.Join(projectDir, "paper.md")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if _, err := file.WriteString(content); err != nil {
		t.Fatal(err)
	}
}

func diagnosticsContain(result Result, code string) bool {
	for _, diag := range result.Diagnostics {
		if diag.Code == code {
			return true
		}
	}
	return false
}
