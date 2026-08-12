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
- `scripts/`, `installer/windows/`, `README.md` (Simplified Chinese),
  `README.en.md` (English) and `examples/cumcm-single-file`
  (fictional test content generated for this project; no real persons,
  schools, teams or papers).

## Go standard library and toolchain — BSD-3-Clause

`nodepaper.exe` is compiled with the Go toolchain and contains Go standard
library code. The Go standard library is distributed under a BSD-3-Clause
license; the Go-specific copyright and license text is in
`licenses/BSD-3-Clause.txt`. The Go project's license information is available
at <https://go.dev/LICENSE>.

## gopkg.in/yaml.v3 — MIT AND Apache-2.0

The Go module `gopkg.in/yaml.v3` (v3.0.1) is used for `nodepaper.yaml`
parsing. It is covered by two licenses: the MIT License (files ported from
libyaml) and the Apache License 2.0. The exact upstream v3.0.1 copyright and
dual-license notice is in `licenses/YAML-V3-LICENSE.txt`; the Apache 2.0 full
text is in `licenses/Apache-2.0.txt`. Upstream:
<https://github.com/go-yaml/yaml/tree/v3.0.1>. (The separate
`licenses/MIT.txt` is NodePaper's own MIT text and does not replace the YAML
upstream notice.)

## Pandoc 3.9 — GPL-2.0-or-later (bundled binary)

The release package bundles `tools/windows-x64/pandoc/pandoc.exe` (version
3.9, pinned by the Profile) so that testers do not need to install Pandoc.
Pandoc is a separate executable invoked by NodePaper at build time; it is not
linked into `nodepaper.exe`.

- Copyright (C) 2006-2024 John MacFarlane
- License: GNU General Public License version 2 or later; full text in
  `licenses/GPL-2.0.txt`
- Exact upstream 3.9 copyright and per-component notices:
  `licenses/PANDOC-COPYRIGHT.txt`
- Corresponding source archive included in the package:
  `tools/windows-x64/sources/pandoc-3.9-source.tar.gz`, SHA-256
  `d8da16e1ad1f685123fbc1a5a83b74766bcfd939dc6989484822f023bb70438f`
- Upstream: <https://github.com/jgm/pandoc/tree/3.9>,
  <https://hackage.haskell.org/package/pandoc-3.9>

## pandoc-crossref 0.3.24 — GPL-2.0-or-later (bundled binary)

The release package bundles
`tools/windows-x64/pandoc-crossref/pandoc-crossref.exe` (version 0.3.24,
pinned by the Profile). It is a separate executable invoked by NodePaper at
build time; it is not linked into `nodepaper.exe`.

- License: GNU General Public License version 2 or later; full text in
  `licenses/GPL-2.0.txt`
- Corresponding source archive included in the package:
  `tools/windows-x64/sources/pandoc-crossref-0.3.24-source.tar.gz`, SHA-256
  `ea9e06e5f95dee428d48005a4776bffa4d02c4936097aff269cafe81ec39105b`
- Upstream: <https://github.com/lierdakil/pandoc-crossref/tree/v0.3.24>

## CSL style — CC BY-SA 3.0

The CUMCM Profile redistributes the citation style
`china-national-standard-gb-t-7714-2015-numeric.csl` from the
citation-style-language/styles repository (upstream Git blob
`cc05707d5feaa7cb78df27a8da7f44de6aeb4934`). It is unchanged and is licensed
under Creative Commons Attribution-ShareAlike 3.0; full text in
`licenses/CC-BY-SA-3.0.txt`. Authors named by the CSL file: 牛耕田;
contributor Zeping Lee. Upstream:
<https://github.com/citation-style-language/styles>.

## Inno Setup 6.7.3 — Inno Setup License (Setup channel only)

The Windows Setup channel
(`NodePaper-Setup-<version>-windows-x64.exe`) is generated with the pinned
Inno Setup 6.7.3 compiler and therefore contains Inno Setup installer code,
copyright (C) 1997-2026 Jordan Russell and (C) 2000-2026 Martijn Laan. The
license text of the pinned version is in
`installer/windows/INNO-SETUP-LICENSE.txt`; the pinned download URL, SHA-256
values and license location are recorded in
`installer/windows/innosetup-toolchain.json`. Upstream:
<https://github.com/jrsoftware/issrc> (tag `is-6_7_3`).

The Inno Setup compiler itself is a build tool. It is not part of the release
payload, not contained in `nodepaper-<version>-windows-x64.zip` and not
installed on user machines.

## Not bundled

NodePaper does **not** bundle a TeX distribution. Users must install TeX Live
or MiKTeX with `xelatex` and `latexmk` themselves; those distributions carry
their own licenses and obligations. The release package contains no user
papers, no real contest submissions, no secrets and no absolute development
machine paths.

## Redistribution conditions

- The bundled GPL executables (Pandoc, pandoc-crossref) are redistributed
  under the GPL-2.0-or-later terms. Their exact-version source archives,
  SHA-256 values, upstream notices and GPL text are distributed in the same
  package; `tools/versions.json` records the binary/source download URLs and
  checksums.
- The CSL style is redistributed under CC BY-SA 3.0 terms; modifications must
  preserve attribution and comply with ShareAlike.
- The Setup channel is redistributed under the Inno Setup License; the Inno
  Setup copyright notices and web addresses embedded by the compiler are left
  untouched.
- Neither the Setup nor the ZIP is Authenticode signed at this stage. The
  unsigned state, the fixed source commit, the file sizes and the SHA-256
  values are published as-is; NodePaper never claims a trusted publisher and
  never asks users to disable security features.
- NodePaper makes no claim of official certification by the competition
  organizers, and the candidate CUMCM Profile is not an official template.
