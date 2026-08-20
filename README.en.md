[简体中文](README.md) | English

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

NodePaper is a Windows command-line tool that builds a Markdown Project identified by `nodepaper.yaml` into PDF.

The current v0.1 targets the CUMCM 2026 electronic-paper workflow, including Chinese typesetting, equations, figures, tables, cross-references, citations, code blocks, appendices, and ordered multi-file projects.

> NodePaper is still under beta development and has not been formally released. It is not endorsed by the competition organizers.

## Showcase

Both examples below are real NodePaper build outputs. Click a preview or its link to open the complete paper PDF.

<table>
  <tr>
    <td width="50%" align="center">
      <a href="https://github.com/Cat5E0/NodePaper/blob/main/docs/assets/showcase/a163-nodepaper-multi-file.pdf">
        <img src="https://raw.githubusercontent.com/Cat5E0/NodePaper/main/docs/assets/showcase/a163-pages-26-27.png" alt="A163 pages 26–27: diagram, equations, and model-result tables" width="100%">
      </a>
      <br>
      <strong>A163 · Multi-file project</strong>
      <br>
      <sub>Pages 26–27 · Diagram, equations, and model-result tables</sub>
      <br>
      <a href="https://github.com/Cat5E0/NodePaper/blob/main/docs/assets/showcase/a163-nodepaper-multi-file.pdf">Open the complete build PDF</a>
    </td>
    <td width="50%" align="center">
      <a href="https://github.com/Cat5E0/NodePaper/blob/main/docs/assets/showcase/c063-nodepaper-single-file-latex-tables.pdf">
        <img src="https://raw.githubusercontent.com/Cat5E0/NodePaper/main/docs/assets/showcase/c063-pages-06-07.png" alt="C063 pages 6–7: LaTeX tables and equations" width="100%">
      </a>
      <br>
      <strong>C063 · Single-file project</strong>
      <br>
      <sub>Pages 6–7 · LaTeX tables and equations</sub>
      <br>
      <a href="https://github.com/Cat5E0/NodePaper/blob/main/docs/assets/showcase/c063-nodepaper-single-file-latex-tables.pdf">Open the complete build PDF</a>
    </td>
  </tr>
</table>

For project structure, the full configuration reference, abstract-page fitting, tables, LaTeX Fragments, Overleaf export, and troubleshooting, see the [user-guide index](https://github.com/Cat5E0/NodePaper/blob/main/docs/guides/README.md) (currently in Chinese).

## Installation

Requires Windows 10/11 x64. The NodePaper Setup is about 52 MB and installs in seconds. **It does not bundle TeX, and the whole "Quick start" below works without a local TeX installation** — export only calls the pandoc shipped inside the release package. Install TeX later if you want a PDF from a single local command; see "Producing a PDF locally: installing TeX".

> The official repository, <https://github.com/Cat5E0/NodePaper>, has no public GitHub Release assets yet. Test candidates are handed out directly by the maintainer. Obtain `NodePaper-Setup-<version>-windows-x64.exe` (or the portable ZIP) together with the matching `release-manifest-<version>.json`; do not use third-party downloads or GitHub's Source code ZIP.

Run the Setup, then open a new terminal:

```powershell
nodepaper doctor
```

It installs into your user profile without administrator rights. Uninstall from Windows Settings → Apps, or the Start-menu entry.

<details>
<summary>Portable build, download verification and the unsigned build</summary>

**Portable ZIP**: extract `nodepaper-<version>-windows-x64.zip` **somewhere you intend to keep it**, then run `.\Install-NodePaper.ps1`.

**NodePaper runs from the extracted folder; the script copies nothing.** It only adds that folder to your user PATH so `nodepaper` works from any directory. So:

- **Do not delete or move the folder** — the command stops working if you do;
- to relocate it, move the whole folder and run `.\Install-NodePaper.ps1` again from its new location; the old entry is removed from PATH automatically;
- to upgrade, extract the new release elsewhere and run its `.\Install-NodePaper.ps1`; the old folder leaves PATH automatically (its files are not touched) and you can then delete it;
- to uninstall, run `.\Uninstall-NodePaper.ps1` from the same folder. It unregisters the folder it sits in, removes the PATH entry only and **keeps the folder**; delete it yourself if you want it gone;
- keeping several extracted folders is fine, but only one of them can answer to `nodepaper`: PATH is searched left to right, so running a folder's `.\Install-NodePaper.ps1` hands the command to that folder and takes the other one off PATH.

You can also skip the script entirely: run `nodepaper.exe` from the extracted folder by full path, or add it to PATH yourself as shown below.

> Note: double-clicking `nodepaper.exe` only opens a window with guidance text and **installs nothing**; press Enter to close it.
> Version behaviour: `Install-NodePaper.ps1` compares against the portable folder your PATH finds first, which is the copy `nodepaper` currently runs. Upgrade and re-registering the same version continue directly; if this package is older than that one (downgrade), it asks for confirmation in an owned console and is rejected in non-interactive (piped/CI) runs.

<details>
<summary>Adding it to PATH by hand</summary>

**With the GUI**

1. Win+R, enter `sysdm.cpl` → Advanced → Environment Variables
2. Under **User variables** (the upper half), select `Path` → Edit → New
3. Paste the full path of the extracted folder and confirm
4. **Open a new terminal** and run `nodepaper`

**From PowerShell**

```powershell
# your extracted folder
$dir = 'D:\Tools\NodePaper'

$key = 'HKCU:\Environment'
$old = [string](Get-Item $key).GetValue('Path', '', 'DoNotExpandEnvironmentNames')
$has = @($old -split ';' | ForEach-Object { $_.Trim().Trim('"').TrimEnd('\') }) -contains $dir.TrimEnd('\')
if (-not $has) {
    $new = if ([string]::IsNullOrWhiteSpace($old)) { $dir } else { $old.TrimEnd(';') + ';' + $dir }
    # ExpandString is required: the user Path is REG_EXPAND_SZ by default, and
    # writing String stops any %VARIABLE% entry of other software from expanding.
    Set-ItemProperty -Path $key -Name Path -Value $new -Type ExpandString
    'Added; open a new terminal'
} else { 'Already present' }
```

Two details are not optional. Read with `DoNotExpandEnvironmentNames`, or another program's `%VARIABLE%` entries get frozen to whatever they pointed at. Read `HKCU:\Environment` rather than `$env:PATH`, which is the user and system values merged — writing that back copies the system PATH into your user PATH.

**Uninstalling**: remove that entry from the same place; the folder is yours to keep or delete.

</details>

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

## Quick start (works without a local TeX installation)

### 1. Check the environment

```powershell
nodepaper doctor
```

Without TeX, XeLaTeX reports a Warning. That is expected: it costs you `nodepaper build` only, and the export in step 5 is unaffected.

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

Content Markdown cannot express — complex tables, TikZ figures, and the like — lives as LaTeX Fragments in directories you create inside the Project (e.g. `tables/`, `figures/`). Declare them in `nodepaper.yaml` and insert them with `\input{...}`; see the Markdown examples below.

### 4. Validate

```powershell
cd D:\papers\cumcm-a
nodepaper validate
```

Fix whatever Validate reports before going on.

### 5. Get the finished document: export, then compile on Overleaf

```powershell
nodepaper export . --to ..\paper-latex.zip
```

What you get is not a PDF but a self-contained LaTeX project: `paper.tex`, `references.bib`, only the images the paper references, any fragments it `\input{}`s, and a `README.txt` spelling out the compile steps and the packages they need.

When `--to` ends in `.zip` (case-insensitive), export creates a ZIP directly. Its files sit at the archive root with no wrapper directory, so the result can be uploaded through **New Project → Upload Project**. Point `--to` at a directory when you want to inspect or edit the export first. Then:

- Set **Menu → Compiler** to **XeLaTeX**. Overleaf defaults to pdfLaTeX, which fails here with an error that does not point at the real cause;
- Chinese fonts follow the machine doing the compiling: Noto CJK on Overleaf, the SimSun families on your own Windows box. **The page differs slightly; no characters are dropped**.

At this point you can see the real typeset result, without a local TeX installation.

**What to do next**: if you only wanted to see the result, or you intended to hand over a LaTeX project anyway, you are done. If you will revise repeatedly — especially during a competition — zipping and uploading every revision costs real time, so install TeX and switch to a single `nodepaper build`.

Export is **one-way**: edits made on Overleaf do not flow back into the Markdown project. Treat it as a handover, not a sync.

## Producing a PDF locally: installing TeX

Installing TeX is the longest step of the whole process, and only `nodepaper build` needs it.

| Option | Download | Installed | Time | Suits |
|---|---|---|---|---|
| **MiKTeX** (try this first) | ~140 MB | ~1 GB | 10–20 min | Limited disk space. Missing packages are downloaded on demand during the first build, so it needs a network connection |
| **TeX Live, full** | ~6.3 GB | ~8–9 GB | 20–60 min (with a nearby mirror) | Plenty of disk space; everything installed up front and fully offline afterwards |

These are order-of-magnitude figures; the real numbers depend on your network and disk.

Download from the official sources:

- MiKTeX: <https://miktex.org/download>
- TeX Live: <https://tug.org/texlive/windows.html>

**Use a nearby CTAN mirror.** TeX Live downloads from its default server can take hours; a local mirror usually brings that down to tens of minutes. Mirror lists and setup instructions are published by each mirror, for example <https://mirrors.tuna.tsinghua.edu.cn/help/CTAN/>.

### Notes

- Install to a path without spaces or non-ASCII characters. TeX tooling handles such paths inconsistently.
- **Open a new terminal after installing.** PATH changes do not apply to already-open windows, and this is the most common reason for "installed but `xelatex` not found".
- Avoid the CTeX suite. It bundles a years-old MiKTeX that may not match current packages. NodePaper only needs a current TeX distribution that can run `xelatex`.

Run `xelatex --version` in a new terminal; a version banner means you are ready. NodePaper drives XeLaTeX directly; **latexmk and Perl are not required**.

### Once it is installed

```powershell
cd D:\papers\cumcm-a
nodepaper validate
nodepaper build
```

After a successful build, the PDF is located at:

```text
dist/paper.pdf
```

## Export: further options

Beyond the default use in step 5 above, exporting also suits submitting the paper, handing it to an advisor or teammate who works in LaTeX, or tuning one spot of typography that Markdown cannot express. The exported project does not refer back to the original, and compiling it does not need NodePaper.

### Choosing a bibliography backend

`--bib` decides how references are handled. The default is `bibtex`:

| Mode | Citation command | Style | Compile order | Trade-off |
|---|---|---|---|---|
| `bibtex` (default) | `\cite{}` | `gbt7714` | `xelatex` → `bibtex` → `xelatex` ×2 | Widest compatibility; present almost everywhere |
| `biblatex` | `\autocite{}` | `biblatex-gb7714-2015` | `xelatex` → `biber` → `xelatex` ×2 | More styles, updated more recently, but needs `biber`, which MiKTeX often does not install by default |
| `inline` | none | already typeset into `paper.tex` | `xelatex` ×2 | No dependencies and no `.bib`, at the cost of a reference list that is dead text and cannot be re-sorted |

The repeated `xelatex` runs are not redundant: the first pass writes the citation and cross-reference data, and the later ones read it back.

### Other options

- `--verify` is off by default. When enabled it **copies the export to a temporary directory** and compiles it there to confirm it works, so no `.aux`, `.log` or `.pdf` is left in the delivered folder. It needs TeX on this machine; when `xelatex`, `bibtex` or `biber` is missing it only reports a Warning and **the export itself still completes**. Note also that compiling here does not guarantee it compiles on the recipient's machine;
- `--force` is required when the `--to` directory is not empty or the target ZIP already exists;
- `nodepaper doctor` additionally reports whether the packages used by export are available; missing ones do not affect ordinary builds.

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
nodepaper export [project-directory] --to <directory-or-zip> [--bib bibtex|biblatex|inline] [--verify] [--force]
nodepaper --help
nodepaper --version
```

- `clean` removes intermediate build files;
- `clean --all` also removes `dist/`;
- running `nodepaper` without arguments suggests the next step for the current location;
- `export` produces an editable LaTeX project rather than a PDF; see "Export: further options" for the `--bib`, `--verify` and `--force` trade-offs.

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

Set the total width of an ordinary Markdown table on its caption; no LaTeX rewrite is needed:

```markdown
| Symbol | Description |
| :---: | :---: |
| $x$ | Decision variable |

: Parameters {#tbl:variables width=80% ratios="1:3"}
```

`width` accepts `auto`, `full`, or a percentage such as `80%`. The optional `ratios` gives one relative weight per column. Separator-line dash counts are only a Pandoc heuristic and are not guaranteed for short tables. Use the LaTeX Fragment route below only for structures Markdown cannot express, such as merged cells or local font-size changes.

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

## TikZ / PGF figures

NodePaper v0.1 supports two kinds of figure Fragments: TikZ code (`\begin{tikzpicture}`, hand-written or tool-generated) and plain PGF command files (`\begin{pgfpicture}`, e.g. the `.pgf` output of Matplotlib's PGF backend). PGF is the low-level drawing language TikZ itself is built on; `pgfplots` (the separate axis/chart package using `\begin{axis}` and `\addplot`) sits on top of it and is not yet supported in v0.1. The shortest route is to allowlist the exported `figures/model.tex` or `figures/model.pgf`:

```yaml
latexFragments:
  - figures/model.pgf
```

Then insert it at the intended location in Markdown:

```markdown
\input{figures/model.pgf}
```

Run the plotting script outside NodePaper; NodePaper only validates and compiles declared Fragments. Full `pgfplots` support is not part of the v0.1 contract. See the [TikZ / PGF Fragment guide](https://github.com/Cat5E0/NodePaper/blob/main/docs/guides/tikz-pgf.md) for Matplotlib export, font and path constraints, the support matrix, and troubleshooting.

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
- `build` requires an external TeX Live or MiKTeX installation providing XeLaTeX; latexmk and Perl are not required. `export` is the exception: it only runs the bundled pandoc and works without TeX;
- the current CUMCM Profile is still a candidate;
- the Setup is not Authenticode signed yet;
- no GUI, HTTP service, DOCX, or Typst output;
- the installer only installs: it provides no paper editor, no drag-and-drop build and no desktop GUI;
- does not execute paper code automatically;
- does not upload paper content;
- does not convert isolated Markdown files directly.

## License

Original NodePaper code is licensed under the MIT License. See [`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md) for third-party components and licenses.
