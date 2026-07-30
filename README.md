# NodePaper

NodePaper is a Windows-oriented Go CLI that builds a Markdown Project identified by `nodepaper.yaml` into PDF.

The current v0.1 development focus is a candidate CUMCM 2026 electronic-paper Profile. NodePaper owns project discovery, configuration, validation, diagnostics, build locking, logging, and artifact publication. Pandoc, pandoc-crossref, Citeproc, PowerShell, latexmk, and XeLaTeX perform conversion and typesetting.

> Status: under development. The candidate CUMCM Profile has not completed the MiKTeX, Windows 10, race-detector, release-ZIP, or manual PDF review gates and is not endorsed by the competition organizers.

## Project workflow

NodePaper operates on Project directories, not isolated Markdown files:

```text
my-paper/
├── nodepaper.yaml
├── paper.md
├── references.bib
├── images/
├── dist/
│   └── paper.pdf
└── .nodepaper/
    ├── build/
    ├── logs/
    └── build.lock
```

Typical commands:

```powershell
nodepaper init D:\papers\cumcm-a
nodepaper doctor D:\papers\cumcm-a
nodepaper validate D:\papers\cumcm-a
nodepaper build D:\papers\cumcm-a
```

Inside a Project or one of its subdirectories, the path may be omitted:

```powershell
cd D:\papers\cumcm-a
nodepaper validate
nodepaper build
```

Clean intermediates or all generated output:

```powershell
nodepaper clean D:\papers\cumcm-a
nodepaper clean D:\papers\cumcm-a --all
```

Machine-readable output:

```powershell
nodepaper build D:\papers\cumcm-a --format json
```

JSON stdout is one object with `schemaVersion`; ordinary logs are not mixed into JSON stdout.

## Configuration

Single source:

```yaml
version: 1
profile: cumcm
source: paper.md
output:
  file: dist/paper.pdf
```

Ordered multiple sources:

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

Sources are processed in declared order. NodePaper does not scan source directories automatically and does not keep a global current-project state.

## Markdown baseline

The first Source contains YAML front matter:

```markdown
---
title: Paper Title
problem: A
keywords:
  - Keyword One
  - Keyword Two
---

# 摘要

Write the abstract here.
```

Citations use Pandoc Citeproc:

```markdown
The method is suitable for demand forecasting [@wang2024].
```

Cross-references use pandoc-crossref IDs:

```markdown
![Result](images/result.png){#fig:result width=80%}

See @fig:result.

$$
x = 1
$$ {#eq:model}

See @eq:model.
```

The formal CUMCM bibliography route is:

```text
references.bib + Pandoc Citeproc + pinned CSL
```

## Current CUMCM behavior

- The first page contains title, abstract, and keywords.
- The electronic-paper Profile does not generate a contents, commitment, or numbering page.
- Ordered single- and multi-source projects, Chinese cross-references, and Citeproc are supported in the current development workspace.
- PDF publication checks non-empty content, header, EOF, and the 20 MB limit.
- Only one write build may run for a Project at a time.
- Profile resources are read-only during a build.

## Confirmed but not yet implemented v0.1 work

The following are approved plans, not current capabilities:

- Pandoc-native syntax highlighting, long-line wrapping, and multi-page code blocks;
- a retained appendix heading with A/B by default and continuous/none alternatives;
- explicitly declared, Project-root-confined LaTeX Fragments for complex tables and equations;
- Profile version and complete resource SHA-256 logging;
- generated-LaTeX contracts, strict missing-character/overflow checks, and non-visual PDF geometry checks;
- a public `layout-stress` E2E and local reference-paper E2E.

NodePaper will not add custom Markdown-in-Markdown Include syntax. Use ordered `sources` for chapters and controlled LaTeX Fragments for advanced typesetting.

## Source-tree tests

```powershell
.\scripts\test-unit.ps1
.\scripts\test-integration.ps1
.\scripts\test-e2e.ps1
.\scripts\test-all.ps1
```

`test-release.ps1` intentionally blocks release while ZIP, MiKTeX, Windows 10, and manual review gates remain incomplete. Source-tree E2E uses test-only environment variables to locate scripts and the Profile; those variables are not part of the public CLI contract.

## Existing SCAU PowerShell compatibility entry

The repository still contains the original SCAU thesis, assignment, and experiment-report PowerShell templates and commands. They have not yet been migrated into formal Project Profiles and must not be confused with support by the new Go CLI.

The previous detailed documentation is preserved in:

- [`README.SCAU-COMPAT.md`](README.SCAU-COMPAT.md)
- [`README.SCAU-COMPAT.zh-CN.md`](README.SCAU-COMPAT.zh-CN.md)

After CUMCM is stable, the architecture is intended to permit migration of these reviewed built-in Profiles:

```text
scau-thesis
scau-assignment
scau-experiment
```

They are not formally supported by the new CLI until metadata, validation, licensing, fixtures, E2E, and manual PDF review are complete.

## Current limitations

- No release ZIP is available yet.
- No GUI, HTTP server, or VS Code extension is provided.
- VS Code's built-in Markdown preview is not the final Pandoc/LaTeX PDF.
- Windows 10/11 x64 is the target, but all release-platform gates are not complete.
- NodePaper does not install a complete TeX distribution, execute paper code, or upload papers.

A thin VS Code adapter may be developed only after CLI semantics, JSON Schema, file/line diagnostics, and Project behavior are stable. It must reuse the CLI or Application Service rather than reimplementing build logic.
