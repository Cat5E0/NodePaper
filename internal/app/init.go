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
		{relPath: "nodepaper.yaml", content: configYAML(), kind: "config"},
		{relPath: "paper.md", content: paperMarkdown(), kind: "markdown"},
		{relPath: "ai-usage.md", content: aiStatementMarkdown(), kind: "ai-statement"},
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

func configYAML() string {
	// aiUsage ships as false and aiStatement ships commented out, which is the
	// combination a team that used no AI tool needs and never has to touch.
	// aiUsage carries no default in the config loader on purpose - an unset
	// value is refused rather than read as false, so nobody can be handed a
	// declaration of non-use they never made. Here in a freshly created Project
	// there is no such risk: the author is looking at the line as they write
	// their first paragraph.
	//
	// Every Project gets ai-usage.md regardless, because a team that ends up
	// using an AI tool should not have to go looking for a template
	// mid-competition; but a team that used none must not hand in a supporting
	// document it never needed, so nothing is built until the line is
	// uncommented.
	return `version: 1
profile: cumcm
source: paper.md
output:
  file: dist/paper.pdf

# 竞赛规定：每份论文都要在参考文献之前声明是否使用了 AI 工具。NodePaper 按下面这行
# 自动生成「AI工具使用声明」一节。未使用 AI 工具的保持原样即可；使用了的改成
# aiUsage: true，并加一行 aiUsagePurpose 写明本队的真实用途，例如
#   aiUsagePurpose: 语言润色、代码调试
aiUsage: false

# 用了 AI 工具的参赛队，除了改上面那行，再取消下面这行注释：nodepaper build 会连同
# 论文一起生成 dist/AI工具使用详情.pdf，文件名就是竞赛规定要求的名字。
# aiStatement: ai-usage.md
`
}

// aiStatementMarkdown is the fill-in-the-blanks supporting document. Two things
// about its wording are deliberate. The paragraph that explains the switch is
// not wrapped in full-width parentheses: with inline code in the same
// paragraph, the closing ） lands at the start of the next line, which is the
// line-start punctuation defect M4-11 recorded and has not been fixed at the
// engine level. And nothing here names a real tool or invents an interaction -
// every value is a placeholder the team replaces with what it actually did.
func aiStatementMarkdown() string {
	return `---
title: AI工具使用详情
---

# 摘要

本参赛队在竞赛期间使用了下表所列的 AI 工具，用于（此处填写实际环节，例如资料检索、
代码调试、文字润色）。竞赛作品的核心建模、求解与分析由本参赛队主导，AI 参与完成的
内容均已由本队逐项人工审查与核实后采用。本参赛队对所提交作品的原创性、真实性和准
确性负全部责任。

以上为填写示例，请按本队真实情况改写。本文件只有在 ` + "`nodepaper.yaml`" + ` 中取消
` + "`aiStatement`" + ` 那行注释后才会被构建；未使用任何 AI 工具的参赛队不需要提交它。

# 工具清单

逐项列出竞赛期间使用过的全部 AI 工具的名称与版本或型号。同一工具在不同日期多次使用
的，可以合并为一行。

| 工具名称 | 版本/型号 |
|---|---|
| （工具名称） | （版本/型号） |
| （工具名称） | （版本/型号） |

: 竞赛期间使用的 AI 工具 {#tbl:ai-tools}

# 使用目的与环节

分工具说明具体用它做什么、在哪个环节用，以及哪些环节由本队主导完成，例如：

- 模型假设、建模思路、求解方案与结果分析由本队主导完成；
- AI 工具仅用于（此处填写，例如查找文献线索、解释报错信息、检查语句通顺）；
- 未使用 AI 工具直接生成论文正文，未改写他人作品或往届获奖成果。

# 提示方式与使用过程

说明本队主要怎么向 AI 工具提问、怎么使用它的输出。规定要求的是“主要提示方式与使用
过程说明”，逐字附上每一次交互并不是必须的；如果附典型交互示例，照抄原文，不做美化，
较长的提问或输出放进代码块。

本队的主要提示方式：（此处填写，例如先给出问题背景与已有假设，再要求给出思路而非
完整代码；对报错信息只提供最小复现片段）。

**典型示例**（可选；工具：填写；时间：填写）

提问：

` + "```" + `text
（此处粘贴提问原文）
` + "```" + `

输出要点：

` + "```" + `text
（此处粘贴关键输出，或概括其要点）
` + "```" + `

# 采纳、修改与核验

说明对 AI 输出的采纳、人工修改和核验的主要情况。**语言润色除外**，不必逐条记录。
例如：

- 上述示例的输出未直接采用，本队据此自行推导后写入论文第 3 节，公式与结论均已重新
  验证；
- AI 协助调试的代码，本队逐行读过并用（此处填写，例如已知解析解／小规模手算样例）
  核对过运行结果。

# 原创性与责任声明

本参赛队声明：竞赛作品的核心建模与分析由本队主导完成，AI 参与完成的内容均已逐项人工
审查与核实；AI 工具的使用情况已如实记录于本文件；本队对所提交作品的原创性、真实性和
准确性负全部责任，并接受竞赛组委会的核查。
`
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

# 参考文献 {-}

::: {#refs}
:::
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

本机没有 TeX 时 ` + "`nodepaper build`" + ` 无法出 PDF，这不是项目错误。此时用 ` + "`nodepaper export . --to <目录或.zip文件>`" + ` 导出可独立编译的 LaTeX 工程（只用内置 pandoc，不需要 TeX），交由有 TeX 的环境编译；导出是单向的，不要改导出结果来代替修改 Markdown 源。
`
}
