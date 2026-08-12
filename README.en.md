[简体中文](README.md) | English

# NodePaper

NodePaper is a Windows command-line tool that builds a Markdown Project identified by `nodepaper.yaml` into PDF.

The current v0.1 targets the CUMCM 2026 electronic-paper workflow, including Chinese typesetting, equations, figures, tables, cross-references, citations, code blocks, appendices, and ordered multi-file projects.

> NodePaper is still under beta development and has not been formally released. It is not endorsed by the competition organizers.

## Installation

Requires Windows 10/11 x64 and either TeX Live or MiKTeX (providing `xelatex`).

Download `NodePaper-Setup-<version>-windows-x64.exe` from [Releases](https://github.com/Cat5E0/NodePaper/releases), run it, then open a new terminal:

```powershell
nodepaper doctor
```

It installs into your user profile without administrator rights. Uninstall from Windows Settings → Apps, or the Start-menu entry.

> Test phase: candidate files are handed out directly by the maintainer; the Releases page has no assets yet.

<details>
<summary>Portable build, download verification and the unsigned build</summary>

**Portable ZIP**: extract `nodepaper-<version>-windows-x64.zip` and run `.\Install-NodePaper.ps1` to register the command, or just run `nodepaper.exe` from the extracted folder without installing or touching PATH. `Uninstall-NodePaper.ps1` sits next to it.

**Verify the download**: every version ships `release-manifest-<version>.json` with the size and SHA-256 of both channels.

```powershell
Get-FileHash .\NodePaper-Setup-<version>-windows-x64.exe -Algorithm SHA256
```

**Unsigned**: this build has no Authenticode signature, so Windows SmartScreen may warn about an unknown publisher. Check the source and hash, then decide for yourself; do not disable security features for it.

**What uninstalling keeps**: it removes only the installation directory, the PATH entry it added, its shortcuts and its uninstall registration. Paper Projects, PDFs in `dist` and your TeX installation are untouched.

</details>

### Let an AI assistant install it

Copy this to Codex, Claude Code, Cursor or another agent that can operate your machine:

```text
Please install NodePaper (Windows). Rules:
1. Use only the local installer path I give you, or the official Releases at https://github.com/Cat5E0/NodePaper - no third-party mirrors, file-sharing links or search results.
2. Before installing, check the SHA-256 with Get-FileHash against the release-manifest for that version; stop and tell me if it differs.
3. This is an unsigned test build and SmartScreen may block it. Leave the decision to continue to me and never disable security software.
4. Run the installer normally with its UI; no silent install and no administrator rights.
5. If a large dependency such as TeX Live or MiKTeX is needed, tell me the size and time cost and ask me first.
6. Afterwards run nodepaper --version and nodepaper doctor in a new terminal and show me the real output.
```

## Quick start

### 1. Check the environment

```powershell
nodepaper doctor
```

### 2. Create a Project

```powershell
nodepaper init D:\papers\cumcm-a
```

To also create a Project-level AI writing guide:

```powershell
nodepaper init D:\papers\cumcm-a --ai-guide
```

### 3. Edit the paper

A basic Project looks like this:

```text
cumcm-a/
├── nodepaper.yaml
├── paper.md
├── references.bib
├── images/
├── dist/
└── .nodepaper/
```

The main files are:

- `paper.md`: paper source;
- `references.bib`: bibliography;
- `images/`: image resources;
- `nodepaper.yaml`: Project configuration.

### 4. Validate and build

```powershell
cd D:\papers\cumcm-a
nodepaper validate
nodepaper build
```

After a successful build, the PDF is located at:

```text
dist/paper.pdf
```

## Ways to start NodePaper

- Start menu “NodePaper”: opens a persistent command-line window that first prints the next step for the current location and then keeps accepting commands;
- Terminal: run `nodepaper` from any directory for read-only guidance that exits immediately;
- Double-clicking `nodepaper.exe` in File Explorer: the window no longer flashes and disappears; it keeps the explanation visible and points to the Start menu or a terminal. Double-clicking installs nothing, creates no Project and changes no system setting.

## Selecting a Project

NodePaper operates on Project directories, not isolated Markdown files.

From a Project root:

```powershell
nodepaper validate
nodepaper build
```

When run from a subdirectory, NodePaper searches upward for `nodepaper.yaml`. A Project directory can also be passed explicitly:

```powershell
nodepaper build D:\papers\cumcm-a
```

NodePaper does not store a global “current Project.”

## Common commands

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

- `clean` removes intermediate build files;
- `clean --all` also removes `dist/`;
- running `nodepaper` without arguments suggests the next step for the current location.

Machine-readable output:

```powershell
nodepaper build D:\papers\cumcm-a --format json
```

JSON output, pipelines, redirected output and CI never wait for input.

## Project configuration

Minimal single-source configuration:

```yaml
version: 1
profile: cumcm
source: paper.md
output:
  file: dist/paper.pdf
```

Ordered multi-source configuration:

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

Sources are processed in the declared order. Directories are not scanned automatically.

Common optional settings:

```yaml
appendix:
  numbering: alpha
highlight:
  style: tango
linespread: 1.25
abstractLinespread: 0.95
mathFont: cm
```

## Markdown examples

The first Source contains YAML front matter:

```markdown
---
title: Paper title
problem: A
keywords:
  - Keyword one
  - Keyword two
---

# 摘要

Write the abstract here.
```

Citation:

```markdown
The method is suitable for demand forecasting [@wang2024].
```

Figure and cross-reference:

```markdown
![Result](images/result.png){#fig:result width=80%}

See @fig:result.
```

Equation and cross-reference:

```markdown
$$
x = 1
$$ {#eq:model}

See @eq:model.
```

Advanced tables or equations can use LaTeX Fragments explicitly declared in the Project configuration.

## Main capabilities

- single-source and ordered multi-source Projects;
- Chinese abstracts, keywords, and section headings;
- equations, figures, tables, and footnotes;
- figure, table, equation, and section cross-references;
- BibTeX citations with numeric superscripts;
- syntax highlighting and long code blocks;
- configurable appendix numbering;
- PDF bookmark outlines;
- Project validation, environment diagnostics, build logs, and build locking;
- human-readable and JSON output.

## Current limitations

- officially targets Windows 10/11 x64 only;
- requires an external TeX Live or MiKTeX installation;
- the current CUMCM Profile is still a candidate;
- the Setup is not Authenticode signed yet;
- no GUI, HTTP service, DOCX, or Typst output;
- the installer only installs: it provides no paper editor, no drag-and-drop build and no desktop GUI;
- does not execute paper code automatically;
- does not upload paper content;
- does not convert isolated Markdown files directly.

The SCAU PowerShell templates retained in this repository have not been migrated into formal NodePaper Profiles. See:

- [`README.SCAU-COMPAT.md`](README.SCAU-COMPAT.md)
- [`README.SCAU-COMPAT.zh-CN.md`](README.SCAU-COMPAT.zh-CN.md)

## License

Original NodePaper code is licensed under the MIT License. See [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md) for third-party components and licenses.
