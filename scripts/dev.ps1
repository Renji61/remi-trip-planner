# Start REMI with auto-restart on file changes.
# 1) Air (recommended): fast rebuild + restart
# 2) go run air@latest: no prior install
# 3) dev-watch.ps1: pure PowerShell FileSystemWatcher fallback
#
# Usage (from repo root):
#   .\scripts\dev.ps1
# Optional: override listen address (default :4122 via run-dev.cmd)

$ErrorActionPreference = "Stop"
$Root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $Root

$airCmd = Get-Command air -ErrorAction SilentlyContinue
if ($airCmd) {
    Write-Host "[dev] Using Air from PATH: $($airCmd.Source)" -ForegroundColor Cyan
    & $airCmd.Source
    exit $LASTEXITCODE
}

$goBinAir = Join-Path $env:USERPROFILE "go\bin\air.exe"
if (Test-Path $goBinAir) {
    Write-Host "[dev] Using Air: $goBinAir" -ForegroundColor Cyan
    & $goBinAir
    exit $LASTEXITCODE
}

Write-Host "[dev] Air not installed; trying go run github.com/air-verse/air@latest (first run may download)..." -ForegroundColor Yellow
go run github.com/air-verse/air@latest
if ($LASTEXITCODE -eq 0) { exit 0 }

Write-Host "[dev] go run air failed; falling back to scripts/dev-watch.ps1" -ForegroundColor Yellow
& (Join-Path $PSScriptRoot "dev-watch.ps1")
