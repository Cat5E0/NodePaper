<#
.SYNOPSIS
    Remove the registered NodePaper folder from the current user's Path.

.DESCRIPTION
    Removes the exact Path entry registered by Install-NodePaper.ps1 and clears
    the registration under HKCU\Software\NodePaper. The folder itself is kept:
    NodePaper runs from where it was extracted, so that directory belongs to
    the user rather than to this script.

    Only a directory this script's own channel registered is touched. An
    installation created by Setup has its own uninstaller and is refused here,
    and without a registration nothing is removed at all.

    Projects, TeX, Pandoc installations and unrelated Path entries are not
    touched, and neither is anything installed by Setup.

.PARAMETER InstallRoot
    Folder to remove from Path. Normally omitted: the folder registered by
    Install-NodePaper.ps1 is used. Pass it explicitly to clean up a leftover
    entry whose registration is already gone.

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

$RegistrationKey = 'HKCU:\Software\NodePaper'
$RegistrationValue = 'PortablePath'

if ([string]::IsNullOrWhiteSpace($InstallRoot)) {
    # Only ever remove what Install-NodePaper.ps1 registered. Falling back to
    # $PSScriptRoot used to look helpful, but it made an unregistered copy of
    # this script remove the Path entry of whatever directory it happened to
    # sit in -- including a Setup installation's -- and then report success.
    $registered = ""
    if (Test-Path -LiteralPath $RegistrationKey) {
        $item = Get-ItemProperty -LiteralPath $RegistrationKey -Name $RegistrationValue -ErrorAction SilentlyContinue
        if ($null -ne $item) { $registered = [string]$item.$RegistrationValue }
    }
    if ([string]::IsNullOrWhiteSpace($registered)) {
        Write-Host "No portable NodePaper installation is registered for the current user,"
        Write-Host "so there is no Path entry to remove. Nothing was changed."
        Write-Host ""
        Write-Host "If NodePaper was installed with Setup, uninstall it through"
        Write-Host "Start menu > NodePaper > Uninstall NodePaper, or Settings > Apps."
        Write-Host "If you know a stale entry is left on your Path, name the folder explicitly:"
        Write-Host "  .\Uninstall-NodePaper.ps1 -InstallRoot <folder>"
        Wait-CloseWindow
        exit 0
    }
    $InstallRoot = $registered
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
