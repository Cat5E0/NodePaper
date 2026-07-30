package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"nodepaper/internal/build"
	"nodepaper/internal/diagnostic"
	"nodepaper/internal/doctor"
	"nodepaper/internal/project"
	"nodepaper/internal/validate"
)

// New returns the default App implementation that orchestrates all NodePaper
// operations without rendering output or depending on CLI globals.
func New() App {
	return &appImpl{
		profileDir: findProfileDir(),
	}
}

type appImpl struct {
	profileDir string
}

func (a *appImpl) Init(ctx context.Context, req InitRequest) (InitResult, error) {
	cfg := projectConfig{
		Version: 1,
		Profile: "cumcm",
		Source:  "paper.md",
		Output:  outputConfig{File: "dist/paper.pdf"},
	}

	paperContent := paperMarkdown()

	refsContent := `% NodePaper references
% Add your BibTeX entries below.
`

	gitignoreContent := `dist/
.nodepaper/
`

	result := InitResult{ProjectRoot: req.ProjectDir}

	if err := os.MkdirAll(req.ProjectDir, 0755); err != nil {
		result.Diagnostics = append(result.Diagnostics, initDiag("NP1101",
			fmt.Sprintf("cannot create project directory: %v", err),
			req.ProjectDir, "Check that the parent directory exists and is writable."))
		return result, nil
	}

	writeFile := func(rel, content string) error {
		path := filepath.Join(req.ProjectDir, rel)
		dir := filepath.Dir(path)
		if err := os.MkdirAll(dir, 0755); err != nil {
			return err
		}
		return os.WriteFile(path, []byte(content), 0644)
	}

	if err := writeConfig(req.ProjectDir, cfg); err != nil {
		result.Diagnostics = append(result.Diagnostics, initDiag("NP1102",
			fmt.Sprintf("cannot write nodepaper.yaml: %v", err),
			req.ProjectDir, ""))
		return result, nil
	}

	if err := writeFile("paper.md", paperContent); err != nil {
		result.Diagnostics = append(result.Diagnostics, initDiag("NP1102",
			fmt.Sprintf("cannot write paper.md: %v", err),
			req.ProjectDir, ""))
		return result, nil
	}

	if err := writeFile("references.bib", refsContent); err != nil {
		result.Diagnostics = append(result.Diagnostics, initDiag("NP1102",
			fmt.Sprintf("cannot write references.bib: %v", err),
			req.ProjectDir, ""))
		return result, nil
	}

	if err := writeFile(".gitignore", gitignoreContent); err != nil {
		result.Diagnostics = append(result.Diagnostics, initDiag("NP1102",
			fmt.Sprintf("cannot write .gitignore: %v", err),
			req.ProjectDir, ""))
		return result, nil
	}

	imagesDir := filepath.Join(req.ProjectDir, "images")
	if err := os.MkdirAll(imagesDir, 0755); err != nil {
		result.Diagnostics = append(result.Diagnostics, initDiag("NP1102",
			fmt.Sprintf("cannot create images/ directory: %v", err),
			req.ProjectDir, ""))
		return result, nil
	}

	p, err := project.DiscoverFrom(req.ProjectDir, "")
	if err == nil {
		result.ProjectRoot = p.Root
		result.Success = true
		result.Artifacts = []Artifact{
			{Kind: "markdown", Path: filepath.Join(p.Root, "paper.md")},
			{Kind: "config", Path: filepath.Join(p.Root, "nodepaper.yaml")},
		}
	}

	return result, nil
}

func (a *appImpl) Doctor(ctx context.Context, req DoctorRequest) (DoctorResult, error) {
	projectRoot := req.ProjectDir
	if projectRoot != "" {
		p, err := project.DiscoverFrom(projectRoot, "")
		if err == nil {
			projectRoot = p.Root
		}
	}

	dr := doctor.Run(ctx, projectRoot, a.profileDir)

	var checks []DoctorCheck
	for _, c := range dr.Checks {
		checks = append(checks, DoctorCheck{
			Name:       c.Name,
			Status:     string(c.Status),
			Message:    c.Message,
			Suggestion: c.Suggestion,
		})
	}

	return DoctorResult{
		Success:     dr.Success,
		ProjectRoot: dr.ProjectRoot,
		Diagnostics: dr.Diagnostics,
		Checks:      checks,
	}, nil
}

func (a *appImpl) Validate(ctx context.Context, req ValidateRequest) (ValidateResult, error) {
	vr := validate.Run(ctx, req.ProjectDir)
	return ValidateResult{
		Success:     vr.Success,
		ProjectRoot: vr.ProjectRoot,
		Diagnostics: vr.Diagnostics,
	}, nil
}

func (a *appImpl) Build(ctx context.Context, req BuildRequest) (BuildResult, error) {
	br := build.Run(ctx, req.ProjectDir)
	var artifacts []Artifact
	for _, art := range br.Artifacts {
		artifacts = append(artifacts, Artifact{Kind: art.Kind, Path: art.Path})
	}
	return BuildResult{
		Success:     br.Success,
		BuildID:     br.BuildID,
		ProjectRoot: br.ProjectRoot,
		Artifacts:   artifacts,
		Diagnostics: br.Diagnostics,
	}, nil
}

func (a *appImpl) Clean(ctx context.Context, req CleanRequest) (CleanResult, error) {
	diags, err := build.Clean(req.ProjectDir, req.All)
	if err != nil {
		return CleanResult{}, err
	}
	return CleanResult{
		Success:     !hasErrorDiags(diags),
		ProjectRoot: req.ProjectDir,
		Diagnostics: diags,
	}, nil
}

func hasErrorDiags(diags []diagnostic.Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == diagnostic.SeverityError {
			return true
		}
	}
	return false
}

func initDiag(code, message, file, suggestion string) diagnostic.Diagnostic {
	return diagnostic.Diagnostic{
		Severity:   diagnostic.SeverityError,
		Code:       code,
		Message:    message,
		File:       file,
		Suggestion: suggestion,
		Source:     "init",
	}
}

type projectConfig struct {
	Version int          `yaml:"version"`
	Profile string       `yaml:"profile"`
	Source  string       `yaml:"source"`
	Output  outputConfig `yaml:"output"`
}

type outputConfig struct {
	File string `yaml:"file"`
}

func writeConfig(root string, cfg projectConfig) error {
	content := fmt.Sprintf(
		"version: %d\nprofile: %s\nsource: %s\noutput:\n  file: %s\n",
		cfg.Version, cfg.Profile, cfg.Source, cfg.Output.File,
	)
	return os.WriteFile(filepath.Join(root, "nodepaper.yaml"), []byte(content), 0644)
}

func findProfileDir() string {
	if override := os.Getenv("NODEPAPER_PROFILE_DIR"); override != "" {
		return override
	}
	// Try profiles/cumcm relative to the executable first.
	if exe, err := os.Executable(); err == nil {
		dir := filepath.Join(filepath.Dir(exe), "profiles", "cumcm")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	// Fallback: look relative to the current working directory (dev mode).
	if wd, err := os.Getwd(); err == nil {
		dir := filepath.Join(wd, "profiles", "cumcm")
		if info, err := os.Stat(dir); err == nil && info.IsDir() {
			return dir
		}
	}
	return ""
}

func paperMarkdown() string {
	return `---
title: 论文标题
problem: A
keywords:
  - 关键词1
  - 关键词2
---

# 摘要

在此撰写摘要内容。

# 问题重述

## 问题背景

## 问题提出

# 模型假设与符号说明

## 模型假设

## 符号说明

# 模型建立与求解

# 模型评价与改进

# 参考文献
`
}
