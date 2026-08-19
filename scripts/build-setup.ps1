<#
.SYNOPSIS
    Build the single-file Windows x64 NodePaper Setup from an already built
    release payload.

.DESCRIPTION
    This script is the controlled Setup build entry. It never compiles
    nodepaper.exe: both distribution channels come from one fixed payload built
    by scripts/build-release.ps1.

      1. verifies the pinned Inno Setup 6 toolchain (SHA-256)
      2. verifies the payload against its own payload-manifest.json
      3. verifies the payload version, source commit and Profile snapshot
      4. writes the ASCII payload checksum list embedded in Setup
      5. compiles packaging/windows/nodepaper.iss with pinned defines
      6. records the Setup SHA-256, size and channel data in the release
         manifest next to the ZIP channel

    The generated Setup is unsigned unless a separate Authenticode signing step
    is performed afterwards; signing changes the bytes and invalidates every
    hash recorded here.

.PARAMETER Version
    Build version, for example 0.1.0-dev.184+g17bdb9e or 0.1.0-rc.1. Must match the payload manifest
    and nodepaper.exe --version.

.PARAMETER PayloadDirectory
    Built payload directory (build\release\nodepaper-<version>-windows-x64).

.PARAMETER OutputDirectory
    Output directory for the Setup. Default: the payload's parent directory.

.PARAMETER ManifestPath
    Release manifest to extend. Default: <OutputDirectory>\release-manifest.json

.EXAMPLE
    .\scripts\build-setup.ps1 -Version 0.1.0-rc.1 `
        -PayloadDirectory .\build\release\nodepaper-0.1.0-rc.1-windows-x64 `
        -FeatureFreeze -NoReleaseBlockers
#>
param(
    [Parameter(Mandatory = $true)]
    [string]$Version,
    [Parameter(Mandatory = $true)]
    [string]$PayloadDirectory,
    [string]$OutputDirectory = "",
    [string]$ManifestPath = "",
    [switch]$FeatureFreeze,
    [switch]$NoReleaseBlockers
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "version-lifecycle.ps1")

$root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
$versionIdentity = ConvertFrom-NodePaperVersion $Version

$payloadDir = (Resolve-Path -LiteralPath $PayloadDirectory).Path.TrimEnd('\')
$expectedPayloadName = Get-NodePaperAssetBaseName $Version
if ([System.IO.Path]::GetFileName($payloadDir) -ne $expectedPayloadName) {
    throw "Payload directory name must be $expectedPayloadName, got $([System.IO.Path]::GetFileName($payloadDir))"
}

if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Split-Path -Parent $payloadDir
}
if (-not (Test-Path -LiteralPath $OutputDirectory -PathType Container)) {
    New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
}
$OutputDirectory = (Resolve-Path -LiteralPath $OutputDirectory).Path
if ([string]::IsNullOrWhiteSpace($ManifestPath)) {
    $ManifestPath = Join-Path $OutputDirectory "release-manifest.json"
}

# ---------- 1. pinned Inno Setup toolchain ---------------------------------

$toolchainPath = Join-Path $root "packaging\windows\innosetup-toolchain.json"
$toolchainsRoot = Join-Path ([Environment]::GetFolderPath([Environment+SpecialFolder]::LocalApplicationData)) "NodePaper\toolchains"
$toolchain = Get-Content -LiteralPath $toolchainPath -Raw -Encoding UTF8 | ConvertFrom-Json
if ([int]$toolchain.schemaVersion -ne 1) {
    throw "Unsupported innosetup-toolchain.json schemaVersion: $($toolchain.schemaVersion)"
}
$pinnedFiles = @()
foreach ($entry in @($toolchain.compiler.files)) {
    $pinnedFiles += @{ Path = [string]$entry.path; SHA256 = [string]$entry.sha256 }
}
foreach ($entry in @($toolchain.languageFiles)) {
    $pinnedFiles += @{ Path = [string]$entry.target; SHA256 = [string]$entry.sha256 }
}
foreach ($entry in $pinnedFiles) {
    $path = Join-Path $toolchainsRoot ($entry.Path -replace '/', '\')
    if (-not (Test-Path -LiteralPath $path -PathType Leaf)) {
        throw "Pinned Inno Setup file missing; run .\scripts\bootstrap-innosetup.ps1 first: $($entry.Path)"
    }
    $actual = (Get-FileHash -LiteralPath $path -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actual -ne $entry.SHA256.ToLowerInvariant()) {
        throw "Pinned Inno Setup SHA-256 mismatch for $($entry.Path): expected $($entry.SHA256), got $actual"
    }
}
$iscc = Join-Path $toolchainsRoot (([string]$toolchain.compiler.compilerExecutable) -replace '/', '\')
Write-Host "Inno Setup $($toolchain.compiler.version) verified: $iscc"

# ---------- 2. payload verification ----------------------------------------

$payloadManifestPath = Join-Path $payloadDir "payload-manifest.json"
if (-not (Test-Path -LiteralPath $payloadManifestPath -PathType Leaf)) {
    throw "payload-manifest.json missing; build the payload with scripts\build-release.ps1 first."
}
$payloadManifest = Get-Content -LiteralPath $payloadManifestPath -Raw -Encoding UTF8 | ConvertFrom-Json
if ([int]$payloadManifest.schemaVersion -ne 1) {
    throw "Unsupported payload manifest schemaVersion: $($payloadManifest.schemaVersion)"
}
if ([string]$payloadManifest.version -ne $Version) {
    throw "Payload manifest version is $($payloadManifest.version), expected $Version"
}
$sourceCommit = [string]$payloadManifest.sourceCommit
$buildIdentity = Assert-NodePaperBuildIdentity -Version $Version -ResolvedCommit $sourceCommit -RepositoryRoot $root -FeatureFreeze:$FeatureFreeze -NoReleaseBlockers:$NoReleaseBlockers
$buildInfoPath = Join-Path $payloadDir "build-info.json"
if (-not (Test-Path -LiteralPath $buildInfoPath -PathType Leaf)) {
    throw "build-info.json missing from payload."
}
$buildInfo = Get-Content -LiteralPath $buildInfoPath -Raw -Encoding UTF8 | ConvertFrom-Json
if ([int]$buildInfo.schemaVersion -ne 1 -or [string]$buildInfo.version -ne $Version -or
    [string]$buildInfo.stage -ne $buildIdentity.Stage -or [string]$buildInfo.sourceCommit -ne $sourceCommit) {
    throw "build-info.json identity does not match the requested payload."
}
$actualPayloadSHA256 = Get-NodePaperPayloadSHA256 $payloadDir
if ([string]$buildInfo.payloadSHA256 -ne $actualPayloadSHA256 -or
    [string]$payloadManifest.payloadSHA256 -ne $actualPayloadSHA256) {
    throw "Payload SHA-256 does not match build-info.json and payload-manifest.json."
}

# Payload files the ZIP channel needs and Setup must never install. Both are
# the portable channel's own Path registration scripts, and a file named
# Uninstall-NodePaper.ps1 sitting inside a Setup installation directory is a
# trap: running it is the obvious thing to do and it removes the Path entry
# while the installation, its Start-menu entries and its entry in Settings all
# stay. One list feeds both the [Files] exclusion and the checksum list Setup
# verifies under {app} after installing, so those two cannot drift apart.
$setupExcludedPayloadFiles = New-Object 'System.Collections.Generic.HashSet[string]' ([System.StringComparer]::OrdinalIgnoreCase)
foreach ($relative in @("Install-NodePaper.ps1", "Uninstall-NodePaper.ps1")) {
    $setupExcludedPayloadFiles.Add($relative) | Out-Null
}

$expected = New-Object 'System.Collections.Generic.HashSet[string]' ([System.StringComparer]::OrdinalIgnoreCase)
$checksumLines = New-Object System.Collections.Generic.List[string]
foreach ($entry in @($payloadManifest.files)) {
    $relative = ([string]$entry.path) -replace '\\', '/'
    if ([string]::IsNullOrWhiteSpace($relative) -or $relative.Contains("..") -or [System.IO.Path]::IsPathRooted($relative)) {
        throw "Unsafe path in payload manifest: $relative"
    }
    # Setup embeds an ASCII checksum list; a non-ASCII payload path would not
    # survive the installer-side text reader and must fail the build instead.
    foreach ($char in $relative.ToCharArray()) {
        if ([int]$char -gt 126 -or [int]$char -lt 32 -or $char -eq '|') {
            throw "Payload path is not usable in the Setup checksum list: $relative"
        }
    }
    if (-not $expected.Add($relative)) {
        throw "Duplicate path in payload manifest: $relative"
    }
    $file = Join-Path $payloadDir ($relative -replace '/', '\')
    if (-not (Test-Path -LiteralPath $file -PathType Leaf)) {
        throw "Payload file missing: $relative"
    }
    $actual = (Get-FileHash -LiteralPath $file -Algorithm SHA256).Hash.ToLowerInvariant()
    if ($actual -ne ([string]$entry.sha256).ToLowerInvariant()) {
        throw "Payload SHA-256 mismatch: $relative"
    }
    # Verified as part of the payload above, then left out of the list Setup
    # checks under {app}: Setup does not install these.
    if ($setupExcludedPayloadFiles.Contains($relative)) { continue }
    $checksumLines.Add(($actual + "|" + $relative)) | Out-Null
}
foreach ($relative in $setupExcludedPayloadFiles) {
    if (-not $expected.Contains($relative)) {
        throw "Setup exclusion list names a file the payload does not contain: $relative"
    }
}
$actualFiles = @(Get-ChildItem -LiteralPath $payloadDir -Recurse -File | ForEach-Object {
    $_.FullName.Substring($payloadDir.Length).TrimStart('\') -replace '\\', '/'
} | Where-Object { $_ -ne "payload-manifest.json" })
foreach ($relative in $actualFiles) {
    if (-not $expected.Contains($relative)) {
        throw "Unverified extra file in payload: $relative"
    }
}
if ($actualFiles.Count -ne $expected.Count) {
    throw "Payload file list does not match its manifest."
}
Write-Host "Payload verified against payload-manifest.json: $($expected.Count) files"
Write-Host "Not installed by Setup (ZIP channel only): $((@($setupExcludedPayloadFiles) | Sort-Object) -join ', ')"

$exe = Join-Path $payloadDir "nodepaper.exe"
$reportedVersion = ((& $exe --version 2>&1 | Out-String).Trim())
if ($LASTEXITCODE -ne 0 -or $reportedVersion -ne "nodepaper $Version") {
    throw "Payload executable reports '$reportedVersion', want 'nodepaper $Version'"
}

# The payload manifest is not covered by its own hash list; the ZIP SHA-256 in
# the release manifest is the outer proof. Record its hash for the Setup
# channel so both channels can be compared file by file.
$payloadManifestSHA256 = (Get-FileHash -LiteralPath $payloadManifestPath -Algorithm SHA256).Hash.ToLowerInvariant()
$checksumLines.Add(($payloadManifestSHA256 + "|payload-manifest.json")) | Out-Null

# ---------- 3. checksum list embedded in Setup ------------------------------

$checksumPath = Join-Path $OutputDirectory "payload-checksums-$Version.txt"
$checksumText = ($checksumLines -join "`r`n") + "`r`n"
[System.IO.File]::WriteAllText($checksumPath, $checksumText, (New-Object System.Text.ASCIIEncoding))
Write-Host "Setup payload checksum list: $checksumPath"

# ---------- 4. compile the Setup --------------------------------------------

$outputBaseName = Get-NodePaperAssetBaseName -Version $Version -Prefix "NodePaper-Setup"
$setupPath = Join-Path $OutputDirectory ($outputBaseName + ".exe")
if (Test-Path -LiteralPath $setupPath) {
    Remove-Item -LiteralPath $setupPath -Force
}
$issPath = Join-Path $root "packaging\windows\nodepaper.iss"
# An Inno [Files] Excludes pattern that contains a backslash is matched against
# the path relative to the source directory, so the leading backslash pins each
# name to the payload root instead of every directory below it.
$payloadExcludes = ((@($setupExcludedPayloadFiles) | Sort-Object | ForEach-Object { "\" + ($_ -replace '/', '\') }) -join ',')

$isccArguments = @(
    "/DNodePaperVersion=$Version",
    "/DPayloadDir=$payloadDir",
    "/DChecksumFile=$checksumPath",
    "/DPayloadExcludes=$payloadExcludes",
    "/DOutputDir=$OutputDirectory",
    "/DOutputBaseName=$outputBaseName",
    "/DSourceCommit=$sourceCommit",
    $issPath
)
Write-Host "Compiling $issPath"
& $iscc @isccArguments
if ($LASTEXITCODE -ne 0) {
    throw "Inno Setup compilation failed with exit code $LASTEXITCODE"
}
if (-not (Test-Path -LiteralPath $setupPath -PathType Leaf)) {
    throw "Setup was not produced: $setupPath"
}

$setupItem = Get-Item -LiteralPath $setupPath
$setupSHA256 = (Get-FileHash -LiteralPath $setupPath -Algorithm SHA256).Hash.ToLowerInvariant()
$signature = Get-AuthenticodeSignature -LiteralPath $setupPath
$signatureStatus = [string]$signature.Status
if ($signatureStatus -eq "NotSigned") {
    Write-Host "Setup is NOT Authenticode signed. SmartScreen and security products may warn; publish the SHA-256 and source commit and never ask users to disable protection."
}

# ---------- 5. record the Setup channel in the release manifest -------------

if (-not (Test-Path -LiteralPath $ManifestPath -PathType Leaf)) {
    throw "Release manifest not found: $ManifestPath"
}
$manifestText = Get-Content -LiteralPath $ManifestPath -Raw -Encoding UTF8
$manifest = $manifestText | ConvertFrom-Json
if ([string]$manifest.version -ne $Version) {
    throw "Release manifest version is $($manifest.version), expected $Version"
}
if ([string]$manifest.sourceCommit -ne $sourceCommit) {
    throw "Release manifest source commit $($manifest.sourceCommit) differs from the payload commit $sourceCommit"
}

$ordered = [ordered]@{}
foreach ($property in $manifest.PSObject.Properties) {
    if ($property.Name -in @("channels", "setupFile", "setupSHA256")) { continue }
    $ordered[$property.Name] = $property.Value
}
$zipFile = [string]$manifest.zipFile
$zipPath = Join-Path (Split-Path -Parent $ManifestPath) $zipFile
$zipSize = 0
if (Test-Path -LiteralPath $zipPath -PathType Leaf) {
    $zipSize = (Get-Item -LiteralPath $zipPath).Length
}
$ordered["channels"] = @(
    [ordered]@{
        channel = "portable-zip"
        file = $zipFile
        size = $zipSize
        sha256 = [string]$manifest.zipSHA256
        audience = "advanced, offline and verifiable use"
        signed = $false
        signatureStatus = "NotSigned"
    },
    [ordered]@{
        channel = "setup-exe"
        file = [System.IO.Path]::GetFileName($setupPath)
        size = $setupItem.Length
        sha256 = $setupSHA256
        audience = "default channel for ordinary users"
        signed = ($signatureStatus -eq "Valid")
        signatureStatus = $signatureStatus
        installerToolchain = "Inno Setup $($toolchain.compiler.version)"
        installerToolchainSHA256 = [string]$toolchain.compiler.downloadSha256
        installerDefinition = "packaging/windows/nodepaper.iss"
        payloadDirectory = [System.IO.Path]::GetFileName($payloadDir)
        payloadManifestSHA256 = $payloadManifestSHA256
        payloadFileCount = $checksumLines.Count
        installScope = "current user"
        requiresAdministrator = $false
        builtAt = (Get-Date).ToString("o")
    }
)
$ordered["setupFile"] = [System.IO.Path]::GetFileName($setupPath)
$ordered["setupSHA256"] = $setupSHA256
$ordered | ConvertTo-Json -Depth 8 | Set-Content -LiteralPath $ManifestPath -Encoding UTF8

$versionedManifestPath = Join-Path (Split-Path -Parent $ManifestPath) "release-manifest-$($versionIdentity.AssetVersion).json"
Copy-Item -LiteralPath $ManifestPath -Destination $versionedManifestPath -Force

Write-Host ""
Write-Host "Setup: $setupPath"
Write-Host "Size: $($setupItem.Length) bytes"
Write-Host "SHA-256: $setupSHA256"
Write-Host "Authenticode: $signatureStatus"
Write-Host "Source commit: $sourceCommit"
Write-Host "Manifest: $ManifestPath"
Write-Host "Manifest copy: $versionedManifestPath"
