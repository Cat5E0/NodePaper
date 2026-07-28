param(
    [Parameter(Mandatory = $true)]
    [string]$ProjectDir,
    [int]$Seconds = 60
)

$nodepaperDir = Join-Path $ProjectDir ".nodepaper"
New-Item -ItemType Directory -Force -Path $nodepaperDir | Out-Null
$lockPath = Join-Path $nodepaperDir "build.lock"

$lock = @{
    buildId = "manual-active-lock"
    pid = $PID
    startedAt = (Get-Date).ToString("o")
    projectRoot = (Resolve-Path $ProjectDir).Path
} | ConvertTo-Json

Set-Content -Path $lockPath -Value $lock -Encoding UTF8
Write-Host "Active lock written to $lockPath with PID $PID"

try {
    Start-Sleep -Seconds $Seconds
}
finally {
    Remove-Item -Force -ErrorAction SilentlyContinue $lockPath
}
