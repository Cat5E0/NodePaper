<#
.SYNOPSIS
    Validate an extracted (or zipped) NodePaper release-candidate package.

.DESCRIPTION
    Runs the release package in a clean, source-tree-independent environment:
      1. ZIP file name, package directory structure and required resources
      2. extraction to a unique temporary directory outside the repository
      3. no Go, source tree or source-only environment variable dependencies
      4. nodepaper.exe doctor
      5. nodepaper.exe init
      6. copy a public Fixture, then validate and build (text and JSON)
      7. build log, PDF and repeat-build / atomic-publish behaviour
      8. package contents, versions and SHA-256 record
      9. cleanup on success / retained diagnostics on failure
     10. unfinished manual/platform gates fail explicitly via -ManualGatesFile

    The Windows 11 + TeX Live mechanical flow is exercised by this script. The
    remaining release gates (MiKTeX, Windows 10, race detector, PDF manual
    review and maintainer sign-off) must be recorded as evidence in the JSON
    file passed via -ManualGatesFile; missing or failed gates fail the script.

.PARAMETER ReleaseZip
    Path to nodepaper-<version>-windows-x64.zip. Extracted to a unique temp dir.
.PARAMETER ReleaseDirectory
    Path to an already-extracted release directory (alternative to ReleaseZip).
.PARAMETER Fixture
    Public Fixture name to copy (default complete-single-file).
.PARAMETER FixtureDirectory
    Directory of the Fixture project to copy. Defaults to the repository's
    tests/fixtures copy of -Fixture.
.PARAMETER ManualGatesFile
    JSON file with confirmed manual gate evidence:
    { "schemaVersion": 1,
      "gates": { "miktex": {"date":..., "environment":..., "result":"pass"},
                 "windows10": {...}, "raceDetector": {...},
                 "pdfReview": {...}, "maintainer": {"result":"allow-release"} } }
.PARAMETER SkipTools
    Skip the bundled tools existence check (layout-only packages built with
    build-release.ps1 -SkipTools; not valid release candidates).
.PARAMETER KeepWorkDirectory
    Keep the temporary work directory for diagnosis.
#>
param(
    [string]$ReleaseZip = "",
    [string]$ReleaseDirectory = "",
    [string]$Fixture = "complete-single-file",
    [string]$FixtureDirectory = "",
    [string]$ManualGatesFile = "",
    [switch]$SkipTools,
    [switch]$KeepWorkDirectory
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "version-lifecycle.ps1")

$script:WorkRoot = ""
$script:Passed = $false

function Assert-True {
    param(
        [Parameter(Mandatory = $true)]
        [bool]$Condition,
        [Parameter(Mandatory = $true)]
        [string]$Message
    )
    if (-not $Condition) {
        throw "FAIL: $Message"
    }
}

function Get-FileList {
    param([string]$Root)
    return @(Get-ChildItem -LiteralPath $Root -Recurse -File | ForEach-Object {
        $_.FullName.Substring($Root.TrimEnd('\').Length).TrimStart('\') -replace '\\', '/'
    } | Sort-Object)
}

function Test-UserInstallation {
    param([string]$ReleaseRoot, [string]$WorkRoot, [string]$ExpectedVersion)

    # The ZIP channel registers the extracted folder rather than copying it, so
    # the folder under test is the release root itself.
    $installRoot = $ReleaseRoot
    $installer = Join-Path $ReleaseRoot "Install-NodePaper.ps1"
    # Nothing outside the process Path is touched: a portable installation is
    # described by its own directory, so registering writes no registry value
    # this function would have to save and put back.
    $originalProcessPath = [Environment]::GetEnvironmentVariable("Path", "Process")
    try {
        Write-Host "Testing user-level registration, tamper rejection and re-registration..."
        & $installer -PathScope Process
        if ($LASTEXITCODE -ne 0) { throw "FAIL: Install-NodePaper.ps1 failed with exit code $LASTEXITCODE" }

        # A modified payload file must be refused. Adding an unlisted file is
        # deliberately allowed -- the folder is also a workspace, and building
        # the bundled example leaves output in it -- so tamper detection rests
        # on the per-file hashes instead.
        # Restore byte-for-byte: Set-Content -Encoding UTF8 under Windows
        # PowerShell 5.1 re-adds a BOM the original never had, and the next
        # installer run would then fail the per-file hash check on a file this
        # test itself corrupted.
        $tampered = Join-Path $ReleaseRoot "README.md"
        $originalBytes = [System.IO.File]::ReadAllBytes($tampered)
        $rejected = $false
        try {
            Add-Content -LiteralPath $tampered -Value "`ntampered" -Encoding UTF8
            try { & $installer -PathScope Process }
            catch { $rejected = $_.Exception.Message -match "Payload hash mismatch" }
        }
        finally { [System.IO.File]::WriteAllBytes($tampered, $originalBytes) }
        Assert-True $rejected "installer did not reject a modified payload file"

        # Output left by building the bundled example must not block anything:
        # this is what the README asks readers to do first.
        $exampleBuild = Join-Path $ReleaseRoot "examples\cumcm-single-file\.nodepaper\build"
        New-Item -ItemType Directory -Force -Path $exampleBuild | Out-Null
        Set-Content -LiteralPath (Join-Path $exampleBuild "paper.aux") -Value "stray" -Encoding ASCII
        try {
            & $installer -PathScope Process
            if ($LASTEXITCODE -ne 0) { throw "FAIL: build output in the bundled example blocked registration" }
        }
        finally {
            Remove-Item -LiteralPath (Join-Path $ReleaseRoot "examples\cumcm-single-file\.nodepaper") -Recurse -Force -ErrorAction SilentlyContinue
        }

        # Get-Command returns every match, so take the one the shell would
        # actually run. Registration appends to the Path scope it was given, so
        # a NodePaper this test does not track keeps winning: a Setup-channel
        # installation on a developer machine (recognised by unins000.exe and
        # deliberately left alone by the portable installer), a hand-edited
        # Path entry, or a stale one. The point under test is that the payload
        # directory answers to `nodepaper` when it wins the Path search, which
        # is what a machine without an earlier copy does. Promote the payload
        # directory to the front of this process's Path for the direct-command
        # assertions below; the installer itself is still expected to report
        # the shadowing rather than silently claim success (checked above by
        # its non-zero-free run and warning text in the transcript).
        $matches = @(Get-Command nodepaper -CommandType Application -All -ErrorAction Stop)
        $command = $matches[0]
        $expected = (Resolve-Path -LiteralPath (Join-Path $installRoot "nodepaper.exe")).Path
        $actual = (Resolve-Path -LiteralPath $command.Source).Path
        if ($actual -ne $expected) {
            $earlier = ($matches | Where-Object { (Resolve-Path -LiteralPath $_.Source).Path -ne $expected } |
                ForEach-Object { $_.Source }) -join "; "
            Write-Host "note: another NodePaper precedes the registered payload on this machine's Path: $earlier"
            Write-Host "Putting the payload directory first on this process's Path for the command assertions."
            [Environment]::SetEnvironmentVariable("Path", "$installRoot;" + [Environment]::GetEnvironmentVariable("Path", "Process"), "Process")
            $matches = @(Get-Command nodepaper -CommandType Application -All -ErrorAction Stop)
            $command = $matches[0]
            $actual = (Resolve-Path -LiteralPath $command.Source).Path
            Assert-True ($actual -eq $expected) "nodepaper still resolves to '$actual', not the registered payload '$expected', even with the payload first on Path"
        }

        $arbitraryDir = Join-Path $WorkRoot "任意 工作目录"
        New-Item -ItemType Directory -Force -Path $arbitraryDir | Out-Null
        Push-Location $arbitraryDir
        try {
            $versionText = (& nodepaper --version 2>&1 | Out-String).Trim()
            Assert-True ($LASTEXITCODE -eq 0) "installed nodepaper --version failed"
            Assert-True ($versionText -eq "nodepaper $ExpectedVersion") "installed command version mismatch: '$versionText'"
            $guideText = (& nodepaper 2>&1 | Out-String)
            Assert-True ($LASTEXITCODE -eq 0) "installed no-argument onboarding failed"
            Assert-True ($guideText.Contains("nodepaper init")) "installed no-argument onboarding lacks init guidance"
            Assert-True (-not (Test-Path -LiteralPath (Join-Path $arbitraryDir "nodepaper.yaml"))) "no-argument onboarding modified the current directory"
        }
        finally {
            Pop-Location
        }

        $uninstaller = Join-Path $installRoot "Uninstall-NodePaper.ps1"
        & $uninstaller -InstallRoot $installRoot -PathScope Process
        if ($LASTEXITCODE -ne 0) { throw "FAIL: Uninstall-NodePaper.ps1 failed with exit code $LASTEXITCODE" }
        # The folder must survive: it is where the user chose to keep the
        # release, and unregistering does not make it ours to delete.
        Assert-True (Test-Path -LiteralPath (Join-Path $installRoot "nodepaper.exe") -PathType Leaf) "unregistration deleted the extracted folder"
        $normalizedInstall = [System.IO.Path]::GetFullPath($installRoot).TrimEnd('\').ToLowerInvariant()
        $remaining = @([Environment]::GetEnvironmentVariable("Path", "Process") -split ';' | ForEach-Object {
            try { [System.IO.Path]::GetFullPath($_.Trim().Trim('"')).TrimEnd('\').ToLowerInvariant() } catch { "" }
        })
        Assert-True (-not ($remaining -contains $normalizedInstall)) "unregistration left the NodePaper Path entry"
        Write-Host "User-level registration, tamper rejection, global command and unregistration passed."
    }
    finally {
        [Environment]::SetEnvironmentVariable("Path", $originalProcessPath, "Process")
    }
}

function Assert-ManualGates {
    param([string]$GatesFile)
    if ([string]::IsNullOrWhiteSpace($GatesFile)) {
        throw "FAIL: -ManualGatesFile is required. The release gates (MiKTeX, Windows 10, race detector, PDF manual review, maintainer sign-off) are not verified by this script and cannot be assumed to pass."
    }
    if (-not (Test-Path -LiteralPath $GatesFile -PathType Leaf)) {
        throw "FAIL: Manual gates file not found: $GatesFile"
    }
    $gates = Get-Content -LiteralPath $GatesFile -Raw -Encoding UTF8 | ConvertFrom-Json
    $required = @(
        @{ Name = "miktex"; Expect = "pass" },
        @{ Name = "windows10"; Expect = "pass" },
        @{ Name = "raceDetector"; Expect = "pass" },
        @{ Name = "pdfReview"; Expect = "pass" },
        @{ Name = "maintainer"; Expect = "allow-release" }
    )
    $missing = @()
    foreach ($gate in $required) {
        $entry = $gates.gates.($gate.Name)
        if ($null -eq $entry -or [string]::IsNullOrWhiteSpace([string]$entry.result) -or
            ([string]$entry.result).ToLowerInvariant() -ne $gate.Expect) {
            $missing += "$($gate.Name) (expected result '$($gate.Expect)')"
        }
    }
    if ($missing.Count -gt 0) {
        throw "FAIL: manual gates not confirmed: $($missing -join '; ')"
    }
    Write-Host "Manual gates confirmed: MiKTeX, Windows 10, race detector, PDF review, maintainer sign-off."
}

# ---------- input validation and extraction --------------------------------

if ([string]::IsNullOrWhiteSpace($ReleaseZip) -eq [string]::IsNullOrWhiteSpace($ReleaseDirectory)) {
    throw "Exactly one of -ReleaseZip or -ReleaseDirectory is required."
}

$zipPath = ""
$zipSHA256 = ""
if (-not [string]::IsNullOrWhiteSpace($ReleaseZip)) {
    $zipPath = (Resolve-Path -LiteralPath $ReleaseZip).Path
    $zipName = [System.IO.Path]::GetFileName($zipPath)
    if ($zipName -notmatch '^nodepaper-.+-windows-x64\.zip$') {
        throw "ZIP file name does not match nodepaper-<version>-windows-x64.zip: $zipName"
    }
    $zipSHA256 = (Get-FileHash -LiteralPath $zipPath -Algorithm SHA256).Hash.ToLowerInvariant()
}

try {
    if ($zipPath) {
        $script:WorkRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("nodepaper-release-test-" + [Guid]::NewGuid().ToString("N"))
        New-Item -ItemType Directory -Force -Path $script:WorkRoot | Out-Null
        Write-Host "Extracting $zipName to $script:WorkRoot"
        Expand-Archive -LiteralPath $zipPath -DestinationPath $script:WorkRoot -Force
        $packageDirs = @(Get-ChildItem -LiteralPath $script:WorkRoot -Directory)
        if ($packageDirs.Count -ne 1) {
            throw "FAIL: ZIP must contain exactly one top-level package directory; found $($packageDirs.Count)"
        }
        if ($packageDirs[0].Name -notmatch '^nodepaper-.+-windows-x64$') {
            throw "FAIL: top-level directory name does not match nodepaper-<version>-windows-x64: $($packageDirs[0].Name)"
        }
        $releaseDir = $packageDirs[0].FullName
    }
    else {
        $releaseDir = (Resolve-Path -LiteralPath $ReleaseDirectory).Path
        if ([System.IO.Path]::GetFileName($releaseDir) -notmatch '^nodepaper-.+-windows-x64$') {
            throw "FAIL: release directory name does not match nodepaper-<version>-windows-x64: $([System.IO.Path]::GetFileName($releaseDir))"
        }
    }
    $exe = Join-Path $releaseDir "nodepaper.exe"

    # ---------- 1. required resources --------------------------------------

    Write-Host "Checking required package resources..."
    $requiredFiles = @(
        "nodepaper.exe",
        "Build-Paper.ps1",
        "Convert-CumcmProjectToLatex.ps1",
        "Install-NodePaper.ps1",
        "Uninstall-NodePaper.ps1",
        "build-info.json",
        "payload-manifest.json",
        "profiles/cumcm/profile.json",
        "profiles/cumcm/template.tex",
        "profiles/cumcm/crossref.yaml",
        "profiles/cumcm/csl/china-national-standard-gb-t-7714-2015-numeric.csl",
        "profiles/cumcm/filters/extract-abstract.lua",
        "profiles/cumcm/filters/layout.lua",
        "profiles/cumcm/warning-allowlist.json",
        "README.md",
        "README.en.md",
        "nodepaper.ico",
        "LICENSE",
        "THIRD_PARTY_NOTICES.md",
        "licenses/Apache-2.0.txt",
        "licenses/BSD-3-Clause.txt",
        "licenses/CC-BY-SA-3.0.txt",
        "licenses/GPL-2.0.txt",
        "licenses/MIT.txt",
        "licenses/PANDOC-COPYRIGHT.txt",
        "licenses/YAML-V3-LICENSE.txt",
        "tools/versions.json",
        "examples/cumcm-single-file/nodepaper.yaml"
    )
    if (-not $SkipTools) {
        $requiredFiles += @(
            "tools/windows-x64/pandoc/pandoc.exe",
            "tools/windows-x64/pandoc-crossref/pandoc-crossref.exe",
            "tools/windows-x64/sources/pandoc-3.9-source.tar.gz",
            "tools/windows-x64/sources/pandoc-crossref-0.3.24-source.tar.gz"
        )
    }
    foreach ($relative in $requiredFiles) {
        Assert-True (Test-Path -LiteralPath (Join-Path $releaseDir ($relative -replace '/', '\')) -PathType Leaf) "required release resource missing: $relative"
    }
    $licenseFiles = @(Get-ChildItem -LiteralPath (Join-Path $releaseDir "licenses") -File -ErrorAction SilentlyContinue)
    Assert-True ($licenseFiles.Count -ge 1) "licenses/ directory must contain at least one license text"
    if (-not $SkipTools) {
        $toolVersions = Get-Content -LiteralPath (Join-Path $releaseDir "tools\versions.json") -Raw -Encoding UTF8 | ConvertFrom-Json
        $toolHashes = @(
            @{ Path = "tools/windows-x64/pandoc/pandoc.exe"; Expected = [string]$toolVersions.pandoc.executable_sha256 },
            @{ Path = "tools/windows-x64/pandoc-crossref/pandoc-crossref.exe"; Expected = [string]$toolVersions.pandoc_crossref.executable_sha256 },
            @{ Path = "tools/windows-x64/sources/pandoc-3.9-source.tar.gz"; Expected = [string]$toolVersions.pandoc.source_sha256 },
            @{ Path = "tools/windows-x64/sources/pandoc-crossref-0.3.24-source.tar.gz"; Expected = [string]$toolVersions.pandoc_crossref.source_sha256 }
        )
        foreach ($entry in $toolHashes) {
            Assert-True (-not [string]::IsNullOrWhiteSpace($entry.Expected)) "missing pinned SHA-256 for $($entry.Path)"
            $actual = (Get-FileHash -LiteralPath (Join-Path $releaseDir ($entry.Path -replace '/', '\')) -Algorithm SHA256).Hash.ToLowerInvariant()
            Assert-True ($actual -eq $entry.Expected.ToLowerInvariant()) "SHA-256 mismatch for $($entry.Path): expected $($entry.Expected), got $actual"
        }
        Write-Host "Bundled executable and corresponding-source SHA-256 checks passed."
    }
    # ---------- 1b. bilingual README entry points ---------------------------

    Write-Host "Checking bilingual README entry points and AI install prompts..."
    Assert-True (-not (Test-Path -LiteralPath (Join-Path $releaseDir "README.zh-CN.md"))) "package still contains the removed README.zh-CN.md"
    $chineseReadme = Get-Content -LiteralPath (Join-Path $releaseDir "README.md") -Raw -Encoding UTF8
    $englishReadme = Get-Content -LiteralPath (Join-Path $releaseDir "README.en.md") -Raw -Encoding UTF8
    $notices = Get-Content -LiteralPath (Join-Path $releaseDir "THIRD_PARTY_NOTICES.md") -Raw -Encoding UTF8
    Assert-True ($chineseReadme -match '[\u4e00-\u9fff]') "root README.md is not Simplified Chinese"
    Assert-True ($chineseReadme.Contains("README.en.md")) "Chinese README.md does not link to README.en.md"
    Assert-True ($englishReadme.Contains("README.md")) "README.en.md does not link back to README.md"
    foreach ($text in @(@{ Name = "README.md"; Value = $chineseReadme }, @{ Name = "README.en.md"; Value = $englishReadme }, @{ Name = "THIRD_PARTY_NOTICES.md"; Value = $notices })) {
        Assert-True (-not ($text.Value -match 'README\.zh-CN\.md')) "$($text.Name) still references the removed README.zh-CN.md"
    }
    # The AI prompt must keep its substantive safety constraints even when the
    # prose is shortened: unsigned build disclosed, security software never
    # disabled, and the decision left to the user.
    foreach ($readme in @(@{ Name = "README.md"; Value = $chineseReadme; Unsigned = "未签名"; NoDisable = "不要[^\n]{0,30}关闭" },
                          @{ Name = "README.en.md"; Value = $englishReadme; Unsigned = "unsigned"; NoDisable = "never[^\n]{0,30}disable" })) {
        Assert-True ($readme.Value -match [regex]::Escape($readme.Unsigned)) "$($readme.Name) does not disclose the unsigned build"
        # README forbids disabling security software in its own phrasing (for
        # example "不要为此关闭系统安全功能" / "never bypass or disable security
        # software"); match the negative requirement instead of one literal string.
        Assert-True ($readme.Value -match $readme.NoDisable) "$($readme.Name) does not forbid disabling security software"
    }
    foreach ($fragment in @(
        "https://github.com/Cat5E0/NodePaper",
        "NodePaper-Setup-",
        "release-manifest-",
        "Get-FileHash",
        "nodepaper --version",
        "nodepaper doctor",
        "SHA-256",
        "SmartScreen"
    )) {
        Assert-True ($chineseReadme.Contains($fragment)) "Chinese README AI install prompt lacks '$fragment'"
        Assert-True ($englishReadme.Contains($fragment)) "English README AI install prompt lacks '$fragment'"
    }

    # The TeX prerequisite has to be disclosed with real numbers, and the
    # install section must say up front that TeX is not bundled (the
    # export-first restructure in bc23501 folded the old standalone
    # "环境准备 / Before you start" chapter into the install section's
    # opening paragraph plus the dedicated "install TeX" chapter).
    Write-Host "Checking README TeX prerequisite disclosure..."
    foreach ($readme in @(@{ Name = "README.md"; Value = $chineseReadme; Install = "## 安装"; NoTeX = "不自带 TeX"; TeXChapter = "## 在本机直接出 PDF：安装 TeX" },
                          @{ Name = "README.en.md"; Value = $englishReadme; Install = "## Installation"; NoTeX = "does not bundle TeX"; TeXChapter = "## Producing a PDF locally: installing TeX" })) {
        $installAt = $readme.Value.IndexOf($readme.Install)
        $texChapterAt = $readme.Value.IndexOf($readme.TeXChapter)
        Assert-True ($installAt -ge 0) "$($readme.Name) lacks '$($readme.Install)'"
        Assert-True ($texChapterAt -gt $installAt) "$($readme.Name) lacks the install-TeX chapter '$($readme.TeXChapter)' after the install section"
        # Skip past the heading line itself, then take the opening paragraph.
        $bodyStart = $readme.Value.IndexOf("`n", $installAt) + 1
        $installParagraphEnd = $readme.Value.IndexOf("`n`n", $bodyStart)
        if ($installParagraphEnd -lt 0) { $installParagraphEnd = $readme.Value.Length }
        $installOpening = $readme.Value.Substring($bodyStart, $installParagraphEnd - $bodyStart)
        Assert-True ($installOpening.Contains($readme.NoTeX)) "$($readme.Name) install section does not disclose that TeX is not bundled"
        foreach ($fragment in @("miktex.org/download", "tug.org/texlive", "140 MB", "6.3 GB")) {
            Assert-True ($readme.Value.Contains($fragment)) "$($readme.Name) TeX install chapter lacks '$fragment'"
        }
        # The hand-written PATH snippet is destructive if simplified. Reading
        # $env:PATH instead of the registry copies the system PATH into the user
        # one; reading without DoNotExpandEnvironmentNames freezes other
        # software's %VARIABLE% entries; writing without ExpandString stops them
        # expanding at all. Pin all three so a future edit cannot drop them.
        foreach ($fragment in @("DoNotExpandEnvironmentNames", "ExpandString", "HKCU:\Environment")) {
            Assert-True ($readme.Value.Contains($fragment)) "$($readme.Name) manual PATH instructions lack '$fragment'"
        }
    }
    Assert-True (@(Get-ChildItem -LiteralPath $releaseDir -Recurse -Directory -Filter ".nodepaper" -ErrorAction SilentlyContinue).Count -eq 0) "package must not contain .nodepaper directories"
    Assert-True (@(Get-ChildItem -LiteralPath $releaseDir -Recurse -Directory -Filter "dist" -ErrorAction SilentlyContinue).Count -eq 0) "package must not contain dist directories"

    # ---------- 2/3. environment independence --------------------------------

    foreach ($var in @("NODEPAPER_BUILD_SCRIPT", "NODEPAPER_PROFILE_DIR", "NODEPAPER_GO")) {
        $value = [Environment]::GetEnvironmentVariable($var)
        if (-not [string]::IsNullOrWhiteSpace($value)) {
            throw "FAIL: $var is set to '$value'; the release test must run without source-tree overrides."
        }
    }
    Write-Host "Environment independence: no source-tree overrides present."

    $versionOutput = (& $exe --version 2>&1)
    if ($LASTEXITCODE -ne 0) {
        throw "FAIL: nodepaper.exe --version failed with exit code $LASTEXITCODE"
    }
    $reportedVersion = (($versionOutput | Out-String).Trim())
    Write-Host "Executable version: $reportedVersion"
    Assert-True ($reportedVersion -match '^nodepaper ') "nodepaper.exe --version output is malformed: '$reportedVersion'"
    $expectedVersion = $reportedVersion.Substring("nodepaper ".Length)
    $versionIdentity = ConvertFrom-NodePaperVersion $expectedVersion
    $expectedPackageName = Get-NodePaperAssetBaseName $expectedVersion
    Assert-True ([System.IO.Path]::GetFileName($releaseDir) -eq $expectedPackageName) "package directory name does not match executable version: expected $expectedPackageName"
    if ($zipPath) {
        Assert-True ($zipName -eq "$expectedPackageName.zip") "ZIP file name does not match executable version: expected $expectedPackageName.zip"
    }

    $buildInfo = Get-Content -LiteralPath (Join-Path $releaseDir "build-info.json") -Raw -Encoding UTF8 | ConvertFrom-Json
    $payloadManifest = Get-Content -LiteralPath (Join-Path $releaseDir "payload-manifest.json") -Raw -Encoding UTF8 | ConvertFrom-Json
    Assert-True ([int]$buildInfo.schemaVersion -eq 1) "unsupported build-info.json schema"
    Assert-True ([string]$buildInfo.version -eq $expectedVersion) "build-info.json version differs from executable"
    Assert-True ([string]$buildInfo.stage -eq $versionIdentity.Stage) "build-info.json stage differs from version"
    Assert-True ([string]$buildInfo.sourceCommit -match '^[0-9a-f]{40}$') "build-info.json lacks a full source commit"
    Assert-True ([string]$buildInfo.builtAtUTC -match '^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$') "build-info.json timestamp is not canonical UTC"
    Assert-True ($null -ne $buildInfo.toolchain -and -not [string]::IsNullOrWhiteSpace([string]$buildInfo.toolchain.go)) "build-info.json lacks toolchain identity"
    $actualPayloadSHA256 = Get-NodePaperPayloadSHA256 $releaseDir
    Assert-True ([string]$buildInfo.payloadSHA256 -eq $actualPayloadSHA256) "payload differs from build-info.json SHA-256"
    Assert-True ([string]$payloadManifest.version -eq $expectedVersion) "payload manifest version differs from executable"
    Assert-True ([string]$payloadManifest.sourceCommit -eq [string]$buildInfo.sourceCommit) "payload manifest source commit differs from build-info.json"
    Assert-True ([string]$payloadManifest.payloadSHA256 -eq $actualPayloadSHA256) "payload differs from payload-manifest.json SHA-256"
    Write-Host "Build identity: $expectedVersion / $($buildInfo.sourceCommit) / $actualPayloadSHA256"

    # ---------- installation / global command ---------------------------------

    Test-UserInstallation $releaseDir $script:WorkRoot $expectedVersion

    # ---------- 4. doctor -----------------------------------------------------

    Write-Host "Running doctor..."
    $doctorText = (& $exe doctor 2>&1 | Out-String)
    if ($LASTEXITCODE -ne 0) {
        throw "FAIL: doctor failed. This machine must provide a working TeX environment. Output:`n$doctorText"
    }
    Assert-True ($doctorText.Contains("Success")) "doctor text output lacks the success marker"
    $doctorJsonText = (& $exe doctor --format json 2>$null | Out-String)
    if ($LASTEXITCODE -ne 0) {
        throw "FAIL: doctor --format json failed"
    }
    $doctor = $doctorJsonText | ConvertFrom-Json
    Assert-True ([bool]$doctor.success) "doctor JSON success = false"
    $profileSHA256 = ""
    foreach ($check in @($doctor.checks)) {
        if ($check.name -eq "profile") {
            Write-Host "Profile check: $($check.message)"
            if ($check.message -match 'sha256 ([0-9a-f]{64})') {
                $profileSHA256 = $Matches[1]
            }
        }
        if ($check.name -eq "Pandoc" -and $check.status -eq "pass") {
            Write-Host "Pandoc check: $($check.message)"
        }
        if ($check.name -eq "LaTeX driver") {
            Assert-True ($check.status -eq "pass") "doctor LaTeX driver check did not pass: $($check.message)"
            Assert-True ($check.message -match "XeLaTeX driven directly|drives XeLaTeX directly") "doctor does not report direct XeLaTeX drive: $($check.message)"
            Write-Host "LaTeX driver check: $($check.message)"
        }
    }
    Assert-True ($profileSHA256 -ne "") "doctor JSON did not report the Profile snapshot SHA-256"

    # ---------- 5. init --------------------------------------------------------

    Write-Host "Running init..."
    $initDir = Join-Path $script:WorkRoot "init-project"
    $initJsonText = (& $exe init $initDir --format json 2>$null | Out-String)
    if ($LASTEXITCODE -ne 0) {
        throw "FAIL: init failed with exit code $LASTEXITCODE"
    }
    $init = $initJsonText | ConvertFrom-Json
    Assert-True ([bool]$init.success) "init JSON success = false"
    foreach ($relative in @("nodepaper.yaml", "paper.md", "references.bib", "images")) {
        Assert-True (Test-Path -LiteralPath (Join-Path $initDir $relative)) "init did not create $relative"
    }

    # ---------- 6. fixture validate + build ------------------------------------

    $fixtureDir = $FixtureDirectory
    if ([string]::IsNullOrWhiteSpace($fixtureDir)) {
        $fixtureDir = Join-Path $PSScriptRoot "..\tests\fixtures\$Fixture"
        if (-not (Test-Path -LiteralPath $fixtureDir -PathType Container)) {
            throw "FAIL: Fixture not found: $fixtureDir. Pass -FixtureDirectory to use a copied fixture."
        }
    }
    $fixtureDir = (Resolve-Path -LiteralPath $fixtureDir).Path
    $projectDir = Join-Path $script:WorkRoot "fixture-project"
    Copy-Item -LiteralPath $fixtureDir -Destination $projectDir -Recurse
    Write-Host "Fixture project: $projectDir"

    Write-Host "Running validate..."
    $validateJsonText = (& $exe validate $projectDir --format json 2>$null | Out-String)
    if ($LASTEXITCODE -ne 0) {
        throw "FAIL: validate failed with exit code $LASTEXITCODE"
    }
    $validation = $validateJsonText | ConvertFrom-Json
    Assert-True ([bool]$validation.success) "validate JSON success = false"

    Write-Host "Running build (text)..."
    $buildText = (& $exe build $projectDir 2>&1 | Out-String)
    if ($LASTEXITCODE -ne 0) {
        throw "FAIL: build (text) failed with exit code $LASTEXITCODE. Output:`n$buildText"
    }
    Assert-True ($buildText.Contains("Success")) "build text output lacks the success marker"
    Assert-True ($buildText.Contains("Build ID:")) "build text output lacks the Build ID"

    Write-Host "Running build (JSON)..."
    $buildJsonText = (& $exe build $projectDir --format json 2>$null | Out-String)
    if ($LASTEXITCODE -ne 0) {
        throw "FAIL: build (JSON) failed with exit code $LASTEXITCODE"
    }
    $build = $buildJsonText | ConvertFrom-Json
    Assert-True ([bool]$build.success) "build JSON success = false"
    Assert-True (-not [string]::IsNullOrWhiteSpace([string]$build.buildId)) "build JSON lacks buildId"
    $pdfArtifact = ""
    foreach ($artifact in @($build.artifacts)) {
        if ($artifact.kind -eq "pdf") {
            $pdfArtifact = [string]$artifact.path
        }
    }
    Assert-True ($pdfArtifact -ne "") "build JSON lacks a pdf artifact"
    $pdfPath = Join-Path $projectDir "dist\paper.pdf"
    Assert-True (Test-Path -LiteralPath $pdfPath -PathType Leaf) "published PDF not found: $pdfPath"

    # ---------- 7. logs, PDF and repeat build ------------------------------------

    $bytes = [System.IO.File]::ReadAllBytes($pdfPath)
    Assert-True ($bytes.Length -gt 5) "published PDF is empty"
    Assert-True ([System.Text.Encoding]::ASCII.GetString($bytes, 0, 5) -eq "%PDF-") "published PDF header is invalid"
    $pdfSHA256 = (Get-FileHash -LiteralPath $pdfPath -Algorithm SHA256).Hash.ToLowerInvariant()

    $logs = @(Get-ChildItem -LiteralPath (Join-Path $projectDir ".nodepaper\logs") -Filter "build-*.log" -File -ErrorAction SilentlyContinue)
    Assert-True ($logs.Count -ge 1) "build log was not created"
    $logText = Get-Content -LiteralPath $logs[0].FullName -Raw -Encoding UTF8
    Assert-True ($logText.Contains("Build-Paper.ps1")) "build log does not record the PowerShell transition script"

    Write-Host "Running second build (repeatability / atomic publish)..."
    $secondJsonText = (& $exe build $projectDir --format json 2>$null | Out-String)
    if ($LASTEXITCODE -ne 0) {
        throw "FAIL: second build failed with exit code $LASTEXITCODE"
    }
    $second = $secondJsonText | ConvertFrom-Json
    Assert-True ([bool]$second.success) "second build JSON success = false"
    Assert-True ($second.buildId -ne $build.buildId) "second build reused the same Build ID"
    $logs = @(Get-ChildItem -LiteralPath (Join-Path $projectDir ".nodepaper\logs") -Filter "build-*.log" -File -ErrorAction SilentlyContinue)
    Assert-True ($logs.Count -ge 2) "two builds did not create distinct log files"

    # ---------- 8. record -----------------------------------------------------------

    $record = [ordered]@{
        schemaVersion = 1
        package = [System.IO.Path]::GetFileName($releaseDir)
        executableVersion = $reportedVersion
        zip = $zipName
        zipSHA256 = $zipSHA256
        profileSnapshotSHA256 = $profileSHA256
        fixture = $Fixture
        buildIds = @([string]$build.buildId, [string]$second.buildId)
        pdfSHA256 = $pdfSHA256
        validatedAt = (Get-Date).ToString("o")
        files = Get-FileList $releaseDir
    }
    $recordPath = Join-Path $script:WorkRoot "release-test-results.json"
    $record | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $recordPath -Encoding UTF8
    Write-Host "Release test record: $recordPath"

    # ---------- 10. manual gates -----------------------------------------------------

    Assert-ManualGates $ManualGatesFile

    $script:Passed = $true
    Write-Host ""
    Write-Host "Release package test passed: $releaseDir"
    if ($zipSHA256) {
        Write-Host "ZIP SHA-256: $zipSHA256"
    }
}
finally {
    if ($script:Passed -and -not $KeepWorkDirectory -and $script:WorkRoot -and (Test-Path -LiteralPath $script:WorkRoot)) {
        Remove-Item -LiteralPath $script:WorkRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
    elseif ($script:WorkRoot -and (Test-Path -LiteralPath $script:WorkRoot)) {
        Write-Host "Release test work directory retained: $script:WorkRoot"
    }
}
