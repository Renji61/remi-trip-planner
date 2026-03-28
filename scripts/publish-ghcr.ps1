#Requires -Version 5.1
<#
.SYNOPSIS
  Build and push remi-trip-planner to GitHub Container Registry.

.DESCRIPTION
  Run from anywhere; the script resolves the repo root (parent of /scripts).
  Log in first:  $env:GITHUB_TOKEN | docker login ghcr.io -u YOUR_USER --password-stdin
  Or pass -Token for a one-shot login (avoid shell history with real tokens).

.PARAMETER Owner
  GitHub username or organization (will be lowercased). Default: $env:GHCR_OWNER

.PARAMETER Tag
  Image tag (default: latest)

.PARAMETER Token
  Optional GitHub PAT with write:packages; used only to run docker login before push.
#>
param(
  [Parameter(Mandatory = $false)]
  [string] $Owner,
  [string] $Tag = "latest",
  [string] $Token
)

$ErrorActionPreference = "Stop"

if (-not $Owner) { $Owner = $env:GHCR_OWNER }
if (-not $Owner) {
  Write-Error "Set -Owner or `$env:GHCR_OWNER to your GitHub user or org (lowercase recommended)."
}

$Owner = $Owner.ToLowerInvariant()
$image = "ghcr.io/${Owner}/remi-trip-planner:${Tag}"

$repoRoot = Split-Path -Parent $PSScriptRoot
Set-Location $repoRoot

if ($Token) {
  $Token | docker login ghcr.io -u $Owner --password-stdin
  if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }
}

Write-Host "Building $image ..."
docker build -t $image $repoRoot
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host "Pushing $image ..."
docker push $image
if ($LASTEXITCODE -ne 0) { exit $LASTEXITCODE }

Write-Host ""
Write-Host "Done. Share this with homelab installers (docker-compose.registry.yml + .env):"
Write-Host "  REMI_IMAGE=$image"
