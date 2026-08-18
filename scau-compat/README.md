# NodePaper SCAU Markdown 转 LaTeX 工具（现有兼容入口）

> 本文保存迁移前的 SCAU PowerShell 使用说明。它不是当前 CUMCM Project CLI 的主 README，也不表示 SCAU 模板已经迁移为正式 Profile。命令和模板在迁移完成前按现有兼容行为保留。

> 下面的命令都**从仓库根目录**运行：`Build-Paper.ps1`、`Bootstrap-Tools.ps1` 和内置的 `tools/` 与 NodePaper CLI 共用，留在根目录；SCAU 专属的脚本、模板、过滤器和示例在 `scau-compat/` 下。
>
> 注意两个脚本的 `-Input` 基准目录不同，各自相对**脚本所在目录**解析：`Build-Paper.ps1` 在仓库根，所以写 `.\scau-compat\examples\paper.md`；`Convert-MarkdownToScauLatex.ps1` 在 `scau-compat\` 里，所以写 `.\examples\paper.md`。

这个项目用于把一个 Markdown 论文文件转换成华南农业大学本科毕业论文 LaTeX 模板格式，并在本机存在 LaTeX 编译环境时自动生成 PDF。

项目优先面向 Windows x64 分发。Pandoc 和 pandoc-crossref 可以放在项目自己的 `tools/windows-x64` 目录中，普通用户不需要手动配置系统 PATH。

## 快速开始

首次准备内置工具：

```powershell
.\Bootstrap-Tools.ps1
```

只生成 LaTeX：

```powershell
.\scau-compat\Convert-MarkdownToScauLatex.ps1 -Input .\examples\paper.md
```

生成 LaTeX 并尝试编译 PDF：

```powershell
.\Build-Paper.ps1 -Input .\scau-compat\examples\paper.md
```

使用作业模板：

```powershell
.\Build-Paper.ps1 -Input .\scau-compat\examples\assignment.md -TemplateName assignment
```

使用实验报告模板：

```powershell
.\Build-Paper.ps1 -Input '实验一 GPIO输出控制实验.md' -TemplateName experiment
```

指定封面和尾页 PDF（可选）：

```powershell
.\Build-Paper.ps1 -Input '实验一 GPIO输出控制实验.md' -TemplateName experiment `
    -CoverPdf '.\封面.pdf' -LastPagePdf '.\尾页.pdf'

# 或者使用同一个文件的第1页做封面、最后1页做尾页：
.\Build-Paper.ps1 -Input '实验一 GPIO输出控制实验.md' -TemplateName experiment `
    -CoverLastPdf '.\封面加尾页.pdf'
```

使用毕业论文模板：

```powershell
.\Build-Paper.ps1 -Input .\scau-compat\examples\paper.md -TemplateName thesis
```

如果你的机器已经安装了 Pandoc 和 pandoc-crossref，也可以在开发时允许使用系统工具：

```powershell
.\Build-Paper.ps1 -Input .\scau-compat\examples\paper.md -AllowSystemPandoc
```

## 项目结构

```text
NodePaper/                       仓库根目录
  Build-Paper.ps1                入口（同时服务 CUMCM 路线）
  Bootstrap-Tools.ps1            下载内置 pandoc
  tools/windows-x64/
    pandoc/pandoc.exe
    pandoc-crossref/pandoc-crossref.exe
  logs/                          构建日志
  scau-compat/                   本目录：迁移前的 SCAU 工具链
    Convert-MarkdownToScauLatex.ps1
    SCAU-Thesis-Template-2026.tex  独立可编译的 2026 版论文模板
    references.bib
    README.md / README.en.md
    examples/
      paper.md
      assignment.md
      assignment-no-references.md
    filters/
      scau-blocks.lua
    templates/
      scau-thesis.template.tex
      scau-assignment.template.tex
      scau-experiment.template.tex
    image/
      SCAU-LOGO.jpg
      SCAU-LOGO.png
```

## Markdown 文件格式

输入文件使用 YAML front matter 描述文档元数据。

作业模板推荐这样写：

```yaml
---
assignment_type: 课程作业
title_zh: OpenStack 云平台机制分析
course: 云计算技术
author_zh: 张三
college: 数学与信息学院、软件学院
major: 信息与计算科学
student_id: "202225810103"
teacher: 李四
date: 2026-06-09
---
```

毕业论文模板使用更完整的论文元数据：

```yaml
---
title_zh: 基于Lyapunov泛函方法的时滞神经网络稳定性分析
title_en: Stability Analysis of Time-Delay Neural Networks Based on Lyapunov Functional Method
author_zh: 陈汝恒
author_en: Chen Ruheng
college: 数学与信息学院、软件学院
major: 信息与计算科学
student_id: "202225810103"
supervisor: 史晨阳
supervisor_title: 讲师
date: 2026-06-09
abstract_zh: |
  这里填写中文摘要。
keywords_zh:
  - 神经网络
  - 指数稳定性
  - Lyapunov-Krasovskii泛函
abstract_en: |
  Write the English abstract here.
keywords_en:
  - Neural networks
  - Exponential stability
  - Lyapunov-Krasovskii functional
acknowledgements: |
  这里填写致谢内容。
references_tex: |
  \bibitem{fridman2014}
  Fridman, E. (2014). \textit{Introduction to Time-Delay Systems}. Birkhäuser.
---
```

正文从一级标题开始：

```markdown
# 绪论 {#sec:intro}

如 @sec:intro 所示。

![系统结构图](image/system.png){#fig:system width=80%}

见 @fig:system。

$$
x(t+1)=Ax(t)+Bu(t)
$$ {#eq:model}

由 @eq:model 可得……
```

## 交叉引用

项目使用 `pandoc-crossref` 处理编号和交叉引用。推荐标签前缀：

- `sec:`：章节，例如 `# 绪论 {#sec:intro}`
- `fig:`：图片，例如 `![标题](a.png){#fig:a}`
- `tbl:`：表格，例如 `: 表题 {#tbl:result}`
- `eq:`：公式，例如 `$$ x=1 $$ {#eq:x}`

引用时直接写：

```markdown
见 @sec:intro、@fig:system、@tbl:result 和 @eq:model。
```

生成的 LaTeX 会继续使用模板中的 `hyperref` 和 `cleveref`。

## 参考文献和文献引用

作业模板中，参考文献是可选的：

- 有参考文献：自动生成“参考文献”页。
- 没有参考文献：不会生成“参考文献”页。

推荐的稳定写法是在 YAML 中写 `references_tex`，正文中用 LaTeX 的 `\cite{key}` 引用：

```yaml
---
references_tex: |
  \bibitem{openstackdocs}
  OpenStack Documentation. OpenStack Docs. \url{https://docs.openstack.org/}.
---
```

正文引用：

```markdown
OpenStack 的官方文档给出了核心服务的部署和配置说明 \cite{openstackdocs}。
```

也可以在 Markdown 文末写一个参考文献章节：

```markdown
# 参考文献

1. OpenStack Documentation. OpenStack Docs. <https://docs.openstack.org/>.
2. Mell, P. and Grance, T. The NIST Definition of Cloud Computing.
```

这种普通列表可以显示参考文献页，但不会自动支持 `\cite{key}` 编号引用。需要编号引用时，优先使用 `references_tex` + `\cite{key}`。

目前项目暂不默认启用 Pandoc citeproc 的 `[@key]` / BibTeX / CSL 自动格式化；这是后续可以继续增加的功能。

## 图片放在哪里

推荐把图片放在 Markdown 文件旁边的 `image/` 目录中：

```text
paper.md
image/
  system.png
  result-a.jpg
```

然后在 Markdown 中这样引用：

```markdown
![系统结构图](image/system.png){#fig:system width=80%}

见 @fig:system。
```

也可以引用项目根目录中的图片，例如当前示例使用：

```markdown
![华南农业大学标识示例](image/SCAU-LOGO.jpg){#fig:logo width=45%}
```

转换脚本已经把以下目录加入 Pandoc 的资源搜索路径：

```text
项目根目录
项目根目录/image
输入 Markdown 所在目录
输入 Markdown 所在目录/image
```

也就是说，图片既可以放在项目根目录的 `image/`，也可以放在 Markdown 文件同级的 `image/`。为了分发和迁移方便，推荐统一使用 `image/` 子目录，并使用相对路径。

## 调整图片大小

在 Markdown 中通过大括号 `{}` 属性控制图片尺寸。项目已启用 Pandoc 的 `link_attributes` 扩展，属性会透传给 LaTeX 的 `\includegraphics` 命令。

### 常用参数速查

| 写法                   | 效果           | 说明                               |
| -------------------- | ------------ | -------------------------------- |
| `{width=80%}`        | 宽度占页面 80%    | 最常用，按比例缩放                        |
| `{width=60%}`        | 宽度占页面 60%    | 同上，适合小图                          |
| `{width=\textwidth}` | 撑满整页宽        | 适合宽图 |
| `{width=6cm}`        | 固定宽度 6cm     | 适合需要精确控制的场合                      |
| `{height=5cm}`       | 固定高度 5cm     | 宽度按比例自动调整                        |
| `{scale=0.5}`        | 缩放到原始尺寸的 50% | 相对原图缩放                           |

### 完整示例

```markdown
![系统结构图](image/system.png){#fig:system width=80%}

![小图标](image/icon.png){#fig:icon width=3cm}

![宽图](image/wide.png){#fig:wide width=\textwidth}
```

### 同时设置多个属性

多个属性用空格分隔：

```markdown
![看图说话](image/demo.jpg){#fig:demo width=70% height=6cm}
```

> **注意**：同时指定 `width` 和 `height` 会改变图片宽高比，可能导致拉伸变形。大多数情况下只指定 `width` 就够了，高度会自动按比例缩放。

### 完整可用参数

任何 LaTeX `\includegraphics` 接受的参数都可以在大括号中使用，常见的有：

| 参数                | 示例值                         |
| ----------------- | --------------------------- |
| `width`           | `80%`, `10cm`, `\textwidth` |
| `height`          | `6cm`, `0.5\textheight`     |
| `scale`           | `0.5`, `1.2`                |
| `angle`           | `90`（旋转角度）                  |
| `keepaspectratio` | `true`                      |

### 为什么可以这样写？

数据流：

```
Markdown {width=80%}  →  Pandoc 解析为 Image 属性  →  LaTeX \includegraphics[width=0.8\textwidth]{...}
                                                                  ↓
                                                      模板 \adjustbox 自动兜底
                                                      防止超出页面
```

Pandoc 把 `width=80%` 转成 `width=0.8\textwidth` 写入 `.tex` 文件，xelatex 编译时直接交给 `\includegraphics`。即使不写任何属性，模板中的 `\pandocbounded` 也会用 `\adjustbox` 自动限制图片最大宽度和高度，不会撑破页面。

## 代码块和语法高亮

普通代码块可以这样写：

```markdown
```text
plain text
```
```

指定语言后，Pandoc 会生成带语法高亮的 LaTeX 代码环境：

```markdown
```python
def stable(delay: float) -> bool:
    return delay < 1.0
```
```

也可以写其他语言，例如：

```markdown
```matlab
A = [1 0; 0 1];
eig(A)
```

```powershell
.\Build-Paper.ps1 -Input .\scau-compat\examples\paper.md
```
```

当前项目默认使用 Pandoc 的 `tango` 代码高亮风格，并在 LaTeX 模板中内置了 `Shaded` / `Highlighting` 环境所需的宏定义。生成 PDF 时，如果本机 TeX 环境包含 `fvextra`，长代码行会自动换行；否则会退回到基础 `fancyvrb` 显示。

## 定理、引理和证明

`filters/scau-blocks.lua` 会把 fenced Div 转换成 LaTeX 环境：

```markdown
::: theorem
这里是定理内容。
:::

::: lemma
这里是引理内容。
:::

::: proof
这里是证明内容。
:::
```

支持的 class 包括：`theorem`、`thm`、`lemma`、`lem`、`remark`、`proof`。

## 实验报告模板

实验报告模板（`-TemplateName experiment`）专为课程实验报告设计：

- **无封面、无目录**：正文直接开始，不需要封面页和目录页。
- **标题全部使用宋体**：一级标题（`#`）四号宋体居中，二、三级标题（`##`、`###`）小四宋体左顶格。
- **正文格式**：宋体小四，1.5 倍行距，首行缩进两字距。
- **页脚**：居中显示"人工智能教研室制"，右侧显示页码。
- **外部 PDF 封面/尾页**：支持 `-CoverPdf`（封面 PDF）、`-LastPagePdf`（尾页 PDF）、`-CoverLastPdf`（同一文件第1页做封面、最后1页做尾页）。基于 LaTeX 的 pdfpages 宏包，无需额外安装工具。

适合用于实验课程报告的快速排版。

## 输出文件

默认输出在 `build/` 目录：

```text
build/Paper.tex
build/Paper.pdf
```

如果没有检测到 `latexmk` 和 `xelatex`，项目仍会生成 `build/Paper.tex`，但会跳过 PDF 编译。

## 日志文件

每次运行 `Build-Paper.ps1` 时，脚本会自动在 `logs/` 目录保存日志：

```text
logs/build-YYYYMMDD-HHMMSS.log
logs/latex-YYYYMMDD-HHMMSS.log
```

其中：

- `build-*.log`：自动记录本次构建入口、转换命令、LaTeX 编译命令、外部命令输出、退出码、生成的 `.tex` / `.pdf` 路径等信息。
- `latex-*.log`：复制自 `build/Paper.log`，保存完整 LaTeX 编译日志，排查宏包、图片、公式、引用错误时优先看这个文件。

如果构建失败，脚本会在终端输出最后一段 LaTeX 日志，并同时把完整日志复制到 `logs/`。

## 分发说明

发布包应包含：

```text
tools/windows-x64/pandoc/pandoc.exe
tools/windows-x64/pandoc-crossref/pandoc-crossref.exe
tools/versions.json
```

这样用户只需要运行 PowerShell 脚本，不需要单独安装 Pandoc 或 pandoc-crossref。
