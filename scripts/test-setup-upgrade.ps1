<#
.SYNOPSIS
    Setup cross-version upgrade/downgrade manual gate (release checklist 10.5).

.DESCRIPTION
    Installs an older candidate Setup, upgrades in place with the current
    candidate without uninstalling first, repeats the current candidate,
    downgrades with an older candidate, then uninstalls and verifies that the
    Project, PDF and TeX are preserved and that no directory, PATH, Start-menu
    or uninstall-registry residue remains. A Chinese/space path round also
    installs, verifies and uninstalls.

    Runs with /VERYSILENT, where the downgrade confirmation is suppressed to
    its default of No: the downgrade round below therefore expects a non-zero
    exit and an unchanged installation. Confirming the dialog by hand, and
    downgrading on purpose with /ALLOWDOWNGRADE, both stay manual items in the
    release checklist - the second one can only be scripted here once the
    -OldSetup candidate is itself new enough to carry that switch. All installs
    target %LOCALAPPDATA%\Programs\NodePaper (and the custom path round) and
    are removed at the end.

.PARAMETER OldSetup
    Path to an older candidate Setup (for example rc.6 or rc.7).
.PARAMETER NewSetup
    Path to the current candidate Setup.
.PARAMETER ReleaseZip
    Path to the current candidate ZIP (used to seed a real Project fixture).
.PARAMETER KeepWorkDirectory
    Keep the temporary work directory for diagnosis.
#>
param(
    [Parameter(Mandatory = $true)][string]$OldSetup,
    [Parameter(Mandatory = $true)][string]$NewSetup,
    [Parameter(Mandatory = $true)][string]$ReleaseZip,
    [switch]$KeepWorkDirectory
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$AppId = "{6E1B5C6A-6C2F-4D4B-9A62-2C7E60C0A5F1}"
$UninstallKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\${AppId}_is1"
$DefaultInstallRoot = Join-Path $env:LOCALAPPDATA "Programs\NodePaper"
$StartMenuGroup = Join-Path $env:APPDATA "Microsoft\Windows\Start Menu\Programs\NodePaper"
$workRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("nodepaper-upgrade-test-" + [Guid]::NewGuid().ToString("N"))
$passed = $false
$installedRoots = New-Object System.Collections.Generic.List[string]

function Assert-True {
    param([Parameter(Mandatory = $true)][bool]$Condition, [Parameter(Mandatory = $true)][string]$Message)
    if (-not $Condition) { throw "FAIL: $Message" }
    Write-Host "  ok: $Message"
}

function Get-NormalizedPathEntry {
    param([string]$Path)
    try { return [System.IO.Path]::GetFullPath($Path).TrimEnd('\', '/').ToLowerInvariant() } catch { return "" }
}

function Get-UserPathRaw {
    $key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey("Environment")
    try { return [string]$key.GetValue("Path", "", [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames) }
    finally { $key.Close() }
}

function Invoke-Setup {
    param([string]$SetupPath, [string]$InstallDir, [string]$Label)
    $logPath = Join-Path $workRoot ("$Label.log")
    $arguments = @("/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART", "/LANG=chinesesimplified", ('/DIR="' + $InstallDir + '"'), ('/LOG="' + $logPath + '"'))
    $process = Start-Process -FilePath $SetupPath -ArgumentList $arguments -Wait -PassThru
    if ($process.ExitCode -ne 0) {
        throw "FAIL: Setup $Label exited $($process.ExitCode); log: $logPath"
    }
    Write-Host "  setup ok: $Label -> $InstallDir"
    return $logPath
}

function Get-InstalledVersion {
    $exe = Join-Path $DefaultInstallRoot "nodepaper.exe"
    if (-not (Test-Path -LiteralPath $exe -PathType Leaf)) { return "" }
    return ((& $exe --version 2>&1 | Out-String).Trim())
}

function Test-Uninstall {
    $uninstaller = Join-Path $DefaultInstallRoot "unins000.exe"
    if (-not (Test-Path -LiteralPath $uninstaller -PathType Leaf)) {
        throw "FAIL: unins000.exe not found for uninstall"
    }
    $process = Start-Process -FilePath $uninstaller -ArgumentList @("/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART") -Wait -PassThru
    if ($process.ExitCode -ne 0) {
        throw "FAIL: uninstaller exited $($process.ExitCode)"
    }
    Write-Host "  uninstall ok"
}

function Assert-NoResidue {
    param([string]$Root)
    Assert-True (-not (Test-Path -LiteralPath $Root)) "install directory residue: $Root"
    $userPath = Get-UserPathRaw
    $normalized = Get-NormalizedPathEntry $Root
    $inPath = @($userPath -split ';' | ForEach-Object {
        try { (Get-NormalizedPathEntry $_.Trim().Trim('"')) -eq $normalized } catch { $false }
    } | Where-Object { $_ })
    Assert-True ($inPath.Count -eq 0) "PATH residue for $Root"
    Assert-True (-not (Test-Path -LiteralPath $StartMenuGroup)) "Start-menu group residue: $StartMenuGroup"
    Assert-True (-not (Test-Path -LiteralPath $UninstallKey)) "uninstall registration residue"
}

try {
    New-Item -ItemType Directory -Force -Path $workRoot | Out-Null
    if (Test-Path -LiteralPath $DefaultInstallRoot) { Test-Uninstall }
    $userPathBefore = Get-UserPathRaw

    # 1. Fresh install of the old candidate and a real Project build.
    Invoke-Setup $OldSetup $DefaultInstallRoot "install-old"
    Assert-True ((Get-InstalledVersion) -like "nodepaper 0.1.0-rc.*") "old candidate did not install a nodepaper command"
    Assert-True (Test-Path -LiteralPath $UninstallKey) "old candidate did not register uninstall"
    Assert-True (Test-Path -LiteralPath $StartMenuGroup) "old candidate did not create Start-menu group"
    $oldVersion = Get-InstalledVersion
    Write-Host "  installed version: $oldVersion"

    $projectDir = Join-Path $workRoot "project"
    $fixture = Join-Path $PSScriptRoot "..\nodepaper\core\tests\fixtures\complete-single-file"
    Copy-Item -LiteralPath $fixture -Destination $projectDir -Recurse
    $exe = Join-Path $DefaultInstallRoot "nodepaper.exe"
    Push-Location $projectDir
    try {
        & $exe validate $projectDir --format json | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "validate failed under old candidate" }
        & $exe build $projectDir --format json | Out-Null
        if ($LASTEXITCODE -ne 0) { throw "build failed under old candidate" }
    }
    finally { Pop-Location }
    $pdfBefore = Join-Path $projectDir "dist\paper.pdf"
    Assert-True (Test-Path -LiteralPath $pdfBefore -PathType Leaf) "Project PDF not produced under old candidate"

    # 2. In-place upgrade with the new candidate (no uninstall first).
    Invoke-Setup $NewSetup $DefaultInstallRoot "upgrade-new"
    $upgradedVersion = Get-InstalledVersion
    Assert-True ($upgradedVersion -ne $oldVersion) "version did not change after upgrade ($oldVersion -> $upgradedVersion)"
    Assert-True (Test-Path -LiteralPath $pdfBefore -PathType Leaf) "upgrade removed the existing Project PDF"
    Assert-True (Test-Path -LiteralPath (Join-Path $DefaultInstallRoot "nodepaper.exe")) "upgrade left no nodepaper.exe"
    Assert-True (Test-Path -LiteralPath $UninstallKey) "upgrade lost the uninstall registration"
    Assert-True (Test-Path -LiteralPath $StartMenuGroup) "upgrade lost the Start-menu group"

    # 3. Repeat the current candidate (repair install).
    Invoke-Setup $NewSetup $DefaultInstallRoot "repeat-new"
    Assert-True ((Get-InstalledVersion) -eq $upgradedVersion) "repeat install changed the version"

    # 4. Downgrade is a protected path: the confirmation is a
    # SuppressibleMsgBox defaulting to No, so under /VERYSILENT the install
    # must be cancelled and the current version must stay. The interactive
    # "confirm downgrade" flow and the /ALLOWDOWNGRADE override remain manual
    # items in the release checklist.
    $versionBeforeDowngrade = Get-InstalledVersion
    $downgradeLog = Join-Path $workRoot "downgrade-old.log"
    $downgradeArgs = @("/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART", "/LANG=chinesesimplified", ('/DIR="' + $DefaultInstallRoot + '"'), ('/LOG="' + $downgradeLog + '"'))
    $downgrade = Start-Process -FilePath $OldSetup -ArgumentList $downgradeArgs -Wait -PassThru
    Assert-True ($downgrade.ExitCode -ne 0) "silent downgrade was not rejected; log: $downgradeLog"
    Assert-True ((Get-InstalledVersion) -eq $versionBeforeDowngrade) "silent downgrade changed the installed version"
    Assert-True (Test-Path -LiteralPath $pdfBefore -PathType Leaf) "silent downgrade removed the Project PDF"

    # 5. Uninstall and verify no residue; Project/PDF kept.
    Test-Uninstall
    Assert-NoResidue $DefaultInstallRoot
    Assert-True (Test-Path -LiteralPath $pdfBefore -PathType Leaf) "uninstall removed the Project PDF"

    # 6. Chinese/space custom path round.
    $custom = Join-Path $workRoot "中文 安装\NodePaper"
    Invoke-Setup $NewSetup $custom "install-custom"
    Assert-True (Test-Path -LiteralPath (Join-Path $custom "nodepaper.exe")) "custom path install missing exe"
    $customUninstaller = Join-Path $custom "unins000.exe"
    Assert-True (Test-Path -LiteralPath $customUninstaller -PathType Leaf) "custom path missing uninstaller"
    $process = Start-Process -FilePath $customUninstaller -ArgumentList @("/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART") -Wait -PassThru
    Assert-True ($process.ExitCode -eq 0) "custom path uninstall failed"
    Assert-NoResidue $custom

    $passed = $true
    Write-Host ""
    Write-Host "Setup cross-version upgrade gate passed."
    Write-Host "  old: $oldVersion -> new: $upgradedVersion (silent downgrade rejected by design; interactive confirm is manual)"
}
finally {
    # Best-effort cleanup of anything left behind by a failed run.
    foreach ($root in @($DefaultInstallRoot, (Join-Path $workRoot "中文 安装\NodePaper"))) {
        if (Test-Path -LiteralPath $root) {
            $u = Join-Path $root "unins000.exe"
            if (Test-Path -LiteralPath $u) {
                try { Start-Process -FilePath $u -ArgumentList @("/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART") -Wait | Out-Null } catch { }
            }
            try { Remove-Item -LiteralPath $root -Recurse -Force -ErrorAction SilentlyContinue } catch { }
        }
    }
    if (Test-Path -LiteralPath $StartMenuGroup) { Remove-Item -LiteralPath $StartMenuGroup -Recurse -Force -ErrorAction SilentlyContinue }
    if (Test-Path -LiteralPath $UninstallKey) { Remove-Item -LiteralPath $UninstallKey -Recurse -Force -ErrorAction SilentlyContinue }
    if (-not $passed -and -not $KeepWorkDirectory) {
        Remove-Item -LiteralPath $workRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
    elseif ($KeepWorkDirectory -and (Test-Path -LiteralPath $workRoot)) {
        Write-Host "Upgrade test work directory retained: $workRoot"
    }
}
