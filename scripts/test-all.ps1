param(
    [switch]$Race,
    [switch]$IncludeE2E
)

$ErrorActionPreference = "Stop"
& (Join-Path $PSScriptRoot "test-unit.ps1") -Race:$Race
& (Join-Path $PSScriptRoot "test-integration.ps1") -Race:$Race
if ($IncludeE2E) {
    & (Join-Path $PSScriptRoot "test-e2e.ps1")
}
