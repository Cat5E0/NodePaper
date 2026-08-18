package app

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"nodepaper/internal/diagnostic"
	"nodepaper/internal/project"
)

const (
	codeInitCreateFailed     = "NP1101"
	codeInitWriteFailed      = "NP1102"
	codeAIGuidePreserved     = "NP1103"
	codeInitWouldOverwrite   = "NP1104"
	codeInitInvalidDirectory = "NP1105"
	codeInitCanceled         = "NP1106"
)

type initEntry struct {
	relPath string
	content string
	kind    string
}

func (a *appImpl) Init(ctx context.Context, req InitRequest) (InitResult, error) {
	if req.ProjectDir == "" {
		return InitResult{Diagnostics: []diagnostic.Diagnostic{initDiag(
			codeInitInvalidDirectory,
			"Project directory is empty.",
			"",
			"Pass a directory for the new NodePaper Project.",
		)}}, nil
	}

	root, err := filepath.Abs(filepath.Clean(req.ProjectDir))
	if err != nil {
		return InitResult{ProjectRoot: req.ProjectDir, Diagnostics: []diagnostic.Diagnostic{initDiag(
			codeInitInvalidDirectory,
			fmt.Sprintf("cannot resolve project directory: %v", err),
			req.ProjectDir,
			"Pass a valid local directory path.",
		)}}, nil
	}
	result := InitResult{ProjectRoot: root}
	if err := ctx.Err(); err != nil {
		result.Diagnostics = append(result.Diagnostics, initCanceledDiag(root))
		return result, nil
	}

	entries := defaultInitEntries()
	guidePreserved := false
	info, statErr := os.Lstat(root)
	switch {
	case statErr == nil:
		if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
			result.Diagnostics = append(result.Diagnostics, initDiag(
				codeInitInvalidDirectory,
				"Project path exists but is not a regular directory.",
				root,
				"Choose a new directory or an existing regular directory.",
			))
			return result, nil
		}
		if req.GenerateAIGuide {
			guidePath := filepath.Join(root, "AGENTS.md")
			if _, err := os.Lstat(guidePath); err == nil {
				guidePreserved = true
			} else if !os.IsNotExist(err) {
				result.Diagnostics = append(result.Diagnostics, initDiag(codeInitWriteFailed,
					fmt.Sprintf("cannot inspect AGENTS.md: %v", err), guidePath, "Check the file permissions."))
				return result, nil
			}
		}
		if collision := firstInitCollision(root, entries); collision != "" {
			result.Diagnostics = append(result.Diagnostics, initDiag(
				codeInitWouldOverwrite,
				"Initialization would overwrite an existing Project file.",
				collision,
				"Choose a new directory or move the existing file; NodePaper init never overwrites source files.",
			))
			return result, nil
		}
	case os.IsNotExist(statErr):
		// Created from the fully prepared staging directory below.
	default:
		result.Diagnostics = append(result.Diagnostics, initDiag(codeInitCreateFailed,
			fmt.Sprintf("cannot inspect project directory: %v", statErr), root, "Check the parent directory permissions."))
		return result, nil
	}

	if req.GenerateAIGuide && !guidePreserved {
		entries = append(entries, initEntry{relPath: "AGENTS.md", content: aiGuideMarkdown(), kind: "ai-guide"})
	}

	parent := filepath.Dir(root)
	if err := os.MkdirAll(parent, 0o755); err != nil {
		result.Diagnostics = append(result.Diagnostics, initDiag(codeInitCreateFailed,
			fmt.Sprintf("cannot create parent directory: %v", err), parent, "Check that the parent directory is writable."))
		return result, nil
	}
	stage, err := os.MkdirTemp(parent, ".nodepaper-init-*")
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, initDiag(codeInitCreateFailed,
			fmt.Sprintf("cannot create initialization staging directory: %v", err), parent, "Check that the parent directory is writable."))
		return result, nil
	}
	defer os.RemoveAll(stage)

	if err := writeInitStage(ctx, stage, entries); err != nil {
		code := codeInitWriteFailed
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			result.Diagnostics = append(result.Diagnostics, initCanceledDiag(root))
			return result, nil
		}
		result.Diagnostics = append(result.Diagnostics, initDiag(code,
			fmt.Sprintf("cannot prepare Project files: %v", err), root, "Check available disk space and directory permissions."))
		return result, nil
	}

	if err := ctx.Err(); err != nil {
		result.Diagnostics = append(result.Diagnostics, initCanceledDiag(root))
		return result, nil
	}
	if statErr != nil && os.IsNotExist(statErr) {
		if err := os.Rename(stage, root); err != nil {
			result.Diagnostics = append(result.Diagnostics, initDiag(codeInitCreateFailed,
				fmt.Sprintf("cannot publish initialized Project: %v", err), root, "Check that the destination does not already exist and retry."))
			return result, nil
		}
	} else if err := commitInitIntoExisting(ctx, stage, root, entries); err != nil {
		if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			result.Diagnostics = append(result.Diagnostics, initCanceledDiag(root))
		} else {
			result.Diagnostics = append(result.Diagnostics, initDiag(codeInitWriteFailed,
				fmt.Sprintf("cannot publish initialized Project: %v", err), root, "The attempted files were rolled back; check permissions and retry."))
		}
		return result, nil
	}

	p, err := project.DiscoverFrom(root, "")
	if err != nil {
		result.Diagnostics = append(result.Diagnostics, initDiag(codeInitWriteFailed,
			fmt.Sprintf("initialized files could not be verified: %v", err), root, "Remove the incomplete directory and retry."))
		return result, nil
	}
	result.ProjectRoot = p.Root
	result.Success = true
	for _, entry := range entries {
		if entry.kind != "" {
			result.Artifacts = append(result.Artifacts, Artifact{Kind: entry.kind, Path: filepath.Join(p.Root, entry.relPath)})
		}
	}
	if guidePreserved {
		result.Diagnostics = append(result.Diagnostics, diagnostic.Diagnostic{
			Severity:   diagnostic.SeverityInfo,
			Code:       codeAIGuidePreserved,
			Message:    "Existing AGENTS.md was preserved.",
			File:       filepath.Join(p.Root, "AGENTS.md"),
			Suggestion: "Review the existing guide and add NodePaper writing constraints manually if needed.",
			Source:     "init",
		})
	}
	return result, nil
}

func defaultInitEntries() []initEntry {
	return []initEntry{
		{relPath: "nodepaper.yaml", content: "version: 1\nprofile: cumcm\nsource: paper.md\noutput:\n  file: dist/paper.pdf\n", kind: "config"},
		{relPath: "paper.md", content: paperMarkdown(), kind: "markdown"},
		{relPath: "references.bib", content: "% NodePaper references\n% Add your BibTeX entries below.\n", kind: "bibliography"},
		{relPath: ".gitignore", content: "dist/\n.nodepaper/\n", kind: "gitignore"},
	}
}

func firstInitCollision(root string, entries []initEntry) string {
	for _, entry := range entries {
		path := filepath.Join(root, entry.relPath)
		if _, err := os.Lstat(path); err == nil {
			return path
		} else if !os.IsNotExist(err) {
			return path
		}
	}
	images := filepath.Join(root, "images")
	if info, err := os.Lstat(images); err == nil && !info.IsDir() {
		return images
	} else if err != nil && !os.IsNotExist(err) {
		return images
	}
	return ""
}

func writeInitStage(ctx context.Context, stage string, entries []initEntry) error {
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}
		path := filepath.Join(stage, entry.relPath)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(path, []byte(entry.content), 0o644); err != nil {
			return err
		}
	}
	return os.MkdirAll(filepath.Join(stage, "images"), 0o755)
}

func commitInitIntoExisting(ctx context.Context, stage, root string, entries []initEntry) error {
	created := make([]string, 0, len(entries)+1)
	rollback := func() {
		for i := len(created) - 1; i >= 0; i-- {
			_ = os.RemoveAll(created[i])
		}
	}
	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			rollback()
			return err
		}
		source := filepath.Join(stage, entry.relPath)
		target := filepath.Join(root, entry.relPath)
		in, err := os.Open(source)
		if err != nil {
			rollback()
			return err
		}
		out, err := os.OpenFile(target, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err != nil {
			_ = in.Close()
			rollback()
			return err
		}
		created = append(created, target)
		_, copyErr := io.Copy(out, in)
		closeOutErr := out.Close()
		closeInErr := in.Close()
		if copyErr != nil || closeOutErr != nil || closeInErr != nil {
			rollback()
			return errors.Join(copyErr, closeOutErr, closeInErr)
		}
	}
	if err := ctx.Err(); err != nil {
		rollback()
		return err
	}
	images := filepath.Join(root, "images")
	if info, err := os.Lstat(images); os.IsNotExist(err) {
		if err := os.Mkdir(images, 0o755); err != nil {
			rollback()
			return err
		}
		created = append(created, images)
	} else if err != nil || !info.IsDir() {
		rollback()
		if err != nil {
			return err
		}
		return fmt.Errorf("images path is not a directory: %s", images)
	}
	return nil
}

func initCanceledDiag(root string) diagnostic.Diagnostic {
	return initDiag(codeInitCanceled, "Initialization was canceled before the Project was published.", root, "Run nodepaper init again when ready.")
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

func aiGuideMarkdown() string {
	return `# NodePaper 项目写作说明

本目录是一个 NodePaper Project，` + "`nodepaper.yaml`" + ` 是项目标识。论文正式源码是 Markdown、图片和参考文献，不是 ` + "`.nodepaper/build/`" + ` 中生成的 LaTeX。

## 写作约束

- 按 ` + "`nodepaper.yaml`" + ` 中 ` + "`source`" + ` 或 ` + "`sources`" + ` 的显式顺序编辑文档，不自动扫描或重排章节。
- 第一个 Source 使用 YAML Front Matter，并保留非空的 ` + "`title`" + `、` + "`problem`" + `、` + "`keywords`" + `；摘要使用唯一的一级标题 ` + "`# 摘要`" + `。
- 图片、参考文献和 Fragment 路径都相对于 Project Root；不得使用绝对路径或 ` + "`../`" + ` 逃逸项目。
- 图、表、公式和章节使用稳定且唯一的 Pandoc ID，例如 ` + "`#fig:result`" + `、` + "`#tbl:data`" + `、` + "`#eq:model`" + `、` + "`#sec:method`" + `，引用目标必须存在。
- 文献只使用 ` + "`references.bib`" + ` 中真实存在的键；不得猜造作者、标题、年份、DOI 或引用来源。
- 复杂 LaTeX 只能放入配置显式声明的项目内受控 Fragment；不得加载宏包、嵌套输入、执行命令或修改安装目录中的冻结 Profile。
- 不执行论文正文或附录中的代码，不关闭安全检查，不放宽 Warning/质量门禁来掩盖问题。

## 修改后工作流

` + "```powershell" + `
nodepaper validate
nodepaper build
` + "```" + `

先修复 Validate/Build 报告的 Diagnostic，再检查 ` + "`dist/paper.pdf`" + `。不要直接修改生成的 ` + "`.nodepaper/build/paper.tex`" + ` 作为长期修复。

本机没有 TeX 时 ` + "`nodepaper build`" + ` 无法出 PDF，这不是项目错误。此时用 ` + "`nodepaper export . --to <目录>`" + ` 导出可独立编译的 LaTeX 工程（只用内置 pandoc，不需要 TeX），交由有 TeX 的环境编译；导出是单向的，不要改导出结果来代替修改 Markdown 源。
`
}
