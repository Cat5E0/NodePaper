# 从 docs/assets/logo/logo-transparent.png 重新生成桌面端全套应用图标。
#
# tauri.conf.json 的 bundle.icon 引用 src-tauri/icons/ 下的成套规格：
# icon.icns（mac）、icon.ico（Windows）、多尺寸 PNG（安装器/商店）。
# 这些都是生成物，源图变更后运行本脚本刷新，不要手改 icons/ 内的文件。
#
# 用法（在 nodepaper/desktop 下）：
#   pnpm icons            # 默认源图
#   pnpm icons -- path    # 指定其他源图（需 >= 512x512 方形 RGBA）
param(
    [string]$Source = (Join-Path $PSScriptRoot "..\..\..\docs\assets\logo\logo-transparent.png")
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path -LiteralPath $Source)) {
    throw "源图不存在: $Source"
}

Push-Location (Join-Path $PSScriptRoot "..")
try {
    # tauri icon 会顺带生成 android/ ios/ 移动端目录；桌面端用不到，
    # 生成后立即清理，保持 icons/ 只含 tauri.conf.json 引用的桌面规格。
    pnpm tauri icon $Source
    if ($LASTEXITCODE -ne 0) { throw "tauri icon 失败（exit $LASTEXITCODE）" }

    foreach ($dir in @("src-tauri\icons\android", "src-tauri\icons\ios")) {
        if (Test-Path $dir) { Remove-Item -Recurse -Force $dir }
    }
    Write-Host "[成功] 图标已生成: src-tauri/icons/（icns / ico / 多尺寸 PNG）"
}
finally {
    Pop-Location
}
