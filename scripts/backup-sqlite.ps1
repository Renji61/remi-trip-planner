# Opt-in SQLite backup using .backup (safe while the app is running).
# Usage: .\scripts\backup-sqlite.ps1 [-SqlitePath .\data\trips.db] [-OutDir .\backup]
# Requires sqlite3 on PATH (e.g. SQLite tools for Windows). Schedule via Task Scheduler if desired.

param(
    [string]$SqlitePath = ".\data\trips.db",
    [string]$OutDir = ".\backup"
)

if (-not (Get-Command sqlite3 -ErrorAction SilentlyContinue)) {
    Write-Error "sqlite3 not found on PATH. Install SQLite command-line tools or run backup from WSL/Docker."
    exit 1
}

New-Item -ItemType Directory -Force -Path $OutDir | Out-Null
$ts = (Get-Date).ToUniversalTime().ToString("yyyyMMdd-HHmmss")
$dst = Join-Path $OutDir "remi-trips-$ts.db"
$dstFull = [System.IO.Path]::GetFullPath($dst)
$q = $dstFull.Replace("'", "''")
& sqlite3 $SqlitePath ".backup '$q'"
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
Write-Host "Wrote $dstFull"
