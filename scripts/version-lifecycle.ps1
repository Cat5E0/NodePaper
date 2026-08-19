Set-StrictMode -Version Latest

function ConvertFrom-NodePaperVersion {
    param([Parameter(Mandatory = $true)][string]$Version)
    $pattern = '^(?<major>0|[1-9][0-9]*)\.(?<minor>0|[1-9][0-9]*)\.(?<patch>0|[1-9][0-9]*)(?:(?:-dev\.(?<dev>0|[1-9][0-9]*)\+g(?<commit>[0-9a-fA-F]{7,40}))|(?:-beta\.(?<beta>[1-9][0-9]*))|(?:-rc\.(?<rc>[1-9][0-9]*)))?$'
    if ($Version -notmatch $pattern) {
        throw "Invalid NodePaper version '$Version'. Expected X.Y.Z-dev.N+g<commit>, X.Y.Z-beta.N, X.Y.Z-rc.N, or X.Y.Z."
    }
    $parts = @{} + $Matches
    $stage = "stable"
    $sequence = 0
    if ($parts.ContainsKey("dev")) { $stage = "dev"; $sequence = [int64]$parts.dev }
    elseif ($parts.ContainsKey("beta")) { $stage = "beta"; $sequence = [int64]$parts.beta }
    elseif ($parts.ContainsKey("rc")) { $stage = "rc"; $sequence = [int64]$parts.rc }
    $assetVersion = if ($stage -eq "dev") { "dev$sequence" } else { $Version }
    return [pscustomobject]@{
        Version = $Version
        Major = [int64]$parts.major
        Minor = [int64]$parts.minor
        Patch = [int64]$parts.patch
        Core = "$($parts.major).$($parts.minor).$($parts.patch)"
        Stage = $stage
        Sequence = $sequence
        CommitSuffix = if ($stage -eq "dev") { $parts.commit.ToLowerInvariant() } else { "" }
        IsPublic = ($stage -ne "dev")
        AssetVersion = $assetVersion
    }
}

function Compare-NodePaperVersion {
    param([Parameter(Mandatory = $true)][string]$Left, [Parameter(Mandatory = $true)][string]$Right)
    $a = ConvertFrom-NodePaperVersion $Left
    $b = ConvertFrom-NodePaperVersion $Right
    foreach ($field in @("Major", "Minor", "Patch")) {
        if ($a.$field -lt $b.$field) { return -1 }
        if ($a.$field -gt $b.$field) { return 1 }
    }
    $ranks = @{ dev = 0; beta = 1; rc = 2; stable = 3 }
    if ($ranks[$a.Stage] -lt $ranks[$b.Stage]) { return -1 }
    if ($ranks[$a.Stage] -gt $ranks[$b.Stage]) { return 1 }
    if ($a.Sequence -lt $b.Sequence) { return -1 }
    if ($a.Sequence -gt $b.Sequence) { return 1 }
    return 0
}

function Test-NodePaperVersionTransition {
    param([Parameter(Mandatory = $true)][string]$From, [Parameter(Mandatory = $true)][string]$To)
    $a = ConvertFrom-NodePaperVersion $From
    $b = ConvertFrom-NodePaperVersion $To
    if ((Compare-NodePaperVersion $From $To) -ge 0) { return $false }
    if ($a.Core -ne $b.Core) { return ($a.Stage -eq "stable" -and $b.Stage -eq "dev") }
    if ($a.Stage -eq $b.Stage) { return ($a.Stage -ne "stable" -and $b.Sequence -gt $a.Sequence) }
    return (($a.Stage -eq "dev" -and $b.Stage -eq "beta") -or
        ($a.Stage -eq "beta" -and $b.Stage -eq "rc") -or
        ($a.Stage -eq "rc" -and $b.Stage -eq "stable"))
}

function Get-NodePaperAssetBaseName {
    param([Parameter(Mandatory = $true)][string]$Version, [string]$Prefix = "nodepaper", [string]$Suffix = "windows-x64")
    $identity = ConvertFrom-NodePaperVersion $Version
    return "$Prefix-$($identity.AssetVersion)-$Suffix"
}

function Assert-NodePaperBuildIdentity {
    param(
        [Parameter(Mandatory = $true)][string]$Version,
        [Parameter(Mandatory = $true)][string]$ResolvedCommit,
        [Parameter(Mandatory = $true)][string]$RepositoryRoot,
        [switch]$FeatureFreeze,
        [switch]$NoReleaseBlockers
    )
    $identity = ConvertFrom-NodePaperVersion $Version
    $commit = $ResolvedCommit.ToLowerInvariant()
    if ($commit -notmatch '^[0-9a-f]{40}$') { throw "Resolved commit must be a full 40-character Git object ID, got '$ResolvedCommit'." }
    $gitTag = ""
    if ($identity.Stage -eq "dev") {
        if (-not $commit.StartsWith($identity.CommitSuffix)) {
            throw "Dev version commit suffix g$($identity.CommitSuffix) does not match source commit $commit."
        }
    }
    else {
        $gitTag = "v$Version"
        $tagType = (& git -C $RepositoryRoot cat-file -t "refs/tags/$gitTag" 2>$null | Out-String).Trim()
        if ($LASTEXITCODE -ne 0 -or $tagType -ne "tag") { throw "Public $($identity.Stage) build requires annotated tag $gitTag." }
        $tagCommit = (& git -C $RepositoryRoot rev-parse "refs/tags/$gitTag^{}" 2>$null | Out-String).Trim().ToLowerInvariant()
        if ($LASTEXITCODE -ne 0 -or $tagCommit -ne $commit) { throw "Annotated tag $gitTag does not point to source commit $commit." }
    }
    if ($identity.Stage -eq "rc") {
        if (-not $FeatureFreeze) { throw "RC build requires -FeatureFreeze." }
        if (-not $NoReleaseBlockers) { throw "RC build requires -NoReleaseBlockers." }
    }
    return [pscustomobject]@{
        Version = $identity.Version
        Stage = $identity.Stage
        AssetVersion = $identity.AssetVersion
        SourceCommit = $commit
        GitTag = $gitTag
    }
}

function Get-NodePaperCanonicalBuildTime {
    param([Parameter(Mandatory = $true)][string]$RepositoryRoot, [Parameter(Mandatory = $true)][string]$ResolvedCommit)
    $epochText = (& git -C $RepositoryRoot show -s --format=%ct $ResolvedCommit 2>$null | Out-String).Trim()
    if ($LASTEXITCODE -ne 0 -or $epochText -notmatch '^[0-9]+$') { throw "Cannot read the canonical UTC timestamp for commit $ResolvedCommit." }
    return [DateTimeOffset]::FromUnixTimeSeconds([int64]$epochText).UtcDateTime.ToString("yyyy-MM-dd'T'HH:mm:ss'Z'")
}

function Get-NodePaperPayloadSHA256 {
    param([Parameter(Mandatory = $true)][string]$PayloadDirectory)
    $root = (Resolve-Path -LiteralPath $PayloadDirectory).Path.TrimEnd('\')
    $builder = New-Object System.Text.StringBuilder
    Get-ChildItem -LiteralPath $root -Recurse -File |
        Where-Object { $_.Name -notin @("build-info.json", "payload-manifest.json") } |
        Sort-Object FullName |
        ForEach-Object {
            $relative = $_.FullName.Substring($root.Length).TrimStart('\') -replace '\\', '/'
            [void]$builder.Append($relative).Append([char]0).Append($_.Length).Append([char]0)
            [void]$builder.Append((Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant()).Append("`n")
        }
    $sha = [System.Security.Cryptography.SHA256]::Create()
    try {
        $bytes = [System.Text.Encoding]::UTF8.GetBytes($builder.ToString())
        return ([BitConverter]::ToString($sha.ComputeHash($bytes))).Replace("-", "").ToLowerInvariant()
    }
    finally { $sha.Dispose() }
}
