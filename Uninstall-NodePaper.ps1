<#
.SYNOPSIS
    Remove the registered NodePaper folder from the current user's Path.

.DESCRIPTION
    Removes the exact Path entry registered by Install-NodePaper.ps1 and clears
    the registration under HKCU\Software\NodePaper. The folder itself is kept:
    NodePaper runs from where it was extracted, so that directory belongs to
    the user rather than to this script.

    Projects, TeX, Pandoc installations and unrelated Path entries are not
    touched, and neither is anything installed by Setup.
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

# Read and write the user Path through the registry. The .NET User-target APIs
# expand %VARIABLE% on read and store REG_SZ on write, either of which silently
# breaks Path entries belonging to other software. See the longer note in
# Install-NodePaper.ps1.
$script:UserEnvironmentKey = 'HKCU:\Environment'

function Get-PathValue {
    param([string]$Scope)
    if ($Scope -eq "Process") { return [Environment]::GetEnvironmentVariable("Path", "Process") }
    return [string](Get-Item -LiteralPath $script:UserEnvironmentKey).GetValue(
        "Path", "", [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
}

function Set-PathValue {
    param([string]$Scope, [string]$Value)
    if ($Scope -eq "Process") {
        [Environment]::SetEnvironmentVariable("Path", $Value, "Process")
    }
    else {
        Set-ItemProperty -LiteralPath $script:UserEnvironmentKey -Name "Path" -Value $Value -Type ExpandString
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

$RegistrationKey = 'HKCU:\Software\NodePaper'
$RegistrationValue = 'PortablePath'

if ([string]::IsNullOrWhiteSpace($InstallRoot)) {
    # Default to whatever Install-NodePaper.ps1 registered, falling back to this
    # folder so the script still works from inside an unregistered copy.
    $registered = ""
    if (Test-Path -LiteralPath $RegistrationKey) {
        $item = Get-ItemProperty -LiteralPath $RegistrationKey -Name $RegistrationValue -ErrorAction SilentlyContinue
        if ($null -ne $item) { $registered = [string]$item.$RegistrationValue }
    }
    $InstallRoot = if ([string]::IsNullOrWhiteSpace($registered)) { $PSScriptRoot } else { $registered }
}
$InstallRoot = [System.IO.Path]::GetFullPath($InstallRoot).TrimEnd('\', '/')
$oldPath = Get-PathValue $PathScope
$oldProcessPath = [Environment]::GetEnvironmentVariable("Path", "Process")
$newPath = Remove-PathEntry $oldPath $InstallRoot

try {
    Set-PathValue $PathScope $newPath
    if ($PathScope -eq "User") {
        [Environment]::SetEnvironmentVariable("Path", (Remove-PathEntry $oldProcessPath $InstallRoot), "Process")
    }
    if (Test-Path -LiteralPath $RegistrationKey) {
        Remove-ItemProperty -LiteralPath $RegistrationKey -Name $RegistrationValue -ErrorAction SilentlyContinue
    }
}
catch {
    try { Set-PathValue $PathScope $oldPath } catch { }
    try { [Environment]::SetEnvironmentVariable("Path", $oldProcessPath, "Process") } catch { }
    throw
}

# The folder is deliberately left alone. It is where the user chose to put the
# release, it may sit alongside their own files, and deleting a directory this
# script does not own is not a decision to make on their behalf.
Write-Host "NodePaper was removed from the user Path: $InstallRoot"
Write-Host "The folder itself was kept; delete it yourself if you no longer need it."
Write-Host "NodePaper Projects and TeX installations were not changed."
