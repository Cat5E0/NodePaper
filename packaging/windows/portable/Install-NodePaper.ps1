<#
.SYNOPSIS
    Register this extracted NodePaper release on the current user's Path.

.DESCRIPTION
    Verifies every payload file against payload-manifest.json, then puts this
    folder on the user Path. Nothing is copied: NodePaper runs from where it
    was extracted, which is why this folder must stay where it is. Any other
    portable NodePaper folder already on the Path is taken off it first: one
    folder owns the `nodepaper` command and switching is explicit, so extracting
    a new release beside the old one leaves one entry rather than two whose order
    decides which one answers. The folders themselves are never touched.

    No administrator rights, network access, telemetry, or automatic update is
    used, and nothing outside the user Path is written. Setup installs
    elsewhere, keeps its own uninstaller, and is left untouched.

    Reopen the terminal afterwards, then run: nodepaper

.PARAMETER InstallRoot
    Removed. NodePaper now runs from the folder it was extracted to; passing
    this reports an error rather than silently ignoring it. To change location,
    move the folder and run this script again from there.

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
# prompt and follow the safe default per case (see caller). Only the downgrade
# uses this: an upgrade and a repeat of the same version are carried out
# without asking (see the version comparison below).
function Confirm-Installation {
    param([string]$Message)
    if (-not (Test-InteractivePause)) { return $true }
    Write-Host $Message
    Write-Host "Continue? [Y/n]: " -NoNewline
    $answer = Read-Host
    return ($answer -eq "" -or $answer -match '^[Yy]')
}

# The user Path must be read and written through the registry, never through
# [Environment]::GetEnvironmentVariable/SetEnvironmentVariable with the User
# target. Those two damage unrelated software in different ways:
#
#   Reading expands the value, so a Path containing %JAVA_HOME%\bin comes back
#   as the literal directory. Appending to that and writing it back freezes
#   another program's variable at whatever it happened to point at.
#
#   Writing stores REG_SZ, while the user Path is REG_EXPAND_SZ. Every
#   %VARIABLE% entry then stops expanding at all.
#
# Neither failure reports anything. The Inno installer already handles this
# correctly (see the RegWriteExpandStringValue comment in nodepaper.iss); these
# helpers bring the ZIP channel in line.
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
    # Files outside the manifest are ignored on purpose. This directory is also
    # a place people work: it ships examples/cumcm-single-file/ and the README
    # invites readers to build it, which leaves .nodepaper/build/ behind. The
    # previous rule rejected exactly that and refused to register a payload
    # whose own example had been tried once.
    #
    # Nothing is weakened by allowing it. Every manifest file is still checked
    # for presence and SHA-256, so a modified or deleted file still stops the
    # run, and nothing is copied anywhere, so an added file has nowhere to go.
    return $manifest
}

# A Setup installation is not this script's to register. Setup keeps its own
# uninstaller, Start-menu entry and Windows uninstall registration in that
# directory; registering it here as a portable installation would leave two
# channels claiming one folder, which is exactly the state this script exists
# to avoid (see the note above Test-PortableInstallation below).
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
    Write-Host "This script registers a portable ZIP release, and registering this folder would"
    Write-Host "leave the Setup installation and a portable registration claiming the same"
    Write-Host "directory, which is how an uninstall ends up unable to remove either."
    Write-Host ""
    Write-Host "Nothing was changed. That installation puts itself on your Path already: open"
    Write-Host "a new terminal and run nodepaper. To use a portable release instead, extract the"
    Write-Host "ZIP into its own folder and run this script from there."
    Wait-CloseWindow
    exit 1
}

$sourceRoot = (Resolve-Path -LiteralPath $PSScriptRoot).Path
$manifest = Assert-Payload $sourceRoot

# NodePaper runs from where it was extracted. Nothing is copied.
#
# The previous behaviour copied the payload into %LOCALAPPDATA%\Programs\
# NodePaper, which is also Setup's default directory, and replaced it by moving
# the old directory aside and deleting it. Setup keeps its uninstaller in that
# directory (UninstallFilesDir={app}), so installing from the ZIP over a Setup
# installation deleted unins000.exe while its registry entry survived, leaving
# an entry in Settings that could never be removed.
#
# Registering the extracted directory instead means the two channels never
# share a location. The cost is that this directory is now load-bearing: it
# must not be deleted or moved while registered. Move it and re-run this script
# to change location.

# A portable installation is described by its own directory, and nowhere else.
# Two marks tell the two channels apart:
#
#   installed by Setup     unins000.exe in the directory, or Setup's uninstall
#                          entry naming it as InstallLocation
#   extracted from the ZIP a nodepaper.exe and neither of those marks
#
# This replaces a single global registry value, HKCU\Software\NodePaper\
# PortablePath, which recorded the one directory the last run had registered.
# That value could describe only one portable installation per machine, it went
# on describing a directory after Setup took it over or after the directory was
# emptied, and it did not travel with a folder copied to another drive. A
# directory that says what it is cannot go stale and needs no bookkeeping: what
# is on the Path is what a new terminal will find, and probing those entries
# costs a few file lookups.
function Test-PortableInstallation {
    param([string]$Directory)
    if ([string]::IsNullOrWhiteSpace($Directory)) { return $false }
    try {
        if (-not (Test-Path -LiteralPath (Join-Path $Directory "nodepaper.exe") -PathType Leaf)) { return $false }
        return (-not (Test-SetupInstallation $Directory))
    }
    catch { return $false }
}

# Split a Path value into its entries, each with the directory it names. A user
# Path routinely holds entries that cannot be probed: an unset %VAR%, a
# disconnected drive, a name Join-Path rejects. Those keep an empty Directory
# rather than ending the scan, so they are neither inspected nor rewritten.
function Get-PathEntryList {
    param([string]$Current)
    $list = @()
    foreach ($candidate in @($Current -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })) {
        $directory = ""
        try {
            $expanded = [Environment]::ExpandEnvironmentVariables($candidate.Trim().Trim('"'))
            if (-not [string]::IsNullOrWhiteSpace($expanded)) { $directory = $expanded }
        }
        catch { $directory = "" }
        $list += [pscustomobject]@{ Raw = $candidate; Directory = $directory }
    }
    return $list
}

# The portable NodePaper directories on a Path value, in Path order and without
# repeats. Path order is resolution order: Windows searches left to right and
# stops at the first hit.
function Get-PortableDirectoriesOnPath {
    param([string]$Current)
    $directories = @()
    $seen = @{}
    foreach ($entry in @(Get-PathEntryList $Current)) {
        if ($entry.Directory -eq "") { continue }
        $key = ""
        try { $key = Get-NormalizedPathEntry $entry.Directory } catch { continue }
        if ($key -eq "" -or $seen.ContainsKey($key)) { continue }
        $seen[$key] = $true
        if (Test-PortableInstallation $entry.Directory) { $directories += $entry.Directory }
    }
    return $directories
}

# Drop every portable NodePaper entry except the one being registered, and
# report which ones went. Entries are matched by inspecting the directory they
# name, not by comparing strings with a remembered path: an entry written as
# %NODEPAPER_HOME% names the same directory as its expansion, and the entry
# text is what has to be rewritten.
function Remove-OtherPortableEntries {
    param([string]$Current, [string]$Keep)
    $keepKey = Get-NormalizedPathEntry $Keep
    $kept = @()
    $removed = @()
    foreach ($entry in @(Get-PathEntryList $Current)) {
        $drop = $false
        if ($entry.Directory -ne "") {
            try {
                $drop = (Get-NormalizedPathEntry $entry.Directory) -ne $keepKey -and
                    (Test-PortableInstallation $entry.Directory)
            }
            catch { $drop = $false }
        }
        if ($drop) { $removed += $entry.Directory } else { $kept += $entry.Raw }
    }
    return @{ Path = ($kept -join ';'); Removed = @($removed) }
}

# Releases up to rc.9 recorded the registered directory in HKCU\Software\
# NodePaper\PortablePath. Nothing reads it any more, and a leftover value is
# worse than no value: it names one directory for the whole machine and outlives
# whatever it pointed at. It is deleted rather than migrated, and the key goes
# with it because it existed only to carry that value. A key holding anything
# else is left alone: that is not this script's to delete.
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
        if (-not (Test-Path -LiteralPath $LegacyRegistrationKey)) {
            Write-Host "Removed the obsolete registration key $LegacyRegistrationKey (portable installations are recognised by their own directory now)."
        }
    }
    catch { }
}

if (-not [string]::IsNullOrWhiteSpace($InstallRoot)) {
    throw "-InstallRoot is no longer supported: NodePaper now runs from the directory it was extracted to. Move this folder where you want it and run this script again."
}
$InstallRoot = $sourceRoot

# Read the Path this run will rewrite once, so the version comparison below and
# the pruning further down cannot disagree about what is currently on it.
$oldPath = Get-PathValue $PathScope

# Compare against the portable installation the Path finds first, so upgrading
# by extracting a new ZIP next to the old one still reports the version change.
# First on Path is the copy a new terminal would run, which makes its payload
# manifest the installed version -- including when that copy is this directory
# being registered again, which is the repair case.
$installedVersion = ""
foreach ($candidate in @(Get-PortableDirectoriesOnPath $oldPath)) {
    $candidateManifest = Join-Path $candidate "payload-manifest.json"
    if (-not (Test-Path -LiteralPath $candidateManifest -PathType Leaf)) { continue }
    try {
        $parsed = Get-Content -LiteralPath $candidateManifest -Raw -Encoding UTF8 | ConvertFrom-Json
        $installedVersion = [string]$parsed.version
    }
    catch { $installedVersion = "" }
    if ($installedVersion -ne "") { break }
}
if ($installedVersion -ne "") {
    $versionComparison = Compare-NodePaperVersion ([string]$manifest.version) $installedVersion
    $packageIsDev = ([string]$manifest.version -match '-dev\.')
    $installedIsDev = ($installedVersion -match '-dev\.')
    if ($packageIsDev -ne $installedIsDev) {
        # A dev build and a release-track version are different tracks, not
        # points on the same one: dev.126 is typically newer code than rc.9
        # that simply has not passed the release gates, so "older" would be
        # false. Confirm because the user is crossing tracks; the release-track
        # direction is a benign switch and only needs a word.
        if ($packageIsDev) {
            if (-not (Test-InteractivePause)) {
                throw "A release-track NodePaper ($installedVersion) is installed; this package is the development build $($manifest.version), which has not passed the release gates. Switching requires confirmation; run this script interactively or uninstall the release-track version first."
            }
            if (-not (Confirm-Installation "NodePaper $installedVersion (a release-track version) is installed; this package is the development build $($manifest.version), which has not passed the release gates. Continuing replaces the installation with $($manifest.version).")) {
                throw "Installation cancelled by the user; the existing installation was not changed."
            }
        }
        else {
            Write-Host "Replacing the development build $installedVersion with the release-track version $($manifest.version)."
        }
    }
    elseif ($versionComparison -lt 0) {
        # Downgrade: confirm interactively; reject silently otherwise.
        if (-not (Test-InteractivePause)) {
            throw "A newer NodePaper ($installedVersion) is already installed and this package is the older $($manifest.version). Downgrade requires confirmation; run this script interactively or uninstall the newer version first."
        }
        if (-not (Confirm-Installation "NodePaper $installedVersion is already installed. This package is the older $($manifest.version); continuing will downgrade to $($manifest.version).")) {
            throw "Installation cancelled by the user; the existing installation was not changed."
        }
        Write-Host "Downgrading NodePaper from $installedVersion to $($manifest.version)."
    }
    else {
        # An upgrade and a repeat of the same version are not questions. Eight
        # of nine surveyed Windows installers ask on neither: 7-Zip, IrfanView,
        # Notepad++, electron-builder, winget, Git for Windows and VS Code all
        # upgrade or reinstall within their own channel without a word, the odd
        # one out being JetBrains, which asks only when reinstalling an
        # identical version. Inno's own DirExistsWarning is likewise turned off
        # when the directory belongs to the same application being upgraded.
        # Somebody who ran this script has already said what they want; asking
        # again only trains them to press Enter.
        #
        # One line still says which of the two happened, because the version
        # that ends up registered is the thing they cannot see otherwise. The
        # summary printed at the end reports the same version once more, and
        # nothing is stated twice: this line names the change, that one names
        # the result.
        if ($versionComparison -gt 0) {
            Write-Host "Upgrading NodePaper $installedVersion to $($manifest.version)."
        }
        else {
            Write-Host "Reinstalling NodePaper $($manifest.version)."
        }
    }
}

function Get-EffectiveCommandDirectory {
    <#
    .SYNOPSIS
    Returns the directory whose nodepaper.exe a new terminal would run.

    .DESCRIPTION
    Registration appends, so it cannot promote this copy over one that is
    already earlier on Path. Windows composes a new terminal's Path as machine
    entries followed by user entries, so an entry from either can shadow the one
    just registered -- an installation the script does not track (a hand-edited
    Path entry, or one written by the Setup installer to a different folder)
    keeps answering to `nodepaper` while this script reports success. The Process
    scope is different: that Path is already live, so it is read as-is.
    #>
    param([string]$Scope)

    if ($Scope -eq "Process") {
        $candidates = [Environment]::GetEnvironmentVariable("Path", "Process")
    }
    else {
        $machine = ""
        try {
            $machine = [string](Get-Item -LiteralPath "HKLM:\SYSTEM\CurrentControlSet\Control\Session Manager\Environment").GetValue(
                "Path", "", [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
        }
        catch { }
        $user = Get-PathValue "User"
        $candidates = (@($machine, $user) | Where-Object { -not [string]::IsNullOrWhiteSpace($_) }) -join ';'
    }

    foreach ($entry in ($candidates -split ';')) {
        $trimmed = $entry.Trim().Trim('"')
        if ([string]::IsNullOrWhiteSpace($trimmed)) { continue }
        # A user Path routinely holds entries that cannot be probed: an
        # unset %VAR%, a disconnected drive, a name Test-Path rejects. Any of
        # them must be skipped rather than abort the check.
        try {
            $expanded = [Environment]::ExpandEnvironmentVariables($trimmed)
            if ([string]::IsNullOrWhiteSpace($expanded)) { continue }
            if (Test-Path -LiteralPath (Join-Path $expanded "nodepaper.exe")) { return $expanded }
        }
        catch { continue }
    }
    return ""
}

$oldProcessPath = [Environment]::GetEnvironmentVariable("Path", "Process")
$pathChanged = $false

Remove-LegacyRegistration

try {
    # Take every other portable NodePaper folder off the Path before adding this
    # one, so extracting a new ZIP beside the old one does not leave both on it
    # with the winner decided by ordering.
    #
    # One folder owns the command, and switching is explicit -- this is the rule,
    # not a limitation waiting to be lifted. Keeping any number of extracted
    # folders on disk is fine, and each one says for itself what it is, but only
    # one of them can answer to `nodepaper`: Path is searched left to right, so a
    # second entry does not add a second command, it hides one. Every tool that
    # puts a command on Path settles it the same way -- rustup has the one
    # ~/.cargo/bin, scoop the one shims directory, winget's portable packages the
    # one Links directory, and nvm switches with an explicit `nvm use`. To hand
    # the command to another folder, run that folder's Install-NodePaper.ps1.
    #
    # (The directory marks that replaced the old registry value are what makes a
    # folder recognisable in the first place; they are not a licence to register
    # several at once.)
    #
    # A Setup installation's directory is left where it is. That entry belongs
    # to the other channel, which keeps its own uninstaller, Start-menu entry
    # and entry in Settings; removing it would leave all of those behind with no
    # command to go with them.
    $pruned = Remove-OtherPortableEntries $oldPath $InstallRoot
    $newPath = Add-PathEntry $pruned.Path $InstallRoot
    foreach ($dropped in $pruned.Removed) {
        Write-Host "Removed another portable NodePaper folder from Path: $dropped"
    }

    Set-PathValue $PathScope $newPath
    $pathChanged = $true
    if ($PathScope -eq "User") {
        $prunedProcess = Remove-OtherPortableEntries $oldProcessPath $InstallRoot
        [Environment]::SetEnvironmentVariable("Path", (Add-PathEntry $prunedProcess.Path $InstallRoot), "Process")
    }
}
catch {
    if ($pathChanged) { try { Set-PathValue $PathScope $oldPath } catch { } }
    try { [Environment]::SetEnvironmentVariable("Path", $oldProcessPath, "Process") } catch { }
    Write-Host "NodePaper was not registered: $($_.Exception.Message)"
    Write-Host "The Path was left as it was."
    Wait-CloseWindow
    throw
}

Write-Host "NodePaper $($manifest.version) registered for the current user."
Write-Host "Location: $InstallRoot"
Write-Host "Keep this folder where it is. Deleting or moving it removes the command;"
Write-Host "to relocate, move the folder and run this script again from its new location."
Write-Host "Open a new terminal and run: nodepaper"

# Say so when the command will not be this copy. Reporting success while
# `nodepaper` still runs an older installation is the one failure here that
# nobody catches on their own: the script says registered, the command works,
# and only --version disagrees -- which nobody checks.
$effective = Get-EffectiveCommandDirectory $PathScope
if (-not [string]::IsNullOrWhiteSpace($effective) -and
    (Get-NormalizedPathEntry $effective) -ne (Get-NormalizedPathEntry $InstallRoot)) {
    Write-Host ""
    Write-Warning 'Another NodePaper is earlier on Path and will answer to nodepaper:'
    Write-Warning "  $effective"
    Write-Warning "This copy is registered but shadowed. To use it, remove that entry from Path"
    Write-Warning "(or uninstall that copy), then open a new terminal."
}

# Last statement on purpose: the summary above is what the user came for, and
# waiting before it printed made a double-clicked run flash it away.
Wait-CloseWindow
