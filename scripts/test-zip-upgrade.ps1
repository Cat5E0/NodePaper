<#
.SYNOPSIS
    ZIP channel (Install-NodePaper.ps1) registration gate.

.DESCRIPTION
    The ZIP channel registers the extracted folder on PATH; it does not copy
    anything. These checks cover what that changes and what it must not break:

      1. registering an older release and building a real Project with it;
      2. registering a newer release extracted beside it, asserting the older
         folder's PATH entry is dropped so only one survives;
      3. re-registering the same folder (repair);
      4. rejecting a silent downgrade, matching the Setup's silent default;
      5. unregistering, asserting the PATH entry is gone, the registration is
         cleared, and both the folder and the Project's PDF are left alone;
      6. the same round on a Chinese/space path;
      7. registering a payload whose bundled example has been built once --
         the case that used to abort with "Unverified extra file in payload";
      8. registering while a Setup-style installation exists, asserting its
         directory and uninstaller are untouched.

    Runs use Process PATH scope so the real user PATH is never modified. The
    HKCU registration is real, so it is saved and restored around the run.

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
$RegistrationKey = 'HKCU:\Software\NodePaper'
$RegistrationValue = 'PortablePath'
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

function Get-RegisteredPath {
    if (-not (Test-Path -LiteralPath $RegistrationKey)) { return "" }
    $item = Get-ItemProperty -LiteralPath $RegistrationKey -Name $RegistrationValue -ErrorAction SilentlyContinue
    if ($null -eq $item) { return "" }
    return [string]$item.$RegistrationValue
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

function Invoke-Uninstaller {
    param([string]$PackageDir)
    $uninstaller = Join-Path $PackageDir "Uninstall-NodePaper.ps1"
    if (Test-Path -LiteralPath $uninstaller -PathType Leaf) {
        try { & $uninstaller -InstallRoot $PackageDir -PathScope Process 2>&1 | Out-Null } catch { }
    }
}

function Get-PackageVersion {
    param([string]$PackageDir)
    $exe = Join-Path $PackageDir "nodepaper.exe"
    if (-not (Test-Path -LiteralPath $exe -PathType Leaf)) { return "" }
    return ((& $exe --version 2>&1 | Out-String).Trim())
}

# The registration lives in the real HKCU, so preserve whatever is there.
$savedRegistration = Get-RegisteredPath

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
    Assert-True ((Get-NormalizedPathEntry (Get-RegisteredPath)) -eq (Get-NormalizedPathEntry $oldDir)) "registration records the old folder"

    $oldExe = Join-Path $oldDir "nodepaper.exe"
    $projectDir = Join-Path $workRoot "project"
    $fixture = Join-Path $PSScriptRoot "..\nodepaper-test-fixtures\tests\fixtures\complete-single-file"
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
    # nodepaper.exe wins.
    $installNew = Invoke-Installer $newDir "register-new"
    Assert-True ($installNew.Exit -eq 0) "new candidate registered (exit 0)$($installNew.Error)"
    $newVersion = Get-PackageVersion $newDir
    Assert-True ($newVersion -ne $oldVersion) "version changed: $oldVersion -> $newVersion"
    Assert-True (Test-PathContains $newDir) "new candidate folder is on PATH"
    Assert-True (-not (Test-PathContains $oldDir)) "old candidate folder dropped from PATH"
    Assert-True ((Get-NormalizedPathEntry (Get-RegisteredPath)) -eq (Get-NormalizedPathEntry $newDir)) "registration moved to the new folder"
    Assert-True (Test-Path -LiteralPath (Join-Path $oldDir "nodepaper.exe")) "old folder still on disk (registration removes no files)"
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

    # 5. Unregister. The folder and the Project must both survive: this script
    # never owned either of them.
    Invoke-Uninstaller $newDir
    Assert-True (-not (Test-PathContains $newDir)) "PATH entry removed by unregistration"
    Assert-True ((Get-RegisteredPath) -eq "") "registration record cleared"
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
    Invoke-Uninstaller $customDir
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
    Invoke-Uninstaller $newDir

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
            Invoke-Uninstaller $newDir
        }
        finally {
            Remove-Item -LiteralPath $setupDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    }

    $passed = $true
    Write-Host ""
    Write-Host "ZIP channel registration gate passed."
    Write-Host "  old: $oldVersion -> new: $newVersion (silent downgrade rejected by design; interactive confirm is manual)"
}
finally {
    foreach ($dir in @($oldDir, $newDir, (Join-Path $workRoot "中文 安装\NodePaper"))) {
        if ($dir -and (Test-Path -LiteralPath $dir)) { Invoke-Uninstaller $dir }
    }
    # Restore whatever registration the machine had before this run.
    if ([string]::IsNullOrWhiteSpace($savedRegistration)) {
        if (Test-Path -LiteralPath $RegistrationKey) {
            Remove-ItemProperty -LiteralPath $RegistrationKey -Name $RegistrationValue -ErrorAction SilentlyContinue
        }
    }
    else {
        if (-not (Test-Path -LiteralPath $RegistrationKey)) { New-Item -Path $RegistrationKey -Force | Out-Null }
        Set-ItemProperty -LiteralPath $RegistrationKey -Name $RegistrationValue -Value $savedRegistration -Type String
    }
    if (-not $passed -and -not $KeepWorkDirectory) {
        Remove-Item -LiteralPath $workRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
    elseif ($KeepWorkDirectory -and (Test-Path -LiteralPath $workRoot)) {
        Write-Host "ZIP upgrade test work directory retained: $workRoot"
    }
}
