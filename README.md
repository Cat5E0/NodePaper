# NodePaper SCAU Markdown to LaTeX Converter

This project converts one Markdown thesis file into a complete SCAU-style LaTeX file and, when a LaTeX toolchain is available, builds a PDF.

The project targets Windows x64 for distribution. Pandoc and pandoc-crossref can be bundled in the project's own `tools/windows-x64` directory, so users don't need to configure the system PATH.

## Quick Start

Prepare bundled tools once:

```powershell
.\Bootstrap-Tools.ps1
```

Generate LaTeX only:

```powershell
.\Convert-MarkdownToScauLatex.ps1 -Input .\examples\paper.md
```

Generate LaTeX and compile PDF:

```powershell
.\Build-Paper.ps1 -Input .\examples\paper.md
```

Use the assignment template:

```powershell
.\Build-Paper.ps1 -Input .\examples\assignment.md -TemplateName assignment
```

Use the experiment report template:

```powershell
.\Build-Paper.ps1 -Input '实验一 GPIO输出控制实验.md' -TemplateName experiment
```

Optional cover and last-page PDFs:

```powershell
.\Build-Paper.ps1 -Input 'exp1.md' -TemplateName experiment `
    -CoverPdf '.\cover.pdf' -LastPagePdf '.\last-page.pdf'

# Or use page 1 as cover and last page as last page from the same file:
.\Build-Paper.ps1 -Input 'exp1.md' -TemplateName experiment `
    -CoverLastPdf '.\cover-and-last.pdf'
```

Use the thesis template:

```powershell
.\Build-Paper.ps1 -Input .\examples\paper.md -TemplateName thesis
```

If your machine already has Pandoc and pandoc-crossref installed, you can allow system tools during development:

```powershell
.\Build-Paper.ps1 -Input .\examples\paper.md -AllowSystemPandoc
```

## Project Structure

```text
NodePaper/
  Build-Paper.ps1
  Convert-MarkdownToScauLatex.ps1
  Bootstrap-Tools.ps1
  README.md
  README.zh-CN.md
  examples/
    paper.md
    assignment.md
    assignment-no-references.md
  filters/
    scau-blocks.lua
  templates/
    scau-thesis.template.tex
    scau-assignment.template.tex
    scau-experiment.template.tex
  tools/
    windows-x64/
      pandoc/
        pandoc.exe
      pandoc-crossref/
        pandoc-crossref.exe
  build/
```

## Markdown File Format

Input files use YAML front matter for document metadata.

For the assignment template:

```yaml
---
assignment_type: 课程作业
title_zh: OpenStack Cloud Platform Mechanism Analysis
course: Cloud Computing Technology
author_zh: Zhang San
college: College of Mathematics and Informatics
major: Information and Computing Science
student_id: "202225810103"
teacher: Li Si
date: 2026-06-09
---
```

For the thesis template, use more complete thesis metadata:

```yaml
---
title_zh: Stability Analysis of Time-Delay Neural Networks Based on Lyapunov Functional Method
title_en: Stability Analysis of Time-Delay Neural Networks Based on Lyapunov Functional Method
author_zh: Chen Ruheng
author_en: Chen Ruheng
college: College of Mathematics and Informatics
major: Information and Computing Science
student_id: "202225810103"
supervisor: Shi Chenyang
supervisor_title: Lecturer
date: 2026-06-09
abstract_zh: |
  Write the Chinese abstract here.
keywords_zh:
  - Neural networks
  - Exponential stability
  - Lyapunov-Krasovskii functional
abstract_en: |
  Write the English abstract here.
keywords_en:
  - Neural networks
  - Exponential stability
  - Lyapunov-Krasovskii functional
acknowledgements: |
  Write acknowledgements here.
references_tex: |
  \bibitem{fridman2014}
  Fridman, E. (2014). \textit{Introduction to Time-Delay Systems}. Birkhäuser.
---
```

Body content starts from heading level 1:

```markdown
# Introduction {#sec:intro}

See @sec:intro.

![System Architecture](image/system.png){#fig:system width=80%}

See @fig:system.

$$
x(t+1)=Ax(t)+Bu(t)
$$ {#eq:model}

From @eq:model we obtain…
```

## Cross-References

The project uses `pandoc-crossref` for numbering and cross-references. Recommended label prefixes:

- `sec:` for sections, e.g. `# Introduction {#sec:intro}`
- `fig:` for figures, e.g. `![Caption](a.png){#fig:a}`
- `tbl:` for tables, e.g. `: Title {#tbl:result}`
- `eq:` for equations, e.g. `$$ x=1 $$ {#eq:x}`

In the text, write:

```markdown
See @sec:intro, @fig:system, @tbl:result and @eq:model.
```

The generated LaTeX continues to use the template's `hyperref` and `cleveref`.

## References and Citations

For the assignment template, references are optional:

- With references: a "References" page is automatically generated.
- Without references: no references page is generated.

The recommended stable approach is to write `references_tex` in YAML and use LaTeX `\cite{key}` in the body:

```yaml
---
references_tex: |
  \bibitem{openstackdocs}
  OpenStack Documentation. OpenStack Docs. \url{https://docs.openstack.org/}.
---
```

Body citation:

```markdown
The OpenStack documentation provides deployment and configuration instructions \cite{openstackdocs}.
```

You can also write a references section at the end of the Markdown:

```markdown
# References

1. OpenStack Documentation. OpenStack Docs. <https://docs.openstack.org/>.
2. Mell, P. and Grance, T. The NIST Definition of Cloud Computing.
```

This kind of plain list can display a references page, but won't automatically support `\cite{key}` numbered citations. When you need numbered citations, prefer `references_tex` + `\cite{key}`.

Currently the project does not enable Pandoc citeproc's `[@key]` / BibTeX / CSL automatic formatting by default; this can be added in the future.

## Where to Place Images

We recommend putting images in the `image/` directory next to your Markdown file:

```text
paper.md
image/
  system.png
  result-a.jpg
```

Then reference them in Markdown:

```markdown
![System architecture](image/system.png){#fig:system width=80%}

See @fig:system.
```

The conversion script has added the following directories to Pandoc's resource search path:

```text
project root
project root/image
input Markdown directory
input Markdown directory/image
```

## Adjusting Image Size

Use curly brace `{}` attributes in Markdown to control image dimensions. The project has enabled Pandoc's `link_attributes` extension, which passes attributes through to LaTeX's `\includegraphics`.

### Common Parameters Quick Reference

| Syntax                 | Effect                  | Notes                              |
| ---------------------- | ----------------------- | ---------------------------------- |
| `{width=80%}`          | 80% of page width       | Most common, proportional scaling  |
| `{width=60%}`          | 60% of page width       | For smaller images                 |
| `{width=\textwidth}`   | Full page width         | For wide images                    |
| `{width=6cm}`          | Fixed width 6cm         | For precise sizing                 |
| `{height=5cm}`         | Fixed height 5cm        | Width adjusts proportionally       |
| `{scale=0.5}`          | 50% of original size    | Relative to original               |

### Full Examples

```markdown
![System architecture](image/system.png){#fig:system width=80%}

![Small icon](image/icon.png){#fig:icon width=3cm}

![Wide image](image/wide.png){#fig:wide width=\textwidth}
```

### Setting Multiple Attributes

Separate multiple attributes with spaces:

```markdown
![Demo](image/demo.jpg){#fig:demo width=70% height=6cm}
```

> **Note**: Specifying both `width` and `height` changes the aspect ratio and may cause distortion. In most cases, specifying only `width` is sufficient — the height scales proportionally.

### All Available Parameters

Any parameter accepted by LaTeX's `\includegraphics` can be used in the curly braces. Common ones:

| Parameter          | Example Values              |
| ------------------ | --------------------------- |
| `width`            | `80%`, `10cm`, `\textwidth` |
| `height`           | `6cm`, `0.5\textheight`     |
| `scale`            | `0.5`, `1.2`                |
| `angle`            | `90` (rotation)             |
| `keepaspectratio`  | `true`                      |

### Why This Works

Data flow:

```
Markdown {width=80%}  →  Pandoc parses as Image attribute  →  LaTeX \includegraphics[width=0.8\textwidth]{...}
                                                                       ↓
                                                    Template \adjustbox auto-fallback
                                                    prevents overflow
```

Pandoc converts `width=80%` to `width=0.8\textwidth` in the `.tex` file. Even if no attributes are specified, the template's `\pandocbounded` uses `\adjustbox` to automatically limit maximum image width and height, preventing page overflow.

## Code Blocks and Syntax Highlighting

Plain code blocks:

```markdown
```text
plain text
```
```

Specify a language for syntax highlighting:

```markdown
```python
def stable(delay: float) -> bool:
    return delay < 1.0
```
```

Other languages:

```markdown
```matlab
A = [1 0; 0 1];
eig(A)
```

```powershell
.\Build-Paper.ps1 -Input .\examples\paper.md
```
```

This project defaults to Pandoc's `tango` code highlighting style and builds in the `Shaded` / `Highlighting` environment macros in the LaTeX template. When generating PDF, if the local TeX environment includes `fvextra`, long code lines will automatically wrap; otherwise it falls back to basic `fancyvrb` display.

## Theorems, Lemmas and Proofs

`filters/scau-blocks.lua` converts fenced Div blocks into LaTeX environments:

```markdown
::: theorem
Theorem content here.
:::

::: lemma
Lemma content here.
:::

::: proof
Proof content here.
:::
```

Supported classes: `theorem`, `thm`, `lemma`, `lem`, `remark`, `proof`.

## Experiment Report Template

The experiment report template (`-TemplateName experiment`) is designed for course lab reports:

- **No cover page, no table of contents**: Body content starts directly, no cover or TOC pages.
- **All headings in Songti (宋体)**: Level-1 headings (`#`) are centered, size 4 Songti; level-2/3 headings (`##`, `###`) are left-aligned, size small-4 Songti.
- **Body format**: Songti small-4, 1.5× line spacing, 2-em first-line indent.
- **Footer**: "人工智能教研室制" centered, page number on the right.
- **External PDF cover/last-page**: Supports `-CoverPdf`, `-LastPagePdf`, and `-CoverLastPdf` (page 1 as cover, last page as last page from the same file). Uses LaTeX's pdfpages package — no extra tools required.

Suitable for quickly formatting lab reports for courses.

## Output Files

Default output is in the `build/` directory:

```text
build/Paper.tex
build/Paper.pdf
```

If `latexmk` and `xelatex` are not detected, the project still generates `build/Paper.tex` but skips PDF compilation.

## Log Files

Each time `Build-Paper.ps1` runs, the script saves logs in the `logs/` directory:

```text
logs/build-YYYYMMDD-HHMMSS.log
logs/latex-YYYYMMDD-HHMMSS.log
```

- `build-*.log`: records the build entry point, conversion commands, LaTeX compilation commands, external command output, exit codes, generated `.tex` / `.pdf` paths, etc.
- `latex-*.log`: a copy of `build/Paper.log`, preserving the full LaTeX compilation log. Check this file first when debugging package, image, formula, or reference errors.

If the build fails, the script outputs the last portion of the LaTeX log to the terminal and copies the full log to `logs/`.

## Distribution Notes

The release package should include:

```text
tools/windows-x64/pandoc/pandoc.exe
tools/windows-x64/pandoc-crossref/pandoc-crossref.exe
tools/versions.json
```

This way users only need to run PowerShell scripts — no separate Pandoc or pandoc-crossref installation required.
