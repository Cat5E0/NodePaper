<#
.SYNOPSIS
    Deterministic XeLaTeX pass/convergence tests with a fake xelatex.

.DESCRIPTION
    Drives the real Build-Paper.ps1 transition script with a fake xelatex whose
    paper.log decides whether the pass loop asks for another run. Covers every
    convergence branch without needing a real TeX installation:

      1. converges after 1 pass
      2. converges after 2 passes
      3. converges after 3 passes
      4. converges after 4 passes
      5. still requests a rerun after 4 passes -> build fails, no PDF published
      6. non-zero XeLaTeX exit -> build fails
      7. missing paper.log -> build fails

    Every scenario also asserts the run log never invokes latexmk, proving the
    direct XeLaTeX drive stays in force.

.PARAMETER KeepWorkDirectory
    Keep the temporary work directory for diagnosis.
#>
param(
    [switch]$KeepWorkDirectory
)

$ErrorActionPreference = "Stop"
$passed = $false
$workRoot = ""

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

function Invoke-FakeScenario {
    param(
        [string]$Name,
        [string]$ProjectDir,
        [string]$BuildScript,
        [string]$ProfileDir,
        [hashtable]$FakeEnvironment,
        [int]$ExpectedExitCode,
        [string[]]$RequiredLogMarkers,
        [string]$ForbiddenLogMarker = "latexmk"
    )

    $buildDir = Join-Path $ProjectDir "build"
    $logDir = Join-Path $ProjectDir "logs"
    New-Item -ItemType Directory -Force -Path $buildDir | Out-Null
    New-Item -ItemType Directory -Force -Path $logDir | Out-Null

    $manifestPath = Join-Path $ProjectDir "sources.json"
    @{ sources = @((Join-Path $ProjectDir "paper.md")); latexFragments = @() } |
        ConvertTo-Json | Set-Content -LiteralPath $manifestPath -Encoding UTF8

    $fakeDir = Join-Path $ProjectDir "fake-xelatex"
    New-Item -ItemType Directory -Force -Path $fakeDir | Out-Null
    $fakeSourceDir = Split-Path $PSCommandPath -Parent
    Copy-Item -LiteralPath (Join-Path $fakeSourceDir "fake-xelatex.cmd") -Destination (Join-Path $fakeDir "xelatex.cmd") -Force
    Copy-Item -LiteralPath (Join-Path $fakeSourceDir "fake-xelatex.ps1") -Destination (Join-Path $fakeDir "fake-xelatex.ps1") -Force

    # Remove any directory that already contains a real xelatex (TeX Live or
    # MiKTeX) so Get-Command inside Build-Paper.ps1 resolves the fake first.
    # PATH entries can be empty, quoted or non-rooted; drop those untouched.
    $originalPath = [Environment]::GetEnvironmentVariable("Path", "Process")
    $cleanedPath = @($originalPath -split ';' | Where-Object {
        $entry = $_.Trim().Trim('"')
        if ([string]::IsNullOrWhiteSpace($entry) -or -not [System.IO.Path]::IsPathRooted($entry)) {
            $true
        }
        else {
            try {
                -not (Test-Path -LiteralPath (Join-Path $entry "xelatex.exe") -ErrorAction Stop)
            }
            catch {
                $true
            }
        }
    }) -join ';'
    $env:PATH = "$fakeDir;$cleanedPath"
    foreach ($key in @("NODEPAPER_FAKE_RERUN_PASSES", "NODEPAPER_FAKE_NO_LOG", "NODEPAPER_FAKE_EXIT_CODE")) {
        if ($FakeEnvironment.ContainsKey($key)) {
            Set-Item -Path "Env:$key" -Value ([string]$FakeEnvironment[$key])
        }
        else {
            Remove-Item -Path "Env:$key" -ErrorAction SilentlyContinue
        }
    }

    $output = Join-Path $buildDir "paper.tex"
    $callError = ""
    try {
        & $BuildScript -SourceManifest $manifestPath -ProjectRoot $ProjectDir -ProfileDirectory $ProfileDir `
            -Output $output -BuildDirectory $buildDir -LogDirectory $logDir 2>&1 | ForEach-Object { Write-Host $_ }
        $actualExit = $LASTEXITCODE
        if ($null -eq $actualExit) { $actualExit = 0 }
    }
    catch {
        $actualExit = 1
        $callError = $_.Exception.Message
    }

    # Restore the caller environment before asserting so failures are easy to
    # reproduce and later scenarios start clean.
    [Environment]::SetEnvironmentVariable("Path", $originalPath, "Process")
    $env:PATH = $originalPath
    foreach ($key in @("NODEPAPER_FAKE_RERUN_PASSES", "NODEPAPER_FAKE_NO_LOG", "NODEPAPER_FAKE_EXIT_CODE")) {
        Remove-Item -Path "Env:$key" -ErrorAction SilentlyContinue
    }

    $runLogs = @(Get-ChildItem -LiteralPath $logDir -Filter "build-*.log" -File -ErrorAction SilentlyContinue)
    Assert-True ($runLogs.Count -ge 1) "${Name}: run log was not created"
    $logText = Get-Content -LiteralPath $runLogs[0].FullName -Raw -Encoding UTF8
    Assert-True ($actualExit -eq $ExpectedExitCode) "${Name}: exit code $actualExit, want $ExpectedExitCode. Error: $callError. Log tail:`n$($logText | Select-Object -Last 30)"
    foreach ($marker in $RequiredLogMarkers) {
        Assert-True ($logText.Contains($marker)) "${Name}: run log is missing '$marker'`n$logText"
    }
    if ($ForbiddenLogMarker) {
        Assert-True (-not $logText.Contains($ForbiddenLogMarker)) "${Name}: run log still mentions $ForbiddenLogMarker"
    }
    Write-Host "PASS: ${Name}"
}

try {
    $root = (Resolve-Path -LiteralPath (Join-Path $PSScriptRoot "..")).Path
    $buildScript = Join-Path $root "scripts\build\Build-Paper.ps1"
    $profileDir = Join-Path $root "profiles\cumcm"
    $fixtureRoot = Join-Path $root "tests\fixtures\minimal-valid"
    if (-not (Test-Path -LiteralPath $buildScript -PathType Leaf) -or -not (Test-Path -LiteralPath $profileDir -PathType Container) -or -not (Test-Path -LiteralPath $fixtureRoot -PathType Container)) {
        throw "Missing build script, profile or minimal-valid fixture under $root"
    }

    $unicodePathPart = ([string][char]0x4E2D) + ([char]0x6587)
    $workRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("nodepaper xelatex converge $unicodePathPart " + [Guid]::NewGuid().ToString("N"))
    New-Item -ItemType Directory -Force -Path $workRoot | Out-Null

    $cases = @(
        @{ Name = "converges after 1 pass";       Env = @{};                                  Exit = 0; Markers = @("LaTeX converged after 1 pass") },
        @{ Name = "converges after 2 passes";     Env = @{ NODEPAPER_FAKE_RERUN_PASSES = 1 }; Exit = 0; Markers = @("LaTeX pass 1 requested another run.", "LaTeX converged after 2 pass") },
        @{ Name = "converges after 3 passes";     Env = @{ NODEPAPER_FAKE_RERUN_PASSES = 2 }; Exit = 0; Markers = @("LaTeX converged after 3 pass") },
        @{ Name = "converges after 4 passes";     Env = @{ NODEPAPER_FAKE_RERUN_PASSES = 3 }; Exit = 0; Markers = @("LaTeX pass 4", "LaTeX converged after 4 pass") },
        @{ Name = "unconverged after 4 passes";   Env = @{ NODEPAPER_FAKE_RERUN_PASSES = 9 }; Exit = 1; Markers = @("LaTeX pass 4 requested another run.", "convergence was not reached") },
        @{ Name = "non-zero XeLaTeX exit";        Env = @{ NODEPAPER_FAKE_EXIT_CODE = 2 };    Exit = 1; Markers = @("LaTeX pass 1 exited non-zero") },
        @{ Name = "missing paper.log";            Env = @{ NODEPAPER_FAKE_NO_LOG = 1 };       Exit = 1; Markers = @("produced no log") }
    )

    foreach ($case in $cases) {
        $projectDir = Join-Path $workRoot ($case.Name -replace '[^a-z0-9]+', '-')
        New-Item -ItemType Directory -Force -Path $projectDir | Out-Null
        Copy-Item -Path (Join-Path $fixtureRoot "*") -Destination $projectDir -Recurse -Force
        Invoke-FakeScenario -Name $case.Name -ProjectDir $projectDir -BuildScript $buildScript -ProfileDir $profileDir `
            -FakeEnvironment $case.Env -ExpectedExitCode $case.Exit -RequiredLogMarkers $case.Markers
    }

    $passed = $true
    Write-Host "XeLaTeX convergence suite passed (7 scenarios)."
}
finally {
    if ($passed -and -not $KeepWorkDirectory) {
        Remove-Item -LiteralPath $workRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
    else {
        Write-Host "Convergence test work directory retained: $workRoot"
    }
}
