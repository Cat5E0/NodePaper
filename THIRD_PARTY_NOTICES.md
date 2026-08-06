# NodePaper Third-Party Notices

This file documents the components bundled or redistributed with NodePaper and
their license obligations. NodePaper itself is distributed under the MIT
License (see `LICENSE`); this notice does not change any third-party license.

Full license texts are provided in the `licenses/` directory next to this file
and inside the release package.

## NodePaper original code and resources — MIT License

The following are original NodePaper works licensed under the MIT License
(`LICENSE`, `Copyright (c) 2026 NodePaper contributors`):

- Go source in `cmd/` and `internal/`;
- PowerShell transition scripts `Build-Paper.ps1` and
  `Convert-CumcmProjectToLatex.ps1`;
- the CUMCM Profile under `profiles/cumcm/` (`profile.json`, `template.tex`,
  `crossref.yaml`, `warning-allowlist.json`, `filters/*.lua`);
- `scripts/`, `README.md`, `README.zh-CN.md` and `examples/cumcm-single-file`
  (fictional test content generated for this project; no real persons,
  schools, teams or papers).

## Go standard library and toolchain — BSD-3-Clause

`nodepaper.exe` is compiled with the Go toolchain and contains Go standard
library code. The Go standard library is distributed under a BSD-3-Clause
license; full text in `licenses/BSD-3-Clause.txt`. The Go project's license
information is available at <https://go.dev/LICENSE>.

## gopkg.in/yaml.v3 — MIT AND Apache-2.0

The Go module `gopkg.in/yaml.v3` (v3.0.1) is used for `nodepaper.yaml`
parsing. It is covered by two licenses: the MIT License (files ported from
libyaml) and the Apache License 2.0. Full texts in `licenses/MIT.txt` and
`licenses/Apache-2.0.txt`; upstream:
<https://github.com/go-yaml/yaml>.

## Pandoc 3.9 — GPL-2.0-or-later (bundled binary)

The release package bundles `tools/windows-x64/pandoc/pandoc.exe` (version
3.9, pinned by the Profile) so that testers do not need to install Pandoc.
Pandoc is a separate executable invoked by NodePaper at build time; it is not
linked into `nodepaper.exe`.

- Copyright (C) 2006-2024 John MacFarlane
- License: GNU General Public License version 2 or later; full text in
  `licenses/GPL-2.0.txt`
- Upstream and source: <https://github.com/jgm/pandoc>,
  <https://hackage.haskell.org/package/pandoc-3.9>
- Upstream copyright and per-component notices: the `COPYRIGHT` file at
  <https://github.com/jgm/pandoc/blob/master/COPYRIGHT>

## pandoc-crossref 0.3.24 — GPL-2.0-or-later (bundled binary)

The release package bundles
`tools/windows-x64/pandoc-crossref/pandoc-crossref.exe` (version 0.3.24,
pinned by the Profile). It is a separate executable invoked by NodePaper at
build time; it is not linked into `nodepaper.exe`.

- License: GNU General Public License version 2 or later; full text in
  `licenses/GPL-2.0.txt`
- Upstream and source:
  <https://github.com/lierdakil/pandoc-crossref>

## CSL style — CC BY-SA 3.0

The CUMCM Profile redistributes the citation style
`china-national-standard-gb-t-7714-2015-numeric.csl` from the
citation-style-language/styles repository (upstream Git blob
`cc05707d5feaa7cb78df27a8da7f44de6aeb4934`). It is unchanged and is licensed
under Creative Commons Attribution-ShareAlike 3.0; full text in
`licenses/CC-BY-SA-3.0.txt`. Authors named by the CSL file: 牛耕田;
contributor Zeping Lee. Upstream:
<https://github.com/citation-style-language/styles>.

## Not bundled

NodePaper does **not** bundle a TeX distribution. Users must install TeX Live
or MiKTeX with `xelatex` and `latexmk` themselves; those distributions carry
their own licenses and obligations. The release package contains no user
papers, no real contest submissions, no secrets and no absolute development
machine paths.

## Redistribution conditions

- The bundled GPL executables (Pandoc, pandoc-crossref) are redistributed
  under the GPL-2.0-or-later terms; their complete source is available at the
  upstream URLs above, and this notice plus the license text is distributed
  with the package.
- The CSL style is redistributed under CC BY-SA 3.0 terms; modifications must
  preserve attribution and comply with ShareAlike.
- NodePaper makes no claim of official certification by the competition
  organizers, and the candidate CUMCM Profile is not an official template.
