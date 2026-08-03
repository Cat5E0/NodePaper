# CUMCM Profile sources and scope

## Status

This is a NodePaper-authored candidate profile for the CUMCM 2026 electronic
paper. It is not an official template and is not certified or endorsed by the
competition organizers.

## Official formatting facts

- Format specification page (published 2026-03-03):
  <https://www.mcm.edu.cn/html_cn/node/4cd596519c9eb9fbd866398f6df0caa3.html>
- Competition rules page (published 2026-03-03):
  <https://www.mcm.edu.cn/html_cn/node/9d8e511fe7a1447b35f53a82c908e2e0.html>

The profile implements the electronic-paper requirements relevant to generated
PDF output: A4 paper, margins of at least 2.5 cm, page numbering from the
abstract page, no contents page, and no commitment or numbering pages. The
official documents are cited but are not redistributed with NodePaper.

The official specification does not prescribe a single font, type size, line
spacing, or colour scheme. NodePaper's choices for those properties are
conservative implementation defaults, not official requirements.

## Bibliography style

- File: `csl/china-national-standard-gb-t-7714-2015-numeric.csl`
- Upstream repository: <https://github.com/citation-style-language/styles>
- Upstream Git blob: `cc05707d5feaa7cb78df27a8da7f44de6aeb4934`
- Local SHA-256: `d9cd04f94bc21a99dc320d0a2230d5a1fb47e3634c17c5820cf975c40db3c0c7`
- Authors named by the CSL file: 牛耕田; contributor Zeping Lee
- License declared in the CSL file: Creative Commons Attribution-ShareAlike 3.0
  <https://creativecommons.org/licenses/by-sa/3.0/>

The CSL file is redistributed unchanged. Any later modification must preserve
attribution and comply with the ShareAlike terms.

## NodePaper-authored resources

`template.tex`, `crossref.yaml`, `warning-allowlist.json`,
`filters/extract-abstract.lua`, `filters/layout.lua`, and `profile.json` are
original NodePaper resources distributed under `LICENSES/PROFILE-MIT.txt`.

## Deliberate limitations

- Electronic-paper PDF only; paper commitment and numbering pages are not generated.
- The profile does not imply official approval.
- The 30-page body limit and 20 MB upload limit require post-build checks and,
  for the body/appendix boundary, maintainer review.
- MiKTeX compatibility remains a release gate.
- Pandoc built-in highlighting is used without minted, Python Pygments runtime,
  or `shell-escape`. The reviewed default is `tango`; projects may select the
  built-in `pygments` or `kate` color scheme.
- Hyperref emits a numbered PDF bookmark outline without a printed contents page;
  viewers are requested, but cannot be forced, to open it through level two.
- Controlled LaTeX Fragments are limited to explicitly declared Project-local
  table/equation files. Nested dependencies, file reads, TeX I/O, command
  obfuscation, and shell execution are rejected before Pandoc.
