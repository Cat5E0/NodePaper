// Package profile loads and validates immutable built-in Profile resources.
package profile

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// Definition is the stable metadata stored in profile.json.
type Definition struct {
	SchemaVersion         int    `json:"schemaVersion"`
	Name                  string `json:"name"`
	Version               string `json:"version"`
	RulesVersion          string `json:"rulesVersion"`
	OutputMode            string `json:"outputMode"`
	Template              string `json:"template"`
	CrossrefMetadata      string `json:"crossrefMetadata"`
	AbstractFilter        string `json:"abstractFilter"`
	CSL                   string `json:"csl"`
	WarningAllowlist      string `json:"warningAllowlist"`
	PandocVersion         string `json:"pandocVersion"`
	PandocCrossrefVersion string `json:"pandocCrossrefVersion"`
}

// Loaded contains validated metadata and absolute immutable resource paths.
type Loaded struct {
	Dir              string
	Definition       Definition
	Template         string
	CrossrefMetadata string
	AbstractFilter   string
	CSL              string
	WarningAllowlist string
}

// Load strictly parses a built-in Profile and verifies every declared file is
// a regular file below the Profile directory.
func Load(dir string) (Loaded, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return Loaded{}, fmt.Errorf("resolve profile directory: %w", err)
	}
	info, err := os.Stat(root)
	if err != nil {
		return Loaded{}, fmt.Errorf("profile directory: %w", err)
	}
	if !info.IsDir() {
		return Loaded{}, fmt.Errorf("profile path is not a directory: %s", root)
	}

	metadataPath := filepath.Join(root, "profile.json")
	file, err := os.Open(metadataPath)
	if err != nil {
		return Loaded{}, fmt.Errorf("open profile metadata: %w", err)
	}
	defer file.Close()

	var definition Definition
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&definition); err != nil {
		return Loaded{}, fmt.Errorf("parse profile metadata: %w", err)
	}
	if err := ensureJSONEnd(decoder); err != nil {
		return Loaded{}, err
	}
	if definition.SchemaVersion != 1 {
		return Loaded{}, fmt.Errorf("unsupported profile schema version %d", definition.SchemaVersion)
	}
	if definition.Name != "cumcm" {
		return Loaded{}, fmt.Errorf("unexpected profile name %q", definition.Name)
	}
	if definition.Version == "" || definition.RulesVersion != "2026" || definition.OutputMode != "electronic-paper" {
		return Loaded{}, fmt.Errorf("profile version, 2026 rules version and electronic-paper output mode are required")
	}
	if definition.PandocVersion == "" || definition.PandocCrossrefVersion == "" {
		return Loaded{}, fmt.Errorf("profile tool versions are required")
	}

	loaded := Loaded{Dir: root, Definition: definition}
	resources := []struct {
		name  string
		value string
		dest  *string
	}{
		{"template", definition.Template, &loaded.Template},
		{"crossref metadata", definition.CrossrefMetadata, &loaded.CrossrefMetadata},
		{"abstract filter", definition.AbstractFilter, &loaded.AbstractFilter},
		{"CSL", definition.CSL, &loaded.CSL},
		{"warning allowlist", definition.WarningAllowlist, &loaded.WarningAllowlist},
	}
	for _, resource := range resources {
		resolved, err := resolveRegularFile(root, resource.value)
		if err != nil {
			return Loaded{}, fmt.Errorf("%s: %w", resource.name, err)
		}
		*resource.dest = resolved
	}
	return loaded, nil
}

func ensureJSONEnd(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); err == io.EOF {
		return nil
	} else if err != nil {
		return fmt.Errorf("parse profile metadata: %w", err)
	}
	return fmt.Errorf("profile metadata contains multiple JSON values")
}

func resolveRegularFile(root, relative string) (string, error) {
	if strings.TrimSpace(relative) == "" {
		return "", fmt.Errorf("resource path is empty")
	}
	if filepath.IsAbs(relative) {
		return "", fmt.Errorf("resource path must be relative: %s", relative)
	}
	path := filepath.Join(root, filepath.Clean(relative))
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("resource path escapes profile: %s", relative)
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("resource is not a regular file: %s", path)
	}
	return path, nil
}
