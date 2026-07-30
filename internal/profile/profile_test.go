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
	for _, path := range []string{loaded.Template, loaded.CrossrefMetadata, loaded.AbstractFilter, loaded.CSL, loaded.WarningAllowlist} {
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
