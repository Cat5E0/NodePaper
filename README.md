简体中文 | [English](README.en.md)

# NodePaper

NodePaper 是一个面向 Windows 的命令行工具，用于将包含 `nodepaper.yaml` 的 Markdown Project 构建为 PDF。

当前 v0.1 面向 CUMCM 2026 电子版论文场景，支持中文排版、公式、图表、交叉引用、参考文献、代码块、附录和多文件项目。

> 当前仍处于测试版开发阶段，尚未正式发布。NodePaper 不代表比赛官方认证。

## 环境准备

需要 Windows 10/11 x64。NodePaper 的 Setup 约 52 MB，几秒装完，但它**不自带 TeX**——PDF 排版由你机器上的 TeX 发行版完成，安装 TeX 是整个流程中耗时最长的一步。

TeX 是 `nodepaper build` 的前置，不是全部命令的前置：`nodepaper export`（导出可编辑的 LaTeX 工程）只用发布包里自带的 pandoc，没装 TeX 也能用，编译留给拿到工程的人。

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

## 安装

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
- 升级：把新版本解压到别处，运行那里的 `.\Install-NodePaper.ps1`，旧目录会自动从 PATH 摘掉，随后你可以删掉它；
- 卸载用同目录的 `.\Uninstall-NodePaper.ps1`。它只摘除 PATH 条目，**目录本身保留**，需要的话你自己删。

也可以完全不跑脚本，直接用解压目录里的 `nodepaper.exe`（输全路径），或按下面的说明手动配置 PATH。

> 注意：双击 `nodepaper.exe` 只会打开一个显示引导文字的窗口，**不会安装任何东西**，按回车关闭。
> 版本行为：`Install-NodePaper.ps1` 会与上次注册的目录比较版本。升级和相同版本重复注册直接继续；若本包比上次注册的更旧（降级），在独立控制台运行会要求确认，非交互（管道/CI）运行默认拒绝。

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
$dir = 'D:\Tools\nodepaper-0.1.0-rc.9-windows-x64'

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

## 快速开始

### 1. 检查环境

```powershell
nodepaper doctor
```

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

### 4. 验证和构建

```powershell
cd D:\papers\cumcm-a
nodepaper validate
nodepaper build
```

构建成功后，PDF 位于：

```text
dist/paper.pdf
```

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
nodepaper export [project-directory] --to <dir> [--bib bibtex|biblatex|inline] [--verify] [--force]
nodepaper --help
nodepaper --version
```

- `clean` 删除中间构建文件；
- `clean --all` 额外删除 `dist/`；
- `nodepaper` 无参数运行时会根据当前位置提示下一步；
- `export` 把项目导出成一份可编辑的 LaTeX 工程（`.tex` + `.bib` + 图片 + Fragment + `README.txt`）到 `--to` 指定的目录，而不是 PDF；
- `--bib` 决定导出的文献处理方式：`bibtex`（默认，`\cite{}` + `gbt7714`，编译链 `xelatex → bibtex → xelatex ×2`，兼容性最好）、`biblatex`（`\autocite{}` + `biblatex-gb7714-2015`，编译链 `xelatex → biber → xelatex ×2`，需要 biber）、`inline`（文献已渲染成文本，零依赖）；
- `--verify` 默认关闭；开启后会在导出完成后完整编译一遍确认可用，但本机编译成功不代表收件人机器上也一定能编译；
- `--to` 指定的目录非空时需要加 `--force` 才会导出；
- 导出是单向的：在 LaTeX 工程里做的修改不会回流到 Markdown 项目；
- `nodepaper doctor` 会附带检查导出用到的几个宏包是否可用，供需要时参考，不影响日常构建。

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

仓库中保留的 SCAU PowerShell 模板尚未迁移为正式 NodePaper Profile，详见：

- [`README.SCAU-COMPAT.zh-CN.md`](README.SCAU-COMPAT.zh-CN.md)
- [`README.SCAU-COMPAT.md`](README.SCAU-COMPAT.md)

## 许可证

NodePaper 原创代码采用 MIT License。第三方组件及其许可证见 [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md)。
