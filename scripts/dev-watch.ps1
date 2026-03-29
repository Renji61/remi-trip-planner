# Watch REMI Trip Planner sources and rebuild + restart the server on 127.0.0.1:4122.
# Run from repo root:  .\scripts\dev-watch.ps1
# Ctrl+C stops the watcher and the server on this port.

$ErrorActionPreference = "Stop"
$Root = (Resolve-Path (Join-Path $PSScriptRoot "..")).Path
Set-Location $Root

$Port = 4122
$DebounceMs = 1200

function Stop-ServerOnPort {
    $conn = Get-NetTCPConnection -LocalPort $Port -ErrorAction SilentlyContinue | Select-Object -First 1
    if ($conn) {
        Stop-Process -Id $conn.OwningProcess -Force -ErrorAction SilentlyContinue
        Start-Sleep -Milliseconds 400
    }
}

function Test-WatchedPath {
    param([string]$FullPath)
    if ($FullPath -match '[\\/]\.git[\\/]') { return $false }
    if ($FullPath -match '[\\/]tmp[\\/]') { return $false }
    if ($FullPath -match '[\\/]bin[\\/]') { return $false }
    if ($FullPath -match '[\\/]web[\\/]static[\\/]uploads[\\/]') { return $false }
    $ext = [System.IO.Path]::GetExtension($FullPath).ToLowerInvariant()
    return @(".go", ".html", ".css", ".js", ".sql") -contains $ext
}

function Build-And-Start {
    Write-Host "[dev-watch] Building..."
    & go build -o .\bin\remi-server.exe ./cmd/server
    if ($LASTEXITCODE -ne 0) {
        Write-Host "[dev-watch] Build failed." -ForegroundColor Red
        return
    }
    Stop-ServerOnPort
    $env:APP_ADDR = "127.0.0.1:$Port"
    Start-Process -FilePath ".\bin\remi-server.exe" -WorkingDirectory $Root -WindowStyle Hidden
    Write-Host "[dev-watch] Server started on http://localhost:$Port/" -ForegroundColor Green
}

$watcher = New-Object System.IO.FileSystemWatcher
$watcher.Path = $Root
$watcher.Filter = "*.*"
$watcher.IncludeSubdirectories = $true
$watcher.NotifyFilter = [System.IO.NotifyFilters]::LastWrite -bor [System.IO.NotifyFilters]::FileName
$watcher.EnableRaisingEvents = $true

Write-Host "Watching $Root -- .go, .html, .css, .js, .sql changes trigger rebuild. Ctrl+C to stop."
Build-And-Start

try {
    while ($true) {
        $ev = $watcher.WaitForChanged([System.IO.WatcherChangeTypes]::All, 5000)
        if ($ev.TimedOut) { continue }
        if (-not (Test-WatchedPath $ev.FullPath)) { continue }
        Start-Sleep -Milliseconds $DebounceMs
        while ($true) {
            $extra = $watcher.WaitForChanged([System.IO.WatcherChangeTypes]::All, 300)
            if ($extra.TimedOut) { break }
            if (Test-WatchedPath $extra.FullPath) { Start-Sleep -Milliseconds 400 }
        }
        Build-And-Start
    }
} finally {
    $watcher.EnableRaisingEvents = $false
    $watcher.Dispose()
    Stop-ServerOnPort
}
