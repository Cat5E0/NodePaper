package fragment

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInspectAcceptsSafeFragmentsAndHashes(t *testing.T) {
	root := t.TempDir()
	writeFragment(t, root, "tables/result.tex", "\\begin{longtable}{ll}\na & b\\\\\n\\end{longtable}\n")
	files, issues := Inspect(root, []string{"tables/result.tex"})
	if len(issues) != 0 || len(files) != 1 {
		t.Fatalf("Inspect() files=%#v issues=%#v", files, issues)
	}
	if files[0].Relative != "tables/result.tex" || len(files[0].SHA256) != 64 {
		t.Fatalf("file = %#v", files[0])
	}
	if issues := Verify(files); len(issues) != 0 {
		t.Fatalf("Verify() = %#v", issues)
	}
}

func TestInspectRejectsUnsafeDeclarationsAndCommands(t *testing.T) {
	root := t.TempDir()
	writeFragment(t, root, "bad/document.tex", "\\documentclass{article}\n")
	writeFragment(t, root, "bad/nested.tex", "\\input{other.tex}\n")
	writeFragment(t, root, "bad/execute.tex", "\\immediate\\write18{calc}\n")
	writeFragment(t, root, "bad/obfuscated.tex", "\\in% hidden\nput{other.tex}\n")
	writeFragment(t, root, "bad/external-read.tex", "\\includegraphics{../outside.png}\n")
	writeFragment(t, root, "bad/commented.tex", "% \\input{ignored.tex}\nSafe \\% text\n")

	tests := []struct {
		name  string
		paths []string
		code  string
	}{
		{"absolute", []string{filepath.Join(root, "bad", "document.tex")}, CodeInvalidDeclaration},
		{"escape", []string{"../outside.tex"}, CodePathEscape},
		{"wrong extension", []string{"bad/file.md"}, CodeInvalidDeclaration},
		{"missing", []string{"bad/missing.tex"}, CodeMissing},
		{"duplicate", []string{"bad/commented.tex", "bad/commented.tex"}, CodeDuplicate},
		{"document command", []string{"bad/document.tex"}, CodeDocumentCommand},
		{"nested dependency", []string{"bad/nested.tex"}, CodeNestedDependency},
		{"obfuscated dependency", []string{"bad/obfuscated.tex"}, CodeNestedDependency},
		{"external file read", []string{"bad/external-read.tex"}, CodeNestedDependency},
		{"execution", []string{"bad/execute.tex"}, CodeCommandExecution},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			_, issues := Inspect(root, test.paths)
			if !hasIssue(issues, test.code) {
				t.Fatalf("issues = %#v, want %s", issues, test.code)
			}
		})
	}

	files, issues := Inspect(root, []string{"bad/commented.tex"})
	if len(issues) != 0 || len(files) != 1 {
		t.Fatalf("comment-only command was rejected: files=%#v issues=%#v", files, issues)
	}
}

func TestVerifyDetectsChange(t *testing.T) {
	root := t.TempDir()
	writeFragment(t, root, "equations/model.tex", "x=1\n")
	files, issues := Inspect(root, []string{"equations/model.tex"})
	if len(issues) != 0 {
		t.Fatal(issues)
	}
	writeFragment(t, root, "equations/model.tex", "x=2\n")
	if issues := Verify(files); !hasIssue(issues, CodeChanged) {
		t.Fatalf("Verify() = %#v, want %s", issues, CodeChanged)
	}
}

func TestInspectRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.tex")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "escape.tex")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	_, issues := Inspect(root, []string{"escape.tex"})
	if !hasIssue(issues, CodeSymlinkEscape) {
		t.Fatalf("issues = %#v, want %s", issues, CodeSymlinkEscape)
	}
}

func writeFragment(t *testing.T, root, relative, content string) {
	t.Helper()
	path := filepath.Join(root, filepath.FromSlash(relative))
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func hasIssue(issues []Issue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}
