<#
.SYNOPSIS
    Build NodePaper's independently distributed real-world regression corpus.

.DESCRIPTION
    Validates the A163/C063 public allowlists, scans text for private paths and
    personal contact data, writes a per-file SHA-256 manifest, and creates a
    deterministic uncompressed ZIP. The program ZIP and Setup do not call this
    script and do not contain tests/corpus.
#>
param(
    [Parameter(Mandatory = $true)]
    [string]$Version,
    [string]$OutputDirectory = ""
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$corpusRoot = Join-Path $root "tests\corpus"
if ($Version -notmatch '^[A-Za-z0-9][A-Za-z0-9._+-]*$') {
    throw "Invalid corpus version: $Version"
}
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path $root "build\test-corpus"
}
New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
$OutputDirectory = (Resolve-Path -LiteralPath $OutputDirectory).Path

$projects = @(
    @{ Name = "A163"; Tables = @() },
    @{ Name = "C063"; Tables = @("c063-013-unit-price.tex", "c063-015-revenue.tex") }
)
$textExtensions = @(".md", ".yaml", ".yml", ".bib", ".tex", ".json")
$forbiddenPathPatterns = @(
    '(?im)(?:^|[\s`''"])[A-Z]:\\(?:Users|OneDrive|NodePaper)\\',
    '(?im)/(?:Users|home)/[^/\s]+/',
    '(?im)\\Users\\[^\\\s]+\\',
    '(?im)OneDrive[\\/]Project[\\/]NodePaper'
)
$forbiddenContactPatterns = @(
    '(?im)^\s*(?:author|authors|institution|school|university|email|e-mail|phone|mobile)\s*:',
    '(?im)\b1[3-9][0-9]{9}\b',
    '(?im)\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b'
)

$packageFiles = New-Object System.Collections.Generic.List[object]
foreach ($project in $projects) {
    $projectRoot = Join-Path $corpusRoot ("real-world\" + $project.Name)
    if (-not (Test-Path -LiteralPath $projectRoot -PathType Container)) {
        throw "Corpus project missing: $projectRoot"
    }
    foreach ($required in @("README.md", "CORPUS.json", "nodepaper.yaml", "paper.md", "references.bib")) {
        if (-not (Test-Path -LiteralPath (Join-Path $projectRoot $required) -PathType Leaf)) {
            throw "$($project.Name) is missing required file: $required"
        }
    }

    $paper = Get-Content -LiteralPath (Join-Path $projectRoot "paper.md") -Raw -Encoding UTF8
    $referencedImages = @([regex]::Matches($paper, 'images/([0-9a-f]{64}\.jpg)') |
        ForEach-Object { $_.Groups[1].Value } | Sort-Object -Unique)
    $actualImages = @(Get-ChildItem -LiteralPath (Join-Path $projectRoot "images") -File |
        Select-Object -ExpandProperty Name | Sort-Object)
    if (($referencedImages -join "`n") -ne ($actualImages -join "`n")) {
        throw "$($project.Name) images must exactly match the references in paper.md"
    }

    $allowed = New-Object System.Collections.Generic.HashSet[string]([System.StringComparer]::OrdinalIgnoreCase)
    foreach ($name in @("README.md", "CORPUS.json", "nodepaper.yaml", "paper.md", "references.bib")) {
        [void]$allowed.Add($name)
    }
    foreach ($name in $referencedImages) { [void]$allowed.Add("images/$name") }
    foreach ($name in $project.Tables) { [void]$allowed.Add("tables/$name") }

    foreach ($file in (Get-ChildItem -LiteralPath $projectRoot -Recurse -File)) {
        $relative = $file.FullName.Substring($projectRoot.Length).TrimStart('\') -replace '\\', '/'
        if (-not $allowed.Contains($relative)) {
            throw "$($project.Name) contains a file outside its allowlist: $relative"
        }
        if ($textExtensions -contains $file.Extension.ToLowerInvariant()) {
            $text = Get-Content -LiteralPath $file.FullName -Raw -Encoding UTF8
            foreach ($pattern in ($forbiddenPathPatterns + $forbiddenContactPatterns)) {
                if ($text -match $pattern) {
                    throw "$($project.Name)/$relative contains forbidden private data matching: $pattern"
                }
            }
        }
        $archivePath = "real-world/$($project.Name)/$relative"
        $packageFiles.Add([pscustomobject]@{
            Source = $file.FullName
            Path = $archivePath
            Bytes = $file.Length
            Sha256 = (Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
        })
    }
}

$readmePath = Join-Path $corpusRoot "README.md"
$packageFiles.Add([pscustomobject]@{
    Source = $readmePath
    Path = "README.md"
    Bytes = (Get-Item -LiteralPath $readmePath).Length
    Sha256 = (Get-FileHash -LiteralPath $readmePath -Algorithm SHA256).Hash.ToLowerInvariant()
})
$packageFiles = @($packageFiles | Sort-Object Path)

$packageName = "nodepaper-$Version-test-corpus"
$workRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("nodepaper-test-corpus-" + [Guid]::NewGuid().ToString("N"))
$stageRoot = Join-Path $workRoot $packageName
$zipPath = Join-Path $OutputDirectory "$packageName.zip"
$shaPath = "$zipPath.sha256"

try {
    New-Item -ItemType Directory -Force -Path $stageRoot | Out-Null
    foreach ($file in $packageFiles) {
        $target = Join-Path $stageRoot ($file.Path -replace '/', '\')
        New-Item -ItemType Directory -Force -Path (Split-Path -Parent $target) | Out-Null
        Copy-Item -LiteralPath $file.Source -Destination $target -Force
    }

    $manifest = [ordered]@{
        schemaVersion = 1
        version = $Version
        package = "$packageName.zip"
        purpose = "known real-world regression only; not a general compatibility claim"
        projects = @("A163", "C063")
        files = @($packageFiles | ForEach-Object {
            [ordered]@{ path = $_.Path; bytes = $_.Bytes; sha256 = $_.Sha256 }
        })
    }
    $manifestPath = Join-Path $stageRoot "corpus-manifest.json"
    [System.IO.File]::WriteAllText(
        $manifestPath,
        (($manifest | ConvertTo-Json -Depth 6) + "`n"),
        (New-Object System.Text.UTF8Encoding($false))
    )

    if (Test-Path -LiteralPath $zipPath) { Remove-Item -LiteralPath $zipPath -Force }
    Add-Type -AssemblyName System.IO.Compression
    Add-Type -AssemblyName System.IO.Compression.FileSystem
    $zip = [System.IO.Compression.ZipFile]::Open($zipPath, [System.IO.Compression.ZipArchiveMode]::Create)
    try {
        foreach ($file in (Get-ChildItem -LiteralPath $stageRoot -Recurse -File | Sort-Object FullName)) {
            $relative = $file.FullName.Substring($workRoot.Length).TrimStart('\') -replace '\\', '/'
            $entry = $zip.CreateEntry($relative, [System.IO.Compression.CompressionLevel]::NoCompression)
            $entry.LastWriteTime = [DateTimeOffset]::new(2000, 1, 1, 0, 0, 0, [TimeSpan]::Zero)
            $input = [System.IO.File]::OpenRead($file.FullName)
            $output = $entry.Open()
            try { $input.CopyTo($output) } finally { $output.Dispose(); $input.Dispose() }
        }
    }
    finally {
        $zip.Dispose()
    }

    $zipHash = (Get-FileHash -LiteralPath $zipPath -Algorithm SHA256).Hash.ToLowerInvariant()
    [System.IO.File]::WriteAllText(
        $shaPath,
        "$zipHash  $([System.IO.Path]::GetFileName($zipPath))`n",
        (New-Object System.Text.UTF8Encoding($false))
    )
    Write-Host "Corpus ZIP: $zipPath"
    Write-Host "SHA-256:   $zipHash"
}
finally {
    if (Test-Path -LiteralPath $workRoot) {
        Remove-Item -LiteralPath $workRoot -Recurse -Force
    }
}
