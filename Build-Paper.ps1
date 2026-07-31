param(
    [Alias("Input")]
    [string]$MarkdownPath = "",

    [string]$SourceManifest = "",

    [string]$ProjectRoot = "",

    [string]$ProfileDirectory = "",

    [string]$Output = ".\build\Paper.tex",

    [ValidateSet("thesis", "assignment", "experiment")]
    [string]$TemplateName = "thesis",

    [string]$Template = "",

    [string]$BuildDirectory = ".\build",

    [switch]$AllowSystemPandoc,

    [switch]$SkipPdf,

    [string]$LogDirectory = ".\logs",

    [string]$CoverPdf = "",

    [string]$LastPagePdf = "",

    [string]$CoverLastPdf = ""
)

$ErrorActionPreference = "Stop"

function Resolve-ProjectPath {
    param([string]$Path)
    if ([System.IO.Path]::IsPathRooted($Path)) {
        return [System.IO.Path]::GetFullPath($Path)
    }
    return [System.IO.Path]::GetFullPath((Join-Path $PSScriptRoot $Path))
}

function Find-CommandPath {
    param([string[]]$Candidates)
    foreach ($candidate in $Candidates) {
        if ([System.IO.Path]::IsPathRooted($candidate) -and (Test-Path -LiteralPath $candidate -PathType Leaf)) {
            return (Resolve-Path -LiteralPath $candidate).Path
        }
        $cmd = Get-Command $candidate -ErrorAction SilentlyContinue
        if ($cmd) {
            return $cmd.Source
        }
    }
    return $null
}

function Copy-LatexLog {
    param(
        [string]$BuildDir,
        [string]$OutputPath,
        [string]$LogDir,
        [string]$Stamp
    )

    $latexLog = Join-Path $BuildDir ([System.IO.Path]::GetFileNameWithoutExtension($OutputPath) + ".log")
    if (Test-Path -LiteralPath $latexLog -PathType Leaf) {
        $copyPath = Join-Path $LogDir ("latex-$Stamp.log")
        Copy-Item -LiteralPath $latexLog -Destination $copyPath -Force
        Write-Log "Copied LaTeX log: $copyPath"
    }
}

function Write-Log {
    param(
        [string]$Message,
        [string]$Level = "INFO"
    )

    $line = "[{0}] [{1}] {2}" -f (Get-Date -Format "yyyy-MM-dd HH:mm:ss"), $Level, $Message
    Write-Host $line
    Add-Content -LiteralPath $script:RunLogPath -Value $line -Encoding UTF8
}

function Invoke-LoggedCommand {
    param(
        [string]$FilePath,
        [string[]]$Arguments,
        [string]$StepName
    )

    Write-Log "$StepName command: $FilePath $($Arguments -join ' ')"
    $previousErrorActionPreference = $ErrorActionPreference
    try {
        $ErrorActionPreference = "Continue"
        & $FilePath @Arguments 2>&1 | ForEach-Object {
            $line = $_.ToString()
            Write-Host $line
            Add-Content -LiteralPath $script:RunLogPath -Value $line -Encoding UTF8
        }
        $exitCode = $LASTEXITCODE
        if ($null -eq $exitCode) {
            $exitCode = 0
        }
        Write-Log "$StepName exit code: $exitCode"
        return [int]$exitCode
    }
    catch {
        Write-Log "$StepName failed: $($_.Exception.Message)" "ERROR"
        return 1
    }
    finally {
        $ErrorActionPreference = $previousErrorActionPreference
    }
}

function Invoke-LoggedScript {
    param(
        [string]$FilePath,
        [hashtable]$Parameters,
        [string]$StepName
    )

    $renderedArgs = ($Parameters.GetEnumerator() | ForEach-Object {
        if ($_.Value -is [switch] -or $_.Value -is [bool]) {
            if ($_.Value) { "-$($_.Key)" }
        }
        else {
            "-$($_.Key) $($_.Value)"
        }
    }) -join " "
    Write-Log "$StepName command: $FilePath $renderedArgs"

    try {
        & $FilePath @Parameters 2>&1 | ForEach-Object {
            $line = $_.ToString()
            Write-Host $line
            Add-Content -LiteralPath $script:RunLogPath -Value $line -Encoding UTF8
        }
        Write-Log "$StepName exit code: 0"
        return 0
    }
    catch {
        Write-Log "$StepName failed: $($_.Exception.Message)" "ERROR"
        return 1
    }
}

$buildDir = Resolve-ProjectPath $BuildDirectory
$outputPath = Resolve-ProjectPath $Output
$logDir = Resolve-ProjectPath $LogDirectory
New-Item -ItemType Directory -Force -Path $buildDir | Out-Null
New-Item -ItemType Directory -Force -Path $logDir | Out-Null

$stamp = Get-Date -Format "yyyyMMdd-HHmmss"
$script:RunLogPath = Join-Path $logDir ("build-$stamp.log")
$inputDescription = $MarkdownPath
if (-not [string]::IsNullOrWhiteSpace($SourceManifest)) {
    $inputDescription = "SourceManifest=$(Resolve-ProjectPath $SourceManifest); ProjectRoot=$(Resolve-ProjectPath $ProjectRoot)"
}
elseif (-not [string]::IsNullOrWhiteSpace($MarkdownPath)) {
    $inputDescription = Resolve-ProjectPath $MarkdownPath
}
Set-Content -LiteralPath $script:RunLogPath -Value @(
    "NodePaper build log",
    "Started: $(Get-Date -Format 'yyyy-MM-dd HH:mm:ss')",
    "PowerShell: $($PSVersionTable.PSVersion)",
    "Input: $inputDescription",
    "Output: $outputPath",
    "TemplateName: $TemplateName",
    "Template: $Template",
    "ProfileDirectory: $ProfileDirectory",
    "BuildDirectory: $buildDir"
) -Encoding UTF8
Write-Log "Build log: $script:RunLogPath"

if (-not [string]::IsNullOrWhiteSpace($SourceManifest)) {
    if ([string]::IsNullOrWhiteSpace($ProjectRoot) -or [string]::IsNullOrWhiteSpace($ProfileDirectory)) {
        throw "ProjectRoot and ProfileDirectory are required with SourceManifest."
    }
    $convertScript = Join-Path $PSScriptRoot "Convert-CumcmProjectToLatex.ps1"
    $convertParams = @{
        SourceManifest = (Resolve-ProjectPath $SourceManifest)
        ProjectRoot = (Resolve-ProjectPath $ProjectRoot)
        ProfileDirectory = (Resolve-ProjectPath $ProfileDirectory)
        Output = $outputPath
        BuildDirectory = $buildDir
    }
    if ($AllowSystemPandoc) {
        $convertParams.AllowSystemPandoc = $true
    }
}
else {
    if ([string]::IsNullOrWhiteSpace($MarkdownPath)) {
        throw "MarkdownPath is required for the legacy conversion mode."
    }
    $convertScript = Join-Path $PSScriptRoot "Convert-MarkdownToScauLatex.ps1"
    $convertParams = @{
        MarkdownPath = (Resolve-ProjectPath $MarkdownPath)
        Output = $outputPath
        TemplateName = $TemplateName
        BuildDirectory = $buildDir
    }
    if (-not [string]::IsNullOrWhiteSpace($Template)) {
        $convertParams.Template = (Resolve-ProjectPath $Template)
    }
    if ($AllowSystemPandoc) {
        $convertParams.AllowSystemPandoc = $true
    }
    if ($CoverPdf) {
        $convertParams.CoverPdf = (Resolve-ProjectPath $CoverPdf)
    }
    if ($LastPagePdf) {
        $convertParams.LastPagePdf = (Resolve-ProjectPath $LastPagePdf)
    }
    if ($CoverLastPdf) {
        $convertParams.CoverLastPdf = (Resolve-ProjectPath $CoverLastPdf)
    }
}

$convertCode = Invoke-LoggedScript -FilePath $convertScript -Parameters $convertParams -StepName "Convert"
if ($convertCode -ne 0) {
    throw "Conversion failed."
}

if ($SkipPdf) {
    Write-Log "Generated LaTeX only: $outputPath"
    exit 0
}

$latexmk = Find-CommandPath @(
    (Join-Path $PSScriptRoot "tools\windows-x64\texlive\bin\windows\latexmk.exe"),
    "latexmk.exe",
    "latexmk"
)
$xelatex = Find-CommandPath @(
    (Join-Path $PSScriptRoot "tools\windows-x64\texlive\bin\windows\xelatex.exe"),
    "xelatex.exe",
    "xelatex"
)

if (-not $latexmk -or -not $xelatex) {
    Write-Log "LaTeX compiler not found. Generated LaTeX, but skipped PDF: $outputPath" "WARN"
    exit 0
}

$compileArgs = @(
    "-xelatex",
    "-interaction=nonstopmode",
    "-file-line-error",
    "-f",
    "-outdir=$buildDir",
    $outputPath
)

$code = Invoke-LoggedCommand -FilePath $latexmk -Arguments $compileArgs -StepName "LaTeX"
if ($code -ne 0) {
    $logPath = Join-Path $buildDir ([System.IO.Path]::GetFileNameWithoutExtension($outputPath) + ".log")
    if (Test-Path -LiteralPath $logPath) {
        Write-Host ""
        Write-Host "Last LaTeX log lines:"
        Get-Content -LiteralPath $logPath -Tail 40 -Encoding UTF8
    }
    Copy-LatexLog $buildDir $outputPath $logDir $stamp
    throw "LaTeX build failed with exit code $code. See $logPath"
}

$pdfPath = Join-Path $buildDir ([System.IO.Path]::GetFileNameWithoutExtension($outputPath) + ".pdf")
Write-Log "Generated PDF: $pdfPath"
Copy-LatexLog $buildDir $outputPath $logDir $stamp
Write-Log "Build complete."
