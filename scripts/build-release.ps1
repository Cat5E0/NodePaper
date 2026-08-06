<#
.SYNOPSIS
    Build the Windows x64 NodePaper release-candidate package from a fixed commit.

.DESCRIPTION
    - Resolves the release version (parameter, or the Profile version by default)
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
    Release version string (for example "0.1.0-rc.1"). Defaults to the version
    declared in profiles/cumcm/profile.json.

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

.EXAMPLE
    .\scripts\build-release.ps1 -Version 0.1.0-rc.1
#>
param(
    [string]$Version = "",
    [string]$Commit = "HEAD",
    [string]$OutputDirectory = "",
    [switch]$SkipTools,
    [switch]$KeepWork
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

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

$profileMetadataPath = Join-Path $root "profiles\cumcm\profile.json"
if (-not (Test-Path -LiteralPath $profileMetadataPath -PathType Leaf)) {
    throw "Profile metadata not found: $profileMetadataPath"
}
$profileMetadata = Get-Content -LiteralPath $profileMetadataPath -Raw -Encoding UTF8 | ConvertFrom-Json
if ([string]::IsNullOrWhiteSpace($Version)) {
    $Version = [string]$profileMetadata.version
}
if ($Version -notmatch '^[A-Za-z0-9][A-Za-z0-9._+-]*$') {
    throw "Invalid release version string: $Version"
}

if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path $root "build\release"
}
if (-not (Test-Path -LiteralPath $OutputDirectory -PathType Container)) {
    New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
}
$OutputDirectory = (Resolve-Path -LiteralPath $OutputDirectory).Path

$tagName = "nodepaper-$Version-windows-x64"
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

$script:WorkRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("nodepaper-release-" + [Guid]::NewGuid().ToString("N"))
$script:Worktree = Join-Path $script:WorkRoot "src"
New-Item -ItemType Directory -Force -Path $script:WorkRoot | Out-Null

$worktree = $script:Worktree

try {
    Write-Host "Creating isolated worktree at $resolvedCommit"
    Invoke-Git @("-C", $root, "worktree", "add", "--detach", $worktree, $resolvedCommit) | Out-Null

    # Bundled third-party binaries live outside version control; copy them into
    # the isolated worktree so the package is assembled from one tree.
    if (-not $SkipTools) {
        $bundledTools = @(
            @{ Source = "tools\windows-x64\pandoc\pandoc.exe"; Target = "tools\windows-x64\pandoc\pandoc.exe" },
            @{ Source = "tools\windows-x64\pandoc-crossref\pandoc-crossref.exe"; Target = "tools\windows-x64\pandoc-crossref\pandoc-crossref.exe" }
        )
        foreach ($tool in $bundledTools) {
            $source = Join-Path $root $tool.Source
            if (-not (Test-Path -LiteralPath $source -PathType Leaf)) {
                throw "Bundled tool missing; run .\Bootstrap-Tools.ps1 first: $source"
            }
            $target = Join-Path $worktree $tool.Target
            New-Item -ItemType Directory -Force -Path (Split-Path -Parent $target) | Out-Null
            Copy-Item -LiteralPath $source -Destination $target -Force
            Write-Host "Bundled: $($tool.Source)"
        }
    }

    # ---------- build the executable -----------------------------------------

    if (Test-Path -LiteralPath $packageDir) {
        Remove-Item -LiteralPath $packageDir -Recurse -Force
    }
    New-Item -ItemType Directory -Force -Path $packageDir | Out-Null

    $go = Get-NodePaperGo
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
            & $go build -trimpath -ldflags "-X main.version=$Version" -o (Join-Path $packageDir "nodepaper.exe") ./cmd/nodepaper
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
        "Build-Paper.ps1",
        "Convert-CumcmProjectToLatex.ps1",
        "README.md",
        "README.zh-CN.md",
        "LICENSE",
        "THIRD_PARTY_NOTICES.md",
        "tools\versions.json"
    )
    foreach ($relative in $files) {
        $source = Join-Path $worktree $relative
        if (-not (Test-Path -LiteralPath $source -PathType Leaf)) {
            throw "Release whitelist file missing: $relative"
        }
        $target = Join-Path $packageDir $relative
        New-Item -ItemType Directory -Force -Path (Split-Path -Parent $target) | Out-Null
        Copy-Item -LiteralPath $source -Destination $target -Force
    }

    if (-not $SkipTools) {
        $packageTools = @(
            "tools\windows-x64\pandoc\pandoc.exe",
            "tools\windows-x64\pandoc-crossref\pandoc-crossref.exe"
        )
        foreach ($relative in $packageTools) {
            $source = Join-Path $worktree $relative
            if (-not (Test-Path -LiteralPath $source -PathType Leaf)) {
                throw "Bundled tool missing in worktree: $relative"
            }
            $target = Join-Path $packageDir $relative
            New-Item -ItemType Directory -Force -Path (Split-Path -Parent $target) | Out-Null
            Copy-Item -LiteralPath $source -Destination $target -Force
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
    $exampleSource = Join-Path $worktree "nodepaper-test-fixtures\tests\fixtures\complete-single-file"
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
        '(?i)[a-z]:\\(users|onedrive|software|codex)\\'   # development-machine home paths
        '(?i)^/mnt/'                                     # WSL mount paths
        '-----BEGIN (RSA |OPENSSH |EC |DSA )?PRIVATE KEY-----'
        '(?i)ghp_[A-Za-z0-9]{20,}'
        '(?i)github_pat_[A-Za-z0-9_]{20,}'
        'AKIA[0-9A-Z]{16}'
    )
    $binaryExtensions = @(".exe", ".pdf", ".jpg", ".jpeg", ".png", ".gif", ".zip", ".7z", ".woff", ".woff2", ".otf", ".ttf")
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

    # ---------- ZIP and manifest ------------------------------------------------

    $zipPath = Join-Path $OutputDirectory ($tagName + ".zip")
    if (Test-Path -LiteralPath $zipPath) {
        Remove-Item -LiteralPath $zipPath -Force
    }
    Write-Host "Compressing $packageDir -> $zipPath"
    Compress-Archive -LiteralPath $packageDir -DestinationPath $zipPath -CompressionLevel Optimal
    $zipSHA256 = (Get-FileHash -LiteralPath $zipPath -Algorithm SHA256).Hash.ToLowerInvariant()

    $gitTag = ""
    $previousEAP = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        $described = (& git -C $root describe --tags --exact-match $resolvedCommit 2>$null | Out-String).Trim()
        if ($LASTEXITCODE -eq 0 -and $described) {
            $gitTag = $described
        }
    }
    finally {
        $ErrorActionPreference = $previousEAP
    }

    $fileList = @(Get-ChildItem -LiteralPath $packageDir -Recurse -File | ForEach-Object {
        $_.FullName.Substring($packageDir.Length).TrimStart('\') -replace '\\', '/'
    } | Sort-Object)

    $manifest = [ordered]@{
        schemaVersion = 1
        version = $Version
        sourceCommit = $resolvedCommit
        gitTag = $gitTag
        profileVersion = [string]$profileMetadata.version
        profileRulesVersion = [string]$profileMetadata.rulesVersion
        profileSnapshotSHA256 = $profileSHA256
        goVersion = ((& $go version) -join " ").Trim()
        bundledPandocVersion = [string]$profileMetadata.pandocVersion
        bundledPandocCrossrefVersion = [string]$profileMetadata.pandocCrossrefVersion
        bundledToolsIncluded = (-not $SkipTools)
        packageDirectory = $tagName
        zipFile = [System.IO.Path]::GetFileName($zipPath)
        zipSHA256 = $zipSHA256
        builtAt = (Get-Date).ToString("o")
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
