<#
.SYNOPSIS
    Build the Windows x64 NodePaper release-candidate package from a fixed commit.

.DESCRIPTION
    - Requires an explicit release version independent from the Profile version
    - Creates an isolated git worktree at the requested commit (default HEAD)
    - Copies the pinned bundled Pandoc / pandoc-crossref binaries into the worktree
    - Builds nodepaper.exe for windows/amd64 with -trimpath and version ldflags
    - Assembles nodepaper-<version>-windows-x64/ from an explicit whitelist only
    - Scans the package for absolute development paths, secrets and temp files
    - Creates the ZIP and records its SHA-256 in release-manifest.json

    The ZIP SHA-256 must only be treated as the final release candidate hash
    after LICENSE, THIRD_PARTY_NOTICES.md and the Profile version decision are
    final: adding or changing any packaged file changes the hash and invalidates
    earlier verification.

.PARAMETER Version
    Required build version string (for example "0.1.0-dev.184+g17bdb9e",
    "0.1.0-beta.1", "0.1.0-rc.1", or "0.1.0"). It is
    intentionally independent from the frozen Profile candidate version.

.PARAMETER Commit
    Git commit to package (rev, branch or hash). Default: current HEAD.

.PARAMETER OutputDirectory
    Directory for the ZIP and release-manifest.json. Default: build/release
    under the repository root.

.PARAMETER SkipTools
    Do not copy the bundled Pandoc binaries (fast layout testing only; the
    resulting package cannot build without system Pandoc and is not a release).

.PARAMETER KeepWork
    Keep the temporary worktree, package directory and ZIP for inspection.

.PARAMETER FeatureFreeze
    Required together with NoReleaseBlockers when Version is an rc.

.PARAMETER NoReleaseBlockers
    Required together with FeatureFreeze when Version is an rc.

.EXAMPLE
    .\scripts\build-release.ps1 -Version 0.1.0-dev.184+g17bdb9e

.EXAMPLE
    .\scripts\build-release.ps1 -Version 0.1.0-rc.1 -FeatureFreeze -NoReleaseBlockers
#>
param(
    [Parameter(Mandatory = $true)]
    [string]$Version,
    [string]$Commit = "HEAD",
    [string]$OutputDirectory = "",
    [switch]$SkipTools,
    [switch]$KeepWork,
    [switch]$FeatureFreeze,
    [switch]$NoReleaseBlockers
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "version-lifecycle.ps1")

$script:WorkRoot = ""
$script:Worktree = ""

function Invoke-Git {
    param([string[]]$Arguments)
    $stderrFile = Join-Path $env:TEMP ("nodepaper-git-stderr-" + [Guid]::NewGuid().ToString("N") + ".txt")
    $previousEAP = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        $output = & git @Arguments 2> $stderrFile
        $exitCode = $LASTEXITCODE
    }
    finally {
        $ErrorActionPreference = $previousEAP
    }
    $stderrText = ""
    if (Test-Path -LiteralPath $stderrFile) {
        $stderrText = Get-Content -LiteralPath $stderrFile -Raw -ErrorAction SilentlyContinue
        Remove-Item -LiteralPath $stderrFile -Force -ErrorAction SilentlyContinue
    }
    if ($exitCode -ne 0) {
        throw "git $($Arguments -join ' ') failed with exit code ${exitCode}:`n$stderrText"
    }
    return $output
}

function Get-NodePaperGo {
    if ($env:NODEPAPER_GO) {
        if (-not (Test-Path -LiteralPath $env:NODEPAPER_GO -PathType Leaf)) {
            throw "NODEPAPER_GO does not point to a file: $env:NODEPAPER_GO"
        }
        return (Resolve-Path -LiteralPath $env:NODEPAPER_GO).Path
    }
    foreach ($name in @("go.exe", "go")) {
        $command = Get-Command $name -ErrorAction SilentlyContinue
        if ($command) {
            return $command.Source
        }
    }
    $knownPath = "G:\Software\Go\bin\go.exe"
    if (Test-Path -LiteralPath $knownPath -PathType Leaf) {
        return $knownPath
    }
    throw "Go was not found. Install Go or set NODEPAPER_GO to go.exe."
}

function Get-ProfileSnapshotSHA256 {
    param([string]$ProfileDir)
    $entries = Get-ChildItem -LiteralPath $ProfileDir -Recurse -File | Sort-Object FullName
    $sb = New-Object System.Text.StringBuilder
    foreach ($entry in $entries) {
        $relative = $entry.FullName.Substring($ProfileDir.TrimEnd('\').Length).TrimStart('\')
        $relative = $relative -replace '\\', '/'
        $hash = (Get-FileHash -LiteralPath $entry.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
        [void]$sb.Append($relative).Append("`0").Append($hash).AppendLine()
    }
    $sha = [System.Security.Cryptography.SHA256]::Create()
    try {
        $bytes = [System.Text.Encoding]::UTF8.GetBytes($sb.ToString())
        return ([System.BitConverter]::ToString($sha.ComputeHash($bytes))).Replace("-", "").ToLowerInvariant()
    }
    finally {
        $sha.Dispose()
    }
}

# ---------- resolve inputs -------------------------------------------------

$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
if (-not (Test-Path -LiteralPath (Join-Path $root ".git") -PathType Container)) {
    throw "Not a git repository: $root"
}

$versionIdentity = ConvertFrom-NodePaperVersion $Version

if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path $root "build\release"
}
if (-not (Test-Path -LiteralPath $OutputDirectory -PathType Container)) {
    New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
}
$OutputDirectory = (Resolve-Path -LiteralPath $OutputDirectory).Path

$tagName = Get-NodePaperAssetBaseName $Version
$packageDir = Join-Path $OutputDirectory $tagName

Write-Host "NodePaper release build"
Write-Host "Version: $Version"
Write-Host "Commit: $Commit"

# ---------- isolated worktree at the fixed commit -------------------------

$commitLines = @(Invoke-Git @("-C", $root, "rev-parse", "--verify", "$Commit^{commit}"))
$resolvedCommit = $commitLines[0].Trim()
if ([string]::IsNullOrWhiteSpace($resolvedCommit)) {
    throw "Cannot resolve commit: $Commit"
}
$buildIdentity = Assert-NodePaperBuildIdentity -Version $Version -ResolvedCommit $resolvedCommit -RepositoryRoot $root -FeatureFreeze:$FeatureFreeze -NoReleaseBlockers:$NoReleaseBlockers
$canonicalBuildTime = Get-NodePaperCanonicalBuildTime -RepositoryRoot $root -ResolvedCommit $resolvedCommit

$script:WorkRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("nodepaper-release-" + [Guid]::NewGuid().ToString("N"))
$script:Worktree = Join-Path $script:WorkRoot "src"
New-Item -ItemType Directory -Force -Path $script:WorkRoot | Out-Null

$worktree = $script:Worktree

try {
    Write-Host "Creating isolated worktree at $resolvedCommit"
    Invoke-Git @("-C", $root, "-c", "core.autocrlf=false", "worktree", "add", "--detach", $worktree, $resolvedCommit) | Out-Null

    # The release worktree must preserve every frozen Profile byte exactly.
    $profilePaths = @(Invoke-Git @("-C", $root, "ls-tree", "-r", "--name-only", $resolvedCommit, "profiles/cumcm"))
    foreach ($profilePath in $profilePaths) {
        $expectedBlob = @(Invoke-Git @("-C", $root, "rev-parse", "$resolvedCommit`:$profilePath"))[0].Trim()
        $actualBlob = @(Invoke-Git @("-C", $root, "hash-object", (Join-Path $worktree ($profilePath -replace '/', '\'))))[0].Trim()
        if ($actualBlob -ne $expectedBlob) {
            throw "Frozen Profile byte mismatch after checkout: $profilePath"
        }
    }
    Write-Host "Frozen Profile byte check passed."

    $profileMetadataPath = Join-Path $worktree "profiles\cumcm\profile.json"
    if (-not (Test-Path -LiteralPath $profileMetadataPath -PathType Leaf)) {
        throw "Profile metadata not found at fixed commit: $profileMetadataPath"
    }
    $profileMetadata = Get-Content -LiteralPath $profileMetadataPath -Raw -Encoding UTF8 | ConvertFrom-Json

    # Bundled third-party binaries and corresponding sources live in the
    # per-user toolchain cache, outside version control and the worktree.
    $bundledTools = @()
    if (-not $SkipTools) {
        $bundledTools = @(
            @{ Source = "windows-x64\pandoc\pandoc.exe"; Target = "tools\windows-x64\pandoc\pandoc.exe" },
            @{ Source = "windows-x64\pandoc-crossref\pandoc-crossref.exe"; Target = "tools\windows-x64\pandoc-crossref\pandoc-crossref.exe" },
            @{ Source = "windows-x64\sources\pandoc-3.9-source.tar.gz"; Target = "tools\windows-x64\sources\pandoc-3.9-source.tar.gz" },
            @{ Source = "windows-x64\sources\pandoc-crossref-0.3.24-source.tar.gz"; Target = "tools\windows-x64\sources\pandoc-crossref-0.3.24-source.tar.gz" }
        )
    }

    # ---------- build the executable -----------------------------------------

    if (Test-Path -LiteralPath $packageDir) {
        Remove-Item -LiteralPath $packageDir -Recurse -Force
    }
    New-Item -ItemType Directory -Force -Path $packageDir | Out-Null

    foreach ($tool in $bundledTools) {
        $toolchainsRoot = Join-Path ([Environment]::GetFolderPath([Environment+SpecialFolder]::LocalApplicationData)) "NodePaper\toolchains"
        $source = Join-Path $toolchainsRoot $tool.Source
        if (-not (Test-Path -LiteralPath $source -PathType Leaf)) {
            throw "Bundled tool missing; run .\scripts\dev\Bootstrap-Tools.ps1 first: $source"
        }
        $target = Join-Path $packageDir $tool.Target
        New-Item -ItemType Directory -Force -Path (Split-Path -Parent $target) | Out-Null
        Copy-Item -LiteralPath $source -Destination $target -Force
        Write-Host "Bundled: $($tool.Source)"
    }

    $go = Get-NodePaperGo

    # go.mod pins the toolchain, but GOTOOLCHAIN=local would silently ignore
    # that and build with whatever is installed. release-manifest only records
    # the version it was handed, so a mismatch would ship as a fact rather than
    # as a failure -- which is how rc.3 through rc.7 were built on a different
    # toolchain from rc.2 and rc.8 without anyone noticing.
    $expectedToolchain = ""
    foreach ($line in (Get-Content -LiteralPath (Join-Path $worktree "nodepaper\core\go.mod") -Encoding UTF8)) {
        if ($line -match '^\s*toolchain\s+(go\S+)\s*$') { $expectedToolchain = $Matches[1]; break }
    }
    if ([string]::IsNullOrWhiteSpace($expectedToolchain)) {
        throw "go.mod declares no toolchain directive; the release toolchain must be pinned."
    }
    $actualToolchain = (((& $go version) -join " ") -split '\s+')[2]
    if ($actualToolchain -ne $expectedToolchain) {
        throw "Go toolchain mismatch: go.mod pins $expectedToolchain but the build would use $actualToolchain. Check GOTOOLCHAIN (currently '$((& $go env GOTOOLCHAIN))')."
    }
    Write-Host "Go toolchain verified: $actualToolchain (pinned by go.mod)"

    Push-Location $worktree
    try {
        $previousGoos = $env:GOOS
        $previousGoarch = $env:GOARCH
        $previousCgo = $env:CGO_ENABLED
        try {
            $env:GOOS = "windows"
            $env:GOARCH = "amd64"
            $env:CGO_ENABLED = "0"
            Write-Host 'go build -trimpath -ldflags "-X main.version=' $Version '"'
            & $go build -trimpath -ldflags "-X main.version=$Version" -o (Join-Path $packageDir "nodepaper.exe") ./nodepaper/core/cmd/nodepaper
            if ($LASTEXITCODE -ne 0) {
                throw "go build failed with exit code $LASTEXITCODE"
            }
        }
        finally {
            $env:GOOS = $previousGoos
            $env:GOARCH = $previousGoarch
            $env:CGO_ENABLED = $previousCgo
        }
    }
    finally {
        Pop-Location
    }

    $exe = Join-Path $packageDir "nodepaper.exe"
    $versionOutput = (& $exe --version 2>&1)
    if ($LASTEXITCODE -ne 0 -or ($versionOutput -join " ").Trim() -ne "nodepaper $Version") {
        throw "Built executable reports '$(($versionOutput -join ' ').Trim())', want 'nodepaper $Version'"
    }

    # ---------- assemble the package from the whitelist -----------------------

    $files = @(
        @{ Source = "scripts\build\Build-Paper.ps1"; Target = "Build-Paper.ps1" },
        @{ Source = "scripts\build\Convert-CumcmProjectToLatex.ps1"; Target = "Convert-CumcmProjectToLatex.ps1" },
        @{ Source = "packaging\windows\portable\Install-NodePaper.ps1"; Target = "Install-NodePaper.ps1" },
        @{ Source = "packaging\windows\portable\Uninstall-NodePaper.ps1"; Target = "Uninstall-NodePaper.ps1" },
        @{ Source = "packaging\\windows\\nodepaper.ico"; Target = "nodepaper.ico" },
        @{ Source = "README.md"; Target = "README.md" },
        @{ Source = "README.en.md"; Target = "README.en.md" },
        @{ Source = "LICENSE"; Target = "LICENSE" },
        @{ Source = "THIRD_PARTY_NOTICES.md"; Target = "THIRD_PARTY_NOTICES.md" },
        @{ Source = "packaging\toolchains\windows-x64.json"; Target = "tools\versions.json" }
    )
    foreach ($file in $files) {
        $source = Join-Path $worktree $file.Source
        if (-not (Test-Path -LiteralPath $source -PathType Leaf)) {
            throw "Release whitelist file missing: $($file.Source)"
        }
        $target = Join-Path $packageDir $file.Target
        New-Item -ItemType Directory -Force -Path (Split-Path -Parent $target) | Out-Null
        Copy-Item -LiteralPath $source -Destination $target -Force
    }

    if (-not $SkipTools) {
        $packageTools = @(
            "tools\windows-x64\pandoc\pandoc.exe",
            "tools\windows-x64\pandoc-crossref\pandoc-crossref.exe",
            "tools\windows-x64\sources\pandoc-3.9-source.tar.gz",
            "tools\windows-x64\sources\pandoc-crossref-0.3.24-source.tar.gz"
        )
        foreach ($relative in $packageTools) {
            $target = Join-Path $packageDir $relative
            if (-not (Test-Path -LiteralPath $target -PathType Leaf)) {
                throw "Bundled tool or corresponding source missing from package assembly: $relative"
            }
        }

        $toolVersionsMetadata = Get-Content -LiteralPath (Join-Path $worktree "packaging\toolchains\windows-x64.json") -Raw -Encoding UTF8 | ConvertFrom-Json
        $toolHashes = @(
            @{ Path = "tools\windows-x64\pandoc\pandoc.exe"; Expected = [string]$toolVersionsMetadata.pandoc.executable_sha256 },
            @{ Path = "tools\windows-x64\pandoc-crossref\pandoc-crossref.exe"; Expected = [string]$toolVersionsMetadata.pandoc_crossref.executable_sha256 },
            @{ Path = "tools\windows-x64\sources\pandoc-3.9-source.tar.gz"; Expected = [string]$toolVersionsMetadata.pandoc.source_sha256 },
            @{ Path = "tools\windows-x64\sources\pandoc-crossref-0.3.24-source.tar.gz"; Expected = [string]$toolVersionsMetadata.pandoc_crossref.source_sha256 }
        )
        foreach ($entry in $toolHashes) {
            if ([string]::IsNullOrWhiteSpace($entry.Expected)) {
                throw "Missing pinned SHA-256 for $($entry.Path)"
            }
            $actual = (Get-FileHash -LiteralPath (Join-Path $packageDir $entry.Path) -Algorithm SHA256).Hash.ToLowerInvariant()
            if ($actual -ne $entry.Expected.ToLowerInvariant()) {
                throw "Bundled tool/source SHA-256 mismatch for $($entry.Path): expected $($entry.Expected), got $actual"
            }
        }
    }

    $profileSource = Join-Path $worktree "profiles\cumcm"
    if (-not (Test-Path -LiteralPath $profileSource -PathType Container)) {
        throw "Profile directory missing: $profileSource"
    }
    Copy-Item -LiteralPath $profileSource -Destination (Join-Path $packageDir "profiles\cumcm") -Recurse -Force

    $licensesSource = Join-Path $worktree "licenses"
    if (-not (Test-Path -LiteralPath $licensesSource -PathType Container)) {
        throw "licenses/ directory missing; THIRD_PARTY_NOTICES.md requires it: $licensesSource"
    }
    Copy-Item -LiteralPath $licensesSource -Destination (Join-Path $packageDir "licenses") -Recurse -Force

    # Runnable CUMCM example: the public fictional complete-single-file
    # Fixture is the only example that passes the current CUMCM Profile.
    $exampleSource = Join-Path $worktree "nodepaper\core\tests\fixtures\complete-single-file"
    if (-not (Test-Path -LiteralPath $exampleSource -PathType Container)) {
        throw "Example Fixture missing: $exampleSource"
    }
    Copy-Item -LiteralPath $exampleSource -Destination (Join-Path $packageDir "examples\cumcm-single-file") -Recurse -Force

    # ---------- package scans --------------------------------------------------

    foreach ($forbiddenDir in @(".nodepaper", "dist", "build", "logs", "_downloads")) {
        $hit = Get-ChildItem -LiteralPath $packageDir -Recurse -Directory -Filter $forbiddenDir -ErrorAction SilentlyContinue
        if ($hit) {
            throw "Release package contains a forbidden directory: $($hit.FullName -join ', ')"
        }
    }

    $forbiddenPatterns = @(
        # Development-machine home paths. A drive letter is exactly one letter,
        # so anchor on that: without the lookbehind, the registry root
        # HKCU:\Software\... matched as if "U:" were a drive and blocked a
        # payload whose only offence was naming a registry key.
        '(?i)(?<![a-z])[a-z]:\\(users|onedrive|software|codex)\\'
        '(?i)^/mnt/'                                     # WSL mount paths
        '-----BEGIN (RSA |OPENSSH |EC |DSA )?PRIVATE KEY-----'
        '(?i)ghp_[A-Za-z0-9]{20,}'
        '(?i)github_pat_[A-Za-z0-9_]{20,}'
        'AKIA[0-9A-Z]{16}'
    )
    $binaryExtensions = @(".exe", ".pdf", ".jpg", ".jpeg", ".png", ".gif", ".zip", ".7z", ".gz", ".tar", ".woff", ".woff2", ".otf", ".ttf")
    Get-ChildItem -LiteralPath $packageDir -Recurse -File | ForEach-Object {
        $relative = $_.FullName.Substring($packageDir.Length).TrimStart('\')
        if ($binaryExtensions -contains $_.Extension.ToLowerInvariant()) {
            return
        }
        $text = Get-Content -LiteralPath $_.FullName -Raw -Encoding UTF8 -ErrorAction SilentlyContinue
        if ($null -eq $text) {
            return
        }
        foreach ($pattern in $forbiddenPatterns) {
            if ($text -match $pattern) {
                throw "Release package contains a forbidden pattern ('$pattern') in $relative"
            }
        }
    }
    Write-Host "Package scan passed: no secrets, absolute development paths or temp artifacts."

    # ---------- Profile snapshot hash (authoritative Go-side value) ------------

    # doctor reports the built-in Profile snapshot SHA-256 even when other
    # environment checks fail, so parsing its JSON is TeX-independent.
    $profileSHA256 = ""
    $doctorJson = (& $exe doctor --format json 2>&1 | Out-String)
    try {
        $doctor = $doctorJson | ConvertFrom-Json
        foreach ($check in @($doctor.checks)) {
            if ($check.name -eq "profile") {
                if ($check.message -match 'sha256 ([0-9a-f]{64})') {
                    $profileSHA256 = $Matches[1]
                }
            }
        }
    }
    catch {
        Write-Host "WARNING: could not parse doctor JSON for the Profile snapshot hash; falling back to a local computation."
    }
    if ([string]::IsNullOrWhiteSpace($profileSHA256)) {
        $profileSHA256 = Get-ProfileSnapshotSHA256 $profileSource
        Write-Host "Profile snapshot SHA-256 (locally computed): $profileSHA256"
    }
    else {
        Write-Host "Profile snapshot SHA-256 (Go-side, from doctor): $profileSHA256"
    }

    # ---------- immutable payload manifest --------------------------------------

    # The manifest is inside the payload so installers can verify every other
    # payload file before changing the installation or user Path. The manifest
    # itself is covered by the outer ZIP SHA-256 (a file cannot contain its own
    # hash without a circular definition).
    $payloadSHA256 = Get-NodePaperPayloadSHA256 $packageDir
    $buildInfo = [ordered]@{
        schemaVersion = 1
        version = $Version
        stage = $buildIdentity.Stage
        sourceCommit = $resolvedCommit
        gitTag = $buildIdentity.GitTag
        builtAtUTC = $canonicalBuildTime
        toolchain = [ordered]@{
            go = ((& $go version) -join " ").Trim()
            powershell = [string]$PSVersionTable.PSVersion
            pandoc = [string]$profileMetadata.pandocVersion
            pandocCrossref = [string]$profileMetadata.pandocCrossrefVersion
        }
        payloadSHA256 = $payloadSHA256
    }
    $buildInfoPath = Join-Path $packageDir "build-info.json"
    [System.IO.File]::WriteAllText($buildInfoPath, (($buildInfo | ConvertTo-Json -Depth 5) + [Environment]::NewLine), (New-Object System.Text.UTF8Encoding($false)))

    $payloadFiles = @(Get-ChildItem -LiteralPath $packageDir -Recurse -File | ForEach-Object {
        [ordered]@{
            path = ($_.FullName.Substring($packageDir.Length).TrimStart('\') -replace '\\', '/')
            size = $_.Length
            sha256 = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
        }
    } | Sort-Object { $_.path })
    $payloadManifest = [ordered]@{
        schemaVersion = 1
        channel = "portable-zip"
        version = $Version
        stage = $buildIdentity.Stage
        sourceCommit = $resolvedCommit
        buildInfo = "build-info.json"
        payloadSHA256 = $payloadSHA256
        profileVersion = [string]$profileMetadata.version
        profileSnapshotSHA256 = $profileSHA256
        bundledPandocVersion = [string]$profileMetadata.pandocVersion
        bundledPandocCrossrefVersion = [string]$profileMetadata.pandocCrossrefVersion
        bundledPandocSourceSHA256 = if ($SkipTools) { "" } else { [string]$toolVersionsMetadata.pandoc.source_sha256 }
        bundledPandocCrossrefSourceSHA256 = if ($SkipTools) { "" } else { [string]$toolVersionsMetadata.pandoc_crossref.source_sha256 }
        files = $payloadFiles
    }
    $payloadManifestPath = Join-Path $packageDir "payload-manifest.json"
    $payloadManifest | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath $payloadManifestPath -Encoding UTF8

    # ---------- ZIP and outer release manifest ---------------------------------

    $zipPath = Join-Path $OutputDirectory ($tagName + ".zip")
    if (Test-Path -LiteralPath $zipPath) {
        Remove-Item -LiteralPath $zipPath -Force
    }
    Write-Host "Compressing $packageDir -> $zipPath"
    Compress-Archive -LiteralPath $packageDir -DestinationPath $zipPath -CompressionLevel Optimal
    $zipSHA256 = (Get-FileHash -LiteralPath $zipPath -Algorithm SHA256).Hash.ToLowerInvariant()

    $fileList = @(Get-ChildItem -LiteralPath $packageDir -Recurse -File | ForEach-Object {
        $_.FullName.Substring($packageDir.Length).TrimStart('\') -replace '\\', '/'
    } | Sort-Object)

    $manifest = [ordered]@{
        schemaVersion = 1
        version = $Version
        stage = $buildIdentity.Stage
        sourceCommit = $resolvedCommit
        gitTag = $buildIdentity.GitTag
        buildInfo = "build-info.json"
        payloadSHA256 = $payloadSHA256
        profileVersion = [string]$profileMetadata.version
        profileRulesVersion = [string]$profileMetadata.rulesVersion
        profileSnapshotSHA256 = $profileSHA256
        goVersion = ((& $go version) -join " ").Trim()
        bundledPandocVersion = [string]$profileMetadata.pandocVersion
        bundledPandocCrossrefVersion = [string]$profileMetadata.pandocCrossrefVersion
        bundledPandocSourceSHA256 = if ($SkipTools) { "" } else { [string]$toolVersionsMetadata.pandoc.source_sha256 }
        bundledPandocCrossrefSourceSHA256 = if ($SkipTools) { "" } else { [string]$toolVersionsMetadata.pandoc_crossref.source_sha256 }
        bundledToolsIncluded = (-not $SkipTools)
        payloadManifest = "payload-manifest.json"
        payloadFileCount = $payloadFiles.Count
        packageDirectory = $tagName
        zipFile = [System.IO.Path]::GetFileName($zipPath)
        zipSHA256 = $zipSHA256
        builtAtUTC = $canonicalBuildTime
        files = $fileList
    }
    $manifestPath = Join-Path $OutputDirectory "release-manifest.json"
    $manifest | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $manifestPath -Encoding UTF8

    Write-Host ""
    Write-Host "Release candidate package: $zipPath"
    Write-Host "SHA-256: $zipSHA256"
    Write-Host "Commit: $resolvedCommit"
    Write-Host "Manifest: $manifestPath"
}
finally {
    if (-not $KeepWork -and $script:WorkRoot -and (Test-Path -LiteralPath $script:WorkRoot)) {
        if ($script:Worktree -and (Test-Path -LiteralPath $script:Worktree)) {
            git -C $root worktree remove --force $script:Worktree 2>$null | Out-Null
        }
        git -C $root worktree prune 2>$null | Out-Null
        Remove-Item -LiteralPath $script:WorkRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
    elseif ($KeepWork) {
        Write-Host "Release work directory retained: $script:WorkRoot"
    }
}
