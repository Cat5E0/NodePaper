# NodePaper

NodePaper 是一个面向 Windows 的命令行工具，用于将包含 `nodepaper.yaml` 的 Markdown Project 构建为 PDF。

当前 v0.1 面向 CUMCM 2026 电子版论文场景，支持中文排版、公式、图表、交叉引用、参考文献、代码块、附录和多文件项目。

> 当前仍处于测试版开发阶段，尚未正式发布。NodePaper 不代表比赛官方认证。

## 安装

NodePaper 支持 Windows 10/11 x64。使用前请先安装 TeX Live 或 MiKTeX，并确保 `xelatex` 和 `latexmk` 可用。

解压 NodePaper Windows 发布包后，在解压目录运行：

```powershell
.\Install-NodePaper.ps1
```

重新打开终端，然后在任意目录输入：

```powershell
nodepaper
```

卸载 NodePaper：

```powershell
& "$env:LOCALAPPDATA\Programs\NodePaper\Uninstall-NodePaper.ps1"
```

卸载不会删除论文项目或 TeX 环境。

## 快速开始

### 1. 检查环境

```powershell
nodepaper doctor
```

### 2. 创建项目

```powershell
nodepaper init D:\papers\cumcm-a
```

如需同时生成项目级 AI 写作说明：

```powershell
nodepaper init D:\papers\cumcm-a --ai-guide
```

### 3. 编辑论文

项目的基本结构如下：

```text
cumcm-a/
├── nodepaper.yaml
├── paper.md
├── references.bib
├── images/
├── dist/
└── .nodepaper/
```

主要编辑：

- `paper.md`：论文正文；
- `references.bib`：参考文献；
- `images/`：图片资源；
- `nodepaper.yaml`：项目配置。

### 4. 验证和构建

```powershell
cd D:\papers\cumcm-a
nodepaper validate
nodepaper build
```

构建成功后，PDF 位于：

```text
dist/paper.pdf
```

## 项目选择

NodePaper 的操作对象是 Project 目录，不是单独的 Markdown 文件。

在项目根目录运行：

```powershell
nodepaper validate
nodepaper build
```

在项目子目录运行时，NodePaper 会向上查找 `nodepaper.yaml`。也可以显式指定项目目录：

```powershell
nodepaper build D:\papers\cumcm-a
```

NodePaper 不保存全局“当前项目”。

## 常用命令

```powershell
nodepaper
nodepaper init <project-directory>
nodepaper doctor [project-directory]
nodepaper validate [project-directory]
nodepaper build [project-directory]
nodepaper clean [project-directory]
nodepaper clean [project-directory] --all
nodepaper --help
nodepaper --version
```

- `clean` 删除中间构建文件；
- `clean --all` 额外删除 `dist/`；
- `nodepaper` 无参数运行时会根据当前位置提示下一步。

机器可读输出：

```powershell
nodepaper build D:\papers\cumcm-a --format json
```

## 项目配置

最小单文件配置：

```yaml
version: 1
profile: cumcm
source: paper.md
output:
  file: dist/paper.pdf
```

有序多文件配置：

```yaml
version: 1
profile: cumcm
sources:
  - sections/01-abstract.md
  - sections/02-problem.md
  - sections/03-model.md
output:
  file: dist/paper.pdf
```

Source 按配置顺序处理，不自动扫描目录。

常用可选配置：

```yaml
appendix:
  numbering: alpha
highlight:
  style: tango
linespread: 1.25
abstractLinespread: 0.95
mathFont: cm
```

## Markdown 示例

第一个 Source 使用 YAML Front Matter：

```markdown
---
title: 论文标题
problem: A
keywords:
  - 关键词一
  - 关键词二
---

# 摘要

在此撰写摘要。
```

文献引用：

```markdown
该方法可用于需求预测 [@wang2024]。
```

图片和交叉引用：

```markdown
![结果图](images/result.png){#fig:result width=80%}

见图 @fig:result。
```

公式和交叉引用：

```markdown
$$
x = 1
$$ {#eq:model}

见式 @eq:model。
```

复杂表格或公式可以使用项目配置中显式声明的 LaTeX Fragment。

## 主要能力

- 单文件和有序多文件项目；
- 中文摘要、关键词和多级标题；
- 数学公式、图片、表格和脚注；
- 图、表、公式和章节交叉引用；
- BibTeX 文献与数字上标引用；
- 代码高亮和长代码块；
- 可配置附录编号；
- PDF 书签大纲；
- 项目验证、环境诊断、构建日志和并发锁；
- 文本和 JSON 输出。

## 当前限制

- 仅正式面向 Windows 10/11 x64；
- 需要外部 TeX Live 或 MiKTeX；
- 当前 CUMCM Profile 仍是候选版本；
- 不提供 GUI、HTTP 服务、DOCX 或 Typst 输出；
- 不自动执行论文代码；
- 不上传论文内容；
- 不支持直接转换孤立 Markdown 文件。

仓库中保留的 SCAU PowerShell 模板尚未迁移为正式 NodePaper Profile，详见：

- [`README.SCAU-COMPAT.zh-CN.md`](README.SCAU-COMPAT.zh-CN.md)
- [`README.SCAU-COMPAT.md`](README.SCAU-COMPAT.md)

## 许可证

NodePaper 原创代码采用 MIT License。第三方组件及其许可证见 [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md)。
