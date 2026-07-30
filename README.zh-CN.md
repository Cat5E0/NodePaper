# NodePaper

NodePaper 是一个面向 Windows 的 Go CLI，用于把包含 `nodepaper.yaml` 的 Markdown Project 构建为 PDF。

当前 v0.1 开发重点是 CUMCM 2026 电子版候选 Profile。NodePaper 负责项目发现、配置、验证、诊断、构建锁、日志和产物发布；Pandoc、pandoc-crossref、Citeproc、PowerShell、latexmk 和 XeLaTeX 负责文档转换与排版。

> 当前状态：开发中。CUMCM Profile 尚未完成 MiKTeX、Windows 10、Race Detector、发布 ZIP 和人工 PDF 排版门槛，不代表比赛官方认证。

## 正式工作流

NodePaper 的操作对象是 Project 目录，不是孤立 Markdown 文件：

```text
my-paper/
├── nodepaper.yaml
├── paper.md
├── references.bib
├── images/
├── dist/
│   └── paper.pdf
└── .nodepaper/
    ├── build/
    ├── logs/
    └── build.lock
```

创建、检查、验证和构建：

```powershell
nodepaper init D:\papers\cumcm-a
nodepaper doctor D:\papers\cumcm-a
nodepaper validate D:\papers\cumcm-a
nodepaper build D:\papers\cumcm-a
```

进入项目后可以省略路径：

```powershell
cd D:\papers\cumcm-a
nodepaper validate
nodepaper build
```

也可以从项目子目录运行，NodePaper 会向上查找 `nodepaper.yaml`。

清理中间文件：

```powershell
nodepaper clean D:\papers\cumcm-a
nodepaper clean D:\papers\cumcm-a --all
```

- `clean` 删除 `.nodepaper/build/`；
- `clean --all` 额外删除 `dist/`。

机器可读输出：

```powershell
nodepaper build D:\papers\cumcm-a --format json
```

JSON stdout 是单个带 `schemaVersion` 的对象；普通日志不会混入 JSON stdout。

## `nodepaper.yaml`

单文件：

```yaml
version: 1
profile: cumcm
source: paper.md
output:
  file: dist/paper.pdf
```

多文件：

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

Source 按配置顺序处理，不自动扫描目录，也没有全局“当前项目”状态。

## Markdown 基线

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

交叉引用：

```markdown
![结果图](images/result.png){#fig:result width=80%}

见图 @fig:result。

$$
x = 1
$$ {#eq:model}

见式 @eq:model。
```

当前 CUMCM 正式文献路线是：

```text
references.bib + Pandoc Citeproc + 固定 CSL
```

## 当前 CUMCM 电子版行为

- 第一页为标题、摘要和关键词；
- 不生成目录；
- 不生成纸质版承诺书和编号页；
- 使用固定 CUMCM 2026 候选 Profile；
- 支持有序单文件/多文件、中文 Crossref 和 Citeproc；
- PDF 发布前检查非空、Header、EOF 和 20 MB 上限；
- 同一 Project 同时只允许一个写入型 Build；
- Profile 构建时只读。

## 已确认但尚未实现的 v0.1 工作

以下内容是已确认规划，不是当前可用能力：

- Pandoc 内置代码语法高亮、长行换行和多页代码；
- 保留“附录”总标题，默认附录 A/B，并支持连续编号和不编号；
- 配置中显式声明、限制在 Project Root 内的 LaTeX Fragment，用于复杂表格和公式；
- Profile 版本与资源 SHA-256 日志；
- LaTeX 静态契约、严格缺字/Overflow 检查、PDF 非视觉几何检查；
- `layout-stress` 和本地参考论文 E2E。

NodePaper 不计划开发 Markdown 包含 Markdown 的自定义 Include。多章节使用有序 `sources`；复杂排版使用受控 LaTeX Fragment。

## 从源码测试

统一入口：

```powershell
.\scripts\test-unit.ps1
.\scripts\test-integration.ps1
.\scripts\test-e2e.ps1
.\scripts\test-all.ps1
```

当前 `test-release.ps1` 会主动阻止发布，因为 ZIP、MiKTeX、Windows 10 和人工检查门槛尚未完成。

源码树 E2E 通过测试专用环境变量定位构建脚本和 Profile；这些环境变量不是正式 CLI 协议。

## 现有 SCAU PowerShell 兼容入口

仓库仍保留原有 SCAU 毕业论文、课程作业和实验报告 PowerShell 模板与命令。它们尚未迁移为新的正式 Project Profile，不能等同于 Go CLI 已正式支持。

原有详细说明已保存在：

- [`README.SCAU-COMPAT.zh-CN.md`](README.SCAU-COMPAT.zh-CN.md)
- [`README.SCAU-COMPAT.md`](README.SCAU-COMPAT.md)

CUMCM 稳定后，架构计划保留迁移以下内置 Profile 的能力：

```text
scau-thesis
scau-assignment
scau-experiment
```

在迁移、许可证、Fixture、E2E 和 PDF 人工检查完成前，不宣称它们已由新 CLI 正式支持。

## 当前限制

- 没有正式发布 ZIP；
- 不提供 GUI、HTTP Server 或 VS Code 扩展；
- VS Code 内置 Markdown 预览不等于最终 Pandoc/LaTeX PDF；
- 只正式面向 Windows 10/11 x64，发布前平台门槛仍未全部完成；
- 不自动安装完整 TeX；
- 不自动执行论文代码；
- 不上传论文。

VS Code 扩展将在 CLI、JSON Schema、Diagnostic 文件/行号和 Project 语义稳定后作为薄适配器开发，不在扩展中重新实现构建逻辑。
