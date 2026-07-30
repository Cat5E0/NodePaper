package project

import (
	"path/filepath"
	"strings"
	"testing"
)

func FuzzResolveNeverReturnsLexicalEscape(f *testing.F) {
	for _, seed := range []string{
		"paper.md",
		"sections/01.md",
		"../outside.md",
		"..\\outside.md",
		"",
		"a b/中文.md",
	} {
		f.Add(seed)
	}
	root := f.TempDir()
	project := Project{Root: root}
	f.Fuzz(func(t *testing.T, input string) {
		resolved, err := project.Resolve(input)
		if err != nil {
			return
		}
		relative, err := filepath.Rel(root, resolved)
		if err != nil {
			t.Fatalf("Rel(%q): %v", resolved, err)
		}
		if filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			t.Fatalf("Resolve(%q) escaped %q: %q", input, root, resolved)
		}
	})
}
