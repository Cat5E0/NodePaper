# Fake XeLaTeX for deterministic convergence tests. Invoked through
# xelatex.cmd (scripts/fake-xelatex.cmd) so the pass loop resolves it as a
# real command on PATH, and directly by unit debugging. Writes a tiny PDF and
# a paper.log whose rerun request is controlled by:
#   NODEPAPER_FAKE_RERUN_PASSES  request a rerun for this many passes
#                                (default 0 -> converged on pass 1)
#   NODEPAPER_FAKE_NO_LOG        when "1", do not write paper.log at all
#   NODEPAPER_FAKE_EXIT_CODE     when set, exit with this code instead of 0
$ErrorActionPreference = "Stop"

# No param block: powershell.exe -File binds a bare "-output-directory=..."
# argument positionally here (it is not a parameter of this script), so the
# raw argument list is the only reliable source of the output directory.
$outDir = ""
foreach ($argument in $args) {
    if ($argument -like "-output-directory=*") {
        $outDir = $argument.Substring("-output-directory=".Length)
    }
}
if ([string]::IsNullOrWhiteSpace($outDir)) {
    $outDir = "."
}
New-Item -ItemType Directory -Force -Path $outDir | Out-Null

$countFile = Join-Path $outDir ".fake-xelatex-count"
$current = 0
if (Test-Path -LiteralPath $countFile) {
    $current = [int]((Get-Content -LiteralPath $countFile -Raw).Trim())
}
$next = $current + 1
Set-Content -LiteralPath $countFile -Value ([string]$next) -Encoding ASCII

$pdfPath = Join-Path $outDir "paper.pdf"
Set-Content -LiteralPath $pdfPath -Value ("%PDF-1.4`nfake-xelatex pass $next`n%%EOF`n") -Encoding ASCII

if ($env:NODEPAPER_FAKE_NO_LOG -ne "1") {
    $rerunPasses = 0
    if ($env:NODEPAPER_FAKE_RERUN_PASSES) {
        $rerunPasses = [int]$env:NODEPAPER_FAKE_RERUN_PASSES
    }
    $logPath = Join-Path $outDir "paper.log"
    if ($current -lt $rerunPasses) {
        Set-Content -LiteralPath $logPath -Value "LaTeX Warning: Label(s) may have changed. Rerun to get cross-references right." -Encoding UTF8
    }
    else {
        Set-Content -LiteralPath $logPath -Value "no rerun requested" -Encoding UTF8
    }
}

if ($env:NODEPAPER_FAKE_EXIT_CODE) {
    exit ([int]$env:NODEPAPER_FAKE_EXIT_CODE)
}
exit 0
