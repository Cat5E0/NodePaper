param(
    [Parameter(Mandatory = $true)]
    [string]$OutputDirectory
)

$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "test-common.ps1")

New-Item -ItemType Directory -Force -Path $OutputDirectory | Out-Null
foreach ($style in @("tango", "pygments", "kate")) {
    $styleOutput = Join-Path $OutputDirectory $style
    & (Join-Path $PSScriptRoot "test-e2e.ps1") -Fixture "highlight-showcase" -HighlightStyle $style -ReviewOutput $styleOutput
    $nestedOutput = Join-Path $styleOutput "highlight-showcase"
    Get-ChildItem -LiteralPath $nestedOutput -File | ForEach-Object {
        Move-Item -LiteralPath $_.FullName -Destination (Join-Path $styleOutput $_.Name) -Force
    }
    Remove-Item -LiteralPath $nestedOutput -Recurse -Force
    Write-Host "Highlight sample exported: $styleOutput"
}
