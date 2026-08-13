param(
    [switch]$Race,
    [switch]$IncludeE2E
)

$ErrorActionPreference = "Stop"
& (Join-Path $PSScriptRoot "test-unit.ps1") -Race:$Race
& (Join-Path $PSScriptRoot "test-integration.ps1") -Race:$Race
# Deterministic XeLaTeX pass/convergence suite (fake xelatex, no TeX needed).
& (Join-Path $PSScriptRoot "test-xelatex-convergence.ps1")
if ($IncludeE2E) {
    & (Join-Path $PSScriptRoot "test-e2e.ps1")
}
