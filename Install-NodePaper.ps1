<#
.SYNOPSIS
    Install the verified NodePaper release payload for the current Windows user.

.DESCRIPTION
    Copies this complete release payload to a user-level installation directory,
    verifies every payload file against payload-manifest.json, and registers the
    installation directory on the user Path. No administrator rights, network
    access, telemetry, or automatic update is used.

    Reopen the terminal after installation, then run: nodepaper

.PARAMETER InstallRoot
    User-level destination. Default: %LOCALAPPDATA%\Programs\NodePaper

.PARAMETER PathScope
    User (default) updates the current user's persistent Path. Process exists
    only for isolated release testing and does not persist beyond the shell.
#>
param(
    [string]$InstallRoot = "",
    [ValidateSet("User", "Process")]
    [string]$PathScope = "User"
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Get-NormalizedPathEntry {
    param([string]$Path)
    return [System.IO.Path]::GetFullPath($Path).TrimEnd('\', '/').ToLowerInvariant()
}

function Get-PathValue {
    param([string]$Scope)
    if ($Scope -eq "Process") { return [Environment]::GetEnvironmentVariable("Path", "Process") }
    return [Environment]::GetEnvironmentVariable("Path", "User")
}

function Set-PathValue {
    param([string]$Scope, [string]$Value)
    if ($Scope -eq "Process") {
        [Environment]::SetEnvironmentVariable("Path", $Value, "Process")
    }
    else {
        [Environment]::SetEnvironmentVariable("Path", $Value, "User")
    }
}

function Add-PathEntry {
    param([string]$Current, [string]$Entry)
    $wanted = Get-NormalizedPathEntry $Entry
    $entries = @($Current -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
    foreach ($existing in $entries) {
        try {
            if ((Get-NormalizedPathEntry $existing.Trim().Trim('"')) -eq $wanted) { return ($entries -join ';') }
        }
        catch { }
    }
    return (@($entries) + $Entry) -join ';'
}

function Assert-Payload {
    param([string]$SourceRoot)
    $manifestPath = Join-Path $SourceRoot "payload-manifest.json"
    if (-not (Test-Path -LiteralPath $manifestPath -PathType Leaf)) {
        throw "payload-manifest.json is missing. Install from a complete NodePaper release ZIP."
    }
    $manifest = Get-Content -LiteralPath $manifestPath -Raw -Encoding UTF8 | ConvertFrom-Json
    if ([int]$manifest.schemaVersion -ne 1 -or [string]$manifest.channel -ne "portable-zip") {
        throw "Unsupported payload manifest."
    }
    $exe = Join-Path $SourceRoot "nodepaper.exe"
    if (-not (Test-Path -LiteralPath $exe -PathType Leaf)) {
        throw "nodepaper.exe is missing from the release payload."
    }
    $reported = ((& $exe --version 2>&1 | Out-String).Trim())
    if ($LASTEXITCODE -ne 0 -or $reported -ne "nodepaper $($manifest.version)") {
        throw "Payload version mismatch: executable reports '$reported', manifest expects 'nodepaper $($manifest.version)'."
    }
    $expectedFiles = New-Object 'System.Collections.Generic.HashSet[string]' ([System.StringComparer]::OrdinalIgnoreCase)
    foreach ($entry in @($manifest.files)) {
        $relative = ([string]$entry.path) -replace '\\', '/'
        if ([string]::IsNullOrWhiteSpace($relative) -or $relative.Contains("..") -or [System.IO.Path]::IsPathRooted($relative)) {
            throw "Unsafe path in payload manifest: $relative"
        }
        if (-not $expectedFiles.Add($relative)) { throw "Duplicate path in payload manifest: $relative" }
        $path = Join-Path $SourceRoot ($relative -replace '/', '\')
        if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
            throw "Payload file is missing: $relative"
        }
        $actual = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($actual -ne ([string]$entry.sha256).ToLowerInvariant()) {
            throw "Payload hash mismatch: $relative"
        }
    }
    $actualFiles = @(Get-ChildItem -LiteralPath $SourceRoot -Recurse -File | ForEach-Object {
        $_.FullName.Substring($SourceRoot.TrimEnd('\').Length).TrimStart('\') -replace '\\', '/'
    } | Where-Object { $_ -ne "payload-manifest.json" })
    foreach ($relative in $actualFiles) {
        if (-not $expectedFiles.Contains($relative)) { throw "Unverified extra file in payload: $relative" }
    }
    if ($actualFiles.Count -ne $expectedFiles.Count) { throw "Payload file list does not match its manifest." }
    return $manifest
}

$sourceRoot = (Resolve-Path -LiteralPath $PSScriptRoot).Path
$manifest = Assert-Payload $sourceRoot
if ([string]::IsNullOrWhiteSpace($InstallRoot)) {
    if ([string]::IsNullOrWhiteSpace($env:LOCALAPPDATA)) { throw "LOCALAPPDATA is unavailable." }
    $InstallRoot = Join-Path $env:LOCALAPPDATA "Programs\NodePaper"
}
$InstallRoot = [System.IO.Path]::GetFullPath($InstallRoot).TrimEnd('\', '/')
$sourceNormalized = Get-NormalizedPathEntry $sourceRoot
$targetNormalized = Get-NormalizedPathEntry $InstallRoot
if ($sourceNormalized -eq $targetNormalized -or $targetNormalized.StartsWith($sourceNormalized + '\')) {
    throw "InstallRoot must be outside the extracted release payload: $InstallRoot"
}

$parent = Split-Path -Parent $InstallRoot
New-Item -ItemType Directory -Force -Path $parent | Out-Null
$stage = Join-Path $parent (".nodepaper-install-" + [Guid]::NewGuid().ToString("N"))
$backup = Join-Path $parent (".nodepaper-backup-" + [Guid]::NewGuid().ToString("N"))
$oldPath = Get-PathValue $PathScope
$oldProcessPath = [Environment]::GetEnvironmentVariable("Path", "Process")
$oldInstallMoved = $false
$newInstallPublished = $false

try {
    New-Item -ItemType Directory -Path $stage | Out-Null
    Get-ChildItem -LiteralPath $sourceRoot -Force | ForEach-Object {
        Copy-Item -LiteralPath $_.FullName -Destination $stage -Recurse -Force
    }
    # Verify the copied bytes before replacing a prior installation.
    [void](Assert-Payload $stage)

    if (Test-Path -LiteralPath $InstallRoot) {
        Move-Item -LiteralPath $InstallRoot -Destination $backup
        $oldInstallMoved = $true
    }
    Move-Item -LiteralPath $stage -Destination $InstallRoot
    $newInstallPublished = $true

    $newPath = Add-PathEntry $oldPath $InstallRoot
    Set-PathValue $PathScope $newPath
    if ($PathScope -eq "User") {
        [Environment]::SetEnvironmentVariable("Path", (Add-PathEntry $oldProcessPath $InstallRoot), "Process")
    }

    if ($oldInstallMoved -and (Test-Path -LiteralPath $backup)) {
        Remove-Item -LiteralPath $backup -Recurse -Force
        $oldInstallMoved = $false
    }
}
catch {
    try { Set-PathValue $PathScope $oldPath } catch { }
    try { [Environment]::SetEnvironmentVariable("Path", $oldProcessPath, "Process") } catch { }
    if ($newInstallPublished -and (Test-Path -LiteralPath $InstallRoot)) {
        Remove-Item -LiteralPath $InstallRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
    if ($oldInstallMoved -and (Test-Path -LiteralPath $backup)) {
        Move-Item -LiteralPath $backup -Destination $InstallRoot -ErrorAction SilentlyContinue
    }
    throw
}
finally {
    if (Test-Path -LiteralPath $stage) { Remove-Item -LiteralPath $stage -Recurse -Force -ErrorAction SilentlyContinue }
    if (Test-Path -LiteralPath $backup) { Remove-Item -LiteralPath $backup -Recurse -Force -ErrorAction SilentlyContinue }
}

Write-Host "NodePaper $($manifest.version) installed for the current user."
Write-Host "Location: $InstallRoot"
Write-Host "Open a new terminal and run: nodepaper"
