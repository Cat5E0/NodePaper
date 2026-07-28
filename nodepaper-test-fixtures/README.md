# NodePaper 测试 Fixture 包

这是一套可直接复制到 NodePaper 仓库中的测试素材。

## 主要内容

```text
tests/
├── fixture-manifest.json
├── fixtures/
│   ├── minimal-valid/
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
└── helpers/
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

## 需要在 M3 验证的语法

以下写法依赖最终锁定的 Pandoc/pandoc-crossref 版本：

- `$$ ... $$ {#eq:id}` 公式标签；
- `: 表题 {#tbl:id}` 表格标签；
- `::: {#refs}` 文献列表位置。

锁定版本后应先做最小实验，再将确认语法固化为 E2E 和 Golden。

## Active Lock 测试

```powershell
.\tests\helpers\hold-active-lock.ps1 -ProjectDir <临时项目目录> -Seconds 60
```

保持该进程运行时，在另一个终端执行 `nodepaper build`，预期同项目构建被拒绝。
