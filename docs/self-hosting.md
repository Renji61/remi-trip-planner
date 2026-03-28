# Self-hosting REMI Trip Planner (Docker)

This guide covers running REMI with **Docker Compose**, keeping **SQLite data** on a volume, **updating manually**, and optional **Watchtower** auto-updates from a registry.

## Requirements

- Docker Engine 24+ and Docker Compose V2
- For **registry installs**: a published image (see [Publish to GHCR](#publish-to-github-container-registry-ghcr)) or any OCI registry you control

## Option A — Build from this repository (recommended for developers)

```bash
git clone <your-fork-or-upstream-url>
cd remi-trip-planner
cp .env.example .env   # optional: set REMI_PORT=8051 etc.
docker compose up -d --build
```

Open `http://localhost:8080` (or the host port set in `REMI_PORT`).

Data lives in the **`remi-data`** Docker volume (`/app/data/trips.db` inside the container).

### Manual update (git + rebuild)

```bash
git pull
docker compose up -d --build
```

Watchtower is **not** required for this path; you pick up changes when you pull and rebuild.

The default **`docker-compose.yml` does not run Watchtower**, so a local tag like `remi-trip-planner:latest` is never replaced by an image pulled from Docker Hub by mistake.

## Option B — Install from a registry image (no git clone)

1. Publish an image (e.g. `ghcr.io/your-org/remi-trip-planner:latest`) — see below.
2. On the server, create a directory with:
   - `docker-compose.registry.yml` (from this repo)
   - `.env` containing at least:

     ```env
     REMI_IMAGE=ghcr.io/your-org/remi-trip-planner:latest
     REMI_PORT=8080
     ```

3. Start:

   ```bash
   docker compose -f docker-compose.registry.yml up -d
   ```

### Manual update (pull latest image)

```bash
docker compose -f docker-compose.registry.yml pull
docker compose -f docker-compose.registry.yml up -d
```

This fetches the newest image for your tag (e.g. `:latest`) and recreates the container. Your database remains in the **`remi-data`** volume.

### Auto-updates with Watchtower

Watchtower recreates containers when the **registry digest** for the image changes.

```bash
docker compose -f docker-compose.registry.yml --profile watchtower up -d
```

- Only containers with the label `com.centurylinklabs.watchtower.enable=true` are managed (the `remi` service has this label).
- Default poll interval is **3600** seconds; override with `WATCHTOWER_POLL_INTERVAL` in `.env`.
- **Private GHCR**: run `docker login ghcr.io` on the host (or configure a credential helper) before Watchtower can pull.

**Caution:** `:latest` will auto-deploy whatever you last pushed to that tag. For stricter control, pin `REMI_IMAGE` to a version tag (e.g. `:v1.2.3`) and bump it when you choose.

## Health checks

The image includes **`wget`** and reports healthy when `GET /healthz` succeeds (`Dockerfile` + Compose `healthcheck`). Registry images must be built from the current `Dockerfile` for the same check to work.

## Publish to GitHub Container Registry (GHCR)

**Automated (CI):** `.github/workflows/docker-publish.yml` pushes on every push to **`main`** and on SemVer tags **`v*.*.*`**. After you push this repo to GitHub, run the workflow once (or push to `main`). See **[docs/publish-image.md](publish-image.md)** for permissions, **public package** visibility for anonymous `docker pull`, and troubleshooting.

**Manual (your machine):** from the repo root, after `docker login ghcr.io`, run:

- `.\scripts\publish-ghcr.ps1 -Owner your-github-username` (Windows), or
- `./scripts/publish-ghcr.sh your-github-username` (Linux/macOS).

Images use:

`ghcr.io/<lowercase-github-username-or-org>/remi-trip-planner`

Set `REMI_IMAGE` in `.env` to that path plus `:latest` or a version tag.

## TLS and reverse proxy

For production, terminate HTTPS in front of the app (Caddy, Traefik, nginx). Proxy to the container port you mapped (default **8080** on the container). See [deploy/remote_access.md](../deploy/remote_access.md) and [Deployment & HTTPS](../README.md#deployment--https) in the README.

## Backup

- Copy the SQLite file from the volume, e.g.:

  ```bash
  docker run --rm -v remi-trip-planner_remi-data:/data -v "$(pwd):/out" alpine \
    cp /data/trips.db /out/trips.db.backup
  ```

  (Volume name may be prefixed with your Compose project name.)

- Back up `web/static/uploads/` if you use attachments (host path or a bind mount you configure).

## Security reminder

REMI is intended for **trusted networks**. Do not expose it publicly without **TLS**, **access control**, or **VPN**. See [Security notes](../README.md#security-notes) in the README.
