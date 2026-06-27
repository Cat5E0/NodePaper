param(
    [string]$PandocVersion = "3.9",
    [string]$PandocCrossrefVersion = "0.3.24",
    [switch]$Force,
    [switch]$KeepDownloads
)

$ErrorActionPreference = "Stop"

function Invoke-Download {
    param(
        [string]$Uri,
        [string]$OutFile
    )
    Write-Host "Downloading $Uri"
    Invoke-WebRequest -Uri $Uri -OutFile $OutFile
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
$pandocTarget = Join-Path $platformRoot "pandoc\pandoc.exe"
$crossrefTarget = Join-Path $platformRoot "pandoc-crossref\pandoc-crossref.exe"

New-Item -ItemType Directory -Force -Path $cacheRoot | Out-Null

$pandocFile = "pandoc-$PandocVersion-windows-x86_64.zip"
$pandocUrl = "https://github.com/jgm/pandoc/releases/download/$PandocVersion/$pandocFile"
$crossrefFile = "pandoc-crossref-Windows-X64.7z"
$crossrefTag = "v$PandocCrossrefVersion"
$crossrefUrl = "https://github.com/lierdakil/pandoc-crossref/releases/download/$crossrefTag/$crossrefFile"

if ($Force -or -not (Test-Path -LiteralPath $pandocTarget -PathType Leaf)) {
    $pandocArchive = Join-Path $cacheRoot $pandocFile
    $pandocExtract = Join-Path $cacheRoot "pandoc-$PandocVersion"
    Invoke-Download $pandocUrl $pandocArchive
    if (Test-Path -LiteralPath $pandocExtract) {
        Remove-Item -LiteralPath $pandocExtract -Recurse -Force
    }
    Expand-ArchivePortable $pandocArchive $pandocExtract
    Copy-FirstExecutable $pandocExtract "pandoc.exe" $pandocTarget
}
else {
    Write-Host "Pandoc already exists: $pandocTarget"
}

if ($Force -or -not (Test-Path -LiteralPath $crossrefTarget -PathType Leaf)) {
    $crossrefArchive = Join-Path $cacheRoot $crossrefFile
    $crossrefExtract = Join-Path $cacheRoot "pandoc-crossref-$PandocCrossrefVersion"
    Invoke-Download $crossrefUrl $crossrefArchive
    if (Test-Path -LiteralPath $crossrefExtract) {
        Remove-Item -LiteralPath $crossrefExtract -Recurse -Force
    }
    Expand-ArchivePortable $crossrefArchive $crossrefExtract
    Copy-FirstExecutable $crossrefExtract "pandoc-crossref.exe" $crossrefTarget
}
else {
    Write-Host "pandoc-crossref already exists: $crossrefTarget"
}

$versions = [ordered]@{
    platform = "windows-x64"
    pandoc = [ordered]@{
        version = $PandocVersion
        url = $pandocUrl
        target = "tools/windows-x64/pandoc/pandoc.exe"
    }
    pandoc_crossref = [ordered]@{
        version = $PandocCrossrefVersion
        url = $crossrefUrl
        target = "tools/windows-x64/pandoc-crossref/pandoc-crossref.exe"
    }
}

$versionsPath = Join-Path $toolsRoot "versions.json"
$versions | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $versionsPath -Encoding UTF8

if (-not $KeepDownloads -and (Test-Path -LiteralPath $cacheRoot)) {
    Remove-Item -LiteralPath $cacheRoot -Recurse -Force
}

Write-Host "Tool bootstrap complete."
