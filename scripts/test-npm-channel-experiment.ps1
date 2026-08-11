<#
.SYNOPSIS
    Build and test a local-only npm wrapper around an existing Windows payload.

.DESCRIPTION
    This is an M4-04 channel experiment, not a publishing command. It copies an
    already-built Release Payload unchanged, creates a thin Node.js launcher,
    runs npm pack, installs the .tgz into an isolated prefix, checks cwd/stdio/
    exit-code forwarding, and uninstalls it. It never runs npm publish and has
    no install/postinstall network hook.

.PARAMETER SourceCommit
    Full source commit for legacy payloads without payload-manifest.json. New
    payloads read and verify this value from their own manifest.
#>
param(
    [Parameter(Mandatory = $true)]
    [string]$ReleaseDirectory,
    [string]$SourceCommit = "",
    [string]$OutputDirectory = "",
    [switch]$KeepWorkDirectory
)

Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Assert-True {
    param([bool]$Condition, [string]$Message)
    if (-not $Condition) { throw "FAIL: $Message" }
}

function Invoke-NativeCapture {
    param([string]$Command, [string[]]$Arguments)
    $previousEAP = $ErrorActionPreference
    $ErrorActionPreference = "Continue"
    try {
        $text = (& $Command @Arguments 2>&1 | Out-String)
        $exitCode = $LASTEXITCODE
    }
    finally { $ErrorActionPreference = $previousEAP }
    return [pscustomobject]@{ Text = $text; ExitCode = $exitCode }
}

$releaseRoot = (Resolve-Path -LiteralPath $ReleaseDirectory).Path
$exe = Join-Path $releaseRoot "nodepaper.exe"
if (-not (Test-Path -LiteralPath $exe -PathType Leaf)) { throw "nodepaper.exe is missing: $releaseRoot" }
$reportedVersion = ((& $exe --version 2>&1 | Out-String).Trim())
if ($LASTEXITCODE -ne 0 -or $reportedVersion -notmatch '^nodepaper (.+)$') { throw "Cannot read payload version: $reportedVersion" }
$version = $Matches[1]
$payloadManifestPath = Join-Path $releaseRoot "payload-manifest.json"
if (Test-Path -LiteralPath $payloadManifestPath -PathType Leaf) {
    $payloadManifest = Get-Content -LiteralPath $payloadManifestPath -Raw -Encoding UTF8 | ConvertFrom-Json
    if ([string]$payloadManifest.version -ne $version) { throw "Payload manifest and executable versions differ." }
    $SourceCommit = [string]$payloadManifest.sourceCommit
}
if ($SourceCommit -notmatch '^[0-9a-fA-F]{40}$') {
    throw "SourceCommit is required for a payload without payload-manifest.json and must be a full 40-character Git commit."
}
$SourceCommit = $SourceCommit.ToLowerInvariant()
$npm = Get-Command npm.cmd -ErrorAction SilentlyContinue
if (-not $npm) { $npm = Get-Command npm -ErrorAction Stop }

if ([string]::IsNullOrWhiteSpace($OutputDirectory)) {
    $OutputDirectory = Join-Path ([System.IO.Path]::GetTempPath()) ("nodepaper-npm-result-" + [Guid]::NewGuid().ToString("N"))
}
New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
$OutputDirectory = (Resolve-Path -LiteralPath $OutputDirectory).Path
$work = Join-Path ([System.IO.Path]::GetTempPath()) ("nodepaper-npm-experiment-" + [Guid]::NewGuid().ToString("N"))
$packageRoot = Join-Path $work "package"
$payloadRoot = Join-Path $packageRoot "payload"
$prefix = Join-Path $work "prefix"
$originalPath = $env:PATH
$passed = $false

try {
    New-Item -ItemType Directory -Force -Path $payloadRoot, (Join-Path $packageRoot "bin"), $prefix | Out-Null
    Get-ChildItem -LiteralPath $releaseRoot -Force | ForEach-Object {
        Copy-Item -LiteralPath $_.FullName -Destination $payloadRoot -Recurse -Force
    }

    $packageJson = [ordered]@{
        name = "nodepaper-local-experiment"
        version = $version
        private = $true
        description = "Local-only NodePaper npm channel experiment"
        license = "MIT"
        bin = @{ nodepaper = "bin/nodepaper.js" }
        files = @("bin", "payload", "channel-manifest.json")
    }
    $packageJson | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath (Join-Path $packageRoot "package.json") -Encoding UTF8
    @'
#!/usr/bin/env node
const { spawn } = require("node:child_process");
const path = require("node:path");
const exe = path.join(__dirname, "..", "payload", "nodepaper.exe");
const child = spawn(exe, process.argv.slice(2), {
  cwd: process.cwd(), env: process.env, stdio: "inherit", windowsHide: false
});
child.on("error", (error) => { console.error(`nodepaper launcher: ${error.message}`); process.exit(1); });
child.on("exit", (code) => process.exit(code === null ? 1 : code));
'@ | Set-Content -LiteralPath (Join-Path $packageRoot "bin\nodepaper.js") -Encoding ASCII

    $payloadFiles = @(Get-ChildItem -LiteralPath $payloadRoot -Recurse -File | ForEach-Object {
        [ordered]@{
            path = ($_.FullName.Substring($payloadRoot.Length).TrimStart('\') -replace '\\', '/')
            size = $_.Length
            sha256 = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant()
        }
    } | Sort-Object { $_.path })
    $channelManifest = [ordered]@{
        schemaVersion = 1
        channel = "npm-local-experiment"
        packageName = "nodepaper-local-experiment"
        version = $version
        executableVersion = $reportedVersion
        sourceCommit = $SourceCommit
        payloadFiles = $payloadFiles
    }
    $channelManifest | ConvertTo-Json -Depth 6 | Set-Content -LiteralPath (Join-Path $packageRoot "channel-manifest.json") -Encoding UTF8

    Push-Location $packageRoot
    try {
        $packCall = Invoke-NativeCapture $npm.Source @("pack", "--json")
        if ($packCall.ExitCode -ne 0) { throw "npm pack failed:`n$($packCall.Text)" }
        $jsonStart = $packCall.Text.IndexOf('[')
        if ($jsonStart -lt 0) { throw "npm pack did not return JSON:`n$($packCall.Text)" }
        $pack = $packCall.Text.Substring($jsonStart) | ConvertFrom-Json
        $tgzSource = Join-Path $packageRoot ([string]$pack[0].filename)
    }
    finally { Pop-Location }
    $tgz = Join-Path $OutputDirectory ([System.IO.Path]::GetFileName($tgzSource))
    Copy-Item -LiteralPath $tgzSource -Destination $tgz -Force

    $installCall = Invoke-NativeCapture $npm.Source @("install", "--global", "--prefix", $prefix, "--ignore-scripts", $tgz)
    if ($installCall.ExitCode -ne 0) { throw "isolated npm install failed:`n$($installCall.Text)" }
    $env:PATH = "$prefix;$originalPath"

    $testDir = Join-Path $work "中文 cwd"
    New-Item -ItemType Directory -Path $testDir | Out-Null
    Push-Location $testDir
    try {
        $versionCall = Invoke-NativeCapture "nodepaper" @("--version")
        $installedVersion = $versionCall.Text.Trim()
        Assert-True ($versionCall.ExitCode -eq 0 -and $installedVersion -eq $reportedVersion) "npm command did not forward --version: $($versionCall.Text)"
        $guideCall = Invoke-NativeCapture "nodepaper" @()
        Assert-True ($guideCall.ExitCode -eq 0 -and $guideCall.Text.Contains("nodepaper init")) "npm command did not preserve cwd/onboarding"
        $errorCall = Invoke-NativeCapture "nodepaper" @("definitely-not-a-command")
        Assert-True ($errorCall.ExitCode -eq 2) "npm wrapper did not preserve CLI exit code 2"
    }
    finally { Pop-Location }

    $uninstallCall = Invoke-NativeCapture $npm.Source @("uninstall", "--global", "--prefix", $prefix, "nodepaper-local-experiment")
    if ($uninstallCall.ExitCode -ne 0) { throw "isolated npm uninstall failed:`n$($uninstallCall.Text)" }

    $tgzInfo = Get-Item -LiteralPath $tgz
    [long]$payloadSize = 0
    foreach ($payloadEntry in $payloadFiles) { $payloadSize += [long]$payloadEntry["size"] }
    $result = [ordered]@{
        schemaVersion = 1
        channel = "npm-local-experiment"
        version = $version
        sourceCommit = $SourceCommit
        payloadBytes = $payloadSize
        packageBytes = $tgzInfo.Length
        packageSHA256 = (Get-FileHash -LiteralPath $tgz -Algorithm SHA256).Hash.ToLowerInvariant()
        nodeVersion = ((& node --version) -join " ").Trim()
        npmVersion = ((& $npm.Source --version) -join " ").Trim()
        installPrefix = "isolated temporary prefix"
        publishPerformed = $false
        result = "pass"
        testedAt = (Get-Date).ToString("o")
    }
    $resultPath = Join-Path $OutputDirectory "npm-channel-experiment.json"
    $result | ConvertTo-Json -Depth 5 | Set-Content -LiteralPath $resultPath -Encoding UTF8
    Write-Host "Local npm channel experiment passed."
    Write-Host "TGZ: $tgz"
    Write-Host "Result: $resultPath"
    $passed = $true
}
finally {
    $env:PATH = $originalPath
    if (($passed -and -not $KeepWorkDirectory) -and (Test-Path -LiteralPath $work)) {
        Remove-Item -LiteralPath $work -Recurse -Force -ErrorAction SilentlyContinue
    }
    elseif (Test-Path -LiteralPath $work) {
        Write-Host "Experiment work directory retained: $work"
    }
}
