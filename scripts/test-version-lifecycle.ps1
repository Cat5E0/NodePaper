Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"
. (Join-Path $PSScriptRoot "version-lifecycle.ps1")

function Assert-Equal {
    param($Actual, $Expected, [string]$Message)
    if ($Actual -ne $Expected) { throw "FAIL: $Message (expected '$Expected', got '$Actual')" }
}

function Assert-Throws {
    param([scriptblock]$Action, [string]$Pattern, [string]$Message)
    try { & $Action; throw "FAIL: $Message (no error was raised)" }
    catch {
        if ($_.Exception.Message -like "FAIL: $Message*") { throw }
        if ($_.Exception.Message -notmatch $Pattern) {
            throw "FAIL: $Message (unexpected error: $($_.Exception.Message))"
        }
    }
}

$valid = @(
    @{ Version = "0.1.0-dev.184+g17bdb9e"; Stage = "dev"; Asset = "dev184" },
    @{ Version = "0.1.0-beta.1"; Stage = "beta"; Asset = "0.1.0-beta.1" },
    @{ Version = "0.1.0-rc.1"; Stage = "rc"; Asset = "0.1.0-rc.1" },
    @{ Version = "0.1.0"; Stage = "stable"; Asset = "0.1.0" }
)
foreach ($case in $valid) {
    $parsed = ConvertFrom-NodePaperVersion $case.Version
    Assert-Equal $parsed.Stage $case.Stage "stage for $($case.Version)"
    Assert-Equal $parsed.AssetVersion $case.Asset "asset version for $($case.Version)"
}
foreach ($invalid in @("1.0", "01.0.0", "0.1.0-dev.01+gabcdef0", "0.1.0-dev.1", "0.1.0-beta.0", "0.1.0-rc.01", "v0.1.0")) {
    Assert-Throws { ConvertFrom-NodePaperVersion $invalid } "Invalid NodePaper version" "reject $invalid"
}

$ordered = @("0.1.0-dev.1+gabcdef0", "0.1.0-dev.2+gabcdef0", "0.1.0-beta.1", "0.1.0-rc.1", "0.1.0")
for ($i = 0; $i -lt $ordered.Count - 1; $i++) {
    Assert-Equal (Compare-NodePaperVersion $ordered[$i] $ordered[$i + 1]) -1 "version order at index $i"
    Assert-Equal (Test-NodePaperVersionTransition $ordered[$i] $ordered[$i + 1]) $true "legal transition at index $i"
}
foreach ($transition in @(
    @{ From = "0.1.0-dev.2+gabcdef0"; To = "0.1.0-dev.1+gabcdef0" },
    @{ From = "0.1.0-beta.1"; To = "0.1.0" },
    @{ From = "0.1.0-rc.1"; To = "0.2.0-dev.1+gabcdef0" }
)) {
    Assert-Equal (Test-NodePaperVersionTransition $transition.From $transition.To) $false "illegal transition $($transition.From) -> $($transition.To)"
}
Assert-Equal (Test-NodePaperVersionTransition "0.1.0" "0.2.0-dev.1+gabcdef0") $true "stable to next development line"
Assert-Equal (Get-NodePaperAssetBaseName "0.1.0-dev.184+g17bdb9e") "nodepaper-dev184-windows-x64" "short dev asset name"

$tempRepo = Join-Path ([System.IO.Path]::GetTempPath()) ("nodepaper-version-test-" + [Guid]::NewGuid().ToString("N"))
try {
    New-Item -ItemType Directory -Path $tempRepo | Out-Null
    & git -C $tempRepo init --quiet
    & git -C $tempRepo config user.name "NodePaper Test"
    & git -C $tempRepo config user.email "test@nodepaper.invalid"
    Set-Content -LiteralPath (Join-Path $tempRepo "payload.txt") -Value "one" -Encoding UTF8
    & git -C $tempRepo add payload.txt
    & git -C $tempRepo commit --quiet -m "fixture"
    $commit = (& git -C $tempRepo rev-parse HEAD | Out-String).Trim().ToLowerInvariant()
    $devVersion = "0.1.0-dev.1+g$($commit.Substring(0, 7))"
    $identity = Assert-NodePaperBuildIdentity -Version $devVersion -ResolvedCommit $commit -RepositoryRoot $tempRepo
    Assert-Equal $identity.SourceCommit $commit "dev build source commit"
    Assert-Throws { Assert-NodePaperBuildIdentity -Version "0.1.0-dev.1+g0000000" -ResolvedCommit $commit -RepositoryRoot $tempRepo } "does not match" "reject mismatched dev commit"
    Assert-Throws { Assert-NodePaperBuildIdentity -Version "0.1.0-beta.1" -ResolvedCommit $commit -RepositoryRoot $tempRepo } "requires annotated tag" "reject public build without tag"

    & git -C $tempRepo tag v0.1.0-beta.1
    Assert-Throws { Assert-NodePaperBuildIdentity -Version "0.1.0-beta.1" -ResolvedCommit $commit -RepositoryRoot $tempRepo } "requires annotated tag" "reject lightweight public tag"
    & git -C $tempRepo tag -d v0.1.0-beta.1 | Out-Null
    & git -C $tempRepo tag -a v0.1.0-beta.1 -m "beta fixture"
    $beta = Assert-NodePaperBuildIdentity -Version "0.1.0-beta.1" -ResolvedCommit $commit -RepositoryRoot $tempRepo
    Assert-Equal $beta.GitTag "v0.1.0-beta.1" "annotated beta tag"

    & git -C $tempRepo tag -a v0.1.0-rc.1 -m "rc fixture"
    Assert-Throws { Assert-NodePaperBuildIdentity -Version "0.1.0-rc.1" -ResolvedCommit $commit -RepositoryRoot $tempRepo } "FeatureFreeze" "reject RC without freeze assertion"
    Assert-Throws { Assert-NodePaperBuildIdentity -Version "0.1.0-rc.1" -ResolvedCommit $commit -RepositoryRoot $tempRepo -FeatureFreeze } "NoReleaseBlockers" "reject RC without blocker assertion"
    $rc = Assert-NodePaperBuildIdentity -Version "0.1.0-rc.1" -ResolvedCommit $commit -RepositoryRoot $tempRepo -FeatureFreeze -NoReleaseBlockers
    Assert-Equal $rc.Stage "rc" "accepted RC identity"

    $hash1 = Get-NodePaperPayloadSHA256 $tempRepo
    Set-Content -LiteralPath (Join-Path $tempRepo "build-info.json") -Value '{"ignored":1}' -Encoding UTF8
    Set-Content -LiteralPath (Join-Path $tempRepo "payload-manifest.json") -Value '{"ignored":1}' -Encoding UTF8
    Assert-Equal (Get-NodePaperPayloadSHA256 $tempRepo) $hash1 "metadata files excluded from payload hash"
    Set-Content -LiteralPath (Join-Path $tempRepo "payload.txt") -Value "two" -Encoding UTF8
    if ((Get-NodePaperPayloadSHA256 $tempRepo) -eq $hash1) { throw "FAIL: payload hash did not change with payload content" }
}
finally {
    if (Test-Path -LiteralPath $tempRepo) { Remove-Item -LiteralPath $tempRepo -Recurse -Force }
}

Write-Host "NodePaper version lifecycle tests passed."
