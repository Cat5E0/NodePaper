param(
    [string]$Fixture = "powershell-baseline-valid",
    [switch]$KeepWorkDirectory
)

. (Join-Path $PSScriptRoot "test-common.ps1")
$root = Get-NodePaperRepoRoot
$fixtureRoot = Join-Path $root "nodepaper-test-fixtures\tests\fixtures\$Fixture"
if (-not (Test-Path -LiteralPath $fixtureRoot -PathType Container)) {
    throw "Fixture not found: $fixtureRoot"
}

# Every real E2E runs through a Unicode path containing spaces so the
# PowerShell transition boundary is continuously checked for path quoting.
# Character codes avoid Windows PowerShell 5.1 misreading a BOM-less script.
$unicodePathPart = ([string][char]0x4E2D) + ([char]0x6587)
$workRoot = Join-Path ([System.IO.Path]::GetTempPath()) ("nodepaper e2e $unicodePathPart " + [Guid]::NewGuid().ToString("N"))
$projectDir = Join-Path $workRoot "project"
$exePath = Join-Path $workRoot "nodepaper.exe"
$passed = $false

function Get-FixtureSnapshot([string]$Path) {
    $result = [ordered]@{}
    Get-ChildItem -LiteralPath $Path -Recurse -File | Sort-Object FullName | ForEach-Object {
        $relative = $_.FullName.Substring($Path.Length).TrimStart('\')
        $result[$relative] = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash
    }
    return ($result | ConvertTo-Json -Compress)
}

try {
    New-Item -ItemType Directory -Force -Path $projectDir | Out-Null
    Copy-Item -Path (Join-Path $fixtureRoot "*") -Destination $projectDir -Recurse -Force
    $before = Get-FixtureSnapshot $fixtureRoot

    $go = Get-NodePaperGo
    Push-Location $root
    try {
        & $go build -o $exePath ./cmd/nodepaper
        if ($LASTEXITCODE -ne 0) {
            throw "go build failed with exit code $LASTEXITCODE"
        }
    }
    finally {
        Pop-Location
    }

    $toolPaths = @(
        (Join-Path $root "tools\windows-x64\pandoc"),
        (Join-Path $root "tools\windows-x64\pandoc-crossref")
    )
    $env:PATH = ($toolPaths -join ';') + ';' + $env:PATH

    # M2 runs the source-tree transition script. M4 ZIP tests will instead
    # verify the default adjacent-to-executable packaged resource layout.
    $env:NODEPAPER_BUILD_SCRIPT = Join-Path $root "Build-Paper.ps1"

    & $exePath doctor $projectDir --format json
    if ($LASTEXITCODE -ne 0) {
        throw "doctor failed with exit code $LASTEXITCODE"
    }
    & $exePath validate $projectDir --format json
    if ($LASTEXITCODE -ne 0) {
        throw "validate failed with exit code $LASTEXITCODE"
    }
    & $exePath build $projectDir --format json
    if ($LASTEXITCODE -ne 0) {
        throw "build failed with exit code $LASTEXITCODE"
    }

    $tex = Join-Path $projectDir ".nodepaper\build\paper.tex"
    if (-not (Test-Path -LiteralPath $tex -PathType Leaf)) {
        throw "generated LaTeX not found: $tex"
    }
    if (-not (Select-String -LiteralPath $tex -SimpleMatch "\documentclass" -Quiet)) {
        throw "generated LaTeX is not standalone: $tex"
    }

    $logs = @(Get-ChildItem -LiteralPath (Join-Path $projectDir ".nodepaper\logs") -Filter "build-*.log" -File)
    if ($logs.Count -lt 1 -or $logs[0].Length -eq 0) {
        throw "complete build log was not created"
    }
    $goBuildLog = Get-Content -LiteralPath $logs[0].FullName -Raw -Encoding UTF8
    if (-not $goBuildLog.Contains("Command: powershell.exe") -or -not $goBuildLog.Contains("Build-Paper.ps1")) {
        throw "Go build log does not prove delegation to the PowerShell transition script"
    }
    $transitionLogs = @(Get-ChildItem -LiteralPath (Join-Path $projectDir ".nodepaper\logs") -Directory -Filter "*-powershell" |
        ForEach-Object { Get-ChildItem -LiteralPath $_.FullName -Filter "build-*.log" -File })
    if ($transitionLogs.Count -lt 1 -or $transitionLogs[0].Length -eq 0) {
        throw "PowerShell transition log was not created"
    }

    # A second immediate build verifies repeatability and Build ID/log uniqueness.
    & $exePath build $projectDir --format json
    if ($LASTEXITCODE -ne 0) {
        throw "second build failed with exit code $LASTEXITCODE"
    }
    $logs = @(Get-ChildItem -LiteralPath (Join-Path $projectDir ".nodepaper\logs") -Filter "build-*.log" -File)
    if ($logs.Count -lt 2) {
        throw "two builds did not create distinct log files"
    }

    $pdf = Join-Path $projectDir "dist\paper.pdf"
    if (-not (Test-Path -LiteralPath $pdf -PathType Leaf)) {
        throw "PDF not found: $pdf"
    }
    $bytes = [System.IO.File]::ReadAllBytes($pdf)
    if ($bytes.Length -lt 5 -or [System.Text.Encoding]::ASCII.GetString($bytes, 0, 5) -ne "%PDF-") {
        throw "Generated artifact is not a valid PDF header: $pdf"
    }

    $pdfInfo = Get-Command "pdfinfo.exe" -ErrorAction SilentlyContinue
    $pdfToText = Get-Command "pdftotext.exe" -ErrorAction SilentlyContinue
    if (-not $pdfInfo -or -not $pdfToText) {
        throw "pdfinfo.exe and pdftotext.exe are required for PDF structure checks"
    }
    $infoOutput = & $pdfInfo.Source $pdf 2>&1
    if ($LASTEXITCODE -ne 0 -or -not ($infoOutput -match '^Pages:\s+[1-9][0-9]*')) {
        throw "PDF parser did not report a positive page count:`n$($infoOutput -join [Environment]::NewLine)"
    }

    $textPath = Join-Path $workRoot "paper.txt"
    & $pdfToText.Source -enc UTF-8 $pdf $textPath
    if ($LASTEXITCODE -ne 0 -or -not (Test-Path -LiteralPath $textPath -PathType Leaf)) {
        throw "pdftotext failed"
    }
    $pdfText = Get-Content -LiteralPath $textPath -Raw -Encoding UTF8
    $abstractHeading = ([string][char]0x6458) + ([char]0x8981)
    if ([string]::IsNullOrWhiteSpace($pdfText) -or -not $pdfText.Contains($abstractHeading)) {
        throw "PDF text is empty or missing the abstract heading"
    }
    if ($pdfText -match '@(fig|tbl|eq|sec):' -or $pdfText -match '@[A-Za-z][A-Za-z0-9_.:+/-]*') {
        throw "PDF contains an unresolved citation or cross-reference"
    }

    $latexLog = Join-Path $projectDir ".nodepaper\build\paper.log"
    $latexLogText = Get-Content -LiteralPath $latexLog -Raw -ErrorAction Stop
    if ($latexLogText -match 'There were undefined references' -or $latexLogText -match 'Citation.+undefined') {
        throw "LaTeX log contains unresolved references"
    }

    $after = Get-FixtureSnapshot $fixtureRoot
    if ($after -ne $before) {
        throw "Source Fixture was modified by E2E"
    }

    $passed = $true
    Write-Host "E2E passed: $Fixture"
}
finally {
    if ($passed -and -not $KeepWorkDirectory) {
        Remove-Item -LiteralPath $workRoot -Recurse -Force -ErrorAction SilentlyContinue
    }
    else {
        Write-Host "E2E work directory retained: $workRoot"
    }
}
