param(
    [Parameter(Mandatory = $true)]
    [string]$SourceManifest,

    [Parameter(Mandatory = $true)]
    [string]$ProjectRoot,

    [Parameter(Mandatory = $true)]
    [string]$ProfileDirectory,

    [Parameter(Mandatory = $true)]
    [string]$Output,

    [Parameter(Mandatory = $true)]
    [string]$BuildDirectory,

    # citeproc is the main build route and must stay the default: Pandoc
    # resolves every citation before the .tex exists, so LaTeX never needs a
    # bibtex/biber stage. natbib and biblatex exist for `nodepaper export`,
    # which hands the user an editable project with a live references.bib.
    [ValidateSet("citeproc", "natbib", "biblatex")]
    [string]$CiteMethod = "citeproc",

    # Set by `nodepaper export` and by nothing else. The exported .tex is
    # compiled on a machine this script cannot probe, so it asks the template
    # for a preamble that chooses its Chinese fonts at compile time. `nodepaper
    # build` must never pass this: its .tex is compiled here, where the probe
    # below is the better answer, and its output has to stay byte-for-byte what
    # it has always been.
    [switch]$ExportMode,

    [switch]$AllowSystemPandoc
)

$ErrorActionPreference = "Stop"

function Get-FullPath([string]$Path) {
    return [System.IO.Path]::GetFullPath($Path)
}

function Find-Executable {
    param(
        [string]$BundledRelativePath,
        [string]$CommandName,
        [switch]$AllowSystem
    )

    $bundled = Join-Path $PSScriptRoot $BundledRelativePath
    if (Test-Path -LiteralPath $bundled -PathType Leaf) {
        return (Resolve-Path -LiteralPath $bundled).Path
    }
    if ($AllowSystem) {
        $command = Get-Command $CommandName -ErrorAction SilentlyContinue
        if ($command) {
            return $command.Source
        }
    }
    throw "Missing $CommandName. Expected '$bundled'."
}

function Assert-FileUnderRoot {
    param(
        [string]$Path,
        [string]$Root,
        [string]$Description
    )

    $fullPath = Get-FullPath $Path
    $fullRoot = Get-FullPath $Root
    $rootPrefix = $fullRoot.TrimEnd('\', '/') + [System.IO.Path]::DirectorySeparatorChar
    if (-not $fullPath.StartsWith($rootPrefix, [System.StringComparison]::OrdinalIgnoreCase)) {
        throw "$Description escapes Project Root: $fullPath"
    }
    if (-not (Test-Path -LiteralPath $fullPath -PathType Leaf)) {
        throw "$Description not found: $fullPath"
    }
    return $fullPath
}

$project = Get-FullPath $ProjectRoot
$profile = Get-FullPath $ProfileDirectory
$manifest = Assert-FileUnderRoot $SourceManifest $project "Source manifest"
$outputPath = Get-FullPath $Output
$buildDir = Get-FullPath $BuildDirectory

if (-not (Test-Path -LiteralPath $profile -PathType Container)) {
    throw "CUMCM Profile directory not found: $profile"
}

$profileConfigPath = Join-Path $profile "profile.json"
if (-not (Test-Path -LiteralPath $profileConfigPath -PathType Leaf)) {
    throw "CUMCM Profile metadata not found: $profileConfigPath"
}
$profileConfig = Get-Content -LiteralPath $profileConfigPath -Raw -Encoding UTF8 | ConvertFrom-Json
if ($profileConfig.schemaVersion -ne 1 -or $profileConfig.name -ne "cumcm") {
    throw "Unsupported CUMCM Profile metadata: $profileConfigPath"
}

$templateKey = switch ($CiteMethod) {
    "natbib"   { "bibtexTemplate" }
    "biblatex" { "biblatexTemplate" }
    default    { "template" }
}
$templateValue = [string]$profileConfig.$templateKey
if ([string]::IsNullOrWhiteSpace($templateValue)) {
    throw "CUMCM Profile declares no '$templateKey' for -CiteMethod $CiteMethod : $profileConfigPath"
}
$template = Join-Path $profile $templateValue
$crossrefMetadata = Join-Path $profile ([string]$profileConfig.crossrefMetadata)
$abstractFilter = Join-Path $profile ([string]$profileConfig.abstractFilter)
$layoutFilter = Join-Path $profile ([string]$profileConfig.layoutFilter)
$csl = Join-Path $profile ([string]$profileConfig.csl)
$warningAllowlist = Join-Path $profile ([string]$profileConfig.warningAllowlist)
foreach ($resource in @($template, $crossrefMetadata, $abstractFilter, $layoutFilter, $csl, $warningAllowlist)) {
    if (-not (Test-Path -LiteralPath $resource -PathType Leaf)) {
        throw "CUMCM Profile resource not found: $resource"
    }
}

$manifestValue = Get-Content -LiteralPath $manifest -Raw -Encoding UTF8 | ConvertFrom-Json
$sources = @($manifestValue.sources)
if ($sources.Count -eq 0) {
    throw "Source manifest contains no sources: $manifest"
}
$resolvedSources = @()
foreach ($source in $sources) {
    if ([string]::IsNullOrWhiteSpace([string]$source)) {
        throw "Source manifest contains an empty source path"
    }
    $resolvedSources += Assert-FileUnderRoot ([string]$source) $project "Markdown Source"
}

$resolvedFragments = @()
foreach ($fragment in @($manifestValue.latexFragments)) {
    if ([string]::IsNullOrWhiteSpace([string]$fragment)) {
        throw "Source manifest contains an empty LaTeX Fragment path"
    }
    $resolvedFragments += Assert-FileUnderRoot ([string]$fragment) $project "LaTeX Fragment"
}
$appendixNumbering = [string]$manifestValue.appendixNumbering
if ([string]::IsNullOrWhiteSpace($appendixNumbering)) {
    $appendixNumbering = "alpha"
}
if ($appendixNumbering -notin @("alpha", "continuous", "none")) {
    throw "Unsupported appendix numbering mode: $appendixNumbering"
}
$appendixNewPage = $true
if ($null -ne $manifestValue.appendixNewPage) {
    $appendixNewPage = [bool]$manifestValue.appendixNewPage
}
$appendixNewPageMetadata = if ($appendixNewPage) { "true" } else { "false" }
$highlightStyle = [string]$manifestValue.highlightStyle
if ([string]::IsNullOrWhiteSpace($highlightStyle)) {
    $highlightStyle = [string]$profileConfig.highlightStyle
}
if ($highlightStyle -notin @("tango", "pygments", "kate")) {
    throw "Unsupported highlight style: $highlightStyle"
}
$linespread = 1.25
if ($null -ne $manifestValue.linespread) {
    $linespread = [double]$manifestValue.linespread
}
if ($linespread -lt 1.0 -or $linespread -gt 1.3) {
    throw "Unsupported linespread value: $linespread (allowed [1.0, 1.3])"
}
$linespreadMetadata = $linespread.ToString([System.Globalization.CultureInfo]::InvariantCulture)
$abstractLinespread = 0.95
if ($null -ne $manifestValue.abstractLinespread) {
    $abstractLinespread = [double]$manifestValue.abstractLinespread
}
if ($abstractLinespread -lt 0.85 -or $abstractLinespread -gt $linespread) {
    throw "Unsupported abstractLinespread value: $abstractLinespread (allowed [0.85, linespread=$linespread])"
}
$abstractLinespreadMetadata = $abstractLinespread.ToString([System.Globalization.CultureInfo]::InvariantCulture)
$mathFont = [string]$manifestValue.mathFont
if ([string]::IsNullOrWhiteSpace($mathFont)) {
    $mathFont = "cm"
}
if ($mathFont -notin @("cm", "newtx")) {
    throw "Unsupported math font route: $mathFont (allowed cm or newtx)"
}
$mathFontNewtx = if ($mathFont -eq "newtx") { "true" } else { "false" }

# ctex binds its Chinese families while \documentclass runs, so whether SimHei
# and KaiTi exist has to be settled before the .tex is written. Probing the two
# font directories is deliberately cheap and conservative: an unreadable
# directory counts as "present" so a probe failure never switches a working
# machine onto the fallback.
function Test-InstalledFontFile {
    param([Parameter(Mandatory = $true)][string]$FileName)

    foreach ($root in @($env:WINDIR, $env:LOCALAPPDATA)) {
        if ([string]::IsNullOrWhiteSpace($root)) { continue }
        $dir = if ($root -eq $env:WINDIR) {
            Join-Path $root "Fonts"
        } else {
            Join-Path $root "Microsoft\Windows\Fonts"
        }
        if (Test-Path -LiteralPath (Join-Path $dir $FileName)) { return $true }
    }
    return $false
}

$supplementalFonts = @{ SimHei = "simhei.ttf"; KaiTi = "simkai.ttf" }
$missingFonts = @()
foreach ($name in ($supplementalFonts.Keys | Sort-Object)) {
    if (-not (Test-InstalledFontFile $supplementalFonts[$name])) { $missingFonts += $name }
}
$fontFallback = if ($missingFonts.Count -gt 0) { "true" } else { "" }
# The two font branches are mutually exclusive by construction. Under
# -ExportMode this machine's fonts say nothing about the machine that will
# compile the export, and the template's four-step \IfFontExistsTF cascade
# already contains this fallback as its second step, verbatim. Clearing the
# metadata is what selects the cascade instead of a decision made here.
if ($ExportMode) { $fontFallback = "" }

$references = Assert-FileUnderRoot (Join-Path $project "references.bib") $project "Bibliography"
New-Item -ItemType Directory -Force -Path $buildDir | Out-Null
New-Item -ItemType Directory -Force -Path ([System.IO.Path]::GetDirectoryName($outputPath)) | Out-Null

$pandoc = Find-Executable "tools\windows-x64\pandoc\pandoc.exe" "pandoc.exe" -AllowSystem:$AllowSystemPandoc
$crossref = Find-Executable "tools\windows-x64\pandoc-crossref\pandoc-crossref.exe" "pandoc-crossref.exe" -AllowSystem:$AllowSystemPandoc
$pandocVersionLine = [string]((& $pandoc --version 2>&1 | Select-Object -First 1))
$crossrefVersionLine = [string]((& $crossref --version 2>&1 | Select-Object -First 1))
if ($pandocVersionLine -notmatch ("^pandoc " + [regex]::Escape([string]$profileConfig.pandocVersion) + "(?:$|\s)")) {
    throw "Pandoc version does not match Profile: expected $($profileConfig.pandocVersion), got '$pandocVersionLine'"
}
if ($crossrefVersionLine -notmatch ("^pandoc-crossref v?" + [regex]::Escape([string]$profileConfig.pandocCrossrefVersion) + "(?:$|\s)")) {
    throw "pandoc-crossref version does not match Profile: expected $($profileConfig.pandocCrossrefVersion), got '$crossrefVersionLine'"
}

$resourceDirectories = New-Object System.Collections.Generic.List[string]
$resourceDirectories.Add($project)
foreach ($source in $resolvedSources) {
    $directory = [System.IO.Path]::GetDirectoryName($source)
    if (-not $resourceDirectories.Contains($directory)) {
        $resourceDirectories.Add($directory)
    }
}
$resourceDirectories.Add($profile)
$resourcePath = $resourceDirectories -join ";"

$arguments = @()
$arguments += $resolvedSources
$arguments += @(
    "--from", "markdown+yaml_metadata_block+tex_math_dollars+raw_tex+link_attributes+implicit_figures+pipe_tables+fenced_divs",
    "--to", "latex",
    "--standalone",
    "--top-level-division=section",
    "--template", $template,
    "--metadata-file", $crossrefMetadata,
    "--lua-filter", $abstractFilter,
    "--lua-filter", $layoutFilter,
    "--filter", $crossref
)
# --csl is a Citeproc option. Passing it alongside --natbib/--biblatex makes
# Pandoc warn that the option is unused, and --fail-if-warnings turns that
# warning into a failed conversion, so the two branches are exclusive.
#
# nodepaper-bib-method tells filters/layout.lua to emit \bibliography /
# \printbibliography under the references heading. It is deliberately not sent
# on the citeproc route, where Citeproc fills the ::: {#refs} div itself and any
# extra metadata would change the generated .tex.
switch ($CiteMethod) {
    "natbib" {
        $arguments += @("--natbib", "--bibliography", $references, "--metadata", "nodepaper-bib-method=natbib")
    }
    "biblatex" {
        $arguments += @("--biblatex", "--bibliography", $references, "--metadata", "nodepaper-bib-method=biblatex")
    }
    default {
        $arguments += @("--citeproc", "--bibliography", $references, "--csl", $csl)
    }
}
# nodepaper-export switches the template onto the compile-time font cascade and
# asks for \tracinglostchars=3 so a dropped glyph stops the recipient's own
# compile instead of leaving a hole in the PDF. Like -CiteMethod above it is
# only added when asked for, so the build keeps the Pandoc command line it has
# always had.
if ($ExportMode) {
    $arguments += @("--metadata", "nodepaper-export=true")
}
$arguments += @(
    "--syntax-highlighting=$highlightStyle",
    "--metadata", "nodepaper-appendix-numbering=$appendixNumbering",
    "--metadata", "nodepaper-appendix-newpage=$appendixNewPageMetadata",
    "--metadata", "nodepaper-linespread=$linespreadMetadata",
    "--metadata", "nodepaper-abstract-linespread=$abstractLinespreadMetadata",
    "--metadata", "nodepaper-mathfont=$mathFont",
    "--metadata", "nodepaper-mathfont-newtx=$mathFontNewtx",
    "--metadata", "nodepaper-font-fallback=$fontFallback",
    "--metadata", "link-citations=true",
    "--fail-if-warnings",
    "--resource-path", $resourcePath,
    "--output", $outputPath
)

Write-Output "CUMCM Profile: $profile"
Write-Output "CUMCM rules version: $($profileConfig.rulesVersion)"
# The default citeproc route keeps its output stream unchanged so build logs
# stay comparable across versions; the export routes announce themselves.
if ($CiteMethod -ne "citeproc") {
    Write-Output "Citation method: $CiteMethod"
    Write-Output "Citation template: $template"
}
Write-Output "Pandoc version: $pandocVersionLine"
Write-Output "pandoc-crossref version: $crossrefVersionLine"
if ($ExportMode) {
    # Naming this machine's fonts would be a lie about the export: the .tex
    # carries the cascade and decides wherever it is compiled.
    Write-Output "Chinese fonts: chosen at compile time by the exported preamble (SimSun with SimHei/KaiTi, SimSun alone, or Noto CJK)"
} elseif ($missingFonts.Count -gt 0) {
    Write-Output "Chinese fonts: $($missingFonts -join ', ') not installed; bold and italic will be synthesised from SimSun"
} else {
    Write-Output "Chinese fonts: SimHei and KaiTi installed"
}
Write-Output "Ordered Sources: $($resolvedSources -join ' | ')"
Write-Output "LaTeX Fragments: $($resolvedFragments -join ' | ')"
Write-Output "Appendix numbering: $appendixNumbering"
Write-Output "Appendix new page: $appendixNewPageMetadata"
Write-Output "Highlight style: $highlightStyle"
Write-Output "Linespread: $linespreadMetadata"
Write-Output "Abstract linespread: $abstractLinespreadMetadata"
Write-Output "Math font route: $mathFont"
Write-Output "Pandoc command: $pandoc $($arguments -join ' ')"
& $pandoc @arguments
if ($LASTEXITCODE -ne 0) {
    throw "Pandoc CUMCM conversion failed with exit code $LASTEXITCODE"
}
if (-not (Test-Path -LiteralPath $outputPath -PathType Leaf)) {
    throw "Pandoc did not generate LaTeX: $outputPath"
}
Write-Output "Generated CUMCM LaTeX: $outputPath"
