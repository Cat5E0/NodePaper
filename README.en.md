[简体中文](README.md) | English

# NodePaper

NodePaper is a Windows command-line tool that builds a Markdown Project identified by `nodepaper.yaml` into PDF.

The current v0.1 targets the CUMCM 2026 electronic-paper workflow, including Chinese typesetting, equations, figures, tables, cross-references, citations, code blocks, appendices, and ordered multi-file projects.

> NodePaper is still under beta development and has not been formally released. It is not endorsed by the competition organizers.

## Installation

NodePaper targets Windows 10/11 x64. Install TeX Live or MiKTeX first and make sure `xelatex` and `latexmk` are available.

Two channels are produced from one and the same release payload:

| Channel | File | Audience |
| --- | --- | --- |
| Setup installer | `NodePaper-Setup-<version>-windows-x64.exe` | ordinary users, double-click install |
| Portable ZIP | `nodepaper-<version>-windows-x64.zip` | advanced users, offline and verifiable use |

Both channels contain the identical `nodepaper.exe`, Profile, template, bundled tools and licenses; each channel package records its own SHA-256.

> How to obtain it: NodePaper has not been released yet, so the official repository <https://github.com/Cat5E0/NodePaper> has no Release assets. During this test phase the candidate files are handed out directly by the maintainer together with `release-manifest-<version>.json`, which records the version, source commit, file size and SHA-256. Do not obtain a “NodePaper installer” from third-party mirrors, file-sharing links or search results, and do not assemble one from `Source code (zip)`. Download from the official Releases page once it exists.

### Option 1: Setup installer (recommended)

1. Obtain `NodePaper-Setup-<version>-windows-x64.exe` and the matching `release-manifest-<version>.json` from the maintainer.
2. Verify that the file name, size and SHA-256 match the `setup-exe` channel entry in `release-manifest-<version>.json`:

   ```powershell
   Get-FileHash .\NodePaper-Setup-<version>-windows-x64.exe -Algorithm SHA256
   ```

3. The current Setup is not Authenticode signed. Windows SmartScreen or security software may therefore warn about an unknown publisher. Check the source and the SHA-256 first and decide for yourself whether to continue; NodePaper never asks you to disable Defender, SmartScreen or any other protection.
4. Double-click the Setup. It installs for the current user without administrator rights, defaults to `%LOCALAPPDATA%\Programs\NodePaper` and accepts any other user-writable directory. It registers the current user's Path and creates the Start-menu entries “NodePaper” and “卸载 NodePaper” (Uninstall NodePaper); the desktop shortcut is optional and unchecked by default.
5. The final wizard page offers “Launch NodePaper”, which opens a persistent command-line window. You can also open a new terminal and run, from any directory:

   ```powershell
   nodepaper --version
   nodepaper doctor
   ```

Setup performs no network access, no telemetry and no automatic update. It verifies the embedded payload version and file hashes before finishing, and rolls back to the previous installation on failure.

### Option 2: Portable ZIP

Extract `nodepaper-<version>-windows-x64.zip`, then run from the extracted directory:

```powershell
.\Install-NodePaper.ps1
```

Open a new terminal and run from any directory:

```powershell
nodepaper
```

The ZIP can also be used fully portable: run `nodepaper.exe` from the extracted directory without installing anything or changing the Path.

### Uninstalling

- Setup installation: Windows “Settings → Apps → Installed apps → NodePaper → Uninstall”, or the Start-menu “卸载 NodePaper” entry. The uninstaller is stored inside the installation directory, so uninstalling still works after the downloaded Setup has been deleted.
- ZIP installation:

  ```powershell
  & "$env:LOCALAPPDATA\Programs\NodePaper\Uninstall-NodePaper.ps1"
  ```

Uninstalling removes only NodePaper's installation directory, the Path entry it created, its own shortcuts and its uninstall registration. Paper Projects, PDFs in `dist`, TeX Live/MiKTeX and other software are never removed. If a build is still running, the uninstaller asks you to close NodePaper and try again.

### Optional: let an AI assistant install it

If you use Codex, Claude Code, Cursor or another coding agent that can operate your machine, you can copy the prompt below verbatim. It is an optional entry point, not the only way to install NodePaper and not a NodePaper runtime capability; the manual steps above always work.

The current prompt covers the test-phase installation from candidate files that are already on your machine. A prompt for downloading and verifying from the official GitHub Releases will be added once NodePaper is released.

```text
You are a coding agent that can run commands on my Windows machine. Please install NodePaper from the installer file I already have locally, and follow these rules strictly.

1. Use only the local file paths I give you in this message: the Setup NodePaper-Setup-<version>-windows-x64.exe (or the portable nodepaper-<version>-windows-x64.zip), verified against release-manifest-<version>.json from the same batch of files. Do not download any “NodePaper installer” from the internet, do not use third-party mirrors, file-sharing links or search results, and do not compile or assemble an installer from source yourself.
2. If I did not give you a file path or release-manifest-<version>.json, stop and ask me for them instead of looking for a substitute file.
3. Verify before installing and report the results verbatim without drawing the conclusion for me:
   whether the file name matches the file recorded for the setup-exe channel in release-manifest-<version>.json;
   whether Get-FileHash <file path> -Algorithm SHA256 equals the sha256 in that manifest;
   whether the file size equals the size in that manifest;
   what Get-AuthenticodeSignature <file path> reports.
4. Stop and report if the file name, size or SHA-256 disagree in any way.
5. This is an unreleased candidate and the current Setup is not Authenticode signed. Before running it, tell me the version, the source commit from the manifest, the unsigned state and the fact that Windows SmartScreen or security software may block it, and wait for my explicit approval. Never bypass, disable or modify Defender, SmartScreen, the firewall or any other security setting.
6. Run the Setup with its visible user interface. Do not use silent-install switches and do not request administrator rights (NodePaper installs for the current user). If any step needs administrator rights, deletes files, or installs a large dependency such as TeX Live or MiKTeX, explain the disk and time cost first and ask for my approval.
7. After installation, open a new terminal, run nodepaper --version and nodepaper doctor, explain the real output truthfully, and confirm that --version matches the version in the manifest. If doctor reports a missing TeX environment, explain it and ask me before installing anything.
8. Never ask for or use tokens, passwords or credentials; never execute code from a paper appendix; never delete or modify my paper Projects, the PDFs in dist or any other file. Your output is only a record of the installation and is not evidence that tests passed or that quality was verified.
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
