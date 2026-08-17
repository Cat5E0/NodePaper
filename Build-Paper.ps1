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

    [ValidateSet("citeproc", "natbib", "biblatex")]
    [string]$CiteMethod = "citeproc",

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

# Mirrors internal/latexlog CategoryRerun (NP6104) so the pass loop and the
# Go-side log inspection agree on what still needs another XeLaTeX run. The
# hyperref /PageLabels rerun is also included: without it the final pass would
# leave that package warning and Go would classify it as an unknown warning.
function Test-LaTeXNeedsRerun {
    param([string]$LogText)
    if ([string]::IsNullOrWhiteSpace($LogText)) {
        return $false
    }
    return $LogText -match 'Label\(s\) may have changed|Rerun to get cross-references right|Rerun to get /PageLabels entry|Please .*rerun|rerunfilecheck Warning:'
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
    # Only forwarded when it differs from the Convert script's own default, so
    # the citeproc build command line stays exactly what it has always been.
    if ($CiteMethod -ne "citeproc") {
        $convertParams.CiteMethod = $CiteMethod
    }
    if ($AllowSystemPandoc) {
        $convertParams.AllowSystemPandoc = $true
    }
}
else {
    if ([string]::IsNullOrWhiteSpace($MarkdownPath)) {
        throw "MarkdownPath is required for the legacy conversion mode."
    }
    if ($CiteMethod -ne "citeproc") {
        throw "CiteMethod $CiteMethod requires the CUMCM Profile route (SourceManifest); the legacy SCAU conversion only supports citeproc."
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

$xelatex = Find-CommandPath @(
    (Join-Path $PSScriptRoot "tools\windows-x64\texlive\bin\windows\xelatex.exe"),
    "xelatex.exe",
    "xelatex"
)

if (-not $xelatex) {
    Write-Log "LaTeX compiler not found. Generated LaTeX, but skipped PDF: $outputPath" "WARN"
    exit 0
}

# XeLaTeX is driven directly instead of through latexmk. latexmk is a Perl
# script and MiKTeX ships no Perl interpreter, so requiring it made every
# MiKTeX installation fail on a dependency the user cannot see. Its remaining
# value here was only "re-run until cross-references stabilise": citations are
# already resolved by Pandoc's citeproc before the .tex exists, so no
# bibtex/biber stage is needed.
$latexLogPath = Join-Path $buildDir ([System.IO.Path]::GetFileNameWithoutExtension($outputPath) + ".log")
# TeX reads the file name on its command line as TeX tokens instead of as a
# literal string, so an absolute path is not safe to hand it: "~" is an active
# character (a non-breaking space). Windows gives every account name longer
# than eight characters an 8.3 short name and %TEMP% sits under the user
# profile, so a build under a temp directory below such a profile stopped with
# "I can't find file" naming only the part before the tilde - on CI and equally
# for any user whose account name is long enough. Handing XeLaTeX the path relative to the
# working directory keeps the whole absolute prefix - short name, spaces and
# non-ASCII characters alike - out of the TeX tokeniser.
#
# The working directory itself must stay where the caller put it (the Go build
# runs this script in the project root). Pandoc writes image references such as
# \includegraphics{images/plot.png} relative to that directory, and TeX only
# finds them through its own working directory: moving into the build
# directory and pointing TEXINPUTS back at the project breaks every project
# whose path is not pure ASCII, because kpathsea reads that variable in the
# ANSI code page and then cannot resolve it.
$compileTarget = $outputPath
$relativeTarget = Resolve-Path -LiteralPath $outputPath -Relative -ErrorAction SilentlyContinue
if ($relativeTarget) {
    # Windows PowerShell hands back a nonsense ".\C:\..." string when the
    # target sits on another drive, so the candidate is only used when it
    # really does name the same file from the current directory.
    $roundTrip = ""
    try {
        $roundTrip = [System.IO.Path]::GetFullPath((Join-Path (Get-Location).Path $relativeTarget))
    }
    catch {
        $roundTrip = ""
    }
    if ($roundTrip -eq $outputPath) {
        # Forward slashes are mandatory here. TeX normalises the separators of
        # a drive-qualified path, but in a relative one it reads "\build" as a
        # control sequence and dies with "Undefined control sequence".
        $compileTarget = $relativeTarget.Replace("\", "/")
    }
}
$compileArgs = @(
    "-interaction=nonstopmode",
    "-file-line-error",
    "-halt-on-error=0",
    "-output-directory=$buildDir",
    $compileTarget
)

$maxPasses = 4
$code = 0
for ($pass = 1; $pass -le $maxPasses; $pass++) {
    $code = Invoke-LoggedCommand -FilePath $xelatex -Arguments $compileArgs -StepName "LaTeX pass $pass"
    if ($code -ne 0) {
        Write-Log "XeLaTeX pass $pass exited non-zero; stopping." "ERROR"
        break
    }
    if (-not (Test-Path -LiteralPath $latexLogPath -PathType Leaf)) {
        Write-Log "XeLaTeX pass $pass produced no log at $latexLogPath; stopping." "ERROR"
        $code = 1
        break
    }
    # XeLaTeX asks for another pass whenever labels, the TOC, bookmarks or
    # crossref counters moved. Stop as soon as it stops asking.
    $logText = Get-Content -LiteralPath $latexLogPath -Raw -Encoding UTF8 -ErrorAction SilentlyContinue
    if ($null -eq $logText) {
        Write-Log "XeLaTeX pass $pass log could not be read; stopping." "ERROR"
        $code = 1
        break
    }
    if (-not (Test-LaTeXNeedsRerun -LogText $logText)) {
        Write-Log "LaTeX converged after $pass pass(es); no rerun requested."
        break
    }
    Write-Log "LaTeX pass $pass requested another run."
    if ($pass -eq $maxPasses) {
        # Convergence is a NodePaper build contract: an unconverged PDF could
        # contain stale references. Fail and let Go surface NP6104 instead of
        # publishing a PDF whose rerun request we already saw.
        Write-Log "LaTeX still requested a rerun after $maxPasses passes; convergence was not reached." "ERROR"
        $code = 1
    }
}

if ($code -ne 0) {
    $logPath = $latexLogPath
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
