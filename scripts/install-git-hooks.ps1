# 安装 Git Hooks 脚本
Write-Host "========================================" -ForegroundColor Cyan
Write-Host "  安装 Git Hooks" -ForegroundColor Cyan
Write-Host "========================================" -ForegroundColor Cyan
Write-Host ""

$hooksDir = ".git\hooks"
$sourceHooksDir = "scripts\git-hooks"

Write-Host "📦 安装 hooks..." -ForegroundColor Yellow

if (Test-Path $hooksDir) {
    Copy-Item -Path "$sourceHooksDir\*" -Destination $hooksDir -Force
    Write-Host "✅ Hooks 已安装到: $hooksDir" -ForegroundColor Green
} else {
    Write-Host "⚠️  找不到 .git\hooks 目录" -ForegroundColor Yellow
}

Write-Host ""
Write-Host "🎉 Git Hooks 安装完成!" -ForegroundColor Green
Write-Host "   - post-merge: 合并后自动提交" -ForegroundColor Gray
Write-Host "   - pre-commit: 提交前检查" -ForegroundColor Gray
