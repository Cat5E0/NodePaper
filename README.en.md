[简体中文](README.md) | English

# NodePaper

NodePaper is a Windows command-line tool that builds a Markdown Project identified by `nodepaper.yaml` into PDF.

The current v0.1 targets the CUMCM 2026 electronic-paper workflow, including Chinese typesetting, equations, figures, tables, cross-references, citations, code blocks, appendices, and ordered multi-file projects.

> NodePaper is still under beta development and has not been formally released. It is not endorsed by the competition organizers.

## Installation

Requires Windows 10/11 x64 and either TeX Live or MiKTeX (providing `xelatex`). NodePaper drives XeLaTeX directly; **latexmk and Perl are not required**.

> The official repository, <https://github.com/Cat5E0/NodePaper>, has no public GitHub Release assets yet. Test candidates are handed out directly by the maintainer. Obtain `NodePaper-Setup-<version>-windows-x64.exe` (or the portable ZIP) together with the matching `release-manifest-<version>.json`; do not use third-party downloads or GitHub's Source code ZIP.

Run the Setup, then open a new terminal:

```powershell
nodepaper doctor
```

It installs into your user profile without administrator rights. Uninstall from Windows Settings → Apps, or the Start-menu entry.

<details>
<summary>Portable build, download verification and the unsigned build</summary>

**Portable ZIP**: extract `nodepaper-<version>-windows-x64.zip` and run `.\Install-NodePaper.ps1` to register the command, or just run `nodepaper.exe` from the extracted folder without installing or touching PATH. `Uninstall-NodePaper.ps1` sits next to it.

> Note: double-clicking `nodepaper.exe` only opens a window with guidance text and **installs nothing**; press Enter to close it. To install, double-click the Setup, or run `Install-NodePaper.ps1` from a PowerShell window (when it runs in its own window it stays open after showing the result and asks you to press Enter).

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
Please install NodePaper on Windows. Rules:
1. Use only the local candidate files I explicitly provide: the Setup (or portable ZIP) and the matching release-manifest-<version>.json. Do not search online, use mirrors, file-sharing sites, search results or a Source code ZIP, and do not assemble a package from source.
2. If either the package or Manifest is missing, ask me for it instead of finding a substitute.
3. Before installing, report whether the file name and size match the relevant Manifest channel, whether Get-FileHash matches its SHA-256, and the actual Get-AuthenticodeSignature status. Unsigned is expected for the current candidate and does not need to be fixed, but stop on any file-name, size or SHA-256 mismatch.
4. This is an unreleased, unsigned candidate and SmartScreen may block it. Explain that and wait for my explicit approval; never bypass or disable security software.
5. Use the visible installer UI, with no silent installation or administrator rights. Before deleting files or installing a large dependency such as TeX Live or MiKTeX, explain the size and time cost and ask me.
6. Afterwards run nodepaper --version and nodepaper doctor in a new terminal, and show me the real output and version comparison.
7. Do not request tokens or passwords, execute paper code, or delete or modify paper Projects. Do not present AI output as test evidence.
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

Advanced tables or equations can use LaTeX Fragments. Usage has two required steps: declare the safety allowlist in `nodepaper.yaml`, then place `\input{...}` at the intended location in a Markdown Source. **A declaration alone does not insert anything into the PDF.**

```yaml
latexFragments:
  - tables/complex-result.tex
```

```markdown
## Complex result table

\input{tables/complex-result.tex}
```

`nodepaper validate` rejects an undeclared `\input`. If a Fragment is declared but not inserted in any Markdown Source, it returns the `NP2511` Warning with the required `\input{...}` reminder.

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
- requires an external TeX Live or MiKTeX installation providing XeLaTeX; latexmk and Perl are not required;
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
