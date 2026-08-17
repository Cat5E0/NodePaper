package profile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadBuiltinCUMCMProfile(t *testing.T) {
	dir := filepath.Join("..", "..", "profiles", "cumcm")
	loaded, err := Load(dir)
	if err != nil {
		t.Fatalf("Load() error: %v", err)
	}
	if loaded.Definition.RulesVersion != "2026" || loaded.Definition.OutputMode != "electronic-paper" {
		t.Fatalf("definition = %#v", loaded.Definition)
	}
	if len(loaded.Resources) == 0 || len(loaded.SHA256) != 64 {
		t.Fatalf("snapshot resources=%d SHA256=%q", len(loaded.Resources), loaded.SHA256)
	}
	for _, path := range []string{loaded.Template, loaded.CrossrefMetadata, loaded.AbstractFilter, loaded.LayoutFilter, loaded.CSL, loaded.WarningAllowlist} {
		if !filepath.IsAbs(path) {
			t.Fatalf("resource is not absolute: %s", path)
		}
		if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
			t.Fatalf("resource is invalid: %s: %v", path, err)
		}
	}
}

func TestLoadRejectsUnknownMetadataField(t *testing.T) {
	dir := t.TempDir()
	data := `{"schemaVersion":1,"name":"cumcm","unexpected":true}`
	if err := os.WriteFile(filepath.Join(dir, "profile.json"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "unknown field") {
		t.Fatalf("Load() error = %v, want unknown field", err)
	}
}

func TestSnapshotChangesWhenResourceChanges(t *testing.T) {
	source := filepath.Join("..", "..", "profiles", "cumcm")
	dir := filepath.Join(t.TempDir(), "cumcm")
	if err := copyProfileTree(source, dir); err != nil {
		t.Fatal(err)
	}
	before, err := Snapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "template.tex"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	after, err := Snapshot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if before == after {
		t.Fatal("Profile snapshot did not change")
	}
}

// exportFontCascade returns the $if(nodepaper-export)$ arm of one template,
// from the conditional up to and including \tracinglostchars=3.
func exportFontCascade(t *testing.T, name string) string {
	t.Helper()
	path := filepath.Join("..", "..", "profiles", "cumcm", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	text := string(data)
	start := strings.Index(text, "$if(nodepaper-export)$")
	end := strings.Index(text, `\tracinglostchars=3`)
	if start < 0 || end < start {
		t.Fatalf("%s has no $if(nodepaper-export)$ ... \\tracinglostchars=3 arm", path)
	}
	return text[start : end+len(`\tracinglostchars=3`)]
}

// `nodepaper export` produces a .tex that is compiled somewhere NodePaper never
// probed - another Windows machine, a Linux box, Overleaf - so the font choice
// has to happen inside the document. Two failures made this necessary: a bare
// ctexart falls to Fandol on Linux and drops 劼 U+52BC, 黃 U+9EC3 and 內 U+5167
// while still exiting 0, and a hard \setCJKmainfont{SimSun} is a fontspec Error
// with no PDF anywhere SimSun is absent. This guards the cascade that replaced
// both, and guards it in all three templates: --bib inline uses template.tex,
// so fixing only the bibtex copies would leave that route broken.
func TestEveryTemplateCarriesTheSameExportFontCascade(t *testing.T) {
	names := []string{"template.tex", "template-bibtex.tex", "template-biblatex.tex"}
	reference := exportFontCascade(t, names[0])
	for _, name := range names[1:] {
		if got := exportFontCascade(t, name); got != reference {
			t.Errorf("%s carries a different export font cascade than %s", name, names[0])
		}
	}

	// The probes, in order. Order is the whole design: step 1 keeps the real
	// SimHei and KaiTi faces a Windows machine already gets today, and putting
	// a lower-fidelity step first would silently downgrade those users.
	previous := -1
	for _, probe := range []string{
		`\IfFontExistsTF{SimHei}`,
		`\IfFontExistsTF{SimSun}`,
		`\IfFontExistsTF{Noto Serif CJK SC}`,
	} {
		at := strings.Index(reference, probe)
		if at < 0 {
			t.Fatalf("export cascade has no %s probe", probe)
		}
		if at < previous {
			t.Errorf("%s is probed out of order", probe)
		}
		previous = at
	}
	// Nothing installed has to be an error, not a PDF with the Chinese missing.
	if !strings.Contains(reference, `\PackageError{nodepaper}`) {
		t.Error("export cascade does not fail hard when no Chinese font exists")
	}
	// Without fontset=none ctex binds its own families during \documentclass and
	// the cascade below would be decoration.
	if !strings.Contains(reference, "fontset=none") {
		t.Error("export cascade does not disable ctex's own font binding")
	}

	// Every font-binding arm must bind all of them. fontset=none leaves the
	// sans and mono CJK families unset, and a CJK character inside \texttt{}
	// then draws "Unknown CJK family `\CJKttdefault' is being ignored"; the
	// four zh* families and the four \songti-style commands are what the
	// template body and any user's raw LaTeX actually call.
	commands := map[string]int{}
	for _, line := range strings.Split(reference, "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if !strings.HasPrefix(line, `\`) {
			continue
		}
		if brace := strings.Index(line, "{"); brace > 1 {
			commands[line[:brace]]++
		}
	}
	// Three arms bind fonts; the fourth one errors out instead.
	for command, want := range map[string]int{
		`\setCJKmainfont`:   3,
		`\setCJKsansfont`:   3,
		`\setCJKmonofont`:   3,
		`\setCJKfamilyfont`: 3 * 4,
		`\providecommand`:   3 * 4,
	} {
		if commands[command] != want {
			t.Errorf("export cascade uses %s %d times, want %d (one per font arm)", command, commands[command], want)
		}
	}
}

// The second arm is the build's own fallback, and it is only worth having if it
// stays that fallback: a synthesised weight is a cosmetic compromise, a font
// swapped for one with thinner coverage is a wrong document. Compared against
// the profile itself rather than a copy so the two cannot drift apart.
func TestExportCascadeReusesTheBuildFallbackVerbatim(t *testing.T) {
	path := filepath.Join("..", "..", "profiles", "cumcm", "template.tex")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	text := string(data)
	start := strings.Index(text, "$if(nodepaper-font-fallback)$")
	end := strings.Index(text, "$else$")
	if start < 0 || end < start {
		t.Fatalf("%s no longer has a $if(nodepaper-font-fallback)$ ... $else$ arm", path)
	}
	cascade := exportFontCascade(t, "template.tex")

	found := 0
	for _, line := range strings.Split(text[start:end], "\n") {
		line = strings.TrimSpace(strings.TrimSuffix(line, "\r"))
		if !strings.HasPrefix(line, `\setCJK`) && !strings.HasPrefix(line, `\providecommand`) {
			continue
		}
		found++
		if !strings.Contains(cascade, line) {
			t.Errorf("export cascade does not carry the build fallback's line %q", line)
		}
	}
	if found == 0 {
		t.Fatal("no font bindings found in the build fallback arm")
	}
}

func TestLoadRejectsResourceEscape(t *testing.T) {
	parent := t.TempDir()
	dir := filepath.Join(parent, "profile")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	outside := filepath.Join(parent, "outside.tex")
	if err := os.WriteFile(outside, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	data := `{
  "schemaVersion": 1,
  "name": "cumcm",
  "version": "test",
  "rulesVersion": "2026",
  "outputMode": "electronic-paper",
  "template": "../outside.tex",
  "crossrefMetadata": "x",
  "abstractFilter": "x",
  "layoutFilter": "x",
  "highlightStyle": "tango",
  "csl": "x",
  "warningAllowlist": "x",
  "pandocVersion": "3.9",
  "pandocCrossrefVersion": "0.3.24"
}`
	if err := os.WriteFile(filepath.Join(dir, "profile.json"), []byte(data), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(dir); err == nil || !strings.Contains(err.Error(), "escapes profile") {
		t.Fatalf("Load() error = %v, want path escape", err)
	}
}

func copyProfileTree(source, destination string) error {
	return filepath.Walk(source, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		target := filepath.Join(destination, relative)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, info.Mode().Perm())
	})
}
