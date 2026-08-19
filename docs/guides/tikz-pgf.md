# TikZ / PGF Fragment 指南

NodePaper 可以把项目内声明过的 `.tex` 或 `.pgf` Fragment 原样插入论文。推荐流程是：在外部绘图工具中导出、把文件放进 Project、加入 `latexFragments` 白名单，再在 Markdown 中用 `\input{...}` 指定插入位置。NodePaper 不执行绘图脚本，也不会自动扫描或插入文件。

## 最短可运行项目

把 Fragment 放在 `figures/model.tex`：

```tex
\begin{figure}
\centering
\begin{tikzpicture}
  \draw[->] (0,0) -- (3,0) node[right] {$x$};
  \draw[thick, blue] (0,0) -- (1,1) -- (2,1.5);
\end{tikzpicture}
\caption{模型结果}\label{fig:model}
\end{figure}
```

在 `nodepaper.yaml` 中声明它：

```yaml
version: 1
profile: cumcm
source: paper.md
latexFragments:
  - figures/model.tex
output:
  file: dist/paper.pdf
```

在 `paper.md` 的目标位置插入并用原生 LaTeX 标签引用：

```markdown
\input{figures/model.tex}

图 \ref{fig:model} 展示模型结果。
```

然后运行：

```powershell
nodepaper validate .
nodepaper build .
```

Fragment 内部的标签不会进入 Pandoc 的交叉引用索引，因此使用 `\label` / `\ref`，不要用 `@fig:model` 引用 Fragment 内的标签。

## 从 Matplotlib 导出 PGF

Matplotlib 官方 PGF backend 可以把图保存成低层 PGF 绘图命令：

```python
import matplotlib
matplotlib.use("pgf")
import matplotlib.pyplot as plt

plt.plot([0, 1, 2], [0, 1, 0.6])
plt.xlabel("time")
plt.tight_layout()
plt.savefig("figures/model.pgf")
```

随后将 `figures/model.pgf` 加入 `latexFragments` 并在 Markdown 中写 `\input{figures/model.pgf}`。Matplotlib 提醒，导出时和论文中字体配置不同可能造成文字对齐偏差；自定义 `pgf.preamble` 也可能引入当前 Profile 不具备的包。优先从无自定义 preamble 的小图开始。参考 [Matplotlib PGF backend 官方文档](https://matplotlib.org/stable/users/explain/text/pgf.html)。

`tikzplotlib` 一类转换器通常输出 `pgfplots` 语法，而不只是基础 PGF。NodePaper v0.1 没有承诺完整 `pgfplots` 支持；导出前先确认产物没有 `\begin{axis}`、`\addplot` 等依赖，或改用 Matplotlib 官方 PGF backend。`pgfplots` 本身是建立在 PGF/TikZ 之上的独立 TeX 包，参见其[官方仓库](https://github.com/pgf-tikz/pgfplots)。

## 当前支持矩阵

| 内容 | v0.1 状态 | 说明 |
|---|---|---|
| 基础 `tikzpicture`、`\draw`、节点 | 已验证 | Profile 预加载 TikZ；最小 Fixture 在 CI/E2E 编译 |
| Fragment 内 `\usetikzlibrary` | 已验证但按库逐案确认 | `layout-stress` 验证 shapes、arrows.meta、positioning |
| 低层 `pgfpicture` / Matplotlib 风格 `.pgf` | 已验证 | 不需要 `pgfplots` 的产物 |
| `pgfplots`、`axis`、`addplot` | 未承诺 | 当前不作为 v0.1 支持面，可能构建失败 |
| Fragment 自行 `\usepackage` / `\documentclass` | 禁止 | 包由 Profile 管理，完整文档命令会被 Validate 拒绝 |
| shell 命令、嵌套 `\input`、越界路径 | 禁止 | Fragment 是受控内容，不是任意 LaTeX 执行入口 |

## 字体、路径与包

- Fragment 路径相对 Project 根目录书写，使用 `/`；文件必须位于 Project 内并明确列入 `latexFragments`。
- 中文文本使用项目构建环境可用的 CJK 字体。导出工具测量文字时使用的字体应尽量与最终 PDF 一致。
- Fragment 不能通过 `\usepackage` 自行增加包。若导出文件要求额外包，它不属于当前稳定支持面。
- 大型散点图可能生成非常大的 `.pgf` 并耗尽 TeX 内存；此类图建议栅格化密集图层或直接导出 PDF/PNG。
- Fragment 是论文内容的一部分；只加入可信、可审查的文本文件，不要从不可信来源直接复制 LaTeX。

## 常见诊断

| Code | 含义与处理 |
|---|---|
| `NP2503` | Fragment 路径越出 Project；改为项目内相对路径 |
| `NP2504` | 已声明文件不存在；检查拼写和扩展名 |
| `NP2506` | 出现 `\documentclass`、`\usepackage` 等完整文档命令；删除并使用 Profile 已提供能力 |
| `NP2507` | Fragment 嵌套包含其他文件；展开为单个受控文件 |
| `NP2508` | 出现命令执行能力；删除该命令 |
| `NP2509` | Markdown 输入了未声明文件；加入白名单或移除输入 |
| `NP2511` | 文件已声明但未插入；在 Markdown 写对应的 `\input{...}` |
| `NP3202` | Markdown 交叉引用目标不存在；Fragment 标签改用 `\ref` |
| `NP6102` / `NP6105` | 构建日志发现未解析引用或 LaTeX 致命错误；先缩减为最小 Fragment，再检查字体、库和包依赖 |

仓库中的 `tests/fixtures/tikz-basic`、`pgf-basic` 是最小正向工程；`fragment-missing`、`fragment-path-traversal`、`fragment-forbidden-command`、`fragment-nested`、`fragment-command-execution`、`fragment-undeclared-input` 和 `unknown-crossref` 固定负向边界。`layout-stress` 继续承担组合排版回归。
