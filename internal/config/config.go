// Package config reads, parses and validates nodepaper.yaml project
// configuration. All paths in the configuration are relative to the
// project root and are not resolved here.
package config

import (
	"bytes"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ProjectConfig is the deserialised nodepaper.yaml.
type ProjectConfig struct {
	Version int          `yaml:"version"`
	Profile string       `yaml:"profile"`
	Source  string       `yaml:"source,omitempty"`
	Sources []string     `yaml:"sources,omitempty"`
	Output  OutputConfig `yaml:"output,omitempty"`
}

// OutputConfig controls the destination of built artifacts.
type OutputConfig struct {
	File string `yaml:"file,omitempty"`
}

// Load reads and validates a project configuration file. The returned
// configuration has been parsed but paths have not been resolved against the
// project root.
func Load(path string) (ProjectConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ProjectConfig{}, fmt.Errorf("cannot read config file %s: %w", path, err)
	}
	return Parse(data)
}

// Parse unmarshals and validates raw YAML bytes into a ProjectConfig.
func Parse(data []byte) (ProjectConfig, error) {
	var cfg ProjectConfig
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	decoder.KnownFields(true)
	if err := decoder.Decode(&cfg); err != nil {
		return ProjectConfig{}, fmt.Errorf("cannot parse YAML: %w", err)
	}

	if err := validate(cfg); err != nil {
		return ProjectConfig{}, err
	}

	return cfg, nil
}

func validate(cfg ProjectConfig) error {
	if cfg.Version != 1 {
		return fmt.Errorf("unsupported config version %d; only version 1 is supported", cfg.Version)
	}
	if cfg.Profile == "" {
		return fmt.Errorf("profile is required")
	}
	if cfg.Profile != "cumcm" {
		return fmt.Errorf("unsupported profile %q; v0.1 only supports cumcm", cfg.Profile)
	}

	hasSource := cfg.Source != ""
	hasSources := len(cfg.Sources) > 0

	if hasSource && hasSources {
		return fmt.Errorf("source and sources are mutually exclusive")
	}
	if !hasSource && !hasSources {
		return fmt.Errorf("source or sources is required")
	}
	if hasSources && len(cfg.Sources) == 0 {
		return fmt.Errorf("sources must not be empty when present")
	}

	for i, s := range cfg.Sources {
		if s == "" {
			return fmt.Errorf("sources[%d] is empty", i)
		}
	}

	return nil
}

// SourceFiles returns the ordered list of Markdown source files. In
// single-source mode the result is a slice containing the single entry.
func (cfg ProjectConfig) SourceFiles() []string {
	if cfg.Source != "" {
		return []string{cfg.Source}
	}
	return cfg.Sources
}
