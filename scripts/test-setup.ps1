<#
.SYNOPSIS
    Mechanically validate the Windows x64 NodePaper Setup channel.

.DESCRIPTION
    Exercises the installation, start, upgrade/repair, rollback and uninstall
    contracts of NodePaper-Setup-<version>-windows-x64.exe without any
    administrator right:

      1. Setup / ZIP / manifest identity: same version, same source commit,
         same payload file hashes
      2. install into a custom directory containing Chinese characters and a
         space, verify installed payload bytes
      3. installed user Path entry, Start-menu and desktop shortcut contract
      4. `nodepaper --version` from an arbitrary directory using the installed
         Path entry, and the persistent-window launcher command line
      5. non-TTY, pipeline, JSON and CI behaviour of the installed executable
      6. repeat (repair) installation of the same version
      7. uninstall after deleting the downloaded Setup, using only the
         uninstaller registered for the current user
      8. no residue: directory, exact Path entry, shortcuts, uninstall entry
      9. unrelated Path entries, Projects, PDFs and TeX are preserved
     10. results are written to setup-test-results.json

    Silent switches are used so the flow is reproducible; the interactive
    wizard, SmartScreen behaviour, Start-menu double-click and real user
    journeys remain manual gates.

.PARAMETER Setup
    Path to NodePaper-Setup-<version>-windows-x64.exe.
.PARAMETER ReleaseZip
    Matching nodepaper-<version>-windows-x64.zip for cross-channel identity.
.PARAMETER ManifestPath
    Release manifest recording both channels.
.PARAMETER ResultsPath
    Output JSON. Default: setup-test-results.json next to the Setup.
.PARAMETER KeepWorkDirectory
    Keep the temporary work directory for diagnosis.
#>
param(
    [Parameter(Mandatory = $true)]
    [string]$Setup,
    [Parameter(Mandatory = $true)]
    [string]$ReleaseZip,
    [Parameter(Mandatory = $true)]
    [string]$ManifestPath,
    [string]$ResultsPath = "",
    [switch]$KeepWorkDirectory
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

$script:WorkRoot = ""
$script:Passed = $false
$script:InstallRoot = ""
$script:Checks = New-Object System.Collections.Generic.List[object]

$UninstallKey = "HKCU:\Software\Microsoft\Windows\CurrentVersion\Uninstall\{6E1B5C6A-6C2F-4D4B-9A62-2C7E60C0A5F1}_is1"

function Assert-True {
    param([Parameter(Mandatory = $true)][bool]$Condition, [Parameter(Mandatory = $true)][string]$Message)
    $result = "fail"
    if ($Condition) { $result = "pass" }
    $script:Checks.Add([ordered]@{ check = $Message; result = $result }) | Out-Null
    if (-not $Condition) {
        throw "FAIL: $Message"
    }
    Write-Host "  ok: $Message"
}

function Get-NormalizedPathEntry {
    param([string]$Path)
    try { return [System.IO.Path]::GetFullPath($Path).TrimEnd('\', '/').ToLowerInvariant() } catch { return "" }
}

function Get-UserPathRaw {
    # The raw (unexpanded) value is the ground truth: an installer must not
    # rewrite, expand or re-encode unrelated entries.
    $key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey("Environment")
    try {
        return [string]$key.GetValue("Path", "", [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
    }
    finally { $key.Close() }
}

function Get-UserPathKind {
    $key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey("Environment")
    try {
        if ($null -eq $key.GetValue("Path", $null)) { return [Microsoft.Win32.RegistryValueKind]::ExpandString }
        return $key.GetValueKind("Path")
    }
    finally { $key.Close() }
}

function Set-UserPathRaw {
    param([string]$Value, [Microsoft.Win32.RegistryValueKind]$Kind)
    $key = [Microsoft.Win32.Registry]::CurrentUser.OpenSubKey("Environment", $true)
    try { $key.SetValue("Path", $Value, $Kind) }
    finally { $key.Close() }
}

function Get-UserPathEntries {
    $value = Get-UserPathRaw
    if ([string]::IsNullOrEmpty($value)) { return @() }
    return @($value -split ';' | Where-Object { -not [string]::IsNullOrWhiteSpace($_) })
}

function Test-UserPathContains {
    param([string]$Directory)
    $wanted = Get-NormalizedPathEntry $Directory
    foreach ($entry in Get-UserPathEntries) {
        if ((Get-NormalizedPathEntry ($entry.Trim().Trim('"'))) -eq $wanted) { return $true }
    }
    return $false
}

function Invoke-Setup {
    param([string]$SetupPath, [string]$InstallDirectory, [string]$LogPath = "", [string[]]$ExtraArguments = @())
    # Quote explicitly: install directories legitimately contain spaces and
    # Chinese characters.
    # The language is pinned so the mechanical expectations (Chinese shortcut
    # names) are deterministic on any locale; the English wizard remains a
    # manual gate.
    $arguments = @("/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART", "/LANG=chinesesimplified", ('/DIR="' + $InstallDirectory + '"'))
    if (-not [string]::IsNullOrWhiteSpace($LogPath)) {
        $arguments += ('/LOG="' + $LogPath + '"')
    }
    $arguments += $ExtraArguments
    $process = Start-Process -FilePath $SetupPath -ArgumentList $arguments -Wait -PassThru
    $exitCode = 0
    try { $exitCode = $process.ExitCode } catch { $exitCode = 0 }
    return $exitCode
}

function Get-ShortcutTarget {
    param([string]$ShortcutPath)
    $shell = New-Object -ComObject WScript.Shell
    try {
        $shortcut = $shell.CreateShortcut($ShortcutPath)
        return [ordered]@{ target = [string]$shortcut.TargetPath; arguments = [string]$shortcut.Arguments; workingDirectory = [string]$shortcut.WorkingDirectory }
    }
    finally {
        [void][System.Runtime.InteropServices.Marshal]::ReleaseComObject($shell)
    }
}

# ---------- inputs -----------------------------------------------------------

$setupPath = (Resolve-Path -LiteralPath $Setup).Path
$setupName = [System.IO.Path]::GetFileName($setupPath)
if ($setupName -notmatch '^NodePaper-Setup-(.+)-windows-x64\.exe$') {
    throw "Setup file name does not match NodePaper-Setup-<version>-windows-x64.exe: $setupName"
}
$version = $Matches[1]
$zipPath = (Resolve-Path -LiteralPath $ReleaseZip).Path
if ([System.IO.Path]::GetFileName($zipPath) -ne "nodepaper-$version-windows-x64.zip") {
    throw "ZIP file name does not match the Setup version: $([System.IO.Path]::GetFileName($zipPath))"
}
$manifestFullPath = (Resolve-Path -LiteralPath $ManifestPath).Path
$manifest = Get-Content -LiteralPath $manifestFullPath -Raw -Encoding UTF8 | ConvertFrom-Json
if ([string]::IsNullOrWhiteSpace($ResultsPath)) {
    $ResultsPath = Join-Path (Split-Path -Parent $setupPath) "setup-test-results.json"
}

$setupSHA256 = (Get-FileHash -LiteralPath $setupPath -Algorithm SHA256).Hash.ToLowerInvariant()
$zipSHA256 = (Get-FileHash -LiteralPath $zipPath -Algorithm SHA256).Hash.ToLowerInvariant()
$setupSignature = [string](Get-AuthenticodeSignature -LiteralPath $setupPath).Status

if (Test-Path -LiteralPath $UninstallKey) {
    throw "FAIL: NodePaper is already installed for this user. Uninstall it before running the Setup test."
}
$originalUserPath = Get-UserPathRaw
$originalUserPathKind = Get-UserPathKind
# Regression probe: an unrelated entry with non-ASCII characters and a space
# must survive install and uninstall byte for byte.
$probePathEntry = "C:\NodePaper 回归 探针\bin"
Set-UserPathRaw (($originalUserPath.TrimEnd(';')) + ';' + $probePathEntry) $originalUserPathKind
$baselineUserPath = Get-UserPathRaw
$originalPathEntries = @(Get-UserPathEntries | ForEach-Object { Get-NormalizedPathEntry ($_.Trim().Trim('"')) } | Sort-Object)

try {
    $script:WorkRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("nodepaper-setup-test-" + [Guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Force -Path $script:WorkRoot | Out-Null

    # ---------- 1. cross-channel identity -----------------------------------

    Write-Host "Checking Setup / ZIP / manifest identity..."
    Assert-True ([string]$manifest.version -eq $version) "manifest version matches the Setup version"
    $zipChannel = $null
    $setupChannel = $null
    foreach ($channel in @($manifest.channels)) {
        if ([string]$channel.channel -eq "portable-zip") { $zipChannel = $channel }
        if ([string]$channel.channel -eq "setup-exe") { $setupChannel = $channel }
    }
    Assert-True ($null -ne $zipChannel -and $null -ne $setupChannel) "manifest records both the portable-zip and setup-exe channels"
    Assert-True ([string]$setupChannel.sha256 -eq $setupSHA256) "manifest records the actual Setup SHA-256"
    Assert-True ([string]$zipChannel.sha256 -eq $zipSHA256) "manifest records the actual ZIP SHA-256"
    Assert-True ([string]$setupChannel.signatureStatus -eq $setupSignature) "manifest records the actual Authenticode status ($setupSignature)"
    Assert-True (-not [bool]$setupChannel.requiresAdministrator) "manifest records a current-user installation"

    $zipExtract = Join-Path $script:WorkRoot "zip"
    Expand-Archive -LiteralPath $zipPath -DestinationPath $zipExtract -Force
    $zipPayload = (Get-ChildItem -LiteralPath $zipExtract -Directory)[0].FullName
    $zipPayloadManifest = Get-Content -LiteralPath (Join-Path $zipPayload "payload-manifest.json") -Raw -Encoding UTF8 | ConvertFrom-Json
    Assert-True ([string]$zipPayloadManifest.version -eq $version) "ZIP payload manifest version matches"
    Assert-True ([string]$zipPayloadManifest.sourceCommit -eq [string]$manifest.sourceCommit) "ZIP payload and manifest share one source commit"
    $zipPayloadManifestSHA256 = (Get-FileHash -LiteralPath (Join-Path $zipPayload "payload-manifest.json") -Algorithm SHA256).Hash.ToLowerInvariant()
    Assert-True ([string]$setupChannel.payloadManifestSHA256 -eq $zipPayloadManifestSHA256) "Setup channel records the ZIP payload manifest hash (same payload)"

    # ---------- 2. install into a Chinese path containing a space -----------

    $script:InstallRoot = Join-Path $script:WorkRoot "安装 目录\NodePaper"
    Write-Host "Installing to $script:InstallRoot"
    $installLog = Join-Path $script:WorkRoot "install.log"
    $exitCode = Invoke-Setup $setupPath $script:InstallRoot $installLog
    Assert-True ($exitCode -eq 0) "Setup exited with 0 for a Chinese custom directory containing a space"
    Assert-True (Test-Path -LiteralPath (Join-Path $script:InstallRoot "nodepaper.exe") -PathType Leaf) "nodepaper.exe was installed"

    # Setup installs the payload minus the ZIP channel's own registration
    # scripts (installer/windows/nodepaper.iss Excludes, list owned by
    # scripts/build-setup.ps1). The payload manifest describes the ZIP, so it
    # still lists them; under {app} they must be absent. A file named
    # Uninstall-NodePaper.ps1 inside a Setup installation directory invites the
    # user to run it, and it strips the Path entry while the installation, its
    # Start-menu entries and its entry in Settings all stay behind.
    $setupExcludedPayloadFiles = @("Install-NodePaper.ps1", "Uninstall-NodePaper.ps1")
    $installedManifest = Get-Content -LiteralPath (Join-Path $script:InstallRoot "payload-manifest.json") -Raw -Encoding UTF8 | ConvertFrom-Json
    foreach ($entry in @($installedManifest.files)) {
        $manifestPath = ([string]$entry.path) -replace '\\', '/'
        $relative = $manifestPath -replace '/', '\'
        $installedFile = Join-Path $script:InstallRoot $relative
        if ($setupExcludedPayloadFiles -contains $manifestPath) {
            if (Test-Path -LiteralPath $installedFile) {
                throw "FAIL: Setup installed a ZIP-channel script it must exclude: $relative"
            }
            continue
        }
        if (-not (Test-Path -LiteralPath $installedFile -PathType Leaf)) {
            throw "FAIL: installed payload file missing: $relative"
        }
        $actual = (Get-FileHash -LiteralPath $installedFile -Algorithm SHA256).Hash.ToLowerInvariant()
        if ($actual -ne ([string]$entry.sha256).ToLowerInvariant()) {
            throw "FAIL: installed payload byte mismatch: $relative"
        }
    }
    Assert-True $true "every installed payload file matches the payload manifest bytes"
    Assert-True $true "the ZIP channel's own scripts were not installed: $($setupExcludedPayloadFiles -join ', ')"
    $installedExtra = @(Get-ChildItem -LiteralPath $script:InstallRoot -Recurse -File | ForEach-Object {
        $_.FullName.Substring($script:InstallRoot.TrimEnd('\').Length).TrimStart('\') -replace '\\', '/'
    } | Where-Object { $_ -notmatch '^unins\d+\.(exe|dat|msg)$' -and $_ -ne "payload-manifest.json" })
    $payloadPaths = @(@($installedManifest.files) | ForEach-Object { ([string]$_.path) -replace '\\', '/' } |
        Where-Object { $setupExcludedPayloadFiles -notcontains $_ })
    foreach ($relative in $installedExtra) {
        Assert-True ($payloadPaths -contains $relative) "installed file belongs to the payload Setup installs: $relative"
    }
    Assert-True (Test-Path -LiteralPath (Join-Path $script:InstallRoot "unins000.exe") -PathType Leaf) "a standalone uninstaller was stored in the installation directory"

    # ---------- 3. Path, Start-menu and desktop contract --------------------

    Assert-True (Test-UserPathContains $script:InstallRoot) "the installation directory was registered on the user Path"
    $startMenuDir = Join-Path ([Environment]::GetFolderPath("Programs")) "NodePaper"
    $launcher = Join-Path $startMenuDir "NodePaper.lnk"
    Assert-True (Test-Path -LiteralPath $launcher -PathType Leaf) "Start-menu NodePaper entry was created"
    Assert-True (Test-Path -LiteralPath (Join-Path $startMenuDir "卸载 NodePaper.lnk") -PathType Leaf) "Start-menu uninstall entry was created"
    Assert-True (-not (Test-Path -LiteralPath (Join-Path ([Environment]::GetFolderPath("Desktop")) "NodePaper.lnk"))) "no desktop shortcut is created by default"
    $shortcut = Get-ShortcutTarget $launcher
    Assert-True ($shortcut.target -match '(?i)cmd\.exe$') "Start-menu entry opens a persistent command-line window"
    Assert-True ($shortcut.arguments -match '(?i)^/K\s') "Start-menu entry keeps the window open after onboarding (/K)"
    Assert-True ($shortcut.arguments -match '(?i)nodepaper\.exe') "Start-menu entry runs the installed nodepaper.exe"

    Assert-True (Test-Path -LiteralPath $UninstallKey) "the Windows uninstall entry was registered for the current user"
    $uninstallEntry = Get-ItemProperty -LiteralPath $UninstallKey
    $registeredUninstaller = ([string]$uninstallEntry.UninstallString).Trim('"')
    Assert-True ((Get-NormalizedPathEntry $registeredUninstaller) -eq (Get-NormalizedPathEntry (Join-Path $script:InstallRoot "unins000.exe"))) "the uninstall entry points at the installed uninstaller, not the downloaded Setup"
    Assert-True ([string]$uninstallEntry.DisplayVersion -eq $version) "the uninstall entry records the installed version"

    # ---------- 4. installed command and launcher command line --------------

    $installedExe = Join-Path $script:InstallRoot "nodepaper.exe"
    $arbitrary = Join-Path $script:WorkRoot "任意 工作目录"
    New-Item -ItemType Directory -Force -Path $arbitrary | Out-Null
    $originalProcessPath = [Environment]::GetEnvironmentVariable("Path", "Process")
    try {
        # A new terminal combines the machine and user Path. The installed
        # command must be reachable there.
        [Environment]::SetEnvironmentVariable("Path", ([Environment]::GetEnvironmentVariable("Path", "Machine") + ";" + [Environment]::GetEnvironmentVariable("Path", "User")), "Process")
        $resolutions = @(Get-Command nodepaper -CommandType Application -All -ErrorAction SilentlyContinue | ForEach-Object { (Resolve-Path -LiteralPath $_.Source).Path })
        Assert-True ($resolutions -contains (Resolve-Path -LiteralPath $installedExe).Path) "a new terminal reaches the installed nodepaper through the user Path"
        if ($resolutions.Count -gt 1) {
            Write-Host "  note: this machine has $($resolutions.Count) nodepaper.exe on the Path; the first is $($resolutions[0])"
        }
        # Version and onboarding are checked with the installation directory
        # first, so an unrelated pre-existing NodePaper installation on the
        # same machine cannot mask the result.
        [Environment]::SetEnvironmentVariable("Path", ($script:InstallRoot + ";" + [Environment]::GetEnvironmentVariable("Path", "Machine")), "Process")
        Push-Location $arbitrary
        try {
            $reported = (& nodepaper --version 2>&1 | Out-String).Trim()
            Assert-True ($reported -eq "nodepaper $version") "the installed command reports the release version from any directory ($reported)"
            $onboarding = (& nodepaper 2>&1 | Out-String)
            Assert-True ($LASTEXITCODE -eq 0) "no-argument onboarding exits with 0 in a non-TTY context"
            Assert-True ($onboarding.Contains("nodepaper init")) "no-argument onboarding guides the next step"
            Assert-True (-not ($onboarding -match '(?i)press enter')) "no-argument onboarding never waits for input when the console is not owned by nodepaper.exe"
            Assert-True (-not (Test-Path -LiteralPath (Join-Path $arbitrary "nodepaper.yaml"))) "onboarding did not modify the current directory"
        }
        finally {
            Pop-Location
        }
    }
    finally {
        [Environment]::SetEnvironmentVariable("Path", $originalProcessPath, "Process")
    }

    # The Start-menu launcher command line must work verbatim, including the
    # Chinese install path and the space in it. cmd.exe is invoked with the
    # same quoting the shortcut uses, only with /C instead of /K so the probe
    # terminates.
    $launcherOutput = Join-Path $script:WorkRoot "launcher-probe.txt"
    $launcherArguments = ($shortcut.arguments -replace '(?i)^/K', '/C') + " --version"
    $process = Start-Process -FilePath $env:ComSpec -ArgumentList $launcherArguments -Wait -PassThru -WindowStyle Hidden -RedirectStandardOutput $launcherOutput
    $launcherExit = 0
    try { $launcherExit = $process.ExitCode } catch { $launcherExit = 0 }
    $launcherText = (Get-Content -LiteralPath $launcherOutput -Raw -Encoding UTF8).Trim()
    Assert-True ($launcherExit -eq 0 -and $launcherText -eq "nodepaper $version") "the Start-menu command line runs the installed executable verbatim from a Chinese path with a space"

    # File Explorer starts nodepaper.exe in a console window created for that
    # process alone. The output must stay readable instead of flashing away.
    # Windows tears a console down asynchronously, so the hidden cmd.exe probe
    # above can still be attached when the next process starts -- it then joins
    # that console, GetConsoleProcessList returns more than one, and the hold is
    # correctly skipped. Measured: without a pause this assertion failed every
    # time and passed every time with one, while the same binary held in eleven
    # isolated runs. Retrying rather than sleeping a fixed amount keeps a real
    # regression failing: a build that never holds fails on the last attempt.
    $doubleClick = $null
    foreach ($attempt in 1..3) {
        if ($null -ne $doubleClick -and -not $doubleClick.HasExited) { break }
        Start-Sleep -Seconds 2
        $doubleClick = Start-Process -FilePath $installedExe -WorkingDirectory $arbitrary -PassThru
        Start-Sleep -Seconds 4
        $doubleClick.Refresh()
    }
    try {
        Assert-True (-not $doubleClick.HasExited) "a double-clicked nodepaper.exe keeps its own console window open instead of flashing away"
    }
    finally {
        if (-not $doubleClick.HasExited) {
            Stop-Process -Id $doubleClick.Id -Force -ErrorAction SilentlyContinue
        }
    }
    Assert-True (-not (Test-Path -LiteralPath (Join-Path $arbitrary "nodepaper.yaml"))) "the double-clicked run installed nothing and created no Project"

    $jsonText = (& $installedExe doctor --format json 2>$null | Out-String)
    Assert-True ($jsonText.TrimStart().StartsWith("{")) "installed executable still emits pure JSON on a pipe"
    $pipedText = ("" | & $installedExe 2>&1 | Out-String)
    Assert-True (-not ($pipedText -match '(?i)press enter')) "piped stdin never triggers the double-click console hold"
    $previousCI = $env:CI
    try {
        $env:CI = "true"
        $ciText = (& $installedExe 2>&1 | Out-String)
        Assert-True (-not ($ciText -match '(?i)press enter')) "CI never triggers the double-click console hold"
    }
    finally {
        if ($null -eq $previousCI) { Remove-Item Env:CI -ErrorAction SilentlyContinue } else { $env:CI = $previousCI }
    }

    # ---------- 5. failed installation rollback -----------------------------

    Write-Host "Attempting an installation that must fail and roll back..."
    $blocker = Join-Path $script:WorkRoot "blocker.txt"
    Set-Content -LiteralPath $blocker -Value "not a directory" -Encoding ASCII
    $failExit = Invoke-Setup $setupPath (Join-Path $blocker "NodePaper") (Join-Path $script:WorkRoot "install-failure.log")
    Assert-True ($failExit -ne 0) "an impossible installation directory fails instead of half-installing"
    Assert-True (-not (Test-Path -LiteralPath (Join-Path $blocker "NodePaper"))) "the failed installation left no directory behind"
    Assert-True (Test-Path -LiteralPath $installedExe -PathType Leaf) "the failed installation preserved the previous installation"
    Assert-True (Test-UserPathContains $script:InstallRoot) "the failed installation preserved the existing Path entry"
    $entryAfterFailure = Get-ItemProperty -LiteralPath $UninstallKey
    Assert-True ((Get-NormalizedPathEntry (([string]$entryAfterFailure.UninstallString).Trim('"'))) -eq (Get-NormalizedPathEntry (Join-Path $script:InstallRoot "unins000.exe"))) "the failed installation preserved the previous uninstall registration"

    # ---------- 6. repeat / repair installation -----------------------------

    Write-Host "Repeating the same-version installation..."
    $exitCode = Invoke-Setup $setupPath $script:InstallRoot (Join-Path $script:WorkRoot "install-repeat.log")
    Assert-True ($exitCode -eq 0) "repeating the same-version installation succeeds"
    Assert-True (Test-Path -LiteralPath $installedExe -PathType Leaf) "the repeated installation kept nodepaper.exe"
    $pathEntryCount = 0
    foreach ($entry in Get-UserPathEntries) {
        if ((Get-NormalizedPathEntry ($entry.Trim().Trim('"'))) -eq (Get-NormalizedPathEntry $script:InstallRoot)) { $pathEntryCount++ }
    }
    Assert-True ($pathEntryCount -eq 1) "the repeated installation did not duplicate the Path entry"

    # ---------- 7. uninstall after deleting the downloaded Setup ------------

    $downloadedCopy = Join-Path $script:WorkRoot "下载\NodePaper-Setup-copy.exe"
    New-Item -ItemType Directory -Force -Path (Split-Path -Parent $downloadedCopy) | Out-Null
    Copy-Item -LiteralPath $setupPath -Destination $downloadedCopy -Force
    Remove-Item -LiteralPath $downloadedCopy -Force
    Assert-True (-not (Test-Path -LiteralPath $downloadedCopy)) "the downloaded Setup copy was deleted before uninstalling"

    $probeProject = Join-Path $script:WorkRoot "论文项目"
    New-Item -ItemType Directory -Force -Path (Join-Path $probeProject "dist") | Out-Null
    Set-Content -LiteralPath (Join-Path $probeProject "nodepaper.yaml") -Value "version: 1" -Encoding UTF8
    Set-Content -LiteralPath (Join-Path $probeProject "dist\paper.pdf") -Value "%PDF-1.7" -Encoding ASCII

    Write-Host "Uninstalling with the registered uninstaller..."
    $process = Start-Process -FilePath $registeredUninstaller -ArgumentList @("/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART") -Wait -PassThru
    $uninstallExit = 0
    try { $uninstallExit = $process.ExitCode } catch { $uninstallExit = 0 }
    Assert-True ($uninstallExit -eq 0) "the registered uninstaller exited with 0 without the downloaded Setup"
    # Inno Setup's uninstaller deletes itself asynchronously.
    for ($i = 0; $i -lt 40 -and (Test-Path -LiteralPath $script:InstallRoot); $i++) { Start-Sleep -Milliseconds 250 }

    Assert-True (-not (Test-Path -LiteralPath $script:InstallRoot)) "uninstall left no installation directory"
    Assert-True (-not (Test-UserPathContains $script:InstallRoot)) "uninstall removed the exact user Path entry"
    $remainingPathEntries = @(Get-UserPathEntries | ForEach-Object { Get-NormalizedPathEntry ($_.Trim().Trim('"')) } | Sort-Object)
    $before = ($originalPathEntries -join "`n")
    $after = ($remainingPathEntries -join "`n")
    Assert-True ($before -eq $after) "uninstall left unrelated user Path entries untouched (before: $($originalPathEntries.Count) entries, after: $($remainingPathEntries.Count) entries)"
    Assert-True ((Get-UserPathRaw) -eq $baselineUserPath) "install and uninstall restored the user Path value exactly, including non-ASCII entries of unrelated software"
    Assert-True (@(Get-UserPathEntries | Where-Object { $_ -eq $probePathEntry }).Count -eq 1) "the non-ASCII Path probe entry survived unchanged"
    Assert-True ((Get-UserPathKind) -eq [Microsoft.Win32.RegistryValueKind]::ExpandString) "the user Path keeps the Windows default REG_EXPAND_SZ type"
    Assert-True (-not (Test-Path -LiteralPath $startMenuDir)) "uninstall removed the Start-menu entries"
    Assert-True (-not (Test-Path -LiteralPath $UninstallKey)) "uninstall removed the Windows uninstall entry"
    Assert-True (Test-Path -LiteralPath (Join-Path $probeProject "nodepaper.yaml")) "uninstall preserved Project files"
    Assert-True (Test-Path -LiteralPath (Join-Path $probeProject "dist\paper.pdf")) "uninstall preserved published PDFs"
    $tex = Get-Command "xelatex" -CommandType Application -ErrorAction SilentlyContinue
    if ($tex) {
        Assert-True (Test-Path -LiteralPath $tex.Source -PathType Leaf) "uninstall preserved the external TeX installation"
    }

    # ---------- 8. record ---------------------------------------------------

    $record = [ordered]@{
        schemaVersion = 1
        version = $version
        sourceCommit = [string]$manifest.sourceCommit
        setupFile = $setupName
        setupSize = (Get-Item -LiteralPath $setupPath).Length
        setupSHA256 = $setupSHA256
        setupAuthenticodeStatus = $setupSignature
        zipFile = [System.IO.Path]::GetFileName($zipPath)
        zipSHA256 = $zipSHA256
        payloadManifestSHA256 = $zipPayloadManifestSHA256
        installRoot = $script:InstallRoot
        installScope = "current user"
        elevationRequested = $false
        installerToolchain = [string]$setupChannel.installerToolchain
        automatedChecks = $script:Checks
        manualGatesStillOpen = @(
            "interactive wizard journey on Windows 11 with TeX Live",
            "Windows 10 smoke test",
            "standalone MiKTeX environment",
            "SmartScreen / Defender observation for this exact file hash",
            "2-5 external testers: install, Start menu, doctor, init, build, uninstall",
            "final PDF review, defect triage and maintainer sign-off"
        )
        validatedAt = (Get-Date).ToString("o")
    }
    $record | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath $ResultsPath -Encoding UTF8

    $script:Passed = $true
    Write-Host ""
    Write-Host "Setup channel mechanical test passed: $setupName"
    Write-Host "Setup SHA-256: $setupSHA256"
    Write-Host "Authenticode: $setupSignature"
    Write-Host "Results: $ResultsPath"
}
finally {
    if (-not $script:Passed) {
        Write-Host "Setup test failed; cleaning up the test installation without touching user data."
        if ($script:InstallRoot -and (Test-Path -LiteralPath $UninstallKey)) {
            $entry = Get-ItemProperty -LiteralPath $UninstallKey -ErrorAction SilentlyContinue
            if ($entry -and $entry.UninstallString) {
                $uninstaller = ([string]$entry.UninstallString).Trim('"')
                if (Test-Path -LiteralPath $uninstaller -PathType Leaf) {
                    Start-Process -FilePath $uninstaller -ArgumentList @("/VERYSILENT", "/SUPPRESSMSGBOXES", "/NORESTART") -Wait -ErrorAction SilentlyContinue | Out-Null
                }
            }
        }
    }
    # Always restore the exact original user Path value and registry type.
    if ((Get-UserPathRaw) -ne $originalUserPath) {
        Set-UserPathRaw $originalUserPath $originalUserPathKind
    }
    if ($script:Passed -and -not $KeepWorkDirectory -and $script:WorkRoot -and (Test-Path -LiteralPath $script:WorkRoot)) {
        Remove-Item -LiteralPath $script:WorkRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
    elseif ($script:WorkRoot -and (Test-Path -LiteralPath $script:WorkRoot)) {
        Write-Host "Setup test work directory retained: $script:WorkRoot"
    }
}
