Set-StrictMode -Version Latest
$ErrorActionPreference = "Stop"

function Get-NodePaperRepoRoot {
    return (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
}

function Get-NodePaperCoreRoot {
    # Go module root after the web-UI restructure (AGENTS.md directory layout).
    return (Join-Path (Get-NodePaperRepoRoot) "nodepaper\core")
}

function Get-NodePaperGo {
    if ($env:NODEPAPER_GO) {
        if (-not (Test-Path -LiteralPath $env:NODEPAPER_GO -PathType Leaf)) {
            throw "NODEPAPER_GO does not point to a file: $env:NODEPAPER_GO"
        }
        return (Resolve-Path -LiteralPath $env:NODEPAPER_GO).Path
    }

    foreach ($name in @("go.exe", "go")) {
        $command = Get-Command $name -ErrorAction SilentlyContinue
        if ($command) {
            return $command.Source
        }
    }

    $knownPath = "G:\Software\Go\bin\go.exe"
    if (Test-Path -LiteralPath $knownPath -PathType Leaf) {
        return $knownPath
    }

    throw "Go was not found. Install Go or set NODEPAPER_GO to go.exe."
}

function Invoke-NodePaperGo {
    param(
        [Parameter(Mandatory = $true)]
        [string[]]$Arguments
    )

    $go = Get-NodePaperGo
    & $go @Arguments
    if ($LASTEXITCODE -ne 0) {
        throw "go $($Arguments -join ' ') failed with exit code $LASTEXITCODE"
    }
}

function Assert-GoFormatting {
    $go = Get-NodePaperGo
    $gofmt = Join-Path (Split-Path -Parent $go) "gofmt.exe"
    if (-not (Test-Path -LiteralPath $gofmt -PathType Leaf)) {
        $gofmt = "gofmt"
    }

    $root = Get-NodePaperRepoRoot
    $files = Get-ChildItem -LiteralPath (Join-Path $root "nodepaper\core\cmd"), (Join-Path $root "nodepaper\core\internal") -Recurse -Filter "*.go" -File |
        ForEach-Object { $_.FullName }
    $unformatted = @(& $gofmt -l @files)
    if ($LASTEXITCODE -ne 0) {
        throw "gofmt failed with exit code $LASTEXITCODE"
    }
    if ($unformatted.Count -gt 0) {
        throw "Go files are not formatted:`n$($unformatted -join [Environment]::NewLine)"
    }
}
