param(
    [switch]$Race
)

. (Join-Path $PSScriptRoot "test-common.ps1")
$root = Get-NodePaperRepoRoot
$core = Get-NodePaperCoreRoot
Push-Location $core
try {
    $packages = @(
        "./internal/validate",
        "./internal/buildlock",
        "./internal/process",
        "./internal/build",
        "./cmd/nodepaper"
    )
    Invoke-NodePaperGo -Arguments (@("test", "-count=1") + $packages)
    if ($Race) {
        Invoke-NodePaperGo -Arguments (@("test", "-race", "-count=1") + $packages)
    }
}
finally {
    Pop-Location
}
