#!/usr/bin/env bash
# Build and push remi-trip-planner to GHCR.
# Usage: ./scripts/publish-ghcr.sh [owner] [tag]
#   owner defaults to $GHCR_OWNER; tag defaults to latest
# Login first: echo "$GITHUB_TOKEN" | docker login ghcr.io -u OWNER --password-stdin
set -euo pipefail

OWNER="${1:-${GHCR_OWNER:-}}"
TAG="${2:-latest}"

if [[ -z "$OWNER" ]]; then
  echo "Usage: GHCR_OWNER=myuser $0   OR   $0 myuser [tag]" >&2
  exit 1
fi

OWNER=$(echo "$OWNER" | tr '[:upper:]' '[:lower:]')
IMAGE="ghcr.io/${OWNER}/remi-trip-planner:${TAG}"
ROOT="$(cd "$(dirname "$0")/.." && pwd)"
cd "$ROOT"

echo "Building $IMAGE ..."
docker build -t "$IMAGE" "$ROOT"
echo "Pushing $IMAGE ..."
docker push "$IMAGE"
echo ""
echo "Done. Set REMI_IMAGE=$IMAGE in .env for docker-compose.registry.yml"
