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
latexFragments:
  - tables/complex-result.tex
  - equations/objective.tex
appendix:
  numbering: alpha
highlight:
  style: tango
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

不要在摘要正文中再次手写“关键词”段落；Profile 会根据 Front Matter 的 `keywords` 仅生成一次。

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

受控 Fragment 必须在 `latexFragments` 中声明，再从 Markdown 插入：

```markdown
\input{tables/complex-result.tex}

见\cref{tab:complex-result}。
```

Fragment 只允许 Project Root 内的相对 UTF-8 `.tex` 普通文件；禁止完整文档、加载宏包、嵌套 `input`、TeX I/O 和命令执行。附录统一写为：

```markdown
# 附录
## 测试数据
## 程序代码
```

`appendix.numbering` 接受 `alpha`（默认）、`continuous` 和 `none`。`highlight.style` 接受已审查的 Pandoc 内置样式 `tango`（默认）、`pygments`、`kate`；不需要 Python 运行时或 `shell-escape`。

## 当前 CUMCM 电子版行为

- 第一页为标题、摘要和关键词；
- 不生成目录；
- 不生成纸质版承诺书和编号页；
- 使用固定 CUMCM 2026 候选 Profile；
- 支持有序单文件/多文件、中文 Crossref、Citeproc 和受控 LaTeX Fragment；
- 使用 Pandoc 内置代码高亮，默认 Tango，可选 Pygments/Kate 配色，并使用可跨页的浅色细框；代码长行可安全断行；
- 数字上标引用可跳转到对应参考文献条目；
- 保留“附录”总标题，并支持 `alpha`、`continuous`、`none`；
- 构建日志记录 Profile 版本、完整资源 SHA-256 和 Fragment SHA-256，构建前后变化会失败；
- 未知 Warning、Overfull、缺字/字体和未解析引用会阻止发布；
- PDF 发布前检查非空、Header、EOF 和 20 MB 上限；真实 E2E 额外检查 A4、文字边界、字体嵌入和内容顺序；
- 同一 Project 同时只允许一个写入型 Build。

## 剩余 v0.1 工作

以下门槛尚未完成：

- MiKTeX、Windows 10、Race Detector、发布 ZIP 和最终人工 PDF 检查。

MinerU 全文导入和语义忠实度审查已延后至 v0.1 后研究项，不阻塞运行时与发布包。

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
