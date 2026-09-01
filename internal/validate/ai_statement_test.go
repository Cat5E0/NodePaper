package validate

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestValidateAcceptsAIStatementWithoutPaperMetadata(t *testing.T) {
	// The supporting document deliberately carries none of what a paper must
	// have: no problem number, no keywords, no 摘要 section. Only a title.
	projectDir := fragmentProject(t, "aiUsage: true\naiUsagePurpose: 语言润色、代码调试\naiStatement: ai-usage.md\n")
	writeStatement(t, projectDir, "---\ntitle: AI工具使用详情\n---\n\n# 工具清单\n\n（填写）\n")

	result := Run(context.Background(), projectDir)
	if !result.Success {
		t.Fatalf("Run() diagnostics = %#v", result.Diagnostics)
	}
	for _, code := range []string{"NP2101", "NP2103", "NP2104", "NP2201", "NP2603"} {
		if diagnosticsContain(result, code) {
			t.Fatalf("Run() reported %s against the AI statement; diagnostics = %#v", code, result.Diagnostics)
		}
	}
}

func TestValidateReportsMissingAIStatement(t *testing.T) {
	projectDir := fragmentProject(t, "aiUsage: true\naiUsagePurpose: 语言润色、代码调试\naiStatement: ai-usage.md\n")

	result := Run(context.Background(), projectDir)
	if result.Success {
		t.Fatal("Run() succeeded with aiStatement pointing at a file that does not exist")
	}
	if !diagnosticsContain(result, "NP2602") {
		t.Fatalf("Run() diagnostics = %#v, want NP2602", result.Diagnostics)
	}
}

func TestValidateWarnsWhenAIStatementHasNoTitle(t *testing.T) {
	// Without a title the Profile template still starts the body on a fresh
	// page, so the document opens on a page carrying nothing. That is worth a
	// warning and not an error: it builds, and the author may have meant it.
	projectDir := fragmentProject(t, "aiUsage: true\naiUsagePurpose: 语言润色、代码调试\naiStatement: ai-usage.md\n")
	writeStatement(t, projectDir, "# 工具清单\n\n（填写）\n")

	result := Run(context.Background(), projectDir)
	if !result.Success {
		t.Fatalf("Run() failed on a titleless AI statement; diagnostics = %#v", result.Diagnostics)
	}
	if !diagnosticsContain(result, "NP2603") {
		t.Fatalf("Run() diagnostics = %#v, want NP2603", result.Diagnostics)
	}
}

func TestValidateReportsAIStatementOutsideProject(t *testing.T) {
	projectDir := fragmentProject(t, "aiUsage: true\naiUsagePurpose: 语言润色、代码调试\naiStatement: ../ai-usage.md\n")
	if err := os.WriteFile(filepath.Join(filepath.Dir(projectDir), "ai-usage.md"), []byte("# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result := Run(context.Background(), projectDir)
	if result.Success {
		t.Fatal("Run() succeeded with an aiStatement outside the Project")
	}
	if !diagnosticsContain(result, "NP2601") {
		t.Fatalf("Run() diagnostics = %#v, want NP2601", result.Diagnostics)
	}
}

func TestValidateIgnoresAbsentAIStatement(t *testing.T) {
	// A team that used no AI tool leaves the field out. The Markdown file that
	// `nodepaper init` drops in the Project must not be validated or built.
	projectDir := fragmentProject(t, "")
	writeStatement(t, projectDir, "not even valid front matter\n")

	result := Run(context.Background(), projectDir)
	if !result.Success {
		t.Fatalf("Run() diagnostics = %#v", result.Diagnostics)
	}
	if diagnosticsContain(result, "NP2603") {
		t.Fatalf("Run() inspected an AI statement the configuration never declared: %#v", result.Diagnostics)
	}
}

func writeStatement(t *testing.T, projectDir, content string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(projectDir, "ai-usage.md"), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
