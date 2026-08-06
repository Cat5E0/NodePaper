# NodePaper

NodePaper is a Windows-oriented Go CLI that builds a Markdown Project identified by `nodepaper.yaml` into PDF.

The current v0.1 development focus is a candidate CUMCM 2026 electronic-paper Profile. NodePaper owns project discovery, configuration, validation, diagnostics, build locking, logging, and artifact publication. Pandoc, pandoc-crossref, Citeproc, PowerShell, latexmk, and XeLaTeX perform conversion and typesetting.

> Status: under development. The candidate CUMCM Profile has not completed the MiKTeX, Windows 10, race-detector, release-ZIP, or manual PDF review gates and is not endorsed by the competition organizers.

## Run from source

From the repository root, bootstrap the pinned Pandoc and pandoc-crossref tools and build the CLI into the repository root. This layout lets the executable find `Build-Paper.ps1` and `profiles/cumcm` beside it:

```powershell
.\Bootstrap-Tools.ps1
go build -o nodepaper.exe .\cmd\nodepaper

.\nodepaper.exe doctor D:\papers\cumcm-a
.\nodepaper.exe validate D:\papers\cumcm-a
.\nodepaper.exe build D:\papers\cumcm-a
```

Avoid invoking `go run` from an arbitrary Project directory because Go's temporary executable does not have the complete build resources beside it. `nodepaper.exe` is the normal product entrypoint; `scripts/test-all.ps1` is the fast developer regression suite, while `scripts/test-e2e.ps1` runs real Pandoc/LaTeX build regression.

Repository Fixtures are read-only test inputs. Copy one outside the repository before manual CLI testing:

```powershell
Copy-Item -Recurse `
  .\nodepaper-test-fixtures\tests\fixtures\complete-single-file `
  D:\NodePaperTests\complete-single-file

.\nodepaper.exe build D:\NodePaperTests\complete-single-file
```

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
latexFragments:
  - tables/complex-result.tex
  - equations/objective.tex
appendix:
  numbering: alpha
highlight:
  style: tango
linespread: 1.25
abstractLinespread: 0.95
mathFont: cm
output:
  file: dist/paper.pdf
```

`linespread` controls the whole-document line spacing. It is a float with default `1.25` and an allowed range of `[1.0, 1.3]`; larger values make the document airier and longer. `abstractLinespread` overrides the line spacing of the abstract region only (default `0.95`, allowed `[0.85, linespread]`), so a long abstract still fits closer to one page. `mathFont` selects the Latin/math font route: `cm` (default; Latin Modern + Computer Modern) or `newtx` (TeX Gyre Termes + newtxmath, Times style).

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

Do not repeat a manual `Keywords:` paragraph in the abstract; the Profile renders `keywords` from front matter exactly once.

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

### Tables

Tables are horizontally centered on the page by default: the Profile template applies `\centering` to every table (Pandoc emits `longtable` for each Markdown table). Column alignment inside a table is independent and is controlled by the usual Markdown markers:

```markdown
| :---- | :----: | ----: |
| left  | center | right |
```

- Whole-table centering: automatic; the template default `\centering` needs no Markdown marker.
- In-column alignment: `|:----|` left, `|:----:|` center, `|----:|` right (Pandoc maps these to `l`, `c`, `r` column specifiers).
- Complex tables (merged cells and similar) are written as controlled LaTeX Fragments declared in `latexFragments` and inserted with `\input{tables/...}`; do not try to express them in Markdown.

An image that belongs inside a list item must be indented four spaces under the item, otherwise Pandoc closes the list before the image and the figure floats away from the list:

```markdown
1. Step one

    ![Step one result](images/step1.png){#fig:step1 width=80%}

2. Step two
```

The formal CUMCM bibliography route is:

```text
references.bib + Pandoc Citeproc + pinned CSL
```

A controlled Fragment must be declared in `latexFragments` before Markdown can insert it:

```markdown
\input{tables/complex-result.tex}

See \cref{tab:complex-result}.
```

Fragments must be relative UTF-8 regular `.tex` files inside the Project Root. Full documents, package loading, nested `input`, TeX I/O, and command execution are rejected. Appendices use:

```markdown
# 附录
## Test data
## Program code
```

`appendix.numbering` accepts `alpha` (default), `continuous`, or `none`. `highlight.style` accepts the reviewed Pandoc built-in styles `tango` (default), `pygments`, or `kate`; no Python runtime or `shell-escape` is used.

## Current CUMCM behavior

- The first page contains title, abstract, and keywords.
- The electronic-paper Profile does not generate a contents, commitment, or numbering page.
- Ordered single- and multi-source projects, Chinese cross-references, Citeproc, and controlled LaTeX Fragments are supported.
- Pandoc-native highlighting defaults to Tango, allows Pygments/Kate color schemes, and uses a breakable light code frame with safe long-line wrapping.
- Numeric superscript citations link to their corresponding bibliography entries.
- The PDF contains a numbered bookmark outline for the abstract, sections, references, and appendices without adding a contents page; it requests the viewer to open the outline panel to level two.
- The retained appendix heading supports `alpha`, `continuous`, and `none` numbering.
- Build logs record Profile version, complete resource SHA-256, and Fragment SHA-256; build-time mutation fails.
- Unknown warnings, overflow, missing characters/fonts, and unresolved references prevent publication.
- PDF publication checks non-empty content, header, EOF, and the 20 MB limit; real E2E additionally checks A4, text bounds, embedded fonts, and content order.
- Only one write build may run for a Project at a time.

## Remaining v0.1 work

The following gates remain incomplete before a formal release:

- independent MiKTeX E2E (auto-install off);
- Windows 10 smoke test;
- race detector on a C-enabled CI runner;
- final manual PDF review;
- maintainer sign-off of the release checklist.

The release packaging entry (ZIP build, ZIP validation, GitHub Actions CI,
`LICENSE` and `THIRD_PARTY_NOTICES.md`) is implemented; see the next section.

MinerU full-paper import and semantic fidelity review are deferred to a post-v0.1 research task and do not block the runtime or release package.

NodePaper will not add custom Markdown-in-Markdown Include syntax. Use ordered `sources` for chapters and controlled LaTeX Fragments for advanced typesetting.

## Source-tree tests

```powershell
.\scripts\test-unit.ps1
.\scripts\test-integration.ps1
.\scripts\test-e2e.ps1
.\scripts\test-all.ps1
```

`test-release.ps1` validates a built release ZIP in a clean directory and
blocks until the manual/platform gates are recorded (see the next section).
Source-tree E2E uses test-only environment variables to locate scripts and the
Profile; those variables are not part of the public CLI contract.

## Release package

The release candidate is a versioned Windows x64 ZIP built from a fixed commit
with `scripts/build-release.ps1`:

```powershell
.\scripts\build-release.ps1 -Version 0.1.0-rc.1
```

The script compiles `nodepaper.exe` inside an isolated git worktree of the
requested commit, assembles `nodepaper-<version>-windows-x64/` from an
explicit whitelist (runtime scripts, the CUMCM Profile, the pinned bundled
Pandoc binaries, a runnable example, README, LICENSE and
THIRD_PARTY_NOTICES), scans the package for absolute development paths,
secrets and temp artifacts, and records the ZIP SHA-256 in
`release-manifest.json`.

Validate the exact ZIP in a clean environment without Go or the source tree:

```powershell
.\scripts\test-release.ps1 -ReleaseZip .\build\release\nodepaper-0.1.0-rc.1-windows-x64.zip -ManualGatesFile .\gates.json
```

`test-release.ps1` runs doctor / init / validate / build, checks logs, the PDF
and repeat builds, and refuses to pass until MiKTeX, Windows 10, race
detector, PDF manual review and maintainer sign-off are recorded in the gates
file. The GitHub Actions workflow `.github/workflows/release-build.yml`
produces the same ZIP as a workflow artifact on demand; it never
auto-publishes a release.

Package layout:

```text
nodepaper-<version>-windows-x64/
├── nodepaper.exe
├── Build-Paper.ps1
├── Convert-CumcmProjectToLatex.ps1
├── profiles/cumcm/
├── tools/windows-x64/pandoc/ + pandoc-crossref/
├── examples/cumcm-single-file/
├── README.md / README.zh-CN.md
├── LICENSE
├── THIRD_PARTY_NOTICES.md
└── licenses/
```

The bundled Pandoc binaries make the package independent of a global Pandoc
install; a TeX distribution (TeX Live or MiKTeX with xelatex and latexmk) is
still required on the tester's machine.

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

- No formally released ZIP exists yet; release-candidate ZIPs are built by `scripts/build-release.ps1` and remain gated by the release checklist.
- No GUI, HTTP server, or VS Code extension is provided.
- VS Code's built-in Markdown preview is not the final Pandoc/LaTeX PDF.
- Windows 10/11 x64 is the target, but all release-platform gates are not complete.
- NodePaper does not install a complete TeX distribution, execute paper code, or upload papers.

A thin VS Code adapter may be developed only after CLI semantics, JSON Schema, file/line diagnostics, and Project behavior are stable. It must reuse the CLI or Application Service rather than reimplementing build logic.
