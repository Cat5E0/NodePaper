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

$template = Join-Path $profile ([string]$profileConfig.template)
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
    "--filter", $crossref,
    "--citeproc",
    "--bibliography", $references,
    "--csl", $csl,
    "--syntax-highlighting=$highlightStyle",
    "--metadata", "nodepaper-appendix-numbering=$appendixNumbering",
    "--metadata", "nodepaper-linespread=$linespreadMetadata",
    "--metadata", "nodepaper-abstract-linespread=$abstractLinespreadMetadata",
    "--metadata", "nodepaper-mathfont=$mathFont",
    "--metadata", "nodepaper-mathfont-newtx=$mathFontNewtx",
    "--metadata", "link-citations=true",
    "--fail-if-warnings",
    "--resource-path", $resourcePath,
    "--output", $outputPath
)

Write-Output "CUMCM Profile: $profile"
Write-Output "CUMCM rules version: $($profileConfig.rulesVersion)"
Write-Output "Pandoc version: $pandocVersionLine"
Write-Output "pandoc-crossref version: $crossrefVersionLine"
Write-Output "Ordered Sources: $($resolvedSources -join ' | ')"
Write-Output "LaTeX Fragments: $($resolvedFragments -join ' | ')"
Write-Output "Appendix numbering: $appendixNumbering"
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
