<#
.SYNOPSIS
    Remove the user-level NodePaper installation and its exact Path entry.

.DESCRIPTION
    Removes only the selected NodePaper installation directory and the matching
    user Path entry. It does not modify Projects, TeX, Pandoc installations, or
    unrelated Path entries.
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

function Remove-PathEntry {
    param([string]$Current, [string]$Entry)
    $wanted = Get-NormalizedPathEntry $Entry
    $kept = @()
    foreach ($candidate in @($Current -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })) {
        $matches = $false
        try { $matches = (Get-NormalizedPathEntry $candidate.Trim().Trim('"')) -eq $wanted } catch { }
        if (-not $matches) { $kept += $candidate }
    }
    return $kept -join ';'
}

if ([string]::IsNullOrWhiteSpace($InstallRoot)) {
    if ([string]::IsNullOrWhiteSpace($env:LOCALAPPDATA)) { throw "LOCALAPPDATA is unavailable." }
    $InstallRoot = Join-Path $env:LOCALAPPDATA "Programs\NodePaper"
}
$InstallRoot = [System.IO.Path]::GetFullPath($InstallRoot).TrimEnd('\', '/')
$oldPath = Get-PathValue $PathScope
$oldProcessPath = [Environment]::GetEnvironmentVariable("Path", "Process")
$newPath = Remove-PathEntry $oldPath $InstallRoot

# Rename first so a failure to update Path can restore the installation.
$tombstone = Join-Path (Split-Path -Parent $InstallRoot) (".nodepaper-uninstall-" + [Guid]::NewGuid().ToString("N"))
$moved = $false
try {
    if (Test-Path -LiteralPath $InstallRoot) {
        Move-Item -LiteralPath $InstallRoot -Destination $tombstone
        $moved = $true
    }
    Set-PathValue $PathScope $newPath
    if ($PathScope -eq "User") {
        [Environment]::SetEnvironmentVariable("Path", (Remove-PathEntry $oldProcessPath $InstallRoot), "Process")
    }
}
catch {
    if ($moved -and (Test-Path -LiteralPath $tombstone) -and -not (Test-Path -LiteralPath $InstallRoot)) {
        Move-Item -LiteralPath $tombstone -Destination $InstallRoot -ErrorAction SilentlyContinue
    }
    try { Set-PathValue $PathScope $oldPath } catch { }
    try { [Environment]::SetEnvironmentVariable("Path", $oldProcessPath, "Process") } catch { }
    throw
}

if ($moved -and (Test-Path -LiteralPath $tombstone)) {
    Remove-Item -LiteralPath $tombstone -Recurse -Force
}
Write-Host "NodePaper was removed from: $InstallRoot"
Write-Host "NodePaper Projects and TeX installations were not changed."
