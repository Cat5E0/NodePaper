param(
    [string]$ReleaseDirectory
)

$ErrorActionPreference = "Stop"
if (-not $ReleaseDirectory) {
    throw "ReleaseDirectory is required. Test the extracted release directory, not the source tree."
}
if (-not (Test-Path -LiteralPath $ReleaseDirectory -PathType Container)) {
    throw "Release directory not found: $ReleaseDirectory"
}

Write-Host "Release automation is not complete. Required manual gates remain:"
Write-Host "- Windows 11 + TeX Live E2E"
Write-Host "- independent MiKTeX E2E"
Write-Host "- Windows 10 smoke test"
Write-Host "- PDF manual review"
Write-Host "- license and THIRD_PARTY_NOTICES review"
Write-Host "- S0/S1/S2 review and maintainer sign-off"
throw "Release gate intentionally remains closed until the release checklist is completed."
