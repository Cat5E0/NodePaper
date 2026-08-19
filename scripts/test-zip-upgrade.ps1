<#
.SYNOPSIS
    ZIP channel (Install-NodePaper.ps1) registration gate.

.DESCRIPTION
    The ZIP channel registers the extracted folder on PATH; it does not copy
    anything. These checks cover what that changes and what it must not break:

      1. registering an older release and building a real Project with it;
      2. registering a newer release extracted beside it, asserting the older
         folder's PATH entry is dropped so only one survives and that the older
         folder itself is left byte for byte as it was;
      3. re-registering the same folder (repair);
      4. rejecting a silent downgrade, matching the Setup's silent default;
      5. unregistering from inside the folder with no arguments, asserting the
         PATH entry is gone and both the folder and the Project's PDF are left
         alone;
      6. the same round on a Chinese/space path;
      7. registering a payload whose bundled example has been built once --
         the case that used to abort with "Unverified extra file in payload";
      8. registering while a Setup-style installation exists, asserting its
         directory and uninstaller are untouched;
      9. a Setup-marked folder on PATH: registering must not take its entry
         away, and running the uninstall script inside it must refuse;
     10. a leftover HKCU\Software\NodePaper\PortablePath from a pre-release
         script: registering deletes the whole key and reads nothing from it.

    A portable installation is identified by its own directory throughout --
    nodepaper.exe present, unins000.exe absent - and no global record of one
    exists any more, so these checks read PATH rather than the registry.

    Runs use Process PATH scope so the real user PATH is never modified. The
    obsolete HKCU key is real, so whether it existed is saved and restored
    around the run.

.PARAMETER OldZip
    Path to an older release ZIP (for example rc.7 or rc.8).
.PARAMETER NewZip
    Path to the current release ZIP.
.PARAMETER KeepWorkDirectory
    Keep the temporary work directory for diagnosis.
#>
param(
    [Parameter(Mandatory = $true)][string]$OldZip,
    [Parameter(Mandatory = $true)][string]$NewZip,
    [switch]$KeepWorkDirectory
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$workRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("nodepaper-zip-upgrade-" + [Guid]::NewGuid().ToString("N"))
# The key the pre-release scripts wrote their single global PortablePath into.
# Nothing writes or reads it any more; the gate only checks that a leftover one
# is deleted and that registering never recreates it.
$LegacyKey = 'HKCU:\Software\NodePaper'
$LegacyValue = 'PortablePath'
$passed = $false

function Assert-True {
    param([Parameter(Mandatory = $true)][bool]$Condition, [Parameter(Mandatory = $true)][string]$Message)
    if (-not $Condition) { throw "FAIL: $Message" }
    Write-Host "  ok: $Message"
}

function Get-NormalizedPathEntry {
    param([string]$Path)
    try { return [System.IO.Path]::GetFullPath($Path).TrimEnd('\', '/').ToLowerInvariant() } catch { return "" }
}

function Test-PathContains {
    param([string]$Directory)
    $normalized = Get-NormalizedPathEntry $Directory
    $processPath = [Environment]::GetEnvironmentVariable("Path", "Process")
    foreach ($entry in @($processPath -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })) {
        try { if ((Get-NormalizedPathEntry $entry.Trim().Trim('"')) -eq $normalized) { return $true } } catch { }
    }
    return $false
}

# The judgement the installer and the uninstaller both use: a folder holding a
# nodepaper.exe and no Setup uninstaller is a portable release.
function Test-PortableDirectory {
    param([string]$Directory)
    if ([string]::IsNullOrWhiteSpace($Directory)) { return $false }
    try {
        if (-not (Test-Path -LiteralPath (Join-Path $Directory "nodepaper.exe") -PathType Leaf)) { return $false }
        return (-not (Test-Path -LiteralPath (Join-Path $Directory "unins000.exe") -PathType Leaf))
    }
    catch { return $false }
}

# The portable folders on the process PATH that this run created, in PATH order.
# Scoped to the work directory on purpose: the machine running this may have a
# NodePaper of its own on PATH, and that one is none of the gate's business.
function Get-PortableDirsUnderWorkRoot {
    $prefix = (Get-NormalizedPathEntry $workRoot) + [System.IO.Path]::DirectorySeparatorChar
    $found = @()
    foreach ($entry in @([Environment]::GetEnvironmentVariable("Path", "Process") -split ';' |
            Where-Object { -not [string]::IsNullOrWhiteSpace($_) })) {
        $directory = $entry.Trim().Trim('"')
        $normalized = Get-NormalizedPathEntry $directory
        if ($normalized -eq "" -or -not $normalized.StartsWith($prefix)) { continue }
        if (Test-PortableDirectory $directory) { $found += $normalized }
    }
    return $found
}

function Test-LegacyKeyExists {
    return (Test-Path -LiteralPath $LegacyKey)
}

# Every file under a folder with its hash: the upgrade must take the older
# folder off PATH without touching a byte inside it.
function Get-DirectorySnapshot {
    param([string]$Root)
    return @(Get-ChildItem -LiteralPath $Root -Recurse -File | Sort-Object FullName | ForEach-Object {
        $_.FullName.Substring($Root.Length).TrimStart('\') + " " + (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash
    }) -join "`n"
}

function Expand-Package {
    param([string]$ZipPath, [string]$Target)
    New-Item -ItemType Directory -Force -Path $Target | Out-Null
    Expand-Archive -LiteralPath $ZipPath -DestinationPath $Target -Force
    $dirs = @(Get-ChildItem -LiteralPath $Target -Directory)
    if ($dirs.Count -ne 1) { throw "ZIP must contain exactly one package directory: $ZipPath" }
    return $dirs[0].FullName
}

function Invoke-Installer {
    param([string]$PackageDir, [string]$Label)
    $installer = Join-Path $PackageDir "Install-NodePaper.ps1"
    Assert-True (Test-Path -LiteralPath $installer) "${Label}: Install-NodePaper.ps1 present"
    Assert-True (Test-Path -LiteralPath (Join-Path $PackageDir "Uninstall-NodePaper.ps1")) "${Label}: Uninstall-NodePaper.ps1 present"
    $callError = ""
    try {
        & $installer -PathScope Process 2>&1 | ForEach-Object { Write-Host $_ }
        $exit = $LASTEXITCODE
        if ($null -eq $exit) { $exit = 0 }
    }
    catch {
        $exit = 1
        $callError = $_.Exception.Message
    }
    Write-Host "  $Label installer exit: $exit"
    return @{ Exit = $exit; Error = $callError }
}

# No -InstallRoot by default: the folder the script sits in is the folder it
# unregisters, which is the normal way a user runs it. -ExplicitRoot exercises
# the parameter, which has to pass the same ownership check.
function Invoke-Uninstaller {
    param([string]$PackageDir, [switch]$ExplicitRoot, [switch]$Show)
    $uninstaller = Join-Path $PackageDir "Uninstall-NodePaper.ps1"
    if (-not (Test-Path -LiteralPath $uninstaller -PathType Leaf)) { return @{ Exit = 0 } }
    $exit = 0
    try {
        if ($ExplicitRoot) {
            $output = & $uninstaller -InstallRoot $PackageDir -PathScope Process 2>&1
        }
        else {
            $output = & $uninstaller -PathScope Process 2>&1
        }
        $exit = $LASTEXITCODE
        if ($null -eq $exit) { $exit = 0 }
        if ($Show) { $output | ForEach-Object { Write-Host "    $_" } }
    }
    catch { $exit = 1 }
    return @{ Exit = $exit }
}

function Get-PackageVersion {
    param([string]$PackageDir)
    $exe = Join-Path $PackageDir "nodepaper.exe"
    if (-not (Test-Path -LiteralPath $exe -PathType Leaf)) { return "" }
    return ((& $exe --version 2>&1 | Out-String).Trim())
}

# The obsolete key lives in the real HKCU. Nothing here writes a PortablePath
# except the leftover-value check below, but that check and the installer both
# delete the key, so whether it existed is recorded and put back afterwards.
$legacyKeyExisted = Test-LegacyKeyExists

try {
    New-Item -ItemType Directory -Force -Path $workRoot | Out-Null
    $oldDir = Expand-Package $OldZip (Join-Path $workRoot "old")
    $newDir = Expand-Package $NewZip (Join-Path $workRoot "new")

    # 1. Register the older release and build a real Project with it.
    $installOld = Invoke-Installer $oldDir "register-old"
    Assert-True ($installOld.Exit -eq 0) "old candidate registered (exit 0)$($installOld.Error)"
    $oldVersion = Get-PackageVersion $oldDir
    Assert-True ($oldVersion -like "nodepaper 0.1.0-rc.*") "old candidate version: $oldVersion"
    Assert-True (Test-PathContains $oldDir) "old candidate folder is on PATH"
    $onPath = @(Get-PortableDirsUnderWorkRoot)
    Assert-True ($onPath.Count -eq 1 -and $onPath[0] -eq (Get-NormalizedPathEntry $oldDir)) "the old folder is the only portable NodePaper this run put on PATH"
    # No key assertion here on purpose: -OldZip is an arbitrary earlier release,
    # and the ones that predate M4-14 register HKCU\Software\NodePaper by
    # design. Asserting its absence after the old installer runs tests the old
    # package, not this change. The absence is asserted after the upgrade below,
    # where the new installer is the one that had to hold to it.

    $oldExe = Join-Path $oldDir "nodepaper.exe"
    $projectDir = Join-Path $workRoot "project"
    $fixture = Join-Path $PSScriptRoot "..\tests\fixtures\complete-single-file"
    Copy-Item -LiteralPath $fixture -Destination $projectDir -Recurse
    Push-Location $projectDir
    try {
        & $oldExe validate $projectDir --format json | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "validate failed under old candidate" }
        & $oldExe build $projectDir --format json | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "build failed under old candidate" }
    }
    finally { Pop-Location }
    $pdfBefore = Join-Path $projectDir "dist\paper.pdf"
    Assert-True (Test-Path -LiteralPath $pdfBefore -PathType Leaf) "Project PDF produced under old candidate"

    # 2. Register the newer release extracted beside it. Upgrading by extracting
    # a second copy must leave one PATH entry, not two whose order decides which
    # nodepaper.exe wins: one folder owns the command and switching is explicit.
    # The old folder is only taken off PATH -- not a byte inside it may change.
    $oldSnapshot = Get-DirectorySnapshot $oldDir
    $installNew = Invoke-Installer $newDir "register-new"
    Assert-True ($installNew.Exit -eq 0) "new candidate registered (exit 0)$($installNew.Error)"
    $newVersion = Get-PackageVersion $newDir
    Assert-True ($newVersion -ne $oldVersion) "version changed: $oldVersion -> $newVersion"
    Assert-True (Test-PathContains $newDir) "new candidate folder is on PATH"
    Assert-True (-not (Test-PathContains $oldDir)) "old candidate folder dropped from PATH"
    $onPath = @(Get-PortableDirsUnderWorkRoot)
    Assert-True ($onPath.Count -eq 1 -and $onPath[0] -eq (Get-NormalizedPathEntry $newDir)) "the new folder is now the only portable NodePaper this run has on PATH"
    Assert-True (-not (Test-LegacyKeyExists)) "the upgrade needed no global registration key"
    Assert-True (Test-Path -LiteralPath (Join-Path $oldDir "nodepaper.exe")) "old folder still on disk (registration removes no files)"
    Assert-True ((Get-DirectorySnapshot $oldDir) -eq $oldSnapshot) "old folder unchanged file for file after losing its PATH entry"
    Assert-True (Test-Path -LiteralPath $pdfBefore -PathType Leaf) "existing Project PDF preserved"

    # 3. Re-register the same folder (repair).
    $repair = Invoke-Installer $newDir "repair-new"
    Assert-True ($repair.Exit -eq 0) "repair registration succeeded$($repair.Error)"
    Assert-True (Test-PathContains $newDir) "repair kept the PATH entry"

    # 4. A silent downgrade must be rejected. No published candidate is newer
    # than the current one, so raise the registered manifest's version instead;
    # the payload itself is untouched and must stay registered.
    $registeredManifest = Join-Path $newDir "payload-manifest.json"
    $manifestBackup = Get-Content -LiteralPath $registeredManifest -Raw -Encoding UTF8
    try {
        $parsed = $manifestBackup | ConvertFrom-Json
        $parsed.version = "0.1.0-rc.99"
        $parsed | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath $registeredManifest -Encoding UTF8
        $downgrade = Invoke-Installer $oldDir "downgrade-old"
        Assert-True ($downgrade.Exit -ne 0) "silent downgrade rejected"
        Assert-True (-not (Test-PathContains $oldDir)) "rejected downgrade left PATH unchanged"
    }
    finally {
        Set-Content -LiteralPath $registeredManifest -Value $manifestBackup -Encoding UTF8 -NoNewline
    }

    # 5. Unregister the way a user does it: run the script sitting in the folder,
    # with no arguments. The folder it runs from is the folder it unregisters,
    # and only that one. The folder and the Project must both survive: this
    # script never owned either of them.
    $removeNew = Invoke-Uninstaller $newDir
    Assert-True ($removeNew.Exit -eq 0) "unregistering from inside the folder succeeded (exit 0)"
    Assert-True (-not (Test-PathContains $newDir)) "PATH entry removed by unregistration"
    Assert-True (@(Get-PortableDirsUnderWorkRoot).Count -eq 0) "no portable NodePaper of this run is left on PATH"
    Assert-True (-not (Test-LegacyKeyExists)) "unregistering left no registration key behind"
    Assert-True (Test-Path -LiteralPath (Join-Path $newDir "nodepaper.exe")) "folder kept after unregistration"
    Assert-True (Test-Path -LiteralPath $pdfBefore -PathType Leaf) "Project PDF kept after unregistration"

    # 6. Chinese/space path round.
    $customParent = Join-Path $workRoot "中文 安装"
    New-Item -ItemType Directory -Force -Path $customParent | Out-Null
    $customDir = Join-Path $customParent "NodePaper"
    Copy-Item -LiteralPath $newDir -Destination $customDir -Recurse
    $installCustom = Invoke-Installer $customDir "register-custom"
    Assert-True ($installCustom.Exit -eq 0) "custom path registered$($installCustom.Error)"
    Assert-True (Test-PathContains $customDir) "custom path is on PATH"
    # -InstallRoot names the same folder explicitly; it has to pass the same
    # ownership check the default does.
    $removeCustom = Invoke-Uninstaller $customDir -ExplicitRoot
    Assert-True ($removeCustom.Exit -eq 0) "-InstallRoot unregistered the custom path (exit 0)"
    Assert-True (-not (Test-PathContains $customDir)) "custom path removed from PATH"

    # 7. Regression: a payload whose bundled example has been built once. The
    # README tells readers to try examples/cumcm-single-file, which leaves
    # .nodepaper/build behind, and that used to abort registration entirely.
    $dirtyBuild = Join-Path $newDir "examples\cumcm-single-file\.nodepaper\build"
    New-Item -ItemType Directory -Force -Path $dirtyBuild | Out-Null
    Set-Content -LiteralPath (Join-Path $dirtyBuild "paper.aux") -Value "stray build output" -NoNewline
    $dirty = Invoke-Installer $newDir "register-after-example-build"
    Assert-True ($dirty.Exit -eq 0) "registration succeeded despite build output in the bundled example$($dirty.Error)"
    Assert-True (Test-PathContains $newDir) "dirty payload registered onto PATH"
    $null = Invoke-Uninstaller $newDir

    # 8. Regression: a Setup-style installation must survive untouched. Setup
    # keeps its uninstaller in its install directory, and the previous copying
    # installer deleted that directory wholesale, orphaning the Uninstall entry.
    $setupDir = Join-Path $env:LOCALAPPDATA "Programs\NodePaper"
    $setupPreexisting = Test-Path -LiteralPath $setupDir
    if ($setupPreexisting) {
        Write-Host "  skip: a real installation exists at $setupDir; not simulating Setup over it"
    }
    else {
        New-Item -ItemType Directory -Force -Path $setupDir | Out-Null
        try {
            $uninsPath = Join-Path $setupDir "unins000.exe"
            Set-Content -LiteralPath $uninsPath -Value "setup uninstaller" -NoNewline
            Set-Content -LiteralPath (Join-Path $setupDir "nodepaper.exe") -Value "setup copy" -NoNewline
            $uninsHash = (Get-FileHash -LiteralPath $uninsPath -Algorithm SHA256).Hash

            $crossChannel = Invoke-Installer $newDir "register-over-setup"
            Assert-True ($crossChannel.Exit -eq 0) "registration succeeded alongside a Setup installation$($crossChannel.Error)"
            Assert-True (Test-Path -LiteralPath $uninsPath) "Setup's uninstaller still present"
            Assert-True ((Get-FileHash -LiteralPath $uninsPath -Algorithm SHA256).Hash -eq $uninsHash) "Setup's uninstaller unmodified"
            Assert-True ((Get-Content -LiteralPath (Join-Path $setupDir "nodepaper.exe") -Raw) -eq "setup copy") "Setup's nodepaper.exe untouched"
            $null = Invoke-Uninstaller $newDir
        }
        finally {
            Remove-Item -LiteralPath $setupDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }

    # 9. A Setup-marked folder that is on PATH is another channel's entry. It
    # must survive a registration untouched, and the uninstall script must
    # refuse to run inside it -- taking that entry away would leave a Setup
    # installation listed in Settings with no command, which is the defect
    # M4-13 was opened for.
    $fakeSetupDir = Join-Path $workRoot "setup-marked"
    Copy-Item -LiteralPath $newDir -Destination $fakeSetupDir -Recurse
    Set-Content -LiteralPath (Join-Path $fakeSetupDir "unins000.exe") -Value "setup uninstaller" -NoNewline
    [Environment]::SetEnvironmentVariable("Path",
        ([Environment]::GetEnvironmentVariable("Path", "Process") + ";" + $fakeSetupDir), "Process")
    Assert-True (Test-PathContains $fakeSetupDir) "a Setup-marked folder is on PATH"

    $besideSetup = Invoke-Installer $newDir "register-beside-setup-marked-entry"
    Assert-True ($besideSetup.Exit -eq 0) "registration succeeded with a Setup-marked folder on PATH$($besideSetup.Error)"
    Assert-True (Test-PathContains $fakeSetupDir) "the Setup-marked folder kept its PATH entry"
    Assert-True (Test-PathContains $newDir) "the portable folder was registered"

    $refuseSetup = Invoke-Uninstaller $fakeSetupDir -Show
    Assert-True ($refuseSetup.Exit -eq 1) "the uninstall script refuses to run inside a Setup-marked folder (exit 1)"
    Assert-True (Test-PathContains $fakeSetupDir) "the refused run left the Setup-marked PATH entry alone"
    Assert-True (Test-PathContains $newDir) "the refused run left the portable PATH entry alone"

    $removeBeside = Invoke-Uninstaller $newDir
    Assert-True ($removeBeside.Exit -eq 0) "the portable folder unregisters itself normally afterwards"
    Assert-True (-not (Test-PathContains $newDir)) "portable PATH entry removed"
    Assert-True (Test-PathContains $fakeSetupDir) "the Setup-marked entry still stands after the portable one went"

    # A folder with no nodepaper.exe is nobody's: reported, not acted on.
    $emptyDir = Join-Path $workRoot "not-a-release"
    New-Item -ItemType Directory -Force -Path $emptyDir | Out-Null
    Copy-Item -LiteralPath (Join-Path $newDir "Uninstall-NodePaper.ps1") -Destination $emptyDir
    $notARelease = Invoke-Uninstaller $emptyDir -Show
    Assert-True ($notARelease.Exit -eq 0) "a folder holding no nodepaper.exe is reported, not acted on (exit 0)"
    Assert-True (Test-PathContains $fakeSetupDir) "that run changed no PATH entry either"

    # 10. A leftover PortablePath from a pre-release script. Nothing reads it;
    # registering deletes the whole key, which is all the migration this needs.
    New-Item -Path $LegacyKey -Force | Out-Null
    Set-ItemProperty -LiteralPath $LegacyKey -Name $LegacyValue -Value $oldDir -Type String
    Assert-True (Test-LegacyKeyExists) "a leftover PortablePath was planted"
    $selfHeal = Invoke-Installer $newDir "register-with-leftover-key"
    Assert-True ($selfHeal.Exit -eq 0) "registration succeeded with a leftover key present$($selfHeal.Error)"
    Assert-True (-not (Test-LegacyKeyExists)) "the leftover HKCU\Software\NodePaper key was deleted"
    Assert-True (Test-PathContains $newDir) "the leftover key did not divert the registration"
    Assert-True (Test-PathContains $fakeSetupDir) "the leftover key's directory was not confused with a Setup-marked one"
    $null = Invoke-Uninstaller $newDir

    $passed = $true
    Write-Host ""
    Write-Host "ZIP channel registration gate passed."
    Write-Host "  old: $oldVersion -> new: $newVersion (silent downgrade rejected by design; interactive confirm is manual)"
}
finally {
    foreach ($dir in @($oldDir, $newDir, (Join-Path $workRoot "中文 安装\NodePaper"))) {
        if ($dir -and (Test-Path -LiteralPath $dir)) { $null = Invoke-Uninstaller $dir }
    }
    # Put the obsolete key back exactly as the machine had it: absent if it was
    # absent, and present but empty if it was present. No value is restored --
    # this run only ever plants one itself, and nothing writes one any more.
    if ($legacyKeyExisted) {
        if (-not (Test-Path -LiteralPath $LegacyKey)) { New-Item -Path $LegacyKey -Force | Out-Null }
    }
    elseif (Test-Path -LiteralPath $LegacyKey) {
        Remove-Item -LiteralPath $LegacyKey -Recurse -Force -ErrorAction SilentlyContinue
    }
    if (-not $passed -and -not $KeepWorkDirectory) {
        Remove-Item -LiteralPath $workRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
    elseif ($KeepWorkDirectory -and (Test-Path -LiteralPath $workRoot)) {
        Write-Host "ZIP upgrade test work directory retained: $workRoot"
    }
}
