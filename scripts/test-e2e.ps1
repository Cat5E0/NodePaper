param(
    [string]$Fixture = "",
    [ValidateSet("", "alpha", "continuous", "none")]
    [string]$AppendixNumbering = "",
    [ValidateSet("", "tango", "pygments", "kate")]
    [string]$HighlightStyle = "",
    [string]$ReviewOutput = "",
    [string]$ProfileOverride = "",
    [switch]$KeepWorkDirectory,
    # Builds under an ASCII path containing "~" instead of the usual Unicode
    # one. Guards the M4-13 defect where XeLaTeX read the file name on its
    # command line as TeX tokens, making "~" the active character and
    # truncating the path. Windows gives any account name longer than eight
    # characters such an 8.3 alias, so this is an ordinary user's path.
    [switch]$TildeWorkRoot
)

$ErrorActionPreference = "Stop"
$inspectDir = ""
if ([string]::IsNullOrWhiteSpace($Fixture)) {
    foreach ($case in @("minimal-valid", "complete-single-file", "complete-multi-file", "nocite-only", "citation-shapes", "tikz-basic", "pgf-basic", "layout-stress")) {
        & $PSCommandPath -Fixture $case -HighlightStyle $HighlightStyle -ReviewOutput $ReviewOutput -ProfileOverride $ProfileOverride -KeepWorkDirectory:$KeepWorkDirectory
    }
    # One extra pass over the smallest fixture, under a path containing "~".
    # The full matrix is not repeated: the defect is in how the path reaches
    # XeLaTeX, which is identical for every fixture.
    & $PSCommandPath -Fixture "minimal-valid" -HighlightStyle $HighlightStyle -ProfileOverride $ProfileOverride -KeepWorkDirectory:$KeepWorkDirectory -TildeWorkRoot
    Write-Host "M3 E2E suite passed."
    return
}

. (Join-Path $PSScriptRoot "test-common.ps1")
$root = Get-NodePaperRepoRoot
$fixtureRoot = Join-Path $root "nodepaper\core\tests\fixtures\$Fixture"
if (-not (Test-Path -LiteralPath $fixtureRoot -PathType Container)) {
    throw "Fixture not found: $fixtureRoot"
}

# Every real E2E runs through a Unicode path containing spaces so the
# PowerShell transition boundary is continuously checked for path quoting.
# Character codes avoid Windows PowerShell 5.1 misreading a BOM-less script.
$unicodePathPart = ([string][char]0x4E2D) + ([char]0x6587)
if ($TildeWorkRoot) {
    # Deliberately ASCII. A path combining Chinese, a space and a tilde fails
    # for an unrelated reason -- XeLaTeX cannot put such a value in
    # TEXMF_OUTPUT_DIRECTORY, and the mojibake swallows the tilde -- which
    # would make this scenario report the wrong defect. Chinese alone, and
    # Chinese with a tilde, both build; the ordinary scenarios already cover
    # them.
    $workRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("nodepaper-e2e-tilde~1-" + [Guid]::NewGuid().ToString("N"))
}
else {
    $workRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("nodepaper e2e $unicodePathPart " + [Guid]::NewGuid().ToString("N"))
}
$projectDir = Join-Path $workRoot "project"
$exePath = Join-Path $workRoot "nodepaper.exe"
$passed = $false

function Get-FixtureSnapshot([string]$Path) {
    $result = [ordered]@{}
    Get-ChildItem -LiteralPath $Path -Recurse -File | Sort-Object FullName | ForEach-Object {
        $relative = $_.FullName.Substring($Path.Length).TrimStart('\')
        $result[$relative] = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash
    }
    return ($result | ConvertTo-Json -Compress)
}

function Get-StringSHA256([string]$Value) {
    $sha = [System.Security.Cryptography.SHA256]::Create()
    try {
        return ([System.BitConverter]::ToString($sha.ComputeHash([System.Text.Encoding]::UTF8.GetBytes($Value)))).Replace("-", "").ToLowerInvariant()
    }
    finally {
        $sha.Dispose()
    }
}

try {
    New-Item -ItemType Directory -Force -Path $projectDir | Out-Null
    Copy-Item -Path (Join-Path $fixtureRoot "*") -Destination $projectDir -Recurse -Force
    # Ordinary fixtures are source trees. Developers may have built one in
    # place, but ignored .nodepaper/dist output must never leak into the
    # temporary project and make an E2E assertion inspect stale artifacts.
    # The two lock fixtures intentionally keep .nodepaper/build.lock as input.
    if ($Fixture -notin @("damaged-lock", "stale-lock")) {
        foreach ($generatedDirectory in @(".nodepaper", "dist")) {
            $generatedPath = Join-Path $projectDir $generatedDirectory
            if (Test-Path -LiteralPath $generatedPath) {
                Remove-Item -LiteralPath $generatedPath -Recurse -Force
            }
        }
    }
    if (-not [string]::IsNullOrWhiteSpace($AppendixNumbering)) {
        $configPath = Join-Path $projectDir "nodepaper.yaml"
        $configText = Get-Content -LiteralPath $configPath -Raw -Encoding UTF8
        if ($configText -notmatch '(?m)^\s*numbering:\s*(alpha|continuous|none)\s*$') {
            throw "Fixture does not declare appendix.numbering: $Fixture"
        }
        $configText = $configText -replace '(?m)^(\s*numbering:\s*)(alpha|continuous|none)(\s*)$', "`$1$AppendixNumbering`$3"
        Set-Content -LiteralPath $configPath -Value $configText -Encoding UTF8
    }
    $effectiveAppendixNumbering = if ([string]::IsNullOrWhiteSpace($AppendixNumbering)) { "alpha" } else { $AppendixNumbering }
    $effectiveHighlightStyle = if ([string]::IsNullOrWhiteSpace($HighlightStyle)) { "tango" } else { $HighlightStyle }
    if (-not [string]::IsNullOrWhiteSpace($HighlightStyle)) {
        $configPath = Join-Path $projectDir "nodepaper.yaml"
        $configText = Get-Content -LiteralPath $configPath -Raw -Encoding UTF8
        if ($configText -match '(?m)^\s*style:\s*(tango|pygments|kate)\s*$') {
            $configText = $configText -replace '(?m)^(\s*style:\s*)(tango|pygments|kate)(\s*)$', "`$1$HighlightStyle`$3"
        }
        else {
            $configText = $configText.TrimEnd() + "`r`n" + "highlight:`r`n  style: $HighlightStyle`r`n"
        }
        Set-Content -LiteralPath $configPath -Value $configText -Encoding UTF8
    }
    $before = Get-FixtureSnapshot $fixtureRoot
    $profileDir = if ([string]::IsNullOrWhiteSpace($ProfileOverride)) { Join-Path $root "profiles\cumcm" } else { $ProfileOverride }
    $profileMetadata = Get-Content -LiteralPath (Join-Path $profileDir "profile.json") -Raw -Encoding UTF8 | ConvertFrom-Json
    $profileBefore = Get-FixtureSnapshot $profileDir

    $go = Get-NodePaperGo
    Push-Location (Get-NodePaperCoreRoot)
    try {
        & $go build -o $exePath ./cmd/nodepaper
        if ($LASTEXITCODE -ne 0) {
            throw "go build failed with exit code $LASTEXITCODE"
        }
    }
    finally {
        Pop-Location
    }

    $toolPaths = @(
        (Join-Path ([Environment]::GetFolderPath([Environment+SpecialFolder]::LocalApplicationData)) "NodePaper\toolchains\windows-x64\pandoc"),
        (Join-Path ([Environment]::GetFolderPath([Environment+SpecialFolder]::LocalApplicationData)) "NodePaper\toolchains\windows-x64\pandoc-crossref")
    )
    $env:PATH = ($toolPaths -join ';') + ';' + $env:PATH

    # M2 runs the source-tree transition script. M4 ZIP tests will instead
    # verify the default adjacent-to-executable packaged resource layout.
    $env:NODEPAPER_BUILD_SCRIPT = Join-Path $root "scripts\build\Build-Paper.ps1"
    $env:NODEPAPER_PROFILE_DIR = $profileDir

    & $exePath doctor $projectDir --format json
    if ($LASTEXITCODE -ne 0) {
        throw "doctor failed with exit code $LASTEXITCODE"
    }
    & $exePath validate $projectDir --format json
    if ($LASTEXITCODE -ne 0) {
        throw "validate failed with exit code $LASTEXITCODE"
    }
    & $exePath build $projectDir --format json
    if ($LASTEXITCODE -ne 0) {
        throw "build failed with exit code $LASTEXITCODE"
    }

    $tex = Join-Path $projectDir ".nodepaper\build\paper.tex"
    if (-not (Test-Path -LiteralPath $tex -PathType Leaf)) {
        throw "generated LaTeX not found: $tex"
    }
    if (-not (Select-String -LiteralPath $tex -SimpleMatch "\documentclass" -Quiet)) {
        throw "generated LaTeX is not standalone: $tex"
    }
    $texText = Get-Content -LiteralPath $tex -Raw -Encoding UTF8

    # Which \documentclass line is correct depends on the machine. SimHei and
    # KaiTi ship as an optional Windows feature, and when they are absent the
    # Profile emits its fallback instead (M4-08). Asserting only the ordinary
    # line made this suite fail on every machine without them, including the
    # miktex-e2e runner. Each environment is held to its own contract, so the
    # fallback is checked rather than merely tolerated.
    $supplementalPresent = $true
    foreach ($fontFile in @("simhei.ttf", "simkai.ttf")) {
        $found = @(
            (Join-Path $env:WINDIR "Fonts\$fontFile"),
            (Join-Path $env:LOCALAPPDATA "Microsoft\Windows\Fonts\$fontFile")
        ) | Where-Object { Test-Path -LiteralPath $_ }
        if (-not $found) { $supplementalPresent = $false }
    }
    if ($supplementalPresent) {
        $documentClassContract = @("\documentclass[UTF8,zihao=-4,a4paper]{ctexart}")
    }
    else {
        Write-Host "SimHei/KaiTi absent: expecting the Profile's font fallback"
        $documentClassContract = @(
            "\documentclass[UTF8,zihao=-4,a4paper,fontset=none]{ctexart}",
            "\setCJKmainfont{SimSun}[AutoFakeBold=true, AutoFakeSlant=true]",
            "\setCJKfamilyfont{zhhei}{SimSun}[AutoFakeBold=true]"
        )
    }

    foreach ($required in ($documentClassContract + @("top=2.5cm", "bottom=2.5cm", "left=2.5cm", "right=2.5cm", "bookmarksopenlevel=2", "bookmarksdepth=3", "pdfpagemode=UseOutlines", "{nodepaper.abstract}"))) {
        if (-not $texText.Contains($required)) {
            throw "generated LaTeX contract is missing: $required"
        }
    }
    foreach ($forbidden in @("\tableofcontents", "`$body`$", "`$title`$", "shell-escape")) {
        if ($texText.Contains($forbidden)) {
            throw "generated LaTeX contains a forbidden contract marker: $forbidden"
        }
    }
    if ($texText.Contains($root)) {
        throw "generated LaTeX contains an absolute source-tree path"
    }
    # A nocite-only project has no inline citation by definition, so the linked
    # form never appears; citeproc still renders the nocite entries into the
    # reference list as \bibitem[\citeproctext]{ref-...}. Assert that shape here,
    # and assert the absence of the inline form as well: that absence is exactly
    # what let the M4-23 export defect through, so it belongs in the contract
    # rather than being silently exempted.
    if ($Fixture -eq "nocite-only") {
        if ($texText.Contains("\citeproc{ref-")) {
            throw "nocite-only generated LaTeX contains an inline Citeproc citation"
        }
        foreach ($required in @("\bibitem[\citeproctext]{ref-wang2024bikesharing}", "\bibitem[\citeproctext]{ref-smith2023forecast}")) {
            if (-not $texText.Contains($required)) {
                throw "nocite-only generated LaTeX is missing a nocite reference entry: $required"
            }
        }
        if ($texText.Contains("ref-unused2020entry")) {
            throw "nocite-only generated LaTeX pulled in an entry that is neither cited nor nocited"
        }
    }
    elseif (-not $texText.Contains("\citeproc{ref-")) {
        throw "generated LaTeX does not contain linked Citeproc citations"
    }
    # citation-shapes pins the inline-citation shapes no other fixture carries:
    # a key cited more than once must resolve to one shared target, a key that
    # arrives only through nocite must still reach the reference list, and a key
    # that is neither cited nor nocited must reach neither. The rendered form of
    # the marker itself (spacing next to CJK, and whether two consecutive
    # numbers collapse to a range) is deliberately NOT asserted: those are open
    # typography questions for the maintainer, and pinning today's output would
    # turn an undecided question into a golden.
    if ($Fixture -eq "citation-shapes") {
        $repeated = ([regex]::Matches($texText, [regex]::Escape("\citeproc{ref-zhao2024scheduling}"))).Count
        if ($repeated -ne 3) {
            throw "citation-shapes: the thrice-cited key resolved to $repeated linked citations, want 3"
        }
        if (-not $texText.Contains("ref-listed2020nocite")) {
            throw "citation-shapes: the nocite-only entry is missing from the reference list"
        }
        if ($texText.Contains("unused2019entry")) {
            throw "citation-shapes: an entry that is neither cited nor nocited reached the generated LaTeX"
        }
    }
    if ($Fixture -eq "highlight-showcase") {
        foreach ($required in @("\RecustomVerbatimEnvironment{Highlighting}", "breaknonspaceingroup=true", "\definecolor{nodepapercodeframe}", "\begin{mdframed}")) {
            if (-not $texText.Contains($required)) {
                throw "highlight-showcase LaTeX contract is missing: $required"
            }
        }
    }
    if ($Fixture -eq "layout-stress") {
        # The fixture sets titleAbstractSkip / abstractKeywordsSkip to values that
        # are neither the defaults nor each other, so finding both lengths here
        # proves the whole chain carried them: nodepaper.yaml -> Go ->
        # sources.json -> Convert script -> Pandoc metadata -> template. Before
        # M4-00 D2's two fields were implemented these gaps were literals, and a
        # project could not change them at all.
        foreach ($required in @("\vspace{1.2em}", "\vspace{0.3em}")) {
            if (-not $texText.Contains($required)) {
                throw "layout-stress: a configured abstract gap did not reach the LaTeX: $required"
            }
        }
        foreach ($required in @("\RecustomVerbatimEnvironment{Highlighting}", "breaknonspaceingroup=true", "\definecolor{nodepapercodeframe}", "\begin{mdframed}", "\input{tables/complex-result.tex}", "\input{equations/long-objective.tex}", "\input{figures/tikz-diagram.tex}", "\input{figures/mpl-plot.pgf}")) {
            if (-not $texText.Contains($required)) {
                throw "layout-stress LaTeX contract is missing: $required"
            }
        }
        # The Fragment cannot load tikz itself (NP2506 forbids \usepackage), so a
        # Profile that stopped shipping it would fail compilation with
        # "Environment tikzpicture undefined". Asserting the preamble here names
        # the cause instead of leaving a bare LaTeX error.
        if (-not $texText.Contains("\usepackage{tikz}")) {
            throw "layout-stress needs tikz in the Profile preamble; the TikZ Fragment cannot load it (NP2506)"
        }
        foreach ($requiredTableWidth in @(
            "\begin{longtable}[]{@{}cc@{}}",
            "\real{0.1600}",
            "\real{0.6400}"
        )) {
            if (-not $texText.Contains($requiredTableWidth)) {
                throw "layout-stress Markdown table-width contract is missing: $requiredTableWidth"
            }
        }
        $appendixText = ([string][char]0x9644) + ([char]0x5F55)
        $multilingualCodeText = ([string][char]0x591A) + ([char]0x8BED) + ([char]0x8A00) + ([char]0x4EE3) + ([char]0x7801)
        switch ($effectiveAppendixNumbering) {
            "alpha" {
                if (-not $texText.Contains("\nodepaperAppendixAlpha") -or -not $texText.Contains("\section{$multilingualCodeText}")) {
                    throw "alpha appendix LaTeX contract failed"
                }
            }
            "continuous" {
                if (-not $texText.Contains("\section{$appendixText}") -or -not $texText.Contains("\subsection{$multilingualCodeText}")) {
                    throw "continuous appendix LaTeX contract failed"
                }
            }
            "none" {
                if (-not $texText.Contains("\nodepaperAppendixNone") -or -not $texText.Contains("\subsection*{$multilingualCodeText}")) {
                    throw "unnumbered appendix LaTeX contract failed"
                }
            }
        }
    }
    if ($Fixture -eq "tikz-basic") {
        foreach ($required in @("\input{figures/basic.tex}", "\usepackage{tikz}")) {
            if (-not $texText.Contains($required)) {
                throw "tikz-basic LaTeX contract is missing: $required"
            }
        }
    }
    if ($Fixture -eq "pgf-basic") {
        foreach ($required in @("\input{figures/basic.pgf}", "\usepackage{tikz}")) {
            if (-not $texText.Contains($required)) {
                throw "pgf-basic LaTeX contract is missing: $required"
            }
        }
    }

    $logs = @(Get-ChildItem -LiteralPath (Join-Path $projectDir ".nodepaper\logs") -Filter "build-*.log" -File)
    if ($logs.Count -lt 1 -or $logs[0].Length -eq 0) {
        throw "complete build log was not created"
    }
    $goBuildLog = Get-Content -LiteralPath $logs[0].FullName -Raw -Encoding UTF8
    if (-not $goBuildLog.Contains("Command: powershell.exe") -or -not $goBuildLog.Contains("Build-Paper.ps1")) {
        throw "Go build log does not prove delegation to the PowerShell transition script"
    }
    $transitionLogs = @(Get-ChildItem -LiteralPath (Join-Path $projectDir ".nodepaper\logs") -Directory -Filter "*-powershell" |
        ForEach-Object { Get-ChildItem -LiteralPath $_.FullName -Filter "build-*.log" -File })
    if ($transitionLogs.Count -lt 1 -or $transitionLogs[0].Length -eq 0) {
        throw "PowerShell transition log was not created"
    }
    $transitionLogText = Get-Content -LiteralPath $transitionLogs[0].FullName -Raw -Encoding UTF8
    if (-not $transitionLogText.Contains("Convert-CumcmProjectToLatex.ps1") -or
        -not $transitionLogText.Contains("CUMCM rules version: 2026") -or
        -not $transitionLogText.Contains("--citeproc")) {
        throw "PowerShell log does not prove CUMCM Profile and Citeproc execution"
    }

    # A second immediate build verifies repeatability and Build ID/log uniqueness.
    & $exePath build $projectDir --format json
    if ($LASTEXITCODE -ne 0) {
        throw "second build failed with exit code $LASTEXITCODE"
    }
    $logs = @(Get-ChildItem -LiteralPath (Join-Path $projectDir ".nodepaper\logs") -Filter "build-*.log" -File)
    if ($logs.Count -lt 2) {
        throw "two builds did not create distinct log files"
    }

    # Export-route bibliography regression. --natbib and --biblatex do not read
    # Pandoc's nocite metadata, so entries that arrive only through nocite would
    # leave the exported .tex with no \citation command at all and fail
    # bibtex/biber (M4-23). The Citeproc build above proves the main route; this
    # block covers the export routes that proof cannot reach. Two fixtures drive
    # it, each with its own expectations, so the remaining E2E cases keep their
    # existing shape and runtime:
    #   nocite-only     - nocite is the only source of entries
    #   citation-shapes - inline citations and nocite together, which is the
    #                     combination a faithful real paper needs and which the
    #                     emitted \nocite must not disturb
    $exportExpectations = @{
        "nocite-only" = @{
            RequiredNocite         = "\nocite{wang2024bikesharing,smith2023forecast}"
            ForbiddenKey           = "unused2020entry"
            RequiresInlineCitation = $false
        }
        "citation-shapes" = @{
            RequiredNocite         = "\nocite{listed2020nocite}"
            ForbiddenKey           = "unused2019entry"
            RequiresInlineCitation = $true
        }
    }
    if ($exportExpectations.ContainsKey($Fixture)) {
        $expect = $exportExpectations[$Fixture]
        $exportWorkRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("nodepaper-export-regression-" + [Guid]::NewGuid().ToString("N"))
        $exportSource = Join-Path $exportWorkRoot "project"
        New-Item -ItemType Directory -Force -Path $exportSource | Out-Null
        Copy-Item -Path (Join-Path $fixtureRoot "*") -Destination $exportSource -Recurse -Force
        try {
            foreach ($bibMode in @("bibtex", "biblatex")) {
                $exportTarget = Join-Path $exportWorkRoot "latex-$bibMode"
                & $exePath export $exportSource --to $exportTarget --bib $bibMode --format json
                if ($LASTEXITCODE -ne 0) {
                    throw "$Fixture export --bib $bibMode failed with exit code $LASTEXITCODE"
                }
                $exportedTex = Join-Path $exportTarget "paper.tex"
                if (-not (Test-Path -LiteralPath $exportedTex -PathType Leaf)) {
                    throw "$Fixture export --bib $bibMode produced no paper.tex: $exportedTex"
                }
                $exportedTexText = Get-Content -LiteralPath $exportedTex -Raw -Encoding UTF8
                # The nocite keys must arrive as one \nocite{} command,
                # immediately before \bibliography{references} / \printbibliography,
                # so bibtex/biber record a citation command instead of failing.
                if (-not $exportedTexText.Contains($expect.RequiredNocite)) {
                    throw "$Fixture export --bib $bibMode did not emit $($expect.RequiredNocite): $exportedTex"
                }
                # An entry that is neither cited nor in nocite must not be
                # forced into the reference list; the exported .tex must not
                # name it in a \nocite, nor anywhere else.
                if ($exportedTexText -match ('\\nocite\{[^}]*' + [regex]::Escape($expect.ForbiddenKey))) {
                    throw "$Fixture export --bib $bibMode pulled an uncited entry into \nocite: $exportedTex"
                }
                if ($expect.RequiresInlineCitation) {
                    # The emitted \nocite must not replace or suppress the real
                    # inline citations: both have to reach the same .tex, which
                    # is the shape a paper that cites some entries and merely
                    # lists others depends on.
                    $inlineCommand = if ($bibMode -eq "bibtex") { "\citep{" } else { "\autocite{" }
                    if (-not $exportedTexText.Contains($inlineCommand)) {
                        throw "$Fixture export --bib $bibMode lost its inline citations ($inlineCommand missing): $exportedTex"
                    }
                    if ($exportedTexText.Contains($expect.ForbiddenKey)) {
                        throw "$Fixture export --bib $bibMode carries the neither-cited-nor-nocited key: $exportedTex"
                    }
                }
            }
            # inline mode renders the reference list into the .tex itself
            # and must never carry a \nocite command, which would be dead
            # LaTeX with no bibtex pass to read it.
            & $exePath export $exportSource --to (Join-Path $exportWorkRoot "latex-inline") --bib inline --format json
            if ($LASTEXITCODE -ne 0) {
                throw "$Fixture export --bib inline failed with exit code $LASTEXITCODE"
            }
            $inlineTex = Join-Path $exportWorkRoot "latex-inline\paper.tex"
            $inlineTexText = Get-Content -LiteralPath $inlineTex -Raw -Encoding UTF8
            if ($inlineTexText -match '\\nocite\{') {
                throw "$Fixture export --bib inline emitted a \nocite command: $inlineTex"
            }
        }
        finally {
            Remove-Item -LiteralPath $exportWorkRoot -Recurse -Force -ErrorAction SilentlyContinue
        }
    }

    $outlinePath = Join-Path $projectDir ".nodepaper\build\paper.out"
    if (-not (Test-Path -LiteralPath $outlinePath -PathType Leaf)) {
        throw "PDF bookmark outline file was not generated"
    }
    $outlineText = Get-Content -LiteralPath $outlinePath -Raw -Encoding UTF8
    if (-not $outlineText.Contains("{nodepaper.abstract") -or [regex]::Matches($outlineText, '\\BOOKMARK').Count -lt 2) {
        throw "PDF bookmark outline is missing the abstract or section entries"
    }
    if ($Fixture -eq "layout-stress") {
        switch ($effectiveAppendixNumbering) {
            "alpha" {
                if ($outlineText -notmatch '\\BOOKMARK \[2\].*\{appendix\.A\}' -or $outlineText -notmatch '\\BOOKMARK \[2\].*\{appendix\.B\}') {
                    throw "alpha appendix bookmark hierarchy is missing A/B children"
                }
            }
            "continuous" {
                if ([regex]::Matches($outlineText, '\\BOOKMARK \[2\].*\{subsection\.').Count -lt 2) {
                    throw "continuous appendix bookmark hierarchy is missing child entries"
                }
            }
            "none" {
                if ([regex]::Matches($outlineText, '\\BOOKMARK \[2\].*\{section\*\.').Count -lt 2) {
                    throw "unnumbered appendix bookmark hierarchy is missing child entries"
                }
            }
        }
    }

    $pdf = Join-Path $projectDir "dist\paper.pdf"
    if (-not (Test-Path -LiteralPath $pdf -PathType Leaf)) {
        throw "PDF not found: $pdf"
    }
    $bytes = [System.IO.File]::ReadAllBytes($pdf)
    if ($bytes.Length -lt 5 -or [System.Text.Encoding]::ASCII.GetString($bytes, 0, 5) -ne "%PDF-") {
        throw "Generated artifact is not a valid PDF header: $pdf"
    }
    if ($bytes.Length -gt 20MB) {
        throw "Generated electronic paper exceeds the 20 MB CUMCM limit: $($bytes.Length)"
    }

    $pdfInfo = Get-Command "pdfinfo.exe" -ErrorAction SilentlyContinue
    $pdfToText = Get-Command "pdftotext.exe" -ErrorAction SilentlyContinue
    $pdfToHtml = Get-Command "pdftohtml.exe" -ErrorAction SilentlyContinue
    $pdfFonts = Get-Command "pdffonts.exe" -ErrorAction SilentlyContinue
    if (-not $pdfInfo -or -not $pdfToText -or -not $pdfToHtml -or -not $pdfFonts) {
        throw "pdfinfo.exe, pdftotext.exe, pdftohtml.exe and pdffonts.exe are required for PDF structure checks"
    }

    # The project deliberately lives under a path containing Chinese, because
    # NodePaper has to cope with one. The poppler tools do not: they take file
    # names in the console code page, so on a runner whose locale is not
    # Chinese the name arrives as "nodepaper e2e ??..." and the file cannot be
    # opened. That is a limitation of the inspection tools, not of what is
    # being inspected -- the build itself succeeds on exactly that path. So the
    # artefact is copied to an ASCII-only location and poppler reads the copy,
    # leaving the Chinese path still covered by everything upstream.
    $inspectDir = Join-Path ([System.IO.Path]::GetTempPath()) ("nodepaper-pdf-inspect-" + [Guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Force -Path $inspectDir | Out-Null
    $pdf = Join-Path $inspectDir "paper.pdf"
    Copy-Item -LiteralPath (Join-Path $projectDir "dist\paper.pdf") -Destination $pdf -Force
    $infoOutput = & $pdfInfo.Source -box $pdf 2>&1
    if ($LASTEXITCODE -ne 0 -or -not ($infoOutput -match '^Pages:\s+[1-9][0-9]*')) {
        throw "PDF parser did not report a positive page count:`n$($infoOutput -join [Environment]::NewLine)"
    }
    if (-not ($infoOutput -match '^Page size:\s+595(?:\.[0-9]+)?\s+x\s+842(?:\.[0-9]+)?\s+pts\s+\(A4\)') -and
        -not ($infoOutput -match '^Page size:\s+595\.2[0-9]*\s+x\s+841\.8[0-9]*\s+pts\s+\(A4\)')) {
        throw "PDF is not A4:`n$($infoOutput -join [Environment]::NewLine)"
    }
    $fontOutput = & $pdfFonts.Source $pdf 2>&1
    if ($LASTEXITCODE -ne 0) {
        throw "pdffonts failed"
    }
    foreach ($line in $fontOutput) {
        if ($line -match '\s+(yes|no)\s+(yes|no)\s+(yes|no)\s+\d+\s+\d+\s*$' -and $Matches[1] -ne "yes") {
            throw "PDF contains a non-embedded font: $line"
        }
    }

    # Poppler writes its output through the same code-page-bound path
    # handling it uses for input, so these land beside the PDF copy
    # rather than in the Chinese work directory.
    $textPath = Join-Path $inspectDir "paper.txt"
    & $pdfToText.Source -enc UTF-8 $pdf $textPath
    if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $textPath -PathType Leaf)) {
        throw "pdftotext failed"
    }
    $pdfText = Get-Content -LiteralPath $textPath -Raw -Encoding UTF8
    $abstractHeading = ([string][char]0x6458) + ([char]0x8981)
    if ([string]::IsNullOrWhiteSpace($pdfText) -or -not $pdfText.Contains($abstractHeading)) {
        throw "PDF text is empty or missing the abstract heading"
    }
    $firstPagePath = Join-Path $inspectDir "paper-first-page.txt"
    & $pdfToText.Source -f 1 -l 1 -enc UTF-8 $pdf $firstPagePath
    if ($LASTEXITCODE -ne 0) {
        throw "pdftotext failed to extract the first page"
    }
    $firstPageText = Get-Content -LiteralPath $firstPagePath -Raw -Encoding UTF8
    $keywordsHeading = ([string][char]0x5173) + ([char]0x952E) + ([char]0x8BCD)
    if (-not $firstPageText.Contains($abstractHeading) -or -not $firstPageText.Contains($keywordsHeading)) {
        throw "The electronic paper first page is not the title/abstract/keywords page"
    }
    if ([regex]::Matches($firstPageText, $keywordsHeading).Count -ne 1) {
        throw "The electronic paper first page must contain exactly one keywords label"
    }
    $contentsHeading = ([string][char]0x76EE) + ([char]0x5F55)
    $commitmentHeading = ([string][char]0x627F) + ([char]0x8BFA) + ([char]0x4E66)
    $numberingHeading = ([string][char]0x7F16) + ([char]0x53F7) + ([char]0x4E13) + ([char]0x7528) + ([char]0x9875)
    foreach ($forbidden in @($contentsHeading, $commitmentHeading, $numberingHeading)) {
        if ($pdfText.Contains($forbidden)) {
            throw "Electronic paper contains a forbidden contents/commitment/numbering page marker"
        }
    }
    $referencesHeading = ([string][char]0x53C2) + ([char]0x8003) + ([char]0x6587) + ([char]0x732E)
    # The probe is one bibliography entry that must have rendered, so an empty
    # or dropped reference list fails here rather than passing on the heading
    # alone. Most fixtures share one bib and so share the default probe;
    # citation-shapes ships its own entries and names its own. Keep every probe
    # ASCII so this file needs no [char] construction for it.
    $bibliographyProbe = switch ($Fixture) {
        "citation-shapes" { "2021: 61-68" }
        default           { "Demand Forecasting for Shared Mobility Systems" }
    }
    if (-not $pdfText.Contains($referencesHeading) -or -not $pdfText.Contains($bibliographyProbe)) {
        throw "PDF is missing the Citeproc-generated bibliography (probe: $bibliographyProbe)"
    }
    $linkBase = Join-Path ([System.IO.Path]::GetTempPath()) ("nodepaper-links-" + [Guid]::NewGuid().ToString("N"))
    & $pdfToHtml.Source -xml -hidden -nodrm $pdf $linkBase | Out-Null
    $linkXmlPath = $linkBase + ".xml"
    if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $linkXmlPath -PathType Leaf)) {
        throw "pdftohtml failed to extract PDF links"
    }
    $linkXmlText = Get-Content -LiteralPath $linkXmlPath -Raw -Encoding UTF8
    Remove-Item -LiteralPath $linkXmlPath -Force -ErrorAction SilentlyContinue
    # Poppler may place either both brackets, only the number, or the opening
    # bracket and number inside the anchor depending on glyph positioning.
    $linkedCitationPattern = '(?:<a href="[^"]+#[0-9]+">\[1\]</a>|\[<a href="[^"]+#[0-9]+">1</a>\]|<a href="[^"]+#[0-9]+">\[1</a>\])'
    # A nocite-only project has no inline citation, so there is no [1] in the
    # body that could carry a link; the reference list is still present and still
    # numbered. Assert the inverse of the normal contract instead of skipping:
    # no linked inline citation at all, and the entry that is neither cited nor
    # nocited stays out of the list.
    if ($Fixture -eq "nocite-only") {
        if ($linkXmlText -match $linkedCitationPattern) {
            throw "nocite-only PDF contains a linked inline citation"
        }
        if ($pdfText.Contains("An Uncited Bibliography Entry")) {
            throw "nocite-only PDF lists an entry that is neither cited nor nocited"
        }
    }
    elseif ($linkXmlText -notmatch $linkedCitationPattern) {
        throw "PDF citation [1] does not contain an internal bibliography link"
    }
    if ($pdfText -match '@(fig|tbl|eq|sec):' -or $pdfText -match '@[A-Za-z][A-Za-z0-9_.:+/-]*') {
        throw "PDF contains an unresolved citation or cross-reference"
    }

    if ($Fixture -eq "citation-shapes" -and $pdfText.Contains("Neither Cited")) {
        throw "citation-shapes PDF lists the entry that is neither cited nor nocited"
    }
    if ($Fixture -eq "tikz-basic" -and -not $pdfText.Contains("NP-TIKZ-BASIC-01")) {
        throw "tikz-basic PDF is missing its compiled Fragment marker"
    }
    if ($Fixture -eq "pgf-basic" -and -not $pdfText.Contains("NP-PGF-BASIC-01")) {
        throw "pgf-basic PDF is missing its compiled Fragment marker"
    }

    if ($Fixture -eq "layout-stress") {
        $orderedMarkers = @(
            "NP-LAYOUT-TEXT-01",
            "NP-LAYOUT-EQUATION-01",
            "NP-LAYOUT-TABLE-01",
            "NP-LAYOUT-REFERENCE-01",
            "NP-LAYOUT-CODE-START",
            "NP-LAYOUT-CODE-END",
            "NP-LAYOUT-AFTER-CODE-01",
            "NP-LAYOUT-APPENDIX-END"
        )
        $previousIndex = -1
        foreach ($marker in $orderedMarkers) {
            $index = $pdfText.IndexOf($marker, [System.StringComparison]::Ordinal)
            if ($index -lt 0 -or $index -le $previousIndex) {
                throw "PDF layout marker is missing or out of order: $marker"
            }
            $previousIndex = $index
        }
        $normalizedPdfText = $pdfText -replace '[\s-]', ''
        foreach ($marker in @("NP-LAYOUT-LONGTABLE-FIRST", "NP-LAYOUT-LONGTABLE-LAST", "NP-LAYOUT-PLAIN-CODE-01")) {
            $normalizedMarker = $marker -replace '-', ''
            if (-not $normalizedPdfText.Contains($normalizedMarker)) {
                throw "PDF is missing layout content: $marker"
            }
        }

        # Poppler on Windows cannot reliably create a bbox output file when
        # the output path itself contains non-ASCII characters.
        $bboxPath = Join-Path ([System.IO.Path]::GetTempPath()) ("nodepaper-bbox-" + [Guid]::NewGuid().ToString("N") + ".html")
        & $pdfToText.Source -bbox-layout -enc UTF-8 $pdf $bboxPath
        if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $bboxPath -PathType Leaf)) {
            throw "pdftotext -bbox-layout failed"
        }
        [xml]$bbox = Get-Content -LiteralPath $bboxPath -Raw -Encoding UTF8
        Remove-Item -LiteralPath $bboxPath -Force -ErrorAction SilentlyContinue
        $pages = @($bbox.SelectNodes("//*[local-name()='page']"))
        if ($pages.Count -eq 0) {
            throw "Bounding Box output contains no pages"
        }
        $culture = [System.Globalization.CultureInfo]::InvariantCulture
        foreach ($page in $pages) {
            $pageWidth = [double]::Parse($page.width, $culture)
            $pageHeight = [double]::Parse($page.height, $culture)
            $words = @($page.SelectNodes(".//*[local-name()='word']"))
            if ($words.Count -eq 0) {
                throw "PDF contains an unexpected blank page"
            }
            foreach ($word in $words) {
                $xMin = [double]::Parse($word.xMin, $culture)
                $xMax = [double]::Parse($word.xMax, $culture)
                $yMin = [double]::Parse($word.yMin, $culture)
                if ($yMin -gt ($pageHeight - 55)) {
                    continue # page number/footer exception
                }
                # Geometry sets 70.87 pt margins. Allow 10 pt glyph-ink and
                # extraction tolerance while still catching material overflow.
                if ($xMin -lt 60 -or $xMax -gt ($pageWidth - 60)) {
                    throw "PDF text exceeds the reviewed body boundary: '$($word.InnerText)' x=$xMin..$xMax pageWidth=$pageWidth"
                }
            }
        }
    }

    $latexLog = Join-Path $projectDir ".nodepaper\build\paper.log"
    $latexLogText = Get-Content -LiteralPath $latexLog -Raw -ErrorAction Stop
    if ($latexLogText -match 'There were undefined references' -or
        $latexLogText -match 'Citation.+undefined' -or
        $latexLogText -match 'Overfull \\[hv]box' -or
        $latexLogText -match 'Missing character:' -or
        $latexLogText -match 'LaTeX Font Warning:' -or
        $latexLogText -match 'Package .+ Warning:' -or
        $latexLogText -match 'LaTeX Warning:') {
        throw "Final LaTeX log contains a build-critical warning"
    }

    $after = Get-FixtureSnapshot $fixtureRoot
    if ($after -ne $before) {
        throw "Source Fixture was modified by E2E"
    }
    $profileAfter = Get-FixtureSnapshot $profileDir
    if ($profileAfter -ne $profileBefore) {
        throw "Read-only CUMCM Profile was modified by E2E"
    }

    if (-not [string]::IsNullOrWhiteSpace($ReviewOutput)) {
        $reviewDir = Join-Path $ReviewOutput $Fixture
        if (-not [string]::IsNullOrWhiteSpace($AppendixNumbering)) {
            $reviewDir = Join-Path $reviewDir $AppendixNumbering
        }
        New-Item -ItemType Directory -Force -Path $reviewDir | Out-Null
        Copy-Item -LiteralPath $pdf -Destination (Join-Path $reviewDir "generated.pdf") -Force
        Copy-Item -LiteralPath $tex -Destination (Join-Path $reviewDir "generated.tex") -Force
        Copy-Item -LiteralPath $latexLog -Destination (Join-Path $reviewDir "latex.log") -Force
        $manifest = [ordered]@{
            schemaVersion = 1
            fixture = $Fixture
            appendixNumbering = $effectiveAppendixNumbering
            highlightStyle = $effectiveHighlightStyle
            generatedAt = (Get-Date).ToString("o")
            profileVersion = [string]$profileMetadata.version
            profileSnapshotSHA256 = (Get-StringSHA256 $profileAfter)
            pdfSHA256 = (Get-FileHash -LiteralPath $pdf -Algorithm SHA256).Hash.ToLowerInvariant()
            texSHA256 = (Get-FileHash -LiteralPath $tex -Algorithm SHA256).Hash.ToLowerInvariant()
            latexLogSHA256 = (Get-FileHash -LiteralPath $latexLog -Algorithm SHA256).Hash.ToLowerInvariant()
            checks = @("latex-contract", "keywords-once", "citation-links", "pdf-outline", "breakable-code-frame", "pdf-a4", "pdf-content-order", "font-embedding", "warning-zero")
        }
        $manifest | ConvertTo-Json -Depth 4 | Set-Content -LiteralPath (Join-Path $reviewDir "review-manifest.json") -Encoding UTF8
    }

    $passed = $true
    Write-Host "E2E passed: $Fixture"
}
finally {
    # The ASCII-only copy poppler reads is scratch either way: it holds nothing
    # that is not already in the work directory, so it goes even when the run
    # failed and the work directory is kept for inspection.
    if ($inspectDir -and (Test-Path -LiteralPath $inspectDir)) {
        Remove-Item -LiteralPath $inspectDir -Recurse -Force -ErrorAction SilentlyContinue
    }
    if ($passed -and -not $KeepWorkDirectory) {
        Remove-Item -LiteralPath $workRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
    else {
        Write-Host "E2E work directory retained: $workRoot"
    }
}
