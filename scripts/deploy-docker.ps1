# Build image and run REMI Trip Planner on host port 8051 (maps to container :8080).
# Run from repo root:  .\scripts\deploy-docker.ps1

$ErrorActionPreference = "Stop"
$Root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $Root

Write-Host "[deploy-docker] Building and starting (http://localhost:8051)..." -ForegroundColor Cyan
docker compose up -d --build
if ($LASTEXITCODE -ne 0) {
    Write-Host "[deploy-docker] docker compose failed." -ForegroundColor Red
    exit $LASTEXITCODE
}
Write-Host "[deploy-docker] Done. Open http://localhost:8051/" -ForegroundColor Green
