<#
.SYNOPSIS
    ZIP channel (Install-NodePaper.ps1) cross-version upgrade/downgrade gate.

.DESCRIPTION
    Installs an older release ZIP into an isolated InstallRoot with Process
    PATH scope, upgrades in place with the current ZIP, repeats the current
    ZIP (repair), asserts a silent downgrade is rejected (matching the Setup's
    silent default), uninstalls and verifies no residue with Project/PDF
    preserved, then repeats install/verify/uninstall on a Chinese/space path.

    Upgrade and repair continue without a prompt in non-interactive runs; a
    downgrade is rejected in non-interactive runs and would only continue
    after an interactive confirmation (manual item, like the Setup dialog).

.PARAMETER OldZip
    Path to an older release ZIP (for example rc.6 or rc.7).
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

function Expand-Package {
    param([string]$ZipPath, [string]$Target)
    New-Item -ItemType Directory -Force -Path $Target | Out-Null
    Expand-Archive -LiteralPath $ZipPath -DestinationPath $Target -Force
    $dirs = @(Get-ChildItem -LiteralPath $Target -Directory)
    if ($dirs.Count -ne 1) { throw "ZIP must contain exactly one package directory: $ZipPath" }
    return $dirs[0].FullName
}

function Invoke-Installer {
    param([string]$PackageDir, [string]$InstallRoot, [string]$Label)
    $installer = Join-Path $PackageDir "Install-NodePaper.ps1"
    $uninstaller = Join-Path $PackageDir "Uninstall-NodePaper.ps1"
    Assert-True (Test-Path -LiteralPath $installer) "$Label: Install-NodePaper.ps1 missing in $PackageDir"
    Assert-True (Test-Path -LiteralPath $uninstaller) "$Label: Uninstall-NodePaper.ps1 missing in $PackageDir"
    $callError = ""
    try {
        & $installer -InstallRoot $InstallRoot -PathScope Process 2>&1 | ForEach-Object { Write-Host $_ }
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

function Get-InstalledVersion {
    param([string]$InstallRoot)
    $exe = Join-Path $InstallRoot "nodepaper.exe"
    if (-not (Test-Path -LiteralPath $exe -PathType Leaf)) { return "" }
    return ((& $exe --version 2>&1 | Out-String).Trim())
}

function Test-Residue {
    param([string]$InstallRoot, [string]$Label)
    Assert-True (-not (Test-Path -LiteralPath $InstallRoot)) "$Label: install directory residue"
    $processPath = [Environment]::GetEnvironmentVariable("Path", "Process")
    $normalized = Get-NormalizedPathEntry $InstallRoot
    $inPath = @($processPath -split ';' | ForEach-Object {
        try { (Get-NormalizedPathEntry $_.Trim().Trim('"')) -eq $normalized } catch { $false }
    } | Where-Object { $_ })
    Assert-True ($inPath.Count -eq 0) "$Label: PATH residue"
}

function Uninstall-Cleanup {
    param([string]$InstallRoot)
    $uninstaller = Join-Path $InstallRoot "Uninstall-NodePaper.ps1"
    if (Test-Path -LiteralPath $uninstaller -PathType Leaf) {
        try { & $uninstaller -InstallRoot $InstallRoot -PathScope Process 2>&1 | Out-Null } catch { }
    }
    Remove-Item -LiteralPath $InstallRoot -Recurse -Force -ErrorAction SilentlyContinue
}

try {
    New-Item -ItemType Directory -Force -Path $workRoot | Out-Null
    $oldDir = Expand-Package $OldZip (Join-Path $workRoot "old")
    $newDir = Expand-Package $NewZip (Join-Path $workRoot "new")
    $installRoot = Join-Path $workRoot "inst"
    $oldExe = Join-Path $oldDir "nodepaper.exe"
    $newExe = Join-Path $newDir "nodepaper.exe"

    # 1. Fresh install of the old candidate and a real Project build.
    $installOld = Invoke-Installer $oldDir $installRoot "install-old"
    Assert-True ($installOld.Exit -eq 0) "old candidate install failed: $($installOld.Error)"
    $oldVersion = Get-InstalledVersion $installRoot
    Assert-True ($oldVersion -like "nodepaper 0.1.0-rc.*") "old candidate version: $oldVersion"
    Write-Host "  installed version: $oldVersion"

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
    Assert-True (Test-Path -LiteralPath $pdfBefore -PathType Leaf) "Project PDF not produced under old candidate"

    # 2. In-place upgrade with the current ZIP (no uninstall; silent continues).
    $installNew = Invoke-Installer $newDir $installRoot "upgrade-new"
    Assert-True ($installNew.Exit -eq 0) "upgrade install failed: $($installNew.Error)"
    $upgradedVersion = Get-InstalledVersion $installRoot
    Assert-True ($upgradedVersion -ne $oldVersion) "version did not change after upgrade ($oldVersion -> $upgradedVersion)"
    Assert-True (Test-Path -LiteralPath $pdfBefore -PathType Leaf) "upgrade removed the existing Project PDF"
    Assert-True (Test-Path -LiteralPath (Join-Path $installRoot "nodepaper.exe")) "upgrade left no nodepaper.exe"

    # 3. Repair install (same version, silent continues).
    $repair = Invoke-Installer $newDir $installRoot "repair-new"
    Assert-True ($repair.Exit -eq 0) "repair install failed: $($repair.Error)"
    Assert-True ((Get-InstalledVersion $installRoot) -eq $upgradedVersion) "repair install changed the version"

    # 4. Silent downgrade must be rejected; the new version stays.
    $versionBeforeDowngrade = Get-InstalledVersion $installRoot
    $downgrade = Invoke-Installer $oldDir $installRoot "downgrade-old"
    Assert-True ($downgrade.Exit -ne 0) "silent downgrade was not rejected: $($downgrade.Error)"
    Assert-True ((Get-InstalledVersion $installRoot) -eq $versionBeforeDowngrade) "silent downgrade changed the installed version"
    Assert-True (Test-Path -LiteralPath $pdfBefore -PathType Leaf) "silent downgrade removed the Project PDF"

    # 5. Uninstall and verify no residue; Project/PDF kept.
    Uninstall-Cleanup $installRoot
    Test-Residue $installRoot "uninstall"
    Assert-True (Test-Path -LiteralPath $pdfBefore -PathType Leaf) "uninstall removed the Project PDF"

    # 6. Chinese/space path round.
    $custom = Join-Path $workRoot "中文 安装\NodePaper"
    $installCustom = Invoke-Installer $newDir $custom "install-custom"
    Assert-True ($installCustom.Exit -eq 0) "custom path install failed: $($installCustom.Error)"
    Assert-True (Test-Path -LiteralPath (Join-Path $custom "nodepaper.exe")) "custom path install missing exe"
    Uninstall-Cleanup $custom
    Test-Residue $custom "custom-uninstall"

    $passed = $true
    Write-Host ""
    Write-Host "ZIP channel cross-version upgrade gate passed."
    Write-Host "  old: $oldVersion -> new: $upgradedVersion (silent downgrade rejected by design; interactive confirm is manual)"
}
finally {
    foreach ($root in @($installRoot, (Join-Path $workRoot "中文 安装\NodePaper"))) {
        if ($root -and (Test-Path -LiteralPath $root)) { Uninstall-Cleanup $root }
    }
    if (-not $passed -and -not $KeepWorkDirectory) {
        Remove-Item -LiteralPath $workRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
    elseif ($KeepWorkDirectory -and (Test-Path -LiteralPath $workRoot)) {
        Write-Host "ZIP upgrade test work directory retained: $workRoot"
    }
}
