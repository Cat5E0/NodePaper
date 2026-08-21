# 项目编写、排版与导出

本指南面向 `cumcm` Profile 的 NodePaper v0.1 项目。它补充根目录 README 的最短上手路径，重点说明真实论文在拆分、摘要、表格、受控 LaTeX 和交付时容易踩到的边界。命令和字段以当前 CLI 与 `nodepaper.yaml` 配置契约为准。

如果只需要插入 TikZ / PGF 图，请先看 [TikZ / PGF Fragment 指南](tikz-pgf.md)；这里仅说明它和普通项目源码、导出的关系。

## 1. 先决定源码是单文件还是多文件

一个 NodePaper Project 的根目录必须有 `nodepaper.yaml`。两种组织方式都支持，区别只在论文 Markdown 的组织，不会改变输出 PDF 的能力。

```text
my-paper/
├─ nodepaper.yaml
├─ paper.md                         # 单文件项目使用
├─ sections/                        # 多文件项目使用
│  ├─ 01-frontmatter-abstract.md
│  ├─ 02-introduction.md
│  └─ 03-model.md
├─ figures/                         # 图片、.tex 或 .pgf Fragment 的常见位置
├─ tables/
├─ references.bib
└─ dist/                            # 生成物；不提交
```

### 单文件：适合短论文或先快速成稿

```yaml
version: 1
profile: cumcm
source: paper.md
output:
  file: dist/paper.pdf
```

`source` 只指定一个 Markdown 文件。C063 真实语料就是这种结构：主论文在 `paper.md`，少量无法用 Markdown 表达的表格单独放在 `tables/`。

### 多文件：适合多人协作或篇幅较长的论文

```yaml
version: 1
profile: cumcm
sources:
  - sections/01-frontmatter-abstract.md
  - sections/02-introduction.md
  - sections/03-model.md
  - sections/04-appendix.md
output:
  file: dist/paper.pdf
```

`sources` 的列表顺序就是最终拼接顺序；建议用带序号的文件名，让顺序在文件管理器中也一目了然。A163 真实语料使用了这种方式。

`source` 与 `sources` **互斥**，不能同时写；两者至少要有一个。所有路径都相对 Project 根目录书写，并优先使用 `/`。

### Front Matter 只放在第一个来源文件

首个 Markdown 文件必须以 YAML Front Matter 开始，并提供至少 `title`、`problem` 和 `keywords`：

```markdown
---
title: 基于优化算法的示例模型
problem: C
keywords:
  - 优化
  - 建模
reference-section-title: 参考文献
---

# 摘要
```

单文件项目中它位于 `paper.md` 顶部；多文件项目中它位于 `sources` 的第一项。后续 Markdown 文件不能再出现 Front Matter，否则 `validate` 会报错。论文标题、摘要、章节和附录仍按最终的拼接顺序组成同一篇文档，不应把每个章节当成独立论文。

## 2. `nodepaper.yaml` 配置速查

当前 v0.1 只支持 `profile: cumcm`，并会拒绝拼错或未知的配置字段。下面是所有当前可用字段；没有列出的字段不要猜测添加。

| 字段 | 含义、取值与默认值 |
| --- | --- |
| `version` | 必填，当前只能是 `1`。 |
| `profile` | 必填，当前只能是 `cumcm`。 |
| `source` / `sources` | 二选一。前者是单个 Markdown 文件；后者是按顺序拼接的 Markdown 列表。 |
| `latexFragments` | 可选的 Project 内 `.tex` / `.pgf` 白名单。声明后才能在 Markdown 用 `\input{...}` 插入。 |
| `appendix.numbering` | `alpha`（默认）、`continuous` 或 `none`。 |
| `appendix.newPage` | `true`（默认）时附录从新页开始；设为 `false` 才取消该换页。 |
| `highlight.style` | `tango`（默认）、`pygments` 或 `kate`。控制代码块高亮主题。 |
| `linespread` | 正文行距，范围 `1.0`–`1.3`，默认 `1.25`。 |
| `abstractLinespread` | 摘要与关键词区域的行距，范围 `0.85`–`linespread`，默认 `0.95`。 |
| `titleAbstractSkip` | 摘要标题到摘要正文的垂直间距，单位 em，范围 `0`–`5`，默认 `0.5`。写 `0` 表示不留空隙。 |
| `abstractKeywordsSkip` | 摘要正文到关键词的垂直间距，单位 em，范围 `0`–`5`，默认 `0.8`。写 `0` 表示不留空隙。 |
| `mathFont` | `cm`（默认）或 `newtx`。 |
| `output.file` | 可选；构建 PDF 的目标相对路径，默认 `dist/paper.pdf`。 |

一个含摘要局部调整和 Fragment 的示例：

```yaml
version: 1
profile: cumcm
abstractLinespread: 0.90
sources:
  - sections/01-frontmatter-abstract.md
  - sections/02-background.md
  - sections/03-model.md
latexFragments:
  - tables/complex-result.tex
  - figures/model.pgf
appendix:
  numbering: none
output:
  file: dist/paper.pdf
```

修改配置后先运行：

```powershell
nodepaper validate .
nodepaper build .
```

`validate` 负责尽早发现配置、来源文件和 Fragment 声明问题；`build` 才生成 PDF。需要被脚本读取的稳定结构化结果时，可在任一命令末尾加 `--format json`；默认 `--format text` 用于人读终端输出。

## 3. 摘要挤到第二页时：只压缩摘要区域

如果关键词刚好被挤到第二页，不要先降低整篇论文的 `linespread`。优先在 `nodepaper.yaml` 设置较小的 `abstractLinespread`，例如 A163 使用的：

```yaml
abstractLinespread: 0.90
```

行距压到 `0.85`（下限）仍不够时，再收摘要区的两处垂直间距——它们比行距更不容易被读者察觉：

```yaml
titleAbstractSkip: 0.2      # 摘要标题到正文，默认 0.5em
abstractKeywordsSkip: 0.3   # 摘要正文到关键词，默认 0.8em
```

单位是 em，范围 `0`–`5`，写 `0` 就是完全不留空隙。三者都改完仍装不下，才应考虑精简摘要本身。

它只作用于摘要和关键词区域，正文仍保持 `linespread` 的设定。建议从默认 `0.95` 开始，每次以 `0.01` 或 `0.02` 小步下调，运行 `nodepaper build .` 后人工检查 PDF 首页。`0.90` 是已被真实多文件语料采用的值，不是所有论文都必须使用的固定答案。

当前 Profile **没有** `abstractParagraphSpacing`、关键词间距或摘要独立页边距等单独配置项。达到 `0.85` 下限后仍放不下时，应优先压缩摘要文字、删减关键词或调整内容本身；需要突破模板边界的局部 TeX 调整，先保留最小复现并作为模板/产品问题评估，不要直接修改 `.nodepaper` 里的生成文件。

## 4. Markdown 表格：先控制总宽度，再控制列比例

普通表格应优先保持为 Markdown。表格下方的表题属性是稳定的排版控制入口：

```markdown
| 符号 | 含义 |
| :---: | :--- |
| $x$ | 决策变量 |
| $y$ | 目标函数值 |

: 变量符号说明 {#tbl:variables width=80% ratios="1:4"}
```

`#tbl:variables` 是可供交叉引用使用的表格标识；`width` 和 `ratios` 是 NodePaper 的表格布局属性。

### `width` 的三种选择

| 写法 | 效果 | 适用情况 |
| --- | --- | --- |
| `width=auto` | 保留自然宽度，不强制把表格拉满正文。 | 符号表、短字段、小表。 |
| `width=80%` | 表格总宽度为正文宽度的 80%。 | 需要限制占页面积、但仍要明确宽度的表。 |
| `width=full` | 表格总宽度为完整正文宽度。 | 列多或内容较长、需要尽量减小换行的表。 |

百分比必须大于 `0%` 且不超过 `100%`。`width=auto` 与 `ratios` 不能同时使用；没有 `width` 时也不能只写 `ratios`。

### `ratios` 是唯一可靠的相对列宽控制

对 `width=full` 或百分比宽度，可加与列数相同的正数比例：

```markdown
: 平均销售单价表 {#tbl:unit-price width=85% ratios="1:1:2:1:1:2"}
```

上例六列按 `1:1:2:1:1:2` 分配 85% 的总宽度；比例可以用 `:` 或 `,` 分隔。比例项必须刚好一项对应一列，且每项大于零。长描述列应得到更大的比例，代码、符号或短数值列通常可更窄。

管道表分隔行中的横杠数量，例如 `| --- | ------- |`，**不是**列宽比例契约；冒号仅表示对齐方式。Pandoc 有时会从分隔行推断相对宽度，但短表或自然宽度表可能丢失这项信息，所以同一张表会在一次文本改动后突然占满页面，或看似留下许多空白。需要可重复的结果时，改用 `width` 与 `ratios`，不要反复增减横杠来试排版。

### 表格排得不好时的判断顺序

1. 表不该铺满页面：先写 `width=auto`，而不是调分隔行横杠。
2. 表需要占用确定宽度：写 `width=80%` 或 `width=full`。
3. 某列换行太多、其他列空白：保持总宽度，再为长文本列增大 `ratios`；单纯扩大总宽度不一定解决比例失衡。
4. 内容本身很长：精简表头、缩短单位写法，或把说明移到表外。过窄列中的中文必然换行，这不是 Markdown 空格造成的。
5. 需要合并单元格、局部字号、分组竖线等：转入下一节的 LaTeX Fragment。

表题或表注里的 `**文字**` 是语义加粗，不是普通文字的另一种写法。标准 Windows 字体分支会把宋体的粗体映射到黑体，字体回退分支也可能合成粗体，所以误加粗会让局部字形、字重与周围文字明显不一致。原文没有强调含义时应去掉 `**...**`；确实需要强调时可以保留。表头中确实需要强调的字段同样属于内容级样式，C063 的复杂表格 Fragment 就对表头使用了 `\textbf{...}`。

当前 v0.1 没有供 Markdown 单独设置“表题字体”或“表注字体”的配置字段，表题的基础样式由 Profile 统一控制。若表注的字号、位置或与复杂表格的关系必须精确控制，应使用 Fragment 自己管理该表；不要把不存在的 YAML 字段当作配置项。

## 5. Markdown 的边界：受控 LaTeX Fragment

当表格只需总宽度和列比例时，Markdown 更易维护；下列需求才值得使用 `.tex` Fragment：

- 跨页表、合并单元格（例如 `\multirow`）；
- 局部字号、固定列宽、重复表头；
- 明确的分组竖线或其他 Markdown 无法表示的表格结构；
- TikZ、纯 PGF 命令文件等需要原样交给 LaTeX 的图形内容。

以复杂表为例，先把文件作为源码放在 Project 内：

```text
my-paper/
├─ paper.md
└─ tables/
   └─ complex-result.tex
```

再在配置中明确白名单：

```yaml
latexFragments:
  - tables/complex-result.tex
```

最后在 Markdown 希望出现的位置插入：

```markdown
如表所示：

\input{tables/complex-result.tex}
```

Fragment 路径相对 Project 根目录书写，必须位于 Project 内并且与 `latexFragments` 的条目一致。`.tex` / `.pgf` Fragment 是你维护的**源码**，应与 Markdown、图片、`nodepaper.yaml` 一起进入 Git；它不是构建产物。

Fragment 被刻意限制在文档正文片段的范围内：不要在其中放 `\documentclass`、`\usepackage`、命令执行或嵌套 `\input`。先运行 `nodepaper validate .`，它会在构建之前报告路径逃逸、未声明输入、嵌套输入等问题。复杂交叉引用也应以可验证的最小形式先构建成功后再扩展。

若表或图的 `\label{...}` 位于 Fragment 内，Markdown 的交叉引用检查无法读取那个标签。此时在正文中使用 LaTeX 的 `\ref{...}` 引用 Fragment 内的标签；不要把它误写成只能识别 Markdown 标签的 `@tbl:...` 或 `@fig:...`。例如 Fragment 内写 `\caption{结果}\label{tbl:complex-result}`，正文写“见表 `\ref{tbl:complex-result}`”。

## 6. 图形、TikZ 和 PGF

普通图片以相对 Project 根目录的 Markdown 路径引用；图像文件是 Project 源码，应提交 Git。例如：

```markdown
![模型结果图](figures/result.png){#fig:result width=80%}

见图 @fig:result。
```

`width=80%` 控制图片占正文宽度的比例，`#fig:result` 供交叉引用使用。构建前应确认该文件与 Markdown 一起存在于项目中；不要把图片仅放在 `.nodepaper/`、`dist/` 或项目外的临时目录。TikZ 或纯 PGF 命令文件则必须按与表格 Fragment 相同的“文件在项目内 → `latexFragments` 显式声明 → Markdown `\input` 插入”流程处理。

外部工具如何导出、Matplotlib PGF 的字体一致性、可用 TikZ 库和诊断码，见 [TikZ / PGF Fragment 指南](tikz-pgf.md)。当前已验证的是基本 TikZ 与纯 PGF 命令文件（`pgfpicture`）；`pgfplots` 不在当前支持承诺内，不能把能否偶然编译当作兼容性保证。

## 7. 哪些文件提交，哪些文件只留在本机

| 类别 | 典型内容 | Git / 公开语料包 |
| --- | --- | --- |
| 应提交的源码 | `nodepaper.yaml`、Markdown、图片、`.bib`、已声明的 `.tex` / `.pgf` Fragment | 提交；需要时随项目交付。 |
| 构建状态 | `.nodepaper/` 中的中间 TeX、日志、锁和工作目录 | 不提交。 |
| 成品输出 | `dist/` 中的 PDF 等可重新生成文件 | 不提交，也不进入测试 Fixture / Corpus 源包。 |
| 导出交付物 | `nodepaper export` 生成的可编辑 LaTeX 文件夹或 ZIP | 按交付需要另行保存；不要混入源项目。 |

`.nodepaper/` 和 `dist/` 会同时出现在项目里，但含义不同：前者是 NodePaper 的工作状态，后者是最终 PDF 的通常位置。二者都不应作为长期修改点；修改 `.nodepaper/build/` 中生成的 `paper.tex` 后，下次构建会被覆盖。`nodepaper init` 生成的 `.gitignore` 默认忽略这两类目录；手工建立项目时也应加入同样规则。

清理临时构建状态可用：

```powershell
nodepaper clean .
```

若确认也要删除可重新生成的 `dist/` 输出，再使用：

```powershell
nodepaper clean . --all
```

这会删除本地生成物，先确认其中没有只存在于本机、尚未备份的 PDF。

## 8. 导出可编辑的 LaTeX 工程

`build` 生成 PDF；`export` 生成的是可编辑 LaTeX 交付物，而不是 PDF。基本的文件夹导出：

```powershell
nodepaper export . --to ..\paper-latex
```

需要一个压缩包交付（打包发给别人，或上传到接受 ZIP 的平台）：

```powershell
nodepaper export . --to ..\paper-latex.zip
```

`--to` 以 `.zip` 结尾时（不区分大小写）生成 ZIP；其中的文件直接放在 ZIP 根目录，不额外套一层目录。`--to` 指向文件夹时生成目录，适合先检查或继续编辑；手动压缩时应压缩该目录的内容，而非再多包一层目录。

导出是给**已经会用 LaTeX、要接手精修或搬进别的环境**的人用的。想要的只是 PDF 时，装本地 TeX 走 `nodepaper build` 更短。

导出的文件是从 Markdown 源项目**单向**生成的。之后在 LaTeX 侧改动，不会回写到 Markdown 项目；长期维护仍应改源项目，再重新导出。

常用可选项：

```powershell
# 用 BibLaTeX / biber 交付；不写 --bib 时默认 bibtex
nodepaper export . --to ..\paper-latex --bib biblatex

# 生成后在临时副本中完整编译交付树；不在交付目录留下 PDF、.aux 或 .log
nodepaper export . --to ..\paper-latex.zip --verify

# 非空目录或目标 ZIP 已存在时，明确允许写入/替换
nodepaper export . --to ..\paper-latex.zip --force
```

`--bib` 可取 `bibtex`、`biblatex` 或 `inline`。**如果论文没有任何行内引用、也没有 `nocite`**（例如参考文献是手工写在正文里的编号列表），导出不会生成文献表，命令链里也不会有 `bibtex`／`biber` 这一步，并会给出一条 `NP8012` 信息提示——这是正常结果，不是错误：没有 bib 条目就无法产生引用。`references.bib` 仍会随包交付，`README.txt` 里会说明它未被使用。`--verify` 需要本机相应的 TeX 工具；工具缺失时会给出 Warning，导出本身仍完成。若目标不是空目录或 ZIP 已存在，默认会拒绝覆盖，应确认目标后再加 `--force`。将导出目标放在 Project 根目录内会触发提醒，因为它容易被误提交；建议使用项目外的交付目录。

### Overleaf

导出的工程可以上传 Overleaf（**New Project → Upload Project**，编译器切换为 **XeLaTeX**，必要时把主文档设为 `paper.tex`），但先看清这条限制：**Overleaf 免费版编译限时 10 秒**（[官方 Plan Limits](https://docs.overleaf.com/getting-started/free-and-premium-plans/plan-limits)，付费版 240 秒）。一份完整论文几十页、要跑多遍 XeLaTeX，在 10 秒内跑不完；瓶颈在 preamble 的 ctex + xeCJK 字体实例化，冷启动下连三页的最小样本也会超时，所以**把文档改小不是可行的绕法**。要在 Overleaf 编译完整论文，需要付费会员或其 7 天免费试用；否则更顺的路线是装本地 TeX 用 `nodepaper build`。

Linux/Overleaf 与本机的字体回退不同，版面可能有轻微差异，因此重要交付仍应在目标环境检查首页、宽表和参考文献。

## 9. 构建前后的最小检查清单

```powershell
# 1. 先检查项目结构、Front Matter、资源和 Fragment 声明
nodepaper validate .

# 2. 生成 PDF
nodepaper build .

# 3. 需要把 LaTeX 工程交付给他人时，导出并在本机可用时验证
nodepaper export . --to ..\paper-latex.zip --verify
```

打开生成的 PDF 后，至少人工检查：

- 首页的标题、摘要与关键词是否都在同一页；
- 宽表是否有异常留白、过密换行或溢出；
- 公式密集页、参考文献和附录的分页；
- 每个已声明的 Fragment、图片与引用是否出现在正确位置。

## 10. 常见问题定位

| 现象 | 先做什么 |
| --- | --- |
| 多文件项目报 Front Matter 错误 | 确认只有 `sources` 第一项含 `--- ... ---`，并含 `title`、`problem`、`keywords`。 |
| `source and sources are mutually exclusive` | 二选一，删除另一种来源字段。 |
| 关键词跑到第二页 | 小步降低 `abstractLinespread`（例如 `0.95` 到 `0.90`），重建并检查首页；不要先改正文 `linespread`。 |
| Markdown 表格忽然变宽或列宽不稳定 | 明确写 `width=auto`、百分比或 `full`；需要比例时写 `ratios`，不要依赖分隔行横杠长度。 |
| 复杂表无法用 Markdown 排好 | 先判断是否真需要合并单元格、局部字号或分组线；需要时使用已声明的 LaTeX Fragment。 |
| Fragment 未被插入或校验失败 | 检查路径是否相对 Project 根目录、是否在 `latexFragments` 白名单、`\input{...}` 是否完全一致，以及 Fragment 是否含嵌套输入或不允许的导言区命令。 |
| 改了 `.nodepaper/build/paper.tex` 但下次又丢失 | 这是生成文件。把修改转回 Markdown、`nodepaper.yaml`、图片或已声明 Fragment。 |
| `export` 拒绝已有目标 | 先确认不会覆盖不相关交付物，再加 `--force`。 |
| 上传 Overleaf 后无法编译 | 先看是不是免费版 10 秒超时（见 §8 的 Overleaf 小节）；不是超时则检查编译器为 XeLaTeX、主文档为 `paper.tex`，再查看导出目录中的 `README.txt`。 |

## 当前边界与尚未提供的设置

以下不是遗漏的配置键，而是当前 v0.1 没有提供的能力：Markdown 表题与表注字体的独立 YAML 设置，以及对 `pgfplots` 的兼容性保证。（摘要区的两处间距原先也在此列，已于 2026-08-21 实现为 `titleAbstractSkip` 与 `abstractKeywordsSkip`，见上表。论文标题到摘要标题之间的间距仍是模板固定值，没有配置入口。）不要自行添加看似合理的字段；配置解析会拒绝未知字段。

这些能力如进入后续版本，会在发布说明和本指南中明确标为可用。在此之前，应通过现有 Markdown 表格属性、受控 Fragment 或最小复现问题来处理，而不是依赖未承诺的行为。
