package export

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A machine with no TeX is exactly the case `nodepaper export` exists for, so a
// Project that declares an AI tool usage statement has to get that document out
// of the export too - otherwise the rules-mandated material is reachable only
// from a machine that can already build it locally.
func TestExportCarriesTheAIStatement(t *testing.T) {
	projectDir := fixtureProject(t)
	enableAIStatement(t, projectDir)
	target := filepath.Join(t.TempDir(), "out")

	executor := &fakeExecutor{}
	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target, Bib: BibBibTeX}, executor)
	if !result.Success {
		t.Fatalf("export failed: %#v", result.Diagnostics)
	}

	statementTex := filepath.Join(target, "ai-statement.tex")
	if _, err := os.Stat(statementTex); err != nil {
		t.Fatalf("ai-statement.tex was not delivered: %v", err)
	}
	if !hasArtifact(result, "ai-statement-tex") {
		t.Fatalf("result does not report the AI statement artifact: %#v", result.Artifacts)
	}

	// The statement is converted from its own Source list, not from the paper's.
	statementCalls := 0
	for _, call := range executor.calls {
		if argumentValue(call.args, "-Output") == "" {
			continue
		}
		manifest := argumentValue(call.args, "-SourceManifest")
		if strings.Contains(filepath.Base(manifest), "ai-statement") {
			statementCalls++
			data, err := os.ReadFile(manifest)
			if err == nil && strings.Contains(string(data), "paper.md") {
				t.Fatalf("the AI statement manifest carries the paper sources: %s", data)
			}
		}
	}
	if statementCalls != 1 {
		t.Fatalf("AI statement conversions = %d, want 1", statementCalls)
	}

	readmeText, err := os.ReadFile(filepath.Join(target, "README.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readmeText), "ai-statement.tex") {
		t.Fatalf("README.txt does not mention the second document:\n%s", readmeText)
	}
}

func TestExportWithoutAIStatementDeliversPaperOnly(t *testing.T) {
	projectDir := fixtureProject(t)
	target := filepath.Join(t.TempDir(), "out")

	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target, Bib: BibBibTeX}, &fakeExecutor{})
	if !result.Success {
		t.Fatalf("export failed: %#v", result.Diagnostics)
	}
	if _, err := os.Stat(filepath.Join(target, "ai-statement.tex")); !os.IsNotExist(err) {
		t.Fatalf("ai-statement.tex was delivered for a Project that declares none; err=%v", err)
	}
	readmeText, err := os.ReadFile(filepath.Join(target, "README.txt"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(readmeText), "ai-statement.tex") {
		t.Fatalf("README.txt describes a document that was not exported:\n%s", readmeText)
	}
}

// The recipient compiles by hand from README.txt, and the same list drives the
// terminal's next-step hint and --verify. A second delivered document that no
// chain ever mentions is a document the recipient does not know to build.
func TestExportCompileChainCoversTheAIStatement(t *testing.T) {
	projectDir := fixtureProject(t)
	enableAIStatement(t, projectDir)
	target := filepath.Join(t.TempDir(), "out")

	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target, Bib: BibBibTeX}, &fakeExecutor{})
	if !result.Success {
		t.Fatalf("export failed: %#v", result.Diagnostics)
	}

	statementRuns := 0
	for _, command := range result.CompileCommands {
		if strings.Contains(command, "ai-statement.tex") {
			statementRuns++
		}
		if strings.Contains(command, "bibtex ai-statement") || strings.Contains(command, "biber ai-statement") {
			t.Fatalf("the statement cites nothing but the chain runs a bibliography pass on it: %q", command)
		}
	}
	if statementRuns != 2 {
		t.Fatalf("compile chain runs xelatex on the statement %d times, want 2: %#v", statementRuns, result.CompileCommands)
	}

	readmeText, err := os.ReadFile(filepath.Join(target, "README.txt"))
	if err != nil {
		t.Fatal(err)
	}
	// The published name is prescribed by the rules; the exported .tex is ASCII,
	// so the recipient has to be told to rename the result.
	for _, required := range []string{"ai-statement.pdf", "AI工具使用详情.pdf", "rename"} {
		if !strings.Contains(string(readmeText), required) {
			t.Fatalf("README.txt does not tell the recipient about %q:\n%s", required, readmeText)
		}
	}
}

func TestExportCompileChainStaysPaperOnlyWithoutAIStatement(t *testing.T) {
	projectDir := fixtureProject(t)
	target := filepath.Join(t.TempDir(), "out")

	result := runExport(t, Options{ProjectDir: projectDir, ToPath: target, Bib: BibBibTeX}, &fakeExecutor{})
	if !result.Success {
		t.Fatalf("export failed: %#v", result.Diagnostics)
	}
	for _, command := range result.CompileCommands {
		if strings.Contains(command, "ai-statement") {
			t.Fatalf("compile chain mentions a document that was not exported: %q", command)
		}
	}
}

func enableAIStatement(t *testing.T, projectDir string) {
	t.Helper()
	configPath := filepath.Join(projectDir, "nodepaper.yaml")
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(configPath, append(data, []byte("\naiStatement: ai-usage.md\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasArtifact(result Result, kind string) bool {
	for _, artifact := range result.Artifacts {
		if artifact.Kind == kind {
			return true
		}
	}
	return false
}
