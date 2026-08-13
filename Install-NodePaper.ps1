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

# Keep the window alive after install when the script runs in a console that
# Windows opened only for it (for example right-click "Run with PowerShell");
# piped, redirected, non-console and CI invocations must never block.
function Test-InteractivePause {
    if ($env:CI) { return $false }
    if ($null -eq $Host -or $Host.Name -ne "ConsoleHost") { return $false }
    try {
        if ([Console]::IsOutputRedirected) { return $false }
    }
    catch { return $false }
    return $true
}

function Wait-CloseWindow {
    if (-not (Test-InteractivePause)) { return }
    Write-Host ""
    Write-Host "Press Enter to close this window."
    Read-Host | Out-Null
}

function Get-NormalizedPathEntry {
    param([string]$Path)
    return [System.IO.Path]::GetFullPath($Path).TrimEnd('\', '/').ToLowerInvariant()
}

# Compare NodePaper versions like the Setup does: major.minor.patch numerically,
# then the -rc.N prerelease suffix ordinally. Returns -1 (Left < Right), 0 or 1.
function Compare-NodePaperVersion {
    param([string]$Left, [string]$Right)
    $leftMain = $Left; $leftPre = ""
    $dash = $leftMain.IndexOf('-')
    if ($dash -gt 0) { $leftPre = $leftMain.Substring($dash + 1); $leftMain = $leftMain.Substring(0, $dash) }
    $rightMain = $Right; $rightPre = ""
    $dash = $rightMain.IndexOf('-')
    if ($dash -gt 0) { $rightPre = $rightMain.Substring($dash + 1); $rightMain = $rightMain.Substring(0, $dash) }
    $leftNumbers = @($leftMain.Split('.') | ForEach-Object { $value = 0; [int]::TryParse($_, [ref]$value) | Out-Null; $value })
    $rightNumbers = @($rightMain.Split('.') | ForEach-Object { $value = 0; [int]::TryParse($_, [ref]$value) | Out-Null; $value })
    for ($index = 0; $index -lt 3; $index++) {
        $leftValue = if ($index -lt $leftNumbers.Count) { $leftNumbers[$index] } else { 0 }
        $rightValue = if ($index -lt $rightNumbers.Count) { $rightNumbers[$index] } else { 0 }
        if ($leftValue -lt $rightValue) { return -1 }
        if ($leftValue -gt $rightValue) { return 1 }
    }
    if ($leftPre -eq $rightPre) { return 0 }
    if ($leftPre -eq "") { return 1 }
    if ($rightPre -eq "") { return -1 }
    return [Math]::Sign([string]::CompareOrdinal($leftPre, $rightPre))
}

# Ask for confirmation only when this script runs in a console that Windows
# opened only for it; piped, redirected, non-console and CI invocations never
# prompt and follow the safe default per case (see caller).
function Confirm-Installation {
    param([string]$Message)
    if (-not (Test-InteractivePause)) { return $true }
    Write-Host $Message
    Write-Host "Continue? [Y/n]: " -NoNewline
    $answer = Read-Host
    return ($answer -eq "" -or $answer -match '^[Yy]')
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

# Detect an existing installation and compare versions before touching it.
# Upgrade and repair continue; downgrade must be confirmed interactively and is
# rejected in non-interactive runs, matching the Setup's silent-mode default.
$installedVersion = ""
$installedManifestPath = Join-Path $InstallRoot "payload-manifest.json"
if (Test-Path -LiteralPath $installedManifestPath -PathType Leaf) {
    try {
        $installedManifest = Get-Content -LiteralPath $installedManifestPath -Raw -Encoding UTF8 | ConvertFrom-Json
        $installedVersion = [string]$installedManifest.version
    }
    catch { $installedVersion = "" }
}
if ($installedVersion -ne "") {
    $versionComparison = Compare-NodePaperVersion ([string]$manifest.version) $installedVersion
    if ($versionComparison -lt 0) {
        # Downgrade: confirm interactively; reject silently otherwise.
        if (-not (Test-InteractivePause)) {
            throw "A newer NodePaper ($installedVersion) is already installed and this package is the older $($manifest.version). Downgrade requires confirmation; run this script interactively or uninstall the newer version first."
        }
        if (-not (Confirm-Installation "NodePaper $installedVersion is already installed. This package is the older $($manifest.version); continuing will downgrade to $($manifest.version).")) {
            throw "Installation cancelled by the user; the existing installation was not changed."
        }
        Write-Host "Downgrading NodePaper from $installedVersion to $($manifest.version)."
    }
    elseif ($versionComparison -gt 0) {
        if (-not (Confirm-Installation "NodePaper $installedVersion is already installed. This package will upgrade it to $($manifest.version).")) {
            throw "Installation cancelled by the user; the existing installation was not changed."
        }
        Write-Host "Upgrading NodePaper from $installedVersion to $($manifest.version)."
    }
    else {
        if (-not (Confirm-Installation "NodePaper $($manifest.version) is already installed. Continuing performs a repair install.")) {
            throw "Installation cancelled by the user; the existing installation was not changed."
        }
        Write-Host "Repairing NodePaper $($manifest.version)."
    }
}

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
    Wait-CloseWindow
}

Write-Host "NodePaper $($manifest.version) installed for the current user."
Write-Host "Location: $InstallRoot"
Write-Host "Open a new terminal and run: nodepaper"
