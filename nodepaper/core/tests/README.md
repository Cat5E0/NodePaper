# NodePaper 测试 Fixture 包

这是一套可直接复制到 NodePaper 仓库中的测试素材。

## 主要内容

```text
tests/
├── fixture-manifest.json
├── fixtures/
│   ├── minimal-valid/
│   ├── tikz-basic/
│   ├── pgf-basic/
│   ├── powershell-baseline-valid/
│   ├── complete-single-file/
│   ├── complete-multi-file/
│   ├── nocite-only/
│   ├── citation-shapes/
│   ├── invalid-yaml/
│   ├── missing-frontmatter/
│   ├── missing-abstract/
│   ├── missing-image/
│   ├── missing-bib/
│   ├── missing-citation-key/
│   ├── duplicate-crossref/
│   ├── unknown-crossref/
│   ├── source-and-sources/
│   ├── duplicated-source/
│   ├── raw-latex-warning/
│   ├── 中文路径测试/
│   ├── path with spaces/
│   ├── path-traversal/
│   ├── damaged-lock/
│   └── stale-lock/
├── corpus/
│   └── real-world/{A163,C063}/
└── hold-active-lock.ps1
```

## 合法项目

### `minimal-valid`

快速冒烟项目，包含 Front Matter、摘要、公式、图片、交叉引用和 BibTeX 引用。

### `powershell-baseline-valid`

M2 专用的 PowerShell 过渡构建链基线。它同时包含 Validate 所需字段和旧 assignment 模板所需元数据，不包含尚待 M3 接入的 Citeproc 引用。该 Fixture 只证明 `Go → Build-Paper.ps1 → Pandoc/LaTeX → PDF` 可运行，不代表正式 CUMCM Profile。

### `complete-single-file`

完整单 Markdown 项目，覆盖：

- 标题、题号、关键词；
- 摘要；
- 多级章节；
- 列表、粗体、斜体、脚注和代码块；
- 行内和块级公式；
- 可引用公式；
- 两张程序生成图片；
- 两个表格；
- 图、表、公式、章节引用；
- 6 条 BibTeX，其中 1 条故意不引用；
- 中文和英文文献；
- 附录。

### `complete-multi-file`

拆分为 7 个 Markdown Source，包含跨文件章节、公式、图表和文献引用。

### `nocite-only`

正文不含任何行内文献引用，所有参考文献条目只通过 Front Matter 的 `nocite:` 字段列出（复现 A163 风格的合法写法）。`references.bib` 还含一条既不 nocite 也不引用的条目，用于确认导出只把 nocite 键转成 `\nocite{}`。它专门固定 M4-23 修复的导出缺陷：`nodepaper export --bib bibtex|biblatex` 在 nocite-only 项目上原先会导出一份没有任何 `\citation` 命令的 `.tex`，导致 bibtex/biber 报 “I found no `\citation` commands” 失败。

### `citation-shapes`

固定其他 Fixture 都不覆盖的行内引用写法。既有 Fixture 的引用一律是「空格 + 方括号 + 句末」，而真实论文里的引用常常紧贴中文、出现在句中。本 Fixture 覆盖四种形态：

- **引用标记紧贴 CJK 字符且位于句中**（`……甲类站点网络[@key]和乙类站点网络[@key2]等`）；
- **同一引用键在不同位置被引用三次**，须解析为同一个链接目标与同一序号；
- **行内引用与 `nocite` 并存**——这是「引用了一部分、只列出另一部分」的真实论文所需的组合，`nocite` 补发的 `\nocite{}` 不得挤掉真实的行内引用；
- **两个与三个连续序号**各一组，用于固定合并行为的差异。

`references.bib` 的注释写明了各键的首次出现顺序即预期序号（1～8），另含一条既不引用也不 nocite 的条目，用于确认它既不进文献表也不进导出的 `\nocite{}`。

本 Fixture 同时是两个**尚未定性**的排版问题的复现用例：① citeproc 路由在上标引用标记后留有空隙（`问题[1] 。`），而导出的 gbt7714 路由紧贴（`问题[1]。`）；② 连续序号的合并阈值不同——citeproc 三个连续才压成 `[5–7]`（en dash）、两个留 `[3,4]`，gbt7714 两个即压成 `[3-4]`（hyphen）。**E2E 刻意不断言标记的渲染形态**，只断言结构（同键同目标、nocite 键入表、未引用条目不入表），以免把未拍板的取舍固化成 Golden。

## 图片

图片已经包含在 Fixture 中：

- `demand-trend.png`
- `station-map.png`

它们由 `scripts/generate-test-images.py` 生成，因此不涉及外部图片版权。运行测试不要求安装 Python 或 Matplotlib，脚本只用于复现。

## BibTeX

`references.bib` 中的文献题名、期刊、机构和数据均为虚构测试内容，不可作为真实学术来源。

## 使用原则

1. 不要直接在原始 Fixture 中构建。
2. 测试时先复制到独立临时目录。
3. 原始 Fixture 保持只读。
4. `.nodepaper/`、`dist/`、日志和生成 PDF 不提交 Git。
5. `fixture-manifest.json` 是可执行契约，固定命令、退出码、success 和 Diagnostic Code。

## 源码与生成物边界

一个 Project 在构建后会同时出现两个不同职责的目录，它们不是两套源码，也不会互相替代：

| 路径 | 职责 | 是否提交 |
|---|---|---|
| `paper.md`、`sections/`、`images/`、`tables/*.tex`、`nodepaper.yaml` | 作者编写或审核过的 Project 源码 | 是 |
| `.nodepaper/` | NodePaper 的中间 TeX、日志、锁和其他工作状态 | 否 |
| `dist/` | 最终 PDF 等可重新生成的构建输出 | 否 |

测试必须在 Fixture 或 Corpus 的临时副本中构建。源目录中已有的 `.nodepaper/` 或 `dist/` 只可能是本机遗留生成物：它们被 Git 忽略，也不会进入测试语料 ZIP 或程序发布包；清理它们不改变源码。唯一例外是两个锁损坏测试中刻意提交的 `.nodepaper/build.lock` 输入文件，见根 `.gitignore` 的白名单注释。

## Manifest 契约

`tests/fixture-manifest.json` 的 `schemaVersion` 为 2。Config、Validate、路径和锁场景必须在临时副本上执行，并与 Manifest 中的退出码、success 和 Diagnostic Code 完全一致。

## M3 已固定的构建语法

以下写法已在 Pandoc 3.9、pandoc-crossref 0.3.24、Citeproc 和固定 GB/T 7714-2015 CSL 下通过单文件与多文件真实 E2E：

- `$$ ... $$ {#eq:id}` 公式标签；
- `: 表题 {#tbl:id}` 表格标签；
- `::: {#refs}` 文献列表位置；
- `[@key]` 与 `[@key1; @key2]` 文献引用；
- 跨 Source 的 `@sec:`、`@fig:`、`@tbl:` 和 `@eq:` 引用。

不传 `-Fixture` 时，`scripts/test-e2e.ps1` 串联 `minimal-valid`、`complete-single-file`、`complete-multi-file`、`nocite-only`、`citation-shapes`、`tikz-basic`、`pgf-basic` 和 `layout-stress`，随后用 `-TildeWorkRoot` 再跑一遍 `minimal-valid`，共 9 个场景。其中 `nocite-only` 与 `citation-shapes` 额外触发导出路由的文献回归块（`--bib bibtex|biblatex|inline` 三种模式）。`powershell-baseline-valid` 继续保留为不含 Citeproc 的 M2 旧链基线，不代表候选 CUMCM Profile。

`layout-stress` 覆盖受控 LaTeX Fragment、跨页长表格、多页代码、Pandoc 内置高亮、长 URL/路径、公式、图片、脚注和附录。`highlight-showcase` 只用于 Tango、Pygments、Kate 的聚焦视觉比较，不承担完整排版压力验收。E2E 检查生成 LaTeX 契约、A4、PDF 文字顺序与边界、字体嵌入、稳定标记、零关键 Warning，并支持：

```powershell
.\scripts\test-e2e.ps1 -Fixture layout-stress -AppendixNumbering continuous
.\scripts\test-e2e.ps1 -Fixture layout-stress -AppendixNumbering none
.\scripts\test-e2e.ps1 -Fixture layout-stress -HighlightStyle pygments
.\scripts\test-e2e.ps1 -Fixture layout-stress -HighlightStyle kate
.\scripts\test-e2e.ps1 -Fixture layout-stress -ReviewOutput D:\nodepaper-review
```

`tikz-basic` 与 `pgf-basic` 分别固定最小 TikZ 和低层 PGF 正向契约。`fragment-*` 负向 Fixture 固定 `NP2503`～`NP2509` 的路径、缺失文件、完整文档命令、嵌套依赖、命令执行和未声明输入契约；`unknown-crossref` 固定 `NP3202`。详细使用方法见 [`docs/guides/tikz-pgf.md`](../docs/guides/tikz-pgf.md)。

## Active Lock 测试

```powershell
.\tests\hold-active-lock.ps1 -ProjectDir <临时项目目录> -Seconds 60
```

保持该进程运行时，在另一个终端执行 `nodepaper build`，预期同项目构建被拒绝。
