# Build image and run REMI Trip Planner (default host port 4122 → container :8080 via docker-compose.yml).
# Run from repo root:  .\scripts\deploy-docker.ps1

$ErrorActionPreference = "Stop"
$Root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $Root

Write-Host "[deploy-docker] Building and starting (http://localhost:4122 by default)..." -ForegroundColor Cyan
docker compose up -d --build
if ($LASTEXITCODE -ne 0) {
    Write-Host "[deploy-docker] docker compose failed." -ForegroundColor Red
    exit $LASTEXITCODE
}
Write-Host "[deploy-docker] Done. Open http://localhost:4122/ (or the host port in your .env REMI_PORT)" -ForegroundColor Green
