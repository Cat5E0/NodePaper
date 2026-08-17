<#
.SYNOPSIS
    Register this extracted NodePaper release on the current user's Path.

.DESCRIPTION
    Verifies every payload file against payload-manifest.json, then puts this
    folder on the user Path. Nothing is copied: NodePaper runs from where it
    was extracted, which is why this folder must stay where it is. If a
    previous run registered a different folder, that entry is removed first.

    No administrator rights, network access, telemetry, or automatic update is
    used, and nothing outside the user Path and HKCU\Software\NodePaper is
    written. Setup installs elsewhere and is left untouched.

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
# prompt and follow the safe default per case (see caller).
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
$RegistrationKey = 'HKCU:\Software\NodePaper'
$RegistrationValue = 'PortablePath'

function Get-RegisteredPath {
    if (-not (Test-Path -LiteralPath $RegistrationKey)) { return "" }
    $item = Get-ItemProperty -LiteralPath $RegistrationKey -Name $RegistrationValue -ErrorAction SilentlyContinue
    if ($null -eq $item) { return "" }
    return [string]$item.$RegistrationValue
}

function Set-RegisteredPath {
    param([string]$Value)
    if (-not (Test-Path -LiteralPath $RegistrationKey)) {
        New-Item -Path $RegistrationKey -Force | Out-Null
    }
    Set-ItemProperty -LiteralPath $RegistrationKey -Name $RegistrationValue -Value $Value -Type String
}

if (-not [string]::IsNullOrWhiteSpace($InstallRoot)) {
    throw "-InstallRoot is no longer supported: NodePaper now runs from the directory it was extracted to. Move this folder where you want it and run this script again."
}
$InstallRoot = $sourceRoot

# Compare against whichever directory is currently registered, so upgrading by
# extracting a new ZIP next to the old one still reports the version change.
$previousRoot = Get-RegisteredPath
$installedVersion = ""
if (-not [string]::IsNullOrWhiteSpace($previousRoot)) {
    $previousManifest = Join-Path $previousRoot "payload-manifest.json"
    if (Test-Path -LiteralPath $previousManifest -PathType Leaf) {
        try {
            $parsed = Get-Content -LiteralPath $previousManifest -Raw -Encoding UTF8 | ConvertFrom-Json
            $installedVersion = [string]$parsed.version
        }
        catch { $installedVersion = "" }
    }
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

$oldPath = Get-PathValue $PathScope
$oldProcessPath = [Environment]::GetEnvironmentVariable("Path", "Process")
$pathChanged = $false

try {
    # Drop the directory registered by a previous run before adding this one,
    # so extracting a new ZIP beside the old one does not leave both on Path
    # with the winner decided by ordering.
    $newPath = $oldPath
    if (-not [string]::IsNullOrWhiteSpace($previousRoot) -and
        (Get-NormalizedPathEntry $previousRoot) -ne (Get-NormalizedPathEntry $InstallRoot)) {
        $newPath = Remove-PathEntry $newPath $previousRoot
        Write-Host "Removed the previously registered directory from Path: $previousRoot"
    }
    $newPath = Add-PathEntry $newPath $InstallRoot

    Set-PathValue $PathScope $newPath
    $pathChanged = $true
    if ($PathScope -eq "User") {
        $newProcessPath = $oldProcessPath
        if (-not [string]::IsNullOrWhiteSpace($previousRoot)) {
            $newProcessPath = Remove-PathEntry $newProcessPath $previousRoot
        }
        [Environment]::SetEnvironmentVariable("Path", (Add-PathEntry $newProcessPath $InstallRoot), "Process")
    }

    Set-RegisteredPath $InstallRoot
}
catch {
    if ($pathChanged) { try { Set-PathValue $PathScope $oldPath } catch { } }
    try { [Environment]::SetEnvironmentVariable("Path", $oldProcessPath, "Process") } catch { }
    throw
}
finally {
    Wait-CloseWindow
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
    Write-Warning "Another NodePaper is earlier on Path and will answer to `nodepaper`:"
    Write-Warning "  $effective"
    Write-Warning "This copy is registered but shadowed. To use it, remove that entry from Path"
    Write-Warning "(or uninstall that copy), then open a new terminal."
}
