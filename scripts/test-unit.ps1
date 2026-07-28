param(
    [switch]$Race
)

. (Join-Path $PSScriptRoot "test-common.ps1")
$root = Get-NodePaperRepoRoot
Push-Location $root
try {
    Assert-GoFormatting
    Invoke-NodePaperGo -Arguments @("test", "-count=1", "./...")
    Invoke-NodePaperGo -Arguments @("vet", "./...")
    if ($Race) {
        Invoke-NodePaperGo -Arguments @("test", "-race", "-count=1", "./...")
    }
}
finally {
    Pop-Location
}
