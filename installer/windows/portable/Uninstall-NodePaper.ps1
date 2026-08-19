<#
.SYNOPSIS
    Take this portable NodePaper folder off the current user's Path.

.DESCRIPTION
    Removes this folder's Path entry, the one Install-NodePaper.ps1 added when
    it was run here. The folder itself is kept: NodePaper runs from where it was
    extracted, so that directory belongs to the user rather than to this script.

    Only a portable release is touched, and the folder shows that for itself: it
    holds a nodepaper.exe and none of Setup's marks. An installation created by
    Setup has its own uninstaller and is refused here, and a folder that is no
    portable release at all is reported rather than acted on.

    Projects, TeX, Pandoc installations and unrelated Path entries are not
    touched, and neither is anything installed by Setup.

.PARAMETER InstallRoot
    Folder to unregister instead of the one this script sits in. Normally
    omitted. It goes through the same two checks as this folder does: a Setup
    installation is refused, and a folder holding no nodepaper.exe has nothing
    here to remove.

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

# Keep the window alive after the run when the script runs in a console that
# Windows opened only for it (for example double-clicking it in File Explorer);
# piped, redirected, non-console and CI invocations must never block. Copied
# from Install-NodePaper.ps1: without it every message below, including the
# refusals, flashes away unread.
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
        $matched = $false
        try { $matched = (Get-NormalizedPathEntry $candidate.Trim().Trim('"')) -eq $wanted } catch { }
        if (-not $matched) { $kept += $candidate }
    }
    return $kept -join ';'
}

# A Setup installation is not this script's to remove. Setup keeps its own
# uninstaller, Start-menu entry and Windows uninstall registration, none of
# which this script knows about; taking the Path entry away here leaves an
# installation that is still listed in Settings but has no command.
#
# Both marks are checked because either can be the one present: unins000.exe
# is written into the installation directory (UninstallFilesDir={app}), and
# the uninstall registration records the same directory as InstallLocation.
$SetupAppId = '{6E1B5C6A-6C2F-4D4B-9A62-2C7E60C0A5F1}'
$SetupUninstallKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\$($SetupAppId)_is1"

function Test-SetupInstallation {
    param([string]$Directory)
    if ([string]::IsNullOrWhiteSpace($Directory)) { return $false }
    if (Test-Path -LiteralPath (Join-Path $Directory "unins000.exe") -PathType Leaf) { return $true }
    try {
        if (Test-Path -LiteralPath $SetupUninstallKey) {
            $entry = Get-ItemProperty -LiteralPath $SetupUninstallKey -Name "InstallLocation" -ErrorAction SilentlyContinue
            if ($null -ne $entry) {
                $location = [string]$entry.InstallLocation
                if (-not [string]::IsNullOrWhiteSpace($location) -and
                    (Get-NormalizedPathEntry $location) -eq (Get-NormalizedPathEntry $Directory)) {
                    return $true
                }
            }
        }
    }
    catch { }
    return $false
}

if (Test-SetupInstallation $PSScriptRoot) {
    Write-Host "This folder holds a NodePaper installation created by Setup:"
    Write-Host "  $PSScriptRoot"
    Write-Host ""
    Write-Host "This script only unregisters a portable ZIP release. Running it here would take"
    Write-Host "the command away while the installation itself, its Start-menu entries and its"
    Write-Host "entry in Settings all stayed behind."
    Write-Host ""
    Write-Host "Nothing was changed. Uninstall through one of these instead:"
    Write-Host "  Start menu > NodePaper > Uninstall NodePaper"
    Write-Host "  Settings > Apps > Installed apps > NodePaper > Uninstall"
    Wait-CloseWindow
    exit 1
}

# A portable release is described by its own directory: a nodepaper.exe of its
# own, and none of the marks Test-SetupInstallation looks for. Nothing about it
# is recorded anywhere else, which is why this script needs no registry at all.
function Test-PortableInstallation {
    param([string]$Directory)
    if ([string]::IsNullOrWhiteSpace($Directory)) { return $false }
    try {
        if (-not (Test-Path -LiteralPath (Join-Path $Directory "nodepaper.exe") -PathType Leaf)) { return $false }
        return (-not (Test-SetupInstallation $Directory))
    }
    catch { return $false }
}

# Releases up to rc.9 recorded the registered directory in HKCU\Software\
# NodePaper\PortablePath. Nothing reads it any more, so a leftover value is
# deleted rather than migrated, and the key goes with it because it existed only
# to carry that value. A key holding anything else is left alone.
$LegacyRegistrationKey = 'HKCU:\Software\NodePaper'
$LegacyRegistrationValue = 'PortablePath'

function Remove-LegacyRegistration {
    try {
        if (-not (Test-Path -LiteralPath $LegacyRegistrationKey)) { return }
        if (@(Get-ChildItem -LiteralPath $LegacyRegistrationKey -ErrorAction SilentlyContinue).Count -gt 0) { return }
        $properties = Get-ItemProperty -LiteralPath $LegacyRegistrationKey -ErrorAction SilentlyContinue
        if ($null -ne $properties) {
            $unexpected = @($properties.PSObject.Properties |
                Where-Object { $_.Name -notlike 'PS*' -and $_.Name -ne $LegacyRegistrationValue })
            if ($unexpected.Count -gt 0) { return }
        }
        Remove-Item -LiteralPath $LegacyRegistrationKey -Recurse -Force -ErrorAction SilentlyContinue
    }
    catch { }
}

# Which folder to unregister: the one this script sits in.
#
# M4-13 deliberately removed a $PSScriptRoot fallback, and this is not a return
# to it. What it removed was a blind fallback: the script read a global registry
# value, and when there was none it unregistered $PSScriptRoot anyway -- which
# took the Path entry away from whatever directory this script happened to sit
# in, a Setup installation included, and then reported success. Ownership was
# being guessed. It is now checked: the directory has to show that it is a
# portable release before anything is removed, a Setup installation is refused
# above, and an explicitly named folder goes through both checks too. Under that
# rule $PSScriptRoot is evidence rather than a guess -- a portable release ships
# this script inside the very folder Install-NodePaper.ps1 registers, so the
# folder this script runs from is the folder to unregister -- and it gives the
# natural semantics as well: run the uninstall script in a portable folder and
# that one folder is unregistered.
$explicitRoot = -not [string]::IsNullOrWhiteSpace($InstallRoot)
if (-not $explicitRoot) { $InstallRoot = $PSScriptRoot }
$InstallRoot = [System.IO.Path]::GetFullPath($InstallRoot).TrimEnd('\', '/')

Remove-LegacyRegistration

# The guard above already covered $PSScriptRoot; an explicitly named folder gets
# the same refusal rather than a quieter one.
if ($explicitRoot -and (Test-SetupInstallation $InstallRoot)) {
    Write-Host "That folder holds a NodePaper installation created by Setup:"
    Write-Host "  $InstallRoot"
    Write-Host ""
    Write-Host "This script only unregisters a portable ZIP release. Nothing was changed."
    Write-Host "Uninstall through one of these instead:"
    Write-Host "  Start menu > NodePaper > Uninstall NodePaper"
    Write-Host "  Settings > Apps > Installed apps > NodePaper > Uninstall"
    Wait-CloseWindow
    exit 1
}

if (-not (Test-PortableInstallation $InstallRoot)) {
    Write-Host "This folder holds no portable NodePaper release:"
    Write-Host "  $InstallRoot"
    Write-Host ""
    Write-Host "A portable release is the folder a NodePaper ZIP was extracted to, with"
    Write-Host "nodepaper.exe next to this script. Nothing was changed."
    Write-Host ""
    Write-Host "If NodePaper was installed with Setup, uninstall it through"
    Write-Host "Start menu > NodePaper > Uninstall NodePaper, or Settings > Apps."
    Write-Host "To unregister a portable release, run this script from inside its folder, or"
    Write-Host "name that folder: .\Uninstall-NodePaper.ps1 -InstallRoot <folder>"
    Wait-CloseWindow
    exit 0
}

$oldPath = Get-PathValue $PathScope
$oldProcessPath = [Environment]::GetEnvironmentVariable("Path", "Process")
$newPath = Remove-PathEntry $oldPath $InstallRoot

try {
    Set-PathValue $PathScope $newPath
    if ($PathScope -eq "User") {
        [Environment]::SetEnvironmentVariable("Path", (Remove-PathEntry $oldProcessPath $InstallRoot), "Process")
    }
}
catch {
    try { Set-PathValue $PathScope $oldPath } catch { }
    try { [Environment]::SetEnvironmentVariable("Path", $oldProcessPath, "Process") } catch { }
    Write-Host "NodePaper was not removed from the user Path: $($_.Exception.Message)"
    Write-Host "The Path was left as it was."
    Wait-CloseWindow
    throw
}

# The folder is deliberately left alone. It is where the user chose to put the
# release, it may sit alongside their own files, and deleting a directory this
# script does not own is not a decision to make on their behalf.
Write-Host "NodePaper was removed from the user Path: $InstallRoot"
Write-Host "The folder itself was kept; delete it yourself if you no longer need it."
Write-Host "NodePaper Projects and TeX installations were not changed."
Wait-CloseWindow
exit 0
