<#
.SYNOPSIS
    Build independent NodePaper real-world regression corpus ZIPs.
.DESCRIPTION
    Checks each public project allowlist and private-data patterns, writes a
    per-file manifest, then creates one deterministic ZIP per project. Program
    release packages never call this script and never contain tests/corpus.
#>
param(
    [Parameter(Mandatory = $true)] [string]$Version,
    [string]$OutputDirectory = "",
    [string]$Commit = "HEAD",
    [switch]$FeatureFreeze,
    [switch]$NoReleaseBlockers
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "version-lifecycle.ps1")
$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$corpusRoot = Join-Path $root "nodepaper\core\tests\corpus"
$resolvedCommit = (& git -C $root rev-parse --verify "$Commit^{commit}" 2>$null | Out-String).Trim().ToLowerInvariant()
if ($LASTEXITCODE -ne 0 -or $resolvedCommit -notmatch '^[0-9a-f]{40}$') { throw "Cannot resolve corpus source commit: $Commit" }
$headCommit = (& git -C $root rev-parse HEAD | Out-String).Trim().ToLowerInvariant()
if ($resolvedCommit -ne $headCommit) {
    throw "Corpus packaging requires the repository to be checked out at source commit $resolvedCommit; current HEAD is $headCommit."
}
$corpusChanges = @(& git -C $root status --porcelain --untracked-files=all -- tests/corpus)
if ($corpusChanges.Count -gt 0) {
    throw "Corpus packaging refuses uncommitted tests/corpus content; commit or restore it before building a fixed-commit package."
}
$buildIdentity = Assert-NodePaperBuildIdentity -Version $Version -ResolvedCommit $resolvedCommit -RepositoryRoot $root -FeatureFreeze:$FeatureFreeze -NoReleaseBlockers:$NoReleaseBlockers
$canonicalBuildTime = Get-NodePaperCanonicalBuildTime -RepositoryRoot $root -ResolvedCommit $resolvedCommit
if ([string]::IsNullOrWhiteSpace($OutputDirectory)) { $OutputDirectory = Join-Path $root "build\test-corpus" }
New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
$OutputDirectory = (Resolve-Path -LiteralPath $OutputDirectory).Path

$projects = @(
    @{
        Name = "A163"
        Sources = @(
            "sections/01-frontmatter-abstract.md", "sections/02-problem-background.md",
            "sections/03-analysis.md", "sections/04-preparation.md",
            "sections/05-model-and-solution.md", "sections/06-evaluation.md",
            "sections/07-references.md", "sections/08-appendix.md"
        )
        Tables = @(
            "a163-4-2-position.tex", "a163-4-3-speed.tex",
            "a163-4-6-position.tex", "a163-4-7-speed.tex"
        )
    },
    @{
        Name = "C063"
        Sources = @("paper.md")
        Tables = @("c063-013-unit-price.tex", "c063-015-revenue.tex")
    }
)
$textExtensions = @(".md", ".yaml", ".yml", ".bib", ".tex", ".json")
$forbiddenPatterns = @(
    '(?im)(?:^|[\s`''"])[A-Z]:\\(?:Users|OneDrive|NodePaper)\\',
    '(?im)/(?:Users|home)/[^/\s]+/', '(?im)\\Users\\[^\\\s]+\\',
    '(?im)OneDrive[\\/]Project[\\/]NodePaper',
    '(?im)^\s*(?:author|authors|institution|school|university|email|e-mail|phone|mobile)\s*:',
    '(?im)\b1[3-9][0-9]{9}\b', '(?im)\b[A-Z0-9._%+-]+@[A-Z0-9.-]+\.[A-Z]{2,}\b'
)
Add-Type -AssemblyName System.IO.Compression
Add-Type -AssemblyName System.IO.Compression.FileSystem

foreach ($project in $projects) {
    $projectRoot = Join-Path $corpusRoot ("real-world\" + $project.Name)
    if (-not (Test-Path -LiteralPath $projectRoot -PathType Container)) { throw "Corpus project missing: $projectRoot" }
    foreach ($required in @("CORPUS.json", "nodepaper.yaml", "references.bib") + $project.Sources) {
        if (-not (Test-Path -LiteralPath (Join-Path $projectRoot $required) -PathType Leaf)) {
            throw "$($project.Name) is missing required file: $required"
        }
    }

    $sourceText = (($project.Sources | ForEach-Object {
        Get-Content -LiteralPath (Join-Path $projectRoot $_) -Raw -Encoding UTF8
    }) -join "`n")
    $referencedImages = @([regex]::Matches($sourceText, 'images/([0-9a-f]{64}\.jpg)') |
        ForEach-Object { $_.Groups[1].Value } | Sort-Object -Unique)
    $actualImages = @(Get-ChildItem -LiteralPath (Join-Path $projectRoot "images") -File |
        Select-Object -ExpandProperty Name | Sort-Object)
    if (($referencedImages -join "`n") -ne ($actualImages -join "`n")) {
        throw "$($project.Name) images must exactly match the Markdown references"
    }

    $allowed = New-Object System.Collections.Generic.HashSet[string]([System.StringComparer]::OrdinalIgnoreCase)
    foreach ($name in @("CORPUS.json", "nodepaper.yaml", "references.bib") + $project.Sources) { [void]$allowed.Add($name) }
    foreach ($name in $referencedImages) { [void]$allowed.Add("images/$name") }
    foreach ($name in $project.Tables) { [void]$allowed.Add("tables/$name") }

    $packageFiles = New-Object System.Collections.Generic.List[object]
    foreach ($file in (Get-ChildItem -LiteralPath $projectRoot -Recurse -File)) {
        $relative = $file.FullName.Substring($projectRoot.Length).TrimStart('\') -replace '\\', '/'
        if ($relative.StartsWith(".nodepaper/", [System.StringComparison]::OrdinalIgnoreCase) -or
            $relative.StartsWith("dist/", [System.StringComparison]::OrdinalIgnoreCase)) { continue }
        if (-not $allowed.Contains($relative)) { throw "$($project.Name) contains a file outside its allowlist: $relative" }
        if ($textExtensions -contains $file.Extension.ToLowerInvariant()) {
            $text = Get-Content -LiteralPath $file.FullName -Raw -Encoding UTF8
            foreach ($pattern in $forbiddenPatterns) {
                if ($text -match $pattern) { throw "$($project.Name)/$relative contains forbidden private data matching: $pattern" }
            }
        }
        $packageFiles.Add([pscustomobject]@{
            Source = $file.FullName; Path = $relative; Bytes = $file.Length
            Sha256 = (Get-FileHash -LiteralPath $file.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
        })
    }
    $packageFiles = @($packageFiles | Sort-Object Path)

    $packageName = "nodepaper-$($buildIdentity.AssetVersion)-test-corpus-$($project.Name)"
    $workRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("nodepaper-test-corpus-" + [Guid]::NewGuid().ToString("N"))
    $stageRoot = Join-Path $workRoot $packageName
    $zipPath = Join-Path $OutputDirectory "$packageName.zip"
    try {
        New-Item -ItemType Directory -Force -Path $stageRoot | Out-Null
        foreach ($file in $packageFiles) {
            $target = Join-Path $stageRoot ($file.Path -replace '/', '\')
            New-Item -ItemType Directory -Force -Path (Split-Path -Parent $target) | Out-Null
            Copy-Item -LiteralPath $file.Source -Destination $target -Force
        }
        $manifest = [ordered]@{
            schemaVersion = 1; version = $Version; stage = $buildIdentity.Stage
            sourceCommit = $resolvedCommit; package = "$packageName.zip"
            purpose = "known real-world regression only; not a general compatibility claim"
            project = $project.Name
            files = @($packageFiles | ForEach-Object { [ordered]@{ path = $_.Path; bytes = $_.Bytes; sha256 = $_.Sha256 } })
        }
        [System.IO.File]::WriteAllText((Join-Path $stageRoot "corpus-manifest.json"),
            (($manifest | ConvertTo-Json -Depth 6) + "`n"), (New-Object System.Text.UTF8Encoding($false)))
        $buildInfo = [ordered]@{
            schemaVersion = 1
            version = $Version
            stage = $buildIdentity.Stage
            sourceCommit = $resolvedCommit
            gitTag = $buildIdentity.GitTag
            builtAtUTC = $canonicalBuildTime
            toolchain = [ordered]@{ powershell = [string]$PSVersionTable.PSVersion }
            payloadSHA256 = Get-NodePaperPayloadSHA256 $stageRoot
        }
        [System.IO.File]::WriteAllText((Join-Path $stageRoot "build-info.json"),
            (($buildInfo | ConvertTo-Json -Depth 5) + "`n"), (New-Object System.Text.UTF8Encoding($false)))

        if (Test-Path -LiteralPath $zipPath) { Remove-Item -LiteralPath $zipPath -Force }
        $zip = [System.IO.Compression.ZipFile]::Open($zipPath, [System.IO.Compression.ZipArchiveMode]::Create)
        try {
            foreach ($file in (Get-ChildItem -LiteralPath $stageRoot -Recurse -File | Sort-Object FullName)) {
                $relative = $file.FullName.Substring($workRoot.Length).TrimStart('\') -replace '\\', '/'
                $entry = $zip.CreateEntry($relative, [System.IO.Compression.CompressionLevel]::NoCompression)
                $entry.LastWriteTime = [DateTimeOffset]::new(2000, 1, 1, 0, 0, 0, [TimeSpan]::Zero)
                $input = [System.IO.File]::OpenRead($file.FullName); $output = $entry.Open()
                try { $input.CopyTo($output) } finally { $output.Dispose(); $input.Dispose() }
            }
        } finally { $zip.Dispose() }

        $zipHash = (Get-FileHash -LiteralPath $zipPath -Algorithm SHA256).Hash.ToLowerInvariant()
        [System.IO.File]::WriteAllText("$zipPath.sha256", "$zipHash  $([System.IO.Path]::GetFileName($zipPath))`n",
            (New-Object System.Text.UTF8Encoding($false)))
        Write-Host "Corpus ZIP ($($project.Name)): $zipPath"
        Write-Host "SHA-256: $zipHash"
    } finally {
        if (Test-Path -LiteralPath $workRoot) { Remove-Item -LiteralPath $workRoot -Recurse -Force }
    }
}
