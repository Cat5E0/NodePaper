// Package profile loads, validates and snapshots immutable built-in Profile resources.
package profile

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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
	BibtexTemplate        string `json:"bibtexTemplate"`
	BiblatexTemplate      string `json:"biblatexTemplate"`
	CrossrefMetadata      string `json:"crossrefMetadata"`
	AbstractFilter        string `json:"abstractFilter"`
	LayoutFilter          string `json:"layoutFilter"`
	HighlightStyle        string `json:"highlightStyle"`
	CSL                   string `json:"csl"`
	WarningAllowlist      string `json:"warningAllowlist"`
	PandocVersion         string `json:"pandocVersion"`
	PandocCrossrefVersion string `json:"pandocCrossrefVersion"`
}

// Resource records one file in the complete immutable Profile tree.
type Resource struct {
	Relative string `json:"relative"`
	Path     string `json:"path"`
	SHA256   string `json:"sha256"`
}

// Loaded contains validated metadata, absolute immutable resource paths and a
// complete deterministic Profile snapshot.
type Loaded struct {
	Dir              string
	Definition       Definition
	Template         string
	BibtexTemplate   string
	BiblatexTemplate string
	CrossrefMetadata string
	AbstractFilter   string
	LayoutFilter     string
	CSL              string
	WarningAllowlist string
	Resources        []Resource
	SHA256           string
}

// Load strictly parses a built-in Profile and verifies every declared file and
// every file included in its complete resource snapshot.
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

	resources, digest, err := scanTree(root)
	if err != nil {
		return Loaded{}, err
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
	if definition.HighlightStyle == "" {
		return Loaded{}, fmt.Errorf("profile highlight style is required")
	}

	loaded := Loaded{Dir: root, Definition: definition, Resources: resources, SHA256: digest}
	declared := []struct {
		name  string
		value string
		dest  *string
	}{
		{"template", definition.Template, &loaded.Template},
		{"BibTeX export template", definition.BibtexTemplate, &loaded.BibtexTemplate},
		{"biblatex export template", definition.BiblatexTemplate, &loaded.BiblatexTemplate},
		{"crossref metadata", definition.CrossrefMetadata, &loaded.CrossrefMetadata},
		{"abstract filter", definition.AbstractFilter, &loaded.AbstractFilter},
		{"layout filter", definition.LayoutFilter, &loaded.LayoutFilter},
		{"CSL", definition.CSL, &loaded.CSL},
		{"warning allowlist", definition.WarningAllowlist, &loaded.WarningAllowlist},
	}
	for _, resource := range declared {
		resolved, err := resolveRegularFile(root, resource.value)
		if err != nil {
			return Loaded{}, fmt.Errorf("%s: %w", resource.name, err)
		}
		*resource.dest = resolved
	}
	return loaded, nil
}

// Snapshot recomputes the complete resource-tree SHA-256 without modifying it.
func Snapshot(dir string) (string, error) {
	root, err := filepath.Abs(dir)
	if err != nil {
		return "", fmt.Errorf("resolve profile directory: %w", err)
	}
	_, digest, err := scanTree(root)
	return digest, err
}

func scanTree(root string) ([]Resource, string, error) {
	var resources []Resource
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return fmt.Errorf("profile contains a symbolic link: %s", path)
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("profile contains a non-regular resource: %s", path)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		sum := sha256.Sum256(data)
		resources = append(resources, Resource{
			Relative: filepath.ToSlash(relative),
			Path:     path,
			SHA256:   hex.EncodeToString(sum[:]),
		})
		return nil
	})
	if err != nil {
		return nil, "", fmt.Errorf("snapshot profile resources: %w", err)
	}
	if len(resources) == 0 {
		return nil, "", fmt.Errorf("profile contains no resources: %s", root)
	}
	sort.Slice(resources, func(i, j int) bool { return resources[i].Relative < resources[j].Relative })
	hash := sha256.New()
	for _, resource := range resources {
		fmt.Fprintf(hash, "%s\x00%s\n", resource.Relative, resource.SHA256)
	}
	return resources, hex.EncodeToString(hash.Sum(nil)), nil
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
	info, err := os.Lstat(path)
	if err != nil {
		return "", err
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", fmt.Errorf("resource is not an immutable regular file: %s", path)
	}
	return path, nil
}
