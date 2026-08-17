param(
    [string]$PandocVersion = "3.9",
    [string]$PandocCrossrefVersion = "0.3.24",
    [string]$PandocArchiveSHA256 = "edccdaa95b5a33b3320187f0e291d58a232e21318a8081750bd31f847e598d18",
    [string]$PandocExecutableSHA256 = "3a3f58602ee7bc1237e314e764df07806ea6bb28c24bb9026cd9d09e4e166020",
    [string]$PandocSourceSHA256 = "d8da16e1ad1f685123fbc1a5a83b74766bcfd939dc6989484822f023bb70438f",
    [string]$PandocCrossrefArchiveSHA256 = "f5be3abd946e44184cc8f2a813210e870767379fbaaa03be56c72a637d5a50e7",
    [string]$PandocCrossrefExecutableSHA256 = "6393ed0b495416bfce08a94f6a40d30791aedcb07067d04eba2920d47a4ec280",
    [string]$PandocCrossrefSourceSHA256 = "ea9e06e5f95dee428d48005a4776bffa4d02c4936097aff269cafe81ec39105b",
    [switch]$Force,
    [switch]$KeepDownloads,
    # Skips the corresponding-source archives. They exist for the release
    # package's licence compliance; a CI job that only needs to run pandoc does
    # not, and every extra download is another chance to be rate limited.
    [switch]$SkipSources
)

$ErrorActionPreference = "Stop"

function Invoke-Download {
    <#
    .SYNOPSIS
    Downloads a file, retrying transient failures.

    .DESCRIPTION
    codeload.github.com answers 429 Too Many Requests when a repository has
    dispatched several workflows in quick succession, which took down an
    otherwise healthy run on 2026-08-17 after three of four downloads had
    already succeeded. A rate limit is not a build failure, so back off and try
    again rather than making the operator re-dispatch and hope.
    #>
    param(
        [string]$Uri,
        [string]$OutFile,
        [int]$Attempts = 4
    )
    for ($attempt = 1; $attempt -le $Attempts; $attempt++) {
        if ($attempt -eq 1) { Write-Host "Downloading $Uri" }
        else { Write-Host "Downloading $Uri (attempt $attempt of $Attempts)" }
        try {
            Invoke-WebRequest -Uri $Uri -OutFile $OutFile
            return
        }
        catch {
            if ($attempt -eq $Attempts) { throw }
            $wait = [Math]::Pow(2, $attempt) * 5
            Write-Host "  failed: $($_.Exception.Message)"
            Write-Host "  retrying in $wait seconds"
            Start-Sleep -Seconds $wait
        }
    }
}

function Assert-FileSHA256 {
    param(
        [string]$Path,
        [string]$Expected
    )
    $actual = (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actual -ne $Expected.ToLowerInvariant()) {
        throw "SHA-256 mismatch for $Path. Expected $Expected, got $actual."
    }
    Write-Host "SHA-256 verified: $Path"
}

function Expand-ArchivePortable {
    param(
        [string]$ArchivePath,
        [string]$Destination
    )

    New-Item -ItemType Directory -Force -Path $Destination | Out-Null
    if ($ArchivePath.EndsWith(".zip", [System.StringComparison]::OrdinalIgnoreCase)) {
        Expand-Archive -LiteralPath $ArchivePath -DestinationPath $Destination -Force
        return
    }

    $tar = Get-Command "tar.exe" -ErrorAction SilentlyContinue
    if (-not $tar) {
        throw "Cannot extract $ArchivePath. Windows tar.exe was not found."
    }
    & $tar.Source -xf $ArchivePath -C $Destination
    if ($LASTEXITCODE -ne 0) {
        throw "Failed to extract $ArchivePath with tar.exe"
    }
}

function Copy-FirstExecutable {
    param(
        [string]$SearchRoot,
        [string]$ExeName,
        [string]$Destination
    )

    $exe = Get-ChildItem -Path $SearchRoot -Recurse -Filter $ExeName -File | Select-Object -First 1
    if (-not $exe) {
        throw "Could not find $ExeName under $SearchRoot"
    }
    New-Item -ItemType Directory -Force -Path ([System.IO.Path]::GetDirectoryName($Destination)) | Out-Null
    Copy-Item -LiteralPath $exe.FullName -Destination $Destination -Force
}

$toolsRoot = Join-Path $PSScriptRoot "tools"
$platformRoot = Join-Path $toolsRoot "windows-x64"
$cacheRoot = Join-Path $toolsRoot "_downloads"
$sourcesRoot = Join-Path $platformRoot "sources"
$pandocTarget = Join-Path $platformRoot "pandoc\pandoc.exe"
$crossrefTarget = Join-Path $platformRoot "pandoc-crossref\pandoc-crossref.exe"
$pandocSourceTarget = Join-Path $sourcesRoot "pandoc-$PandocVersion-source.tar.gz"
$crossrefSourceTarget = Join-Path $sourcesRoot "pandoc-crossref-$PandocCrossrefVersion-source.tar.gz"

New-Item -ItemType Directory -Force -Path $cacheRoot | Out-Null

$pandocFile = "pandoc-$PandocVersion-windows-x86_64.zip"
$pandocUrl = "https://github.com/jgm/pandoc/releases/download/$PandocVersion/$pandocFile"
$crossrefFile = "pandoc-crossref-Windows-X64.7z"
$crossrefTag = "v$PandocCrossrefVersion"
$crossrefUrl = "https://github.com/lierdakil/pandoc-crossref/releases/download/$crossrefTag/$crossrefFile"
$pandocSourceUrl = "https://codeload.github.com/jgm/pandoc/tar.gz/refs/tags/$PandocVersion"
$crossrefSourceUrl = "https://codeload.github.com/lierdakil/pandoc-crossref/tar.gz/refs/tags/$crossrefTag"

if ($Force -or -not (Test-Path -LiteralPath $pandocTarget -PathType Leaf)) {
    $pandocArchive = Join-Path $cacheRoot $pandocFile
    $pandocExtract = Join-Path $cacheRoot "pandoc-$PandocVersion"
    Invoke-Download $pandocUrl $pandocArchive
    Assert-FileSHA256 $pandocArchive $PandocArchiveSHA256
    if (Test-Path -LiteralPath $pandocExtract) {
        Remove-Item -LiteralPath $pandocExtract -Recurse -Force
    }
    Expand-ArchivePortable $pandocArchive $pandocExtract
    Copy-FirstExecutable $pandocExtract "pandoc.exe" $pandocTarget
}
else {
    Write-Host "Pandoc already exists: $pandocTarget"
}
Assert-FileSHA256 $pandocTarget $PandocExecutableSHA256

if ($Force -or -not (Test-Path -LiteralPath $crossrefTarget -PathType Leaf)) {
    $crossrefArchive = Join-Path $cacheRoot $crossrefFile
    $crossrefExtract = Join-Path $cacheRoot "pandoc-crossref-$PandocCrossrefVersion"
    Invoke-Download $crossrefUrl $crossrefArchive
    Assert-FileSHA256 $crossrefArchive $PandocCrossrefArchiveSHA256
    if (Test-Path -LiteralPath $crossrefExtract) {
        Remove-Item -LiteralPath $crossrefExtract -Recurse -Force
    }
    Expand-ArchivePortable $crossrefArchive $crossrefExtract
    Copy-FirstExecutable $crossrefExtract "pandoc-crossref.exe" $crossrefTarget
}
else {
    Write-Host "pandoc-crossref already exists: $crossrefTarget"
}
Assert-FileSHA256 $crossrefTarget $PandocCrossrefExecutableSHA256

New-Item -ItemType Directory -Force -Path $sourcesRoot | Out-Null
if ($SkipSources) {
    # Never on the release path: scripts/build-release.ps1 bundles these and
    # would fail the payload manifest without them.
    Write-Host "Skipping the corresponding-source archives (-SkipSources)."
}
else {
    if ($Force -or -not (Test-Path -LiteralPath $pandocSourceTarget -PathType Leaf)) {
        Invoke-Download $pandocSourceUrl $pandocSourceTarget
    }
    Assert-FileSHA256 $pandocSourceTarget $PandocSourceSHA256
    if ($Force -or -not (Test-Path -LiteralPath $crossrefSourceTarget -PathType Leaf)) {
        Invoke-Download $crossrefSourceUrl $crossrefSourceTarget
    }
    Assert-FileSHA256 $crossrefSourceTarget $PandocCrossrefSourceSHA256
}

$versions = [ordered]@{
    platform = "windows-x64"
    pandoc = [ordered]@{
        version = $PandocVersion
        url = $pandocUrl
        archive_sha256 = $PandocArchiveSHA256
        executable_sha256 = $PandocExecutableSHA256
        target = "tools/windows-x64/pandoc/pandoc.exe"
        source_url = $pandocSourceUrl
        source_sha256 = $PandocSourceSHA256
        source_target = "tools/windows-x64/sources/pandoc-$PandocVersion-source.tar.gz"
    }
    pandoc_crossref = [ordered]@{
        version = $PandocCrossrefVersion
        url = $crossrefUrl
        archive_sha256 = $PandocCrossrefArchiveSHA256
        executable_sha256 = $PandocCrossrefExecutableSHA256
        target = "tools/windows-x64/pandoc-crossref/pandoc-crossref.exe"
        source_url = $crossrefSourceUrl
        source_sha256 = $PandocCrossrefSourceSHA256
        source_target = "tools/windows-x64/sources/pandoc-crossref-$PandocCrossrefVersion-source.tar.gz"
    }
}

$versionsPath = Join-Path $toolsRoot "versions.json"
$versionsJSON = ($versions | ConvertTo-Json -Depth 5) -replace "`r`n", "`n"
[System.IO.File]::WriteAllText($versionsPath, $versionsJSON + "`n", (New-Object System.Text.UTF8Encoding($false)))

if (-not $KeepDownloads -and (Test-Path -LiteralPath $cacheRoot)) {
    Remove-Item -LiteralPath $cacheRoot -Recurse -Force
}

Write-Host "Tool bootstrap complete."
