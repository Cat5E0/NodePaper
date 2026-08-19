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

## Manifest 契约

`tests/fixture-manifest.json` 的 `schemaVersion` 为 2。Config、Validate、路径和锁场景必须在临时副本上执行，并与 Manifest 中的退出码、success 和 Diagnostic Code 完全一致。

## M3 已固定的构建语法

以下写法已在 Pandoc 3.9、pandoc-crossref 0.3.24、Citeproc 和固定 GB/T 7714-2015 CSL 下通过单文件与多文件真实 E2E：

- `$$ ... $$ {#eq:id}` 公式标签；
- `: 表题 {#tbl:id}` 表格标签；
- `::: {#refs}` 文献列表位置；
- `[@key]` 与 `[@key1; @key2]` 文献引用；
- 跨 Source 的 `@sec:`、`@fig:`、`@tbl:` 和 `@eq:` 引用。

不传 `-Fixture` 时，`scripts/test-e2e.ps1` 串联 `minimal-valid`、`complete-single-file`、`complete-multi-file`、`tikz-basic`、`pgf-basic` 和 `layout-stress`。`powershell-baseline-valid` 继续保留为不含 Citeproc 的 M2 旧链基线，不代表候选 CUMCM Profile。

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
