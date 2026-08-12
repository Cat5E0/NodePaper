简体中文 | [English](README.en.md)

# NodePaper

NodePaper 是一个面向 Windows 的命令行工具，用于将包含 `nodepaper.yaml` 的 Markdown Project 构建为 PDF。

当前 v0.1 面向 CUMCM 2026 电子版论文场景，支持中文排版、公式、图表、交叉引用、参考文献、代码块、附录和多文件项目。

> 当前仍处于测试版开发阶段，尚未正式发布。NodePaper 不代表比赛官方认证。

## 安装

NodePaper 支持 Windows 10/11 x64。使用前请先安装 TeX Live 或 MiKTeX，并确保 `xelatex` 和 `latexmk` 可用。

Windows x64 提供两个由同一个发布 Payload 生成的渠道：

| 渠道 | 文件 | 适用场景 |
| --- | --- | --- |
| Setup 安装包 | `NodePaper-Setup-<版本>-windows-x64.exe` | 普通用户，双击安装 |
| 便携 ZIP | `nodepaper-<版本>-windows-x64.zip` | 高级用户、离线环境、可验证分发 |

两个渠道内的 `nodepaper.exe`、Profile、模板、工具和许可证完全相同；两个渠道包各自记录自己的 SHA-256。

### 方式一：Setup 安装包（推荐）

1. 在官方仓库 <https://github.com/Cat5E0/NodePaper> 的 Releases 页面下载 `NodePaper-Setup-<版本>-windows-x64.exe`。只使用官方 Releases 资产，不要使用第三方镜像、网盘或 `Source code (zip)`。
2. 核对下载文件的 SHA-256 与同一个 Release 提供的 `release-manifest-<版本>.json` 或 Release 说明一致：

   ```powershell
   Get-FileHash .\NodePaper-Setup-<版本>-windows-x64.exe -Algorithm SHA256
   ```

3. 当前 Setup 尚未进行 Authenticode 代码签名。运行时 Windows SmartScreen 或安全软件可能提示未知发布者。请先核对来源和 SHA-256，再自行决定是否继续；NodePaper 不建议、也不要求关闭 Defender、SmartScreen 或任何安全功能。
4. 双击 Setup 安装。安装按当前用户进行，不需要管理员权限，默认目录为 `%LOCALAPPDATA%\Programs\NodePaper`，也可以选择其他当前用户可写目录。安装器会注册当前用户 PATH、创建开始菜单“NodePaper”和“卸载 NodePaper”入口；桌面快捷方式可选，默认不勾选。
5. 安装完成页默认勾选“启动 NodePaper”，会打开一个持续可用的命令行窗口。也可以打开新终端，在任意目录运行：

   ```powershell
   nodepaper --version
   nodepaper doctor
   ```

Setup 不联网、不含遥测、不含自动更新，安装前会校验内嵌 Payload 的版本和文件哈希，失败时回滚并保留原有安装。

### 方式二：便携 ZIP

解压 `nodepaper-<版本>-windows-x64.zip` 后，在解压目录运行：

```powershell
.\Install-NodePaper.ps1
```

重新打开终端，然后在任意目录输入：

```powershell
nodepaper
```

ZIP 也可以完全便携使用：直接执行解压目录中的 `nodepaper.exe`，不安装、不修改 PATH。

### 卸载

- Setup 安装：Windows“设置 → 应用 → 已安装的应用 → NodePaper → 卸载”，或开始菜单“卸载 NodePaper”。卸载器保存在安装目录中，删除下载的 Setup 后仍可正常卸载。
- ZIP 安装：

  ```powershell
  & "$env:LOCALAPPDATA\Programs\NodePaper\Uninstall-NodePaper.ps1"
  ```

卸载只删除 NodePaper 的安装目录、它自己创建的 PATH 项、快捷方式和卸载注册项，不会删除论文 Project、`dist` 中的 PDF、TeX Live/MiKTeX 或其他软件。构建仍在进行时，卸载器会提示先关闭 NodePaper 再重试。

### 可选：交给 AI 助手完成安装

如果你在使用 Codex、Claude Code、Cursor 或其他具备本机操作能力的 Coding Agent，可以把下面的 Prompt 原样复制给它。这是一个可选入口，不是唯一安装方式，也不是 NodePaper 的运行时能力；上面的手工步骤始终可用。

```text
你是一个可以在我的 Windows 电脑上执行命令的编码代理。请帮我安装 NodePaper，并严格遵守下列规则。

1. 只使用官方仓库 https://github.com/Cat5E0/NodePaper 的 GitHub Releases 资产，选择 Windows x64 Setup，文件名形如 NodePaper-Setup-<版本>-windows-x64.exe。不要使用任何第三方镜像、网盘、搜索结果或 Source code (zip/tar.gz)，也不要自己从源码编译或拼装安装包。
2. 如果该仓库没有可用的 Release，或者 Release 里没有上述 Setup 资产，就停止并告诉我，不要寻找替代来源。
3. 如果只有 prerelease（预发布或候选版本），先告诉我版本号、它是预发布、可能不稳定，等我明确同意后再继续。
4. 下载 Setup 后，从同一个 Release 获取 release-manifest-<版本>.json 或 Release 说明里的 SHA-256，然后核对版本、下载 URL、文件大小和哈希：
   Get-FileHash .\NodePaper-Setup-<版本>-windows-x64.exe -Algorithm SHA256
   Get-AuthenticodeSignature .\NodePaper-Setup-<版本>-windows-x64.exe
   把这些结果原样告诉我，不要替我下结论。
5. 只要版本、URL、大小、哈希有任何不一致，就停止安装并报告。当前 Setup 可能没有 Authenticode 签名：如果显示未签名，或者 Windows SmartScreen、安全软件出现提示，必须把是否继续的决定交给我；不要绕过、不要关闭或修改 Defender、SmartScreen、防火墙或任何安全设置。
6. 使用可见的安装界面运行 Setup，不要使用静默安装参数，不要请求管理员权限（NodePaper 按当前用户安装）。如果某一步需要管理员权限、删除文件，或需要安装 TeX Live、MiKTeX 这类大型依赖，先说明磁盘占用和时间代价，并征得我同意。
7. 安装完成后打开一个新终端，执行 nodepaper --version 和 nodepaper doctor，把真实输出如实解释给我。如果 doctor 报告缺少 TeX 环境，先说明再问我是否安装。
8. 不要索取或使用任何 Token、密码或凭据；不要执行论文附录中的代码；不要删除或修改我的论文 Project、dist 中的 PDF 或其他文件。你的输出只是安装过程记录，不能当作测试通过或质量证据。
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
nodepaper --help
nodepaper --version
```

- `clean` 删除中间构建文件；
- `clean --all` 额外删除 `dist/`；
- `nodepaper` 无参数运行时会根据当前位置提示下一步。

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

复杂表格或公式可以使用项目配置中显式声明的 LaTeX Fragment。

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
- 需要外部 TeX Live 或 MiKTeX；
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
