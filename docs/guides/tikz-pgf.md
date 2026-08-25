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

`tikzplotlib` 一类转换器通常输出 `pgfplots` 语法（`\begin{axis}`、`\addplot`），而不只是基础 PGF。这类产物现在可以直接用：Profile 会在检测到 `pgfplots.sty` 时加载它，见下一节。`pgfplots` 本身是建立在 PGF/TikZ 之上的独立 TeX 包，参见其[官方仓库](https://github.com/pgf-tikz/pgfplots)。

唯一需要自己处理的是**数据**：`\addplot table {data.dat}` 一类引用外部数据文件的写法不受支持——`nodepaper export` 只把 Markdown 里引用到的图片复制进交付目录，从不扫描 Fragment，那个 `.dat` 不会跟着走。把数据内联进 Fragment（`\addplot coordinates {...}`，或 `\addplot table {` 后面直接写出数据行）。`\pgfplotstableread` 会被 Validate 直接拒绝（`NP2507`），原因相同。

## 当前支持矩阵

| 内容 | v0.1 状态 | 说明 |
|---|---|---|
| 基础 `tikzpicture`、`\draw`、节点 | 已验证 | Profile 预加载 TikZ；最小 Fixture 在 CI/E2E 编译 |
| Fragment 内 `\usetikzlibrary` | 已验证但按库逐案确认 | `layout-stress` 验证 shapes、arrows.meta、positioning |
| 低层 `pgfpicture` / Matplotlib 风格 `.pgf` | 已验证 | 不需要 `pgfplots` 的产物 |
| `pgfplots`、`axis`、`addplot` | 已验证，条件可用 | Profile 按 `\IfFileExists{pgfplots.sty}` 加载并钉 `compat=1.18`；`pgfplots-basic` Fixture 在 E2E 编译并断言坐标轴刻度进了 PDF。装了才有——见下面「pgfplots 的条件加载」 |
| Fragment 里的 `\pgfimage`、`\pgfdeclareimage`、`\pgfplotstableread` | 禁止 | 都是读外部文件；这些文件不会被 export 的 staging 复制，会让收件人编译失败。栅格图层请直接导出 PDF/PNG 用 Markdown 插图，数据请内联 |
| Fragment 自行 `\usepackage` / `\documentclass` | 禁止 | 包由 Profile 管理，完整文档命令会被 Validate 拒绝 |
| shell 命令、嵌套 `\input`、越界路径 | 禁止 | Fragment 是受控内容，不是任意 LaTeX 执行入口 |

## pgfplots 的条件加载

`pgfplots` 由 Profile 模板加载，Fragment 不能自己 `\usepackage`（`NP2506`）。但它是**条件加载**的：

```tex
\IfFileExists{pgfplots.sty}{%
  \usepackage{pgfplots}%
  \pgfplotsset{compat=1.18}%
}{}
```

这么写有两个原因：

- **`pgfplots` 不在 MiKTeX 精简安装里。** 无条件 `\usepackage{pgfplots}` 会让任何没装它的机器每一次构建都失败，包括根本不画图的用户。条件加载让「不画图的人完全不受影响」成为可保证的事。
- **`compat` 是必需的，不是装饰。** 不设 `compat` 时 pgfplots 每份文档都会发 `running in backwards compatibility mode` 警告，NodePaper 把未知的 LaTeX 警告归为 `NP6105` 并让构建失败——这正是早期「pgfplots 加进模板会让所有构建失败」那条记录的真实成因。钉在 `1.18` 而不用 `compat=newest`，是为了让同一份源码在不同机器上排出同样的图。

于是：

- 装了 `pgfplots`：`axis` / `addplot` 直接可用。
- 没装：其他一切照常，只有用到 `axis` 的 Fragment 会报 `Environment axis undefined`。

`nodepaper doctor` 会探测并报告 `pgfplots` 是否可用（这一项恒为 Pass——没装是绝大多数用户的正常状态，不是故障）。需要时安装：

```powershell
tlmgr install pgfplots
miktex packages install pgfplots
```

`nodepaper export` 生成的交付 `README.txt` 会在文档确实画了坐标轴时，把 `pgfplots` 写进给收件人的安装命令；`pgf` 则始终在列，因为 Profile 总是加载 TikZ。

## 字体、路径与包

- Fragment 路径相对 Project 根目录书写，使用 `/`；文件必须位于 Project 内并明确列入 `latexFragments`。
- 中文文本使用项目构建环境可用的 CJK 字体。导出工具测量文字时使用的字体应尽量与最终 PDF 一致。
- Fragment 不能通过 `\usepackage` 自行增加包。Profile 已加载 `tikz` 和（条件加载的）`pgfplots`；除此之外的包不属于当前稳定支持面。
- 大型散点图可能生成非常大的 `.pgf` 并耗尽 TeX 内存；此类图建议栅格化密集图层或直接导出 PDF/PNG。
- Fragment 是论文内容的一部分；只加入可信、可审查的文本文件，不要从不可信来源直接复制 LaTeX。

## 常见诊断

| Code | 含义与处理 |
|---|---|
| `NP2503` | Fragment 路径越出 Project；改为项目内相对路径 |
| `NP2504` | 已声明文件不存在；检查拼写和扩展名 |
| `NP2506` | 出现 `\documentclass`、`\usepackage` 等完整文档命令；删除并使用 Profile 已提供能力 |
| `NP2507` | Fragment 嵌套包含或读取其他文件（`\input`、`\includegraphics`、`\pgfimage`、`\pgfplotstableread` 等）；展开为单个受控文件，数据内联，栅格图层改走 Markdown 插图 |
| `NP2508` | 出现命令执行能力；删除该命令 |
| `NP2509` | Markdown 输入了未声明文件；加入白名单或移除输入 |
| `NP2511` | 文件已声明但未插入；在 Markdown 写对应的 `\input{...}` |
| `NP3202` | Markdown 交叉引用目标不存在；Fragment 标签改用 `\ref` |
| `NP6102` / `NP6105` | 构建日志发现未解析引用或 LaTeX 致命错误；先缩减为最小 Fragment，再检查字体、库和包依赖 |

仓库中的 `tests/fixtures/tikz-basic`、`pgf-basic`、`pgfplots-basic` 是最小正向工程；`fragment-missing`、`fragment-path-traversal`、`fragment-forbidden-command`、`fragment-nested`、`fragment-external-image`、`fragment-command-execution`、`fragment-undeclared-input` 和 `unknown-crossref` 固定负向边界。`layout-stress` 继续承担组合排版回归。
