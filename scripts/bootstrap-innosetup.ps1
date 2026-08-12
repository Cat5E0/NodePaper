<#
.SYNOPSIS
    Provision the pinned Inno Setup 6 compiler used to build the Windows Setup.

.DESCRIPTION
    Downloads the Inno Setup version pinned in
    installer/windows/innosetup-toolchain.json, verifies its SHA-256, extracts
    it in portable mode into tools/windows-x64/innosetup (outside version
    control) and verifies the compiler files and the pinned Simplified Chinese
    language file.

    Portable extraction is deliberate: the build machine gets no registry
    entries, no PATH change, no shortcuts and no uninstall entry from the build
    toolchain. Inno Setup is never bundled into the NodePaper release payload.

.PARAMETER Force
    Re-download and re-extract even when the pinned files already verify.

.PARAMETER KeepDownloads
    Keep tools/_downloads after a successful bootstrap.
#>
param(
    [switch]$Force,
    [switch]$KeepDownloads
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$metadataPath = Join-Path $root "installer\windows\innosetup-toolchain.json"
if (-not (Test-Path -LiteralPath $metadataPath -PathType Leaf)) {
    throw "Pinned Inno Setup metadata not found: $metadataPath"
}
$metadata = Get-Content -LiteralPath $metadataPath -Raw -Encoding UTF8 | ConvertFrom-Json
if ([int]$metadata.schemaVersion -ne 1) {
    throw "Unsupported innosetup-toolchain.json schemaVersion: $($metadata.schemaVersion)"
}

function Assert-FileSHA256 {
    param([string]$Path, [string]$Expected, [string]$Label)
    if (-not (Test-Path -LiteralPath $Path -PathType Leaf)) {
        throw "$Label is missing: $Path"
    }
    $actual = (Get-FileHash -LiteralPath $Path -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actual -ne ([string]$Expected).ToLowerInvariant()) {
        throw "$Label SHA-256 mismatch for $Path. Expected $Expected, got $actual."
    }
}

function Test-PinnedFiles {
    $ok = $true
    foreach ($entry in @($metadata.compiler.files)) {
        $path = Join-Path $root (([string]$entry.path) -replace '/', '\')
        if (-not (Test-Path -LiteralPath $path -PathType Leaf)) { return $false }
        $actual = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($actual -ne ([string]$entry.sha256).ToLowerInvariant()) { $ok = $false }
    }
    foreach ($entry in @($metadata.languageFiles)) {
        $path = Join-Path $root (([string]$entry.target) -replace '/', '\')
        if (-not (Test-Path -LiteralPath $path -PathType Leaf)) { return $false }
        $actual = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($actual -ne ([string]$entry.sha256).ToLowerInvariant()) { $ok = $false }
    }
    return $ok
}

$installRoot = Join-Path $root (([string]$metadata.compiler.installRoot) -replace '/', '\')
$cacheRoot = Join-Path $root "tools\_downloads"

if (-not $Force -and (Test-PinnedFiles)) {
    Write-Host "Pinned Inno Setup $($metadata.compiler.version) already present: $installRoot"
}
else {
    New-Item -ItemType Directory -Force -Path $cacheRoot | Out-Null
    $installerPath = Join-Path $cacheRoot ("innosetup-" + [string]$metadata.compiler.version + ".exe")
    if ($Force -or -not (Test-Path -LiteralPath $installerPath -PathType Leaf)) {
        Write-Host "Downloading $($metadata.compiler.downloadUrl)"
        Invoke-WebRequest -Uri ([string]$metadata.compiler.downloadUrl) -OutFile $installerPath
    }
    Assert-FileSHA256 $installerPath ([string]$metadata.compiler.downloadSha256) "Inno Setup installer"
    Write-Host "SHA-256 verified: $installerPath"

    if (Test-Path -LiteralPath $installRoot) {
        Remove-Item -LiteralPath $installRoot -Recurse -Force
    }
    New-Item -ItemType Directory -Force -Path $installRoot | Out-Null
    Write-Host "Extracting Inno Setup in portable mode to $installRoot"
    $process = Start-Process -FilePath $installerPath -ArgumentList @(
        "/PORTABLE=1", "/VERYSILENT", "/SUPPRESSMSGBOXES", "/NOICONS", "/DIR=$installRoot"
    ) -Wait -PassThru
    $exitCode = 0
    try { $exitCode = $process.ExitCode } catch { $exitCode = 0 }
    if ($exitCode -ne 0) {
        throw "Portable Inno Setup extraction failed with exit code $exitCode"
    }

    $languageDir = Join-Path $installRoot "Languages"
    New-Item -ItemType Directory -Force -Path $languageDir | Out-Null
    foreach ($entry in @($metadata.languageFiles)) {
        $target = Join-Path $root (([string]$entry.target) -replace '/', '\')
        Write-Host "Downloading $($entry.downloadUrl)"
        Invoke-WebRequest -Uri ([string]$entry.downloadUrl) -OutFile $target
        Assert-FileSHA256 $target ([string]$entry.sha256) "Inno Setup language file $($entry.name)"
    }
}

foreach ($entry in @($metadata.compiler.files)) {
    $path = Join-Path $root (([string]$entry.path) -replace '/', '\')
    Assert-FileSHA256 $path ([string]$entry.sha256) "Pinned Inno Setup file"
    Write-Host "Verified: $($entry.path)"
}
foreach ($entry in @($metadata.languageFiles)) {
    $path = Join-Path $root (([string]$entry.target) -replace '/', '\')
    Assert-FileSHA256 $path ([string]$entry.sha256) "Pinned Inno Setup language file"
    Write-Host "Verified: $($entry.target)"
}

if (-not $KeepDownloads -and (Test-Path -LiteralPath $cacheRoot)) {
    Remove-Item -LiteralPath $cacheRoot -Recurse -Force -ErrorAction SilentlyContinue
}

Write-Host ""
Write-Host "Inno Setup $($metadata.compiler.version) ready: $(Join-Path $root (([string]$metadata.compiler.compilerExecutable) -replace '/', '\'))"
Write-Host "License: $(Join-Path $root (([string]$metadata.compiler.licenseFile) -replace '/', '\'))"
