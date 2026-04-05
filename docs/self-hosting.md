# Self-hosting REMI Trip Planner (Docker)

This guide covers running REMI with **Docker Compose**, keeping **SQLite data** on a volume, and **updating manually** by rebuilding or pulling a new image.

## Requirements

- Docker Engine 24+ and Docker Compose V2
- For **registry installs**: a published image (see [Publish to GHCR](#publish-to-github-container-registry-ghcr)) or any OCI registry you control

## Option A — Build from this repository (recommended for developers)

```bash
git clone <your-fork-or-upstream-url>
cd remi-trip-planner
cp .env.example .env   # optional: set REMI_PORT if not using default 4122
docker compose up -d --build
```

Open `http://localhost:4122` (or the host port set in `REMI_PORT`). The app listens on **8080 inside the container**; Compose maps **host:4122 → 8080** by default.

Data lives in the **`remi-data`** Docker volume (`/app/data/trips.db` inside the container).

### Volume permissions (SQLite “readonly database”)

The app runs as a **non-root** user. New named volumes often mount as **`root`-owned** directories, so SQLite cannot create `trips.db` or WAL files until ownership is fixed.

**Current images** ship **`docker-entrypoint.sh`**: the container starts as **root**, runs **`chown -R remi:remi`** on **`/app/data`** and **`/app/web/static/uploads`**, then **`su-exec remi`** runs the server. That applies on **every** start, including **`docker compose`** and plain **`docker run`** with the same volume mounts.

If you use an **older** image **without** that entrypoint (or a **custom** image), run a **one-off** root helper container that mounts the same volumes and **`chown`s** those paths, or adjust host-side permissions on bind mounts.

### Manual update (git + rebuild)

```bash
git pull
docker compose up -d --build
```

## Option B — Install from the official registry image (no git clone, no `.env` required)

The **public** image is **`ghcr.io/renji61/remi-trip-planner:latest`** (SemVer tags like `:v1.45.0` are published from Git tags).

### B1 — One file, zero env (Dockhand / copy-paste)

Use **`docker-compose.install.yml`** from this repo (fixed image and **4122:8080** in the file):

```bash
curl -fsSL -o docker-compose.yml https://raw.githubusercontent.com/Renji61/remi-trip-planner/main/docker-compose.install.yml
docker compose -f docker-compose.yml up -d
```

Edit **`4122:8080`** in the YAML if you want another host port.

### B2 — Registry compose with optional overrides

Use **`docker-compose.registry.yml`**. It defaults to the same official image and host port **4122** — **no `.env` file is required**. Add `.env` only to override, for example:

```env
REMI_IMAGE=ghcr.io/your-github-username/remi-trip-planner:latest
REMI_PORT=4122
```

(Forks or private mirrors: set `REMI_IMAGE` to your image reference.)

Start:

```bash
docker compose -f docker-compose.registry.yml up -d
```

### Manual update (pull latest image)

```bash
docker compose -f docker-compose.registry.yml pull
docker compose -f docker-compose.registry.yml up -d
```

This fetches the newest image for your tag (e.g. `:latest`) and recreates the container. Your database remains in the **`remi-data`** volume.

**Pinning:** For production, consider pinning `REMI_IMAGE` to a version tag (e.g. `:v1.45.0`) instead of `:latest`, and bump when you choose.

### Optional auto-updates

This repository **does not** ship a Watchtower (or similar) service. If you want containers recreated when the registry image changes, add your own sidecar or host automation (e.g. Watchtower, systemd timer + `docker compose pull`).

## Health checks

The image includes **`wget`** and reports healthy when `GET /healthz` succeeds (`Dockerfile` + Compose `healthcheck`). Registry images must be built from the current `Dockerfile` for the same check to work.

## Publish to GitHub Container Registry (GHCR)

**Automated (CI):** `.github/workflows/docker-publish.yml` pushes on every push to **`main`** and on SemVer tags **`v*.*.*`**. After you push this repo to GitHub, run the workflow once (or push to `main`). See **[docs/publish-image.md](publish-image.md)** for permissions, **public package** visibility for anonymous `docker pull`, and troubleshooting.

**Manual (your machine):** from the repo root, after `docker login ghcr.io`, run:

- `.\scripts\publish-ghcr.ps1 -Owner your-github-username` (Windows), or
- `./scripts/publish-ghcr.sh your-github-username` (Linux/macOS).

Images use:

`ghcr.io/<lowercase-github-username-or-org>/remi-trip-planner`

Official upstream: **`ghcr.io/renji61/remi-trip-planner`**. The registry Compose file defaults to that; set **`REMI_IMAGE`** in `.env` only for a fork or private mirror.

## TLS and reverse proxy

For production, terminate HTTPS in front of the app (Caddy, Traefik, nginx). Proxy to the **container** port you mapped (the image listens on **8080** inside the container; the default host mapping is **4122**). See [deploy/remote_access.md](../deploy/remote_access.md) and [Deployment & HTTPS](../README.md#deployment--https) in the README.

## Backup

### Online-consistent SQLite (recommended)

While the app is running, use SQLite’s **`.backup`** command so readers get a consistent snapshot (safer than copying the raw `-wal` / `-shm` files while writes occur). From the host, if you have `sqlite3` and the DB file mounted or copied out:

```bash
mkdir -p ./backup
TS=$(date -u +%Y%m%d-%H%M%S)
sqlite3 ./path/to/trips.db ".backup './backup/remi-trips-'"$TS"'.db'"
```

Store the backup on a **different volume or machine** than the live database when you can.

**Docker:** run a one-off container with `sqlite3` installed, mount the **`remi-data`** volume read-only and an output directory, then `.backup` to `/out/…` (see repo **`docker-compose.backup.yml`** for an opt-in Compose profile, or adapt volume names from `docker volume ls`).

### Volume file copy (simple)

- Copy the SQLite file from the volume, e.g.:

  ```bash
  docker run --rm -v remi-trip-planner_remi-data:/data -v "$(pwd):/out" alpine \
    cp /data/trips.db /out/trips.db.backup
  ```

  (Volume name may be prefixed with your Compose project name.) Prefer stopping the app or using `.backup` above for production.

### Uploads

- Back up **`web/static/uploads/`** (or the **`remi-uploads`** volume / bind mount) **together with** the database — attachments are not stored inside SQLite.

Repo scripts (optional): **`scripts/backup-sqlite.sh`** and **`scripts/backup-sqlite.ps1`** for local/dev paths; schedule them with **cron** or **Task Scheduler** as you prefer.

## Security reminder

REMI is intended for **trusted networks**. Do not expose it publicly without **TLS**, **access control**, or **VPN**. See [Security notes](../README.md#security-notes) in the README.
