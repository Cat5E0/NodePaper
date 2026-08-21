简体中文 | [English](README.en.md)

# NodePaper

<p align="left">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset="docs/assets/logo/logo-dark.png">
    <img src="docs/assets/logo/logo-transparent.png" alt="NodePaper logo" width="120">
  </picture>
</p>

[![ci](https://github.com/Cat5E0/NodePaper/actions/workflows/ci.yml/badge.svg)](https://github.com/Cat5E0/NodePaper/actions/workflows/ci.yml)
[![miktex-e2e](https://github.com/Cat5E0/NodePaper/actions/workflows/miktex-e2e.yml/badge.svg)](https://github.com/Cat5E0/NodePaper/actions/workflows/miktex-e2e.yml)
[![export-linux](https://github.com/Cat5E0/NodePaper/actions/workflows/export-linux.yml/badge.svg)](https://github.com/Cat5E0/NodePaper/actions/workflows/export-linux.yml)

NodePaper 是一个面向 Windows 的命令行工具，用于将包含 `nodepaper.yaml` 的 Markdown Project 构建为 PDF。

当前 v0.1 面向 CUMCM 2026 电子版论文场景，支持中文排版、公式、图表、交叉引用、参考文献、代码块、附录和多文件项目。

> 当前仍处于测试版开发阶段，尚未正式发布。NodePaper 不代表比赛官方认证。

## 构建展示

下面均为 NodePaper 的实际构建产物。点击预览图或下方链接可打开完整论文 PDF。

<table>
  <tr>
    <td width="50%" align="center">
      <a href="https://github.com/Cat5E0/NodePaper/blob/main/docs/assets/showcase/a163-nodepaper-multi-file.pdf">
        <img src="https://raw.githubusercontent.com/Cat5E0/NodePaper/main/docs/assets/showcase/a163-pages-26-27.png" alt="A163 第 26–27 页：示意图、公式与模型结果表" width="100%">
      </a>
      <br>
      <strong>A163 · 多文件工程</strong>
      <br>
      <sub>第 26–27 页 · 示意图、公式与模型结果表</sub>
      <br>
      <a href="https://github.com/Cat5E0/NodePaper/blob/main/docs/assets/showcase/a163-nodepaper-multi-file.pdf">打开完整构建 PDF</a>
    </td>
    <td width="50%" align="center">
      <a href="https://github.com/Cat5E0/NodePaper/blob/main/docs/assets/showcase/c063-nodepaper-single-file-latex-tables.pdf">
        <img src="https://raw.githubusercontent.com/Cat5E0/NodePaper/main/docs/assets/showcase/c063-pages-06-07.png" alt="C063 第 6–7 页：LaTeX 表格与公式" width="100%">
      </a>
      <br>
      <strong>C063 · 单文件工程</strong>
      <br>
      <sub>第 6–7 页 · LaTeX 表格与公式</sub>
      <br>
      <a href="https://github.com/Cat5E0/NodePaper/blob/main/docs/assets/showcase/c063-nodepaper-single-file-latex-tables.pdf">打开完整构建 PDF</a>
    </td>
  </tr>
</table>

项目拆分、完整配置、摘要首页、表格、LaTeX Fragment、导出到 Overleaf 与排错，见[用户指南](https://github.com/Cat5E0/NodePaper/blob/main/docs/guides/README.md)。

## 安装

需要 Windows 10/11 x64。NodePaper 的 Setup 约 52 MB，几秒装完。**它不自带 TeX，而下面的「快速开始」全程无需本地 TeX 环境**——导出只调用发布包内置的 pandoc。想在本机一条命令直接出 PDF，再回头装 TeX，见「在本机直接出 PDF：安装 TeX」。

> 官方仓库 <https://github.com/Cat5E0/NodePaper> 当前尚无公开 GitHub Release 资产。测试候选由维护者直接提供；请同时取得 `NodePaper-Setup-<版本>-windows-x64.exe`（或便携 ZIP）和同批 `release-manifest-<版本>.json`，不要从第三方来源或 GitHub 的 Source code ZIP 获取安装包。

双击 Setup 安装，然后打开新终端：

```powershell
nodepaper doctor
```

安装到当前用户目录，不需要管理员权限；卸载走 Windows「设置 → 应用」或开始菜单的「卸载 NodePaper」。

<details>
<summary>便携版、下载校验与未签名说明</summary>

**便携 ZIP**：把 `nodepaper-<版本>-windows-x64.zip` 解压到一个**你打算长期保留的位置**，然后运行 `.\Install-NodePaper.ps1`。

**NodePaper 就在解压目录里运行，脚本不复制任何文件**——它只是把这个目录加入你的用户 PATH，让 `nodepaper` 在任意目录可用。因此：

- **不要删除或移动这个目录**，删了 `nodepaper` 命令就失效；
- 想换位置，把整个目录移过去再运行一次 `.\Install-NodePaper.ps1`，旧位置会自动从 PATH 移除；
- 升级：把新版本解压到别处，运行那里的 `.\Install-NodePaper.ps1`，旧目录会自动从 PATH 摘掉（目录里的文件不动），随后你可以删掉它；
- 卸载用同目录的 `.\Uninstall-NodePaper.ps1`。它注销的就是自己所在的那个目录，只摘除 PATH 条目，**目录本身保留**，需要的话你自己删；
- 磁盘上放多份解压目录没问题，但只有一份能拥有 `nodepaper` 命令：PATH 是从左往右找的，所以运行某一份的 `.\Install-NodePaper.ps1` 就是把命令交给它，同时把另一份从 PATH 摘掉。

也可以完全不跑脚本，直接用解压目录里的 `nodepaper.exe`（输全路径），或按下面的说明手动配置 PATH。

> 注意：双击 `nodepaper.exe` 只会打开一个显示引导文字的窗口，**不会安装任何东西**，按回车关闭。
> 版本行为：`Install-NodePaper.ps1` 会与 PATH 上最先找到的那份便携目录比较版本——那份就是当前 `nodepaper` 实际运行的副本。升级和相同版本重复注册直接继续；若本包比那份更旧（降级），在独立控制台运行会要求确认，非交互（管道/CI）运行默认拒绝。

<details>
<summary>手动配置 PATH（不想跑脚本时）</summary>

**方法一：图形界面**

1. Win+R 输入 `sysdm.cpl` → 「高级」→「环境变量」
2. 在**上半部分「用户变量」**里选中 `Path` → 编辑 → 新建
3. 粘贴解压目录的完整路径，确定
4. **打开一个新终端**，运行 `nodepaper`

**方法二：命令行**

```powershell
# 换成你的解压目录
$dir = 'D:\Tools\NodePaper'

$key = 'HKCU:\Environment'
$old = [string](Get-Item $key).GetValue('Path', '', 'DoNotExpandEnvironmentNames')
$has = @($old -split ';' | ForEach-Object { $_.Trim().Trim('"').TrimEnd('\') }) -contains $dir.TrimEnd('\')
if (-not $has) {
    $new = if ([string]::IsNullOrWhiteSpace($old)) { $dir } else { $old.TrimEnd(';') + ';' + $dir }
    # 必须用 ExpandString：用户 Path 的默认类型是 REG_EXPAND_SZ，
    # 写成 String 会让其他软件的 %VARIABLE% 条目失效。
    Set-ItemProperty -Path $key -Name Path -Value $new -Type ExpandString
    '已添加，请打开新终端'
} else { '已存在，无需重复添加' }
```

两处不能省：用 `DoNotExpandEnvironmentNames` 读，否则会把别的软件的 `%变量%` 固化成当时的路径；读 `HKCU:\Environment` 而不是 `$env:PATH`，后者是用户+系统的合并值，写回会把系统 PATH 复制进用户 PATH。

**卸载**：从同一个位置删掉那条路径即可，解压目录自己决定要不要删。

</details>

**校验下载**：每个版本附带 `release-manifest-<版本>.json`，内含两个渠道的文件大小与 SHA-256。

```powershell
Get-FileHash .\NodePaper-Setup-<版本>-windows-x64.exe -Algorithm SHA256
```

**未签名**：当前构建未做 Authenticode 代码签名，Windows SmartScreen 可能提示未知发布者。请核对来源和哈希后自行决定是否继续；不要为此关闭系统安全功能。

**卸载保留什么**：只删除安装目录、自建的 PATH 项、快捷方式和卸载注册项；论文 Project、`dist` 里的 PDF 和 TeX 环境都不动。

</details>

### 交给 AI 助手安装

把下面这段复制给 Codex、Claude Code、Cursor 等能操作本机的助手：

```text
请帮我安装 NodePaper（Windows）。规则：
1. 只使用我明确给出的本地候选文件：Setup（或便携 ZIP）和同批 release-manifest-<版本>.json。不要联网寻找安装包，不要使用第三方镜像、网盘、搜索结果、Source code ZIP，也不要从源码自行拼装。
2. 缺少安装包或 Manifest 时先向我索取，不要寻找替代文件。
3. 安装前核对并报告：文件名和大小是否等于 Manifest 对应渠道记录、Get-FileHash 的 SHA-256 是否一致、Get-AuthenticodeSignature 的实际状态。未签名是当前候选的已知状态，不要求补签名；但文件名、大小或 SHA-256 任一不一致都必须停止。
4. 这是未正式发布、未签名的候选版，SmartScreen 可能拦截。说明风险并等我明确同意；不要绕过或关闭任何安全软件。
5. 使用可见安装界面，不静默安装、不请求管理员权限。删除文件或安装 TeX Live/MiKTeX 等大型依赖前，先说明体积和耗时并征得同意。
6. 安装后在新终端运行 nodepaper --version 和 nodepaper doctor，把真实输出和版本核对结果发给我。
7. 不索取 Token 或密码，不执行论文代码，不删除或修改论文 Project；不要把 AI 输出当作测试通过证据。
```

## 快速开始（无需本地 TeX 环境）

### 1. 检查环境

```powershell
nodepaper doctor
```

没装 TeX 时 XeLaTeX 会报一条 Warning，这是预期的：它只影响 `nodepaper build`，第 5 步的导出不受影响。

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

复杂表格、TikZ 绘图等 Markdown 表达不了的内容，以 LaTeX Fragment 形式存在项目内自建的目录（如 `tables/`、`figures/`）中，在 `nodepaper.yaml` 声明后用 `\input{...}` 插入正文，见下文「Markdown 示例」。

### 4. 验证

```powershell
cd D:\papers\cumcm-a
nodepaper validate
```

先把 Validate 报出的 Diagnostic 修掉，再往下走。

### 5. 出成品：导出 LaTeX 工程

```powershell
nodepaper export . --to ..\paper-latex
```

导出的不是 PDF，而是一份可独立编译的 LaTeX 工程：`paper.tex`、`references.bib`、论文实际用到的图片、`\input{}` 的 Fragment，以及一份写明编译步骤和所需宏包的 `README.txt`。把 `--to` 指向目录会导出成文件夹；指向 `.zip` 会直接生成可在 Overleaf 上传的 ZIP。

导出是给**已经会用 LaTeX、想接手精修或搬进别的环境**的人用的。它是一条单方向的交接：导出后在别处改的内容不会回流到 Markdown 项目。

> **关于 Overleaf**：导出的工程可以上传 Overleaf（编译器选 XeLaTeX），但一份完整的 CUMCM 论文几十页、要跑多遍 XeLaTeX，在 Overleaf **免费版 10 秒编译限时**（[官方 Plan Limits](https://docs.overleaf.com/getting-started/free-and-premium-plans/plan-limits)）内跑不完。想在 Overleaf 编译完整论文，要么用付费会员（240 秒），要么用会员的 7 天免费试用；否则更顺的路线是装本地 TeX，用下面一条 `nodepaper build` 直接出 PDF。

## 在本机直接出 PDF：安装 TeX

安装 TeX 是整个流程中耗时最长的一步，只有 `nodepaper build` 需要它。

| 方案 | 下载 | 装完占盘 | 耗时 | 适用 |
|---|---|---|---|---|
| **MiKTeX**（推荐先试） | 约 140 MB | 约 1 GB | 约 10～20 分钟 | 磁盘空间有限。首次构建时会自动下载所缺宏包，需联网 |
| **TeX Live 完整版** | 约 6.3 GB | 约 8～9 GB | 20～60 分钟（使用国内镜像） | 磁盘空间充裕，希望一次装全、之后完全离线 |

以上为量级参考，实际取决于网络与磁盘性能。

下载地址（请使用官方来源）：

- MiKTeX：<https://miktex.org/download>
- TeX Live：<https://tug.org/texlive/windows.html>

**国内用户建议更换镜像。** TeX Live 默认从国外服务器下载，可能需要数小时；改用国内镜像后通常几十分钟内完成。配置方法见镜像站说明：

- 清华 TUNA：<https://mirrors.tuna.tsinghua.edu.cn/help/CTAN/>
- 中科大 USTC：<https://mirrors.ustc.edu.cn/CTAN/systems/texlive/tlnet/>

### 注意事项

- 安装路径不要包含中文或空格。TeX 生态对非 ASCII 路径的支持不稳定。
- 安装完成后需要**打开一个新终端**。PATH 变更不作用于已打开的窗口，这是「已安装但提示找不到 `xelatex`」最常见的原因。
- 不建议使用 CTeX 套装。它捆绑的是多年未更新的旧版 MiKTeX，可能与当前宏包版本不兼容。NodePaper 只需要一个能运行 `xelatex` 的现代 TeX 发行版。

在新终端执行 `xelatex --version`，有版本输出即表示环境就绪。NodePaper 直接驱动 XeLaTeX，**不需要 latexmk 或 Perl**。

### 装好之后

```powershell
cd D:\papers\cumcm-a
nodepaper validate
nodepaper build
```

构建成功后，PDF 位于：

```text
dist/paper.pdf
```

## 导出：更多选项

除了上面「快速开始」第 5 步的默认用法，导出还适用于：要投稿、要交给导师或队友在 LaTeX 层面继续改、某处排版需要手工微调而 Markdown 表达不出来。导出的工程不引用原项目，编译它也不需要 NodePaper。

### 选择参考文献后端

`--bib` 决定文献怎么处理，默认 `bibtex`：

| 模式 | 引用命令 | 样式 | 编译顺序 | 取舍 |
|---|---|---|---|---|
| `bibtex`（默认） | `\cite{}` | `gbt7714` | `xelatex` → `bibtex` → `xelatex` ×2 | 兼容性最好，几乎所有环境都有 |
| `biblatex` | `\autocite{}` | `biblatex-gb7714-2015` | `xelatex` → `biber` → `xelatex` ×2 | 样式更全更新，但需要 `biber`（MiKTeX 常常默认没装） |
| `inline` | 无 | 文献已排版进 `paper.tex` | `xelatex` ×2 | 零依赖、没有 `.bib`，代价是文献列表变成死文本，不能再自动重排 |

重复跑 `xelatex` 不是多余：第一遍写出引用和交叉引用数据，后面几遍读回来。

### 其他选项

- `--verify` 默认关闭。开启后会把导出**复制一份到临时目录**完整编译一遍来确认可用，不在交付目录里留下 `.aux`、`.log` 或 `.pdf`。它需要本机有 TeX；找不到 `xelatex`／`bibtex`／`biber` 时只报一条 Warning，**导出本身照常完成**。另外，本机编译成功不代表收件人机器上也一定能编译；
- `--to` 指定的目录非空或目标 ZIP 已存在时，需要加 `--force` 才会导出；
- `nodepaper doctor` 会附带报告导出用到的宏包是否可用，缺了不影响日常构建。

## 启动方式

- 开始菜单“NodePaper”：打开一个持续可用的命令行窗口，先显示当前位置的下一步提示，随后可以继续输入命令；
- 终端：在任意目录直接运行 `nodepaper`，只读输出提示并退出；
- 在文件资源管理器中双击 `nodepaper.exe`：窗口不会一闪而过，而是保留说明并引导你使用开始菜单或终端。双击不会安装任何东西、不会创建 Project、也不会修改系统。

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
nodepaper export [project-directory] --to <directory-or-zip> [--bib bibtex|biblatex|inline] [--verify] [--force]
nodepaper --help
nodepaper --version
```

- `clean` 删除中间构建文件；
- `clean --all` 额外删除 `dist/`；
- `nodepaper` 无参数运行时会根据当前位置提示下一步；
- `export` 产出的是一份可编辑的 LaTeX 工程而不是 PDF；`--bib`、`--verify` 与 `--force` 的取舍见「导出：更多选项」。

机器可读输出：

```powershell
nodepaper build D:\papers\cumcm-a --format json
```

JSON、管道、重定向和 CI 环境永远不会等待输入。

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

`abstractLinespread` 可单独调整摘要与关键词的行距；摘要刚好把关键词挤到第二页时，优先小幅降低它并重新检查首页，而不要先改动正文行距。其他容易踩坑的写法见[用户指南](https://github.com/Cat5E0/NodePaper/blob/main/docs/guides/README.md)。

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

普通 Markdown 表格可以在表题后直接控制总宽度，无需改写为 LaTeX：

```markdown
| 符号 | 说明 |
| :---: | :---: |
| $x$ | 决策变量 |

: 参数说明 {#tbl:variables width=80% ratios="1:3"}
```

`width` 可取 `auto`、`full` 或 `80%` 这样的百分比；可选的 `ratios` 按列给出相对比例，数值数量必须与列数一致。分隔行的横杠长度只是 Pandoc 的启发式提示，短表中不保证生效。只有 Markdown 无法表达的合并单元格、局部字号等结构才需要下面的 LaTeX Fragment。

复杂表格或公式可以使用 LaTeX Fragment。使用时必须完成两步：先在 `nodepaper.yaml` 中声明安全白名单，再在某个 Markdown Source 的目标位置写 `\input{...}`；**仅在配置中声明不会自动插入 PDF**。

```yaml
latexFragments:
  - tables/complex-result.tex
```

```markdown
## 复杂结果表

\input{tables/complex-result.tex}
```

`nodepaper validate` 会拒绝未声明的 `\input`；如果 Fragment 已声明但没有在任何 Markdown Source 中插入，则返回 `NP2511` Warning，并提示需要补写的 `\input{...}`。

## TikZ / PGF 绘图

NodePaper v0.1 支持两类绘图 Fragment：手写或工具生成的 TikZ（`\begin{tikzpicture}`），以及绘图工具导出的纯 PGF 命令文件（`\begin{pgfpicture}`，如 Matplotlib 的 PGF backend 生成的 `.pgf`）。PGF 是 TikZ 底层的绘图语言；`pgfplots`（`\begin{axis}`、`\addplot` 那类坐标轴图表宏包）是更上层的独立宏包，v0.1 尚未支持。最快用法是把导出的 `figures/model.tex` 或 `figures/model.pgf` 加入白名单：

```yaml
latexFragments:
  - figures/model.pgf
```

再在 Markdown 的目标位置插入：

```markdown
\input{figures/model.pgf}
```

绘图脚本在 NodePaper 外部运行；NodePaper 只验证并编译声明过的 Fragment。`pgfplots` 尚不属于 v0.1 完整支持面。Matplotlib 导出、字体与路径约束、支持矩阵和排错见 [TikZ / PGF Fragment 指南](https://github.com/Cat5E0/NodePaper/blob/main/docs/guides/tikz-pgf.md)。

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
- `build` 需要外部 TeX Live 或 MiKTeX 提供 XeLaTeX；不需要 latexmk 或 Perl。`export` 例外，它只跑发布包内置的 pandoc，无 TeX 也能导出；
- 当前 CUMCM Profile 仍是候选版本；
- Setup 尚未进行 Authenticode 代码签名；
- 不提供 GUI、HTTP 服务、DOCX 或 Typst 输出；
- 安装器只负责安装，不提供论文编辑器、拖放构建或桌面图形界面；
- 不自动执行论文代码；
- 不上传论文内容；
- 不支持直接转换孤立 Markdown 文件。

## 许可证

NodePaper 原创代码采用 MIT License。第三方组件及其许可证见 [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md)。
