[简体中文](README.md) | English

# NodePaper

NodePaper is a Windows command-line tool that builds a Markdown Project identified by `nodepaper.yaml` into PDF.

The current v0.1 targets the CUMCM 2026 electronic-paper workflow, including Chinese typesetting, equations, figures, tables, cross-references, citations, code blocks, appendices, and ordered multi-file projects.

> NodePaper is still under beta development and has not been formally released. It is not endorsed by the competition organizers.

## Before you start

Requires Windows 10/11 x64. The NodePaper Setup is about 52 MB and installs in seconds.

**It does not bundle TeX, and you may not need TeX.** Pick a path first:

| You want | TeX required | What to do |
|---|---|---|
| **A PDF on your own machine** | Yes | Install MiKTeX or TeX Live — see "For local PDFs: installing TeX" below |
| **Just a LaTeX project, compiled on Overleaf** | **No** | Skip ahead to "Install", then use `nodepaper export` — see "Compiling on Overleaf" |

The second path skips the rest of this section entirely: `nodepaper export` only uses the pandoc shipped inside the release package, so it works as soon as NodePaper is installed.

### Compiling on Overleaf (no TeX)

```powershell
nodepaper export . --to ..\paper-latex
```

Zip the whole `..\paper-latex` folder, upload it to Overleaf, and then:

- Set **Menu → Compiler** to **XeLaTeX**. Overleaf defaults to pdfLaTeX, which fails here with an error that does not point at the cause;
- Chinese fonts follow the machine doing the compiling: the SimSun families on your Windows box, Noto CJK on Overleaf. **The page differs slightly; no characters are dropped**;
- `README.txt` in the exported folder lists the exact commands and the packages they need.

Export is **one-way**: edits made on Overleaf do not flow back into the Markdown project. Treat it as a handover, not a sync.

### For local PDFs: installing TeX

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

You can install NodePaper before TeX: `nodepaper doctor` reports "PDF output" and "LaTeX export" as separate capabilities, so a missing TeX costs only the first instead of declaring the whole machine unusable.

## Installation

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
$dir = 'D:\Tools\nodepaper-0.1.0-rc.9-windows-x64'

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
nodepaper export [project-directory] --to <dir> [--bib bibtex|biblatex|inline] [--verify] [--force]
nodepaper --help
nodepaper --version
```

- `clean` removes intermediate build files;
- `clean --all` also removes `dist/`;
- running `nodepaper` without arguments suggests the next step for the current location;
- `export` produces an editable LaTeX project (`.tex` + `.bib` + images + Fragments + `README.txt`) in the `--to` directory — not a PDF;
- `--bib` selects how references are handled: `bibtex` (default, `\cite{}` + `gbt7714`, compile chain `xelatex → bibtex → xelatex ×2`, broadest compatibility), `biblatex` (`\autocite{}` + `biblatex-gb7714-2015`, compile chain `xelatex → biber → xelatex ×2`, requires biber), `inline` (references already rendered as plain text, no extra dependencies);
- `--verify` is off by default; when enabled, the exported project is compiled once end-to-end to confirm it works, but a successful local compile does not guarantee the same on a recipient's machine;
- if `--to` points at a non-empty directory, `--force` is required;
- export is one-way: edits made to the LaTeX project do not flow back into the Markdown project;
- `nodepaper doctor` also reports whether the packages used by export are available, for reference when needed — it does not affect ordinary builds.

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
- `build` requires an external TeX Live or MiKTeX installation providing XeLaTeX; latexmk and Perl are not required. `export` is the exception: it only runs the bundled pandoc and works without TeX;
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
