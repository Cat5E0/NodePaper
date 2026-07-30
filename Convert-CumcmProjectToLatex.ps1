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
$csl = Join-Path $profile ([string]$profileConfig.csl)
$warningAllowlist = Join-Path $profile ([string]$profileConfig.warningAllowlist)
foreach ($resource in @($template, $crossrefMetadata, $abstractFilter, $csl, $warningAllowlist)) {
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

$references = Assert-FileUnderRoot (Join-Path $project "references.bib") $project "Bibliography"
New-Item -ItemType Directory -Force -Path $buildDir | Out-Null
New-Item -ItemType Directory -Force -Path ([System.IO.Path]::GetDirectoryName($outputPath)) | Out-Null

$pandoc = Find-Executable "tools\windows-x64\pandoc\pandoc.exe" "pandoc.exe" -AllowSystem:$AllowSystemPandoc
$crossref = Find-Executable "tools\windows-x64\pandoc-crossref\pandoc-crossref.exe" "pandoc-crossref.exe" -AllowSystem:$AllowSystemPandoc

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
    "--filter", $crossref,
    "--citeproc",
    "--bibliography", $references,
    "--csl", $csl,
    "--syntax-highlighting=none",
    "--fail-if-warnings",
    "--resource-path", $resourcePath,
    "--output", $outputPath
)

Write-Output "CUMCM Profile: $profile"
Write-Output "CUMCM rules version: $($profileConfig.rulesVersion)"
Write-Output "Ordered Sources: $($resolvedSources -join ' | ')"
Write-Output "Pandoc command: $pandoc $($arguments -join ' ')"
& $pandoc @arguments
if ($LASTEXITCODE -ne 0) {
    throw "Pandoc CUMCM conversion failed with exit code $LASTEXITCODE"
}
if (-not (Test-Path -LiteralPath $outputPath -PathType Leaf)) {
    throw "Pandoc did not generate LaTeX: $outputPath"
}
Write-Output "Generated CUMCM LaTeX: $outputPath"
