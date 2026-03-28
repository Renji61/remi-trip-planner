# REMI Trip Planner

A **self-hosted** trip planner: one binary (or container), **SQLite** storage, and a **server-rendered** web UI with light **HTMX** enhancements. It targets **low-end hardware** and simple deployment (Docker optional).

---

## Table of contents

- [Why REMI](#why-remi)
- [Stack](#stack)
- [Features](#features)
- [Repository layout](#repository-layout)
- [Requirements](#requirements)
- [Configuration](#configuration)
- [Run locally](#run-locally)
- [Docker & self-hosting](#docker--self-hosting)
- [Development](#development)
- [Deployment & HTTPS](#deployment--https)
- [PWA & offline](#pwa--offline)
- [Sync API](#sync-api)
- [Backup & data](#backup--data)
- [Security notes](#security-notes)
- [License & changelog](#license--changelog)

---

## Why REMI

- **Own your data** — SQLite file on disk, no vendor lock-in.
- **Fast to run** — Go + Chi, minimal JavaScript; suitable for a small VPS or homelab.
- **Practical trip workflow** — itinerary on a map, spends vs budget, stays, rentals, flights, and packing-style checklists in one place.

---

## Stack

| Layer | Technology |
|--------|------------|
| Language & HTTP | Go 1.25, [chi](https://github.com/go-chi/chi) router |
| UI | HTML templates, [HTMX](https://htmx.org/) for partial updates and forms |
| Data | SQLite ([modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)), WAL mode |
| Maps | [Leaflet](https://leafletjs.com/) + OpenStreetMap / Nominatim (optional geocoding) |
| Container | Multi-stage Dockerfile, Alpine runtime |
| Optional TLS | [Caddy](https://caddyserver.com/) example in `deploy/` |

---

## Features

### Trips & dashboard

- Create trips from the home page (optional **place lookup** for trip map center; falls back to site default location); view **active**, **draft**, and **archived** groups.
- **Dashboard customization:** grid vs list cards, sort order, hero background style, heading text (app settings).
- Per-trip: name, description, dates, **cover image URL**, **currency**, archive/delete.

### Itinerary & map

- **Day-grouped** stops with titles, locations, notes, optional cost and times.
- **Per-day descriptions** (labels) editable inline on the trip page.
- **Interactive map** with markers and a route polyline when coordinates exist.
- **Search** across itinerary text from the trip header.
- **Geocoding** can be disabled globally (app settings) for privacy or rate-limit reasons.

### Spends & budget

- Manual expenses (category, amount, date, payment method, notes).
- **Budget summary** on the trip page (budgeted vs spent); dedicated **budget** subpage with transactions and export.
- **Quick spend** entry from the trip sidebar (when Spends is enabled).
- Some spends are **linked** to stay, vehicle, or flight bookings and edited from those flows.

### Stay, vehicle, flights

- Full **accommodation**, **vehicle rental**, and **flights** sections with forms, attachments/images/documents, and links to **itinerary stops** and **expenses** where designed.

### Checklist

- **Categorized** reminder items; mark done/undo; add from the trip page (including multi-item draft list).

### Trip page layout & personalization

- Toggle visibility of **Stay**, **Vehicle**, **Flights**, and **Spends** (and related nav/widgets).
- Rename section labels for nav and headings.
- Control **default expanded** state for itinerary and spend day groups.
- **12h / 24h** clock display per trip.
- **Reorder** main column sections: Trip Map, Itinerary, Spends, Reminder Checklist, Stay, Vehicle, Flights (hero and trip edit panel stay at the top).
- **Reorder** right-sidebar widgets: Add New Stop, Total Budgeted Cost, Quick Spends, Add to Checklist (wide layouts; budget/quick respect Spends toggle).

### App-wide settings

- App title, default currency, **default map location** (place search with short name stored; Tokyo fallback), map zoom, theme (light / dark / system), location lookup, dashboard presentation options — via **Settings** and quick theme POST from the trip shell.

### About & updates

- **About** page with installed version, release notes, and **check for updates** (compares to GitHub Releases for self-hosted instances).

### PWA

- **Manifest** and **service worker** for add-to-home-screen style use and basic static caching (see [PWA & offline](#pwa--offline)).

### Sync (for future / native clients)

- **Read** server change history: `GET /api/v1/trips/{tripID}/changes`.
- **Sync** placeholder: `POST /api/v1/trips/{tripID}/sync` — see [docs/sync_contract.md](docs/sync_contract.md).

---

## Repository layout

```text
cmd/server/          # HTTP entrypoint
cmd/dbpeek/          # Optional SQLite debug CLI
internal/httpapp/    # Routes, handlers, templates wiring
internal/trips/      # Domain types and service logic
internal/storage/sqlite/
migrations/          # Base schema; extra columns migrated in code
web/templates/       # HTML templates
web/static/          # CSS, JS, manifest, service worker (uploads at runtime)
deploy/              # Caddyfile and remote access notes
docs/                # Self-hosting, publish-image, sync contract, CI template, etc.
docker-compose.yml   # Build from clone
docker-compose.registry.yml   # Pull image (defaults to official GHCR; optional .env)
docker-compose.install.yml    # Homelab one-file install (no .env)
```

---

## Requirements

- **Go 1.25+** (see `go.mod`) for local builds.
- **Docker** / **Docker Compose** optional, for containerized runs.

---

## Configuration

Environment variables (all optional except as noted):

| Variable | Default | Purpose |
|----------|---------|---------|
| `APP_ADDR` | `:4122` | HTTP listen address for **native** runs. Inside Docker the image uses `:8080`; map host port in Compose (default **4122**). |
| `SQLITE_PATH` | `./data/trips.db` | SQLite database file path. |
| `REMI_ROOT` | _(unset)_ | Absolute path to repo root if the process cwd is not the module directory. |

Inside Docker, the image sets `APP_ADDR=:8080` and `SQLITE_PATH=/app/data/trips.db`.

---

## Run locally

### With Go (no Docker)

From the directory that contains `go.mod` (the module root):

```bash
mkdir -p data
go run ./cmd/server
```

Open [http://localhost:4122](http://localhost:4122).

Use another port:

```bash
# Unix / Git Bash
APP_ADDR=:9090 go run ./cmd/server
```

```powershell
# Windows PowerShell
$env:APP_ADDR=":9090"; go run ./cmd/server
```

If you open the IDE at a **parent** folder (e.g. a folder containing several projects), the server tries to `chdir` into the checkout that contains `module remi-trip-planner` in `go.mod`, or set `REMI_ROOT` explicitly.

### Quality checks

```bash
gofmt -l .          # should print nothing
go vet ./...
go test ./...
```

---

## Docker & self-hosting

**Official image (public):** `ghcr.io/renji61/remi-trip-planner:latest`  
Version pins: `ghcr.io/renji61/remi-trip-planner:v1.40.0` (and other SemVer tags published by CI).

### Quick start — homelab (no `.env`, no git)

```bash
curl -fsSL -o docker-compose.yml https://raw.githubusercontent.com/Renji61/remi-trip-planner/main/docker-compose.install.yml
docker compose -f docker-compose.yml up -d
```

Or copy [`docker-compose.install.yml`](docker-compose.install.yml) from this repo and run:

```bash
docker compose -f docker-compose.install.yml up -d
```

Open [http://localhost:4122](http://localhost:4122). Edit **`4122:8080`** in the file to change the host port (left side = host, right = container).

### Build from clone (developers)

```bash
cp .env.example .env   # optional: REMI_PORT maps host → container 8080 (default 4122)
docker compose up -d --build
```

- **Data:** named volume **`remi-data`** → `/app/data/trips.db` in the container.
- **Health:** image includes `wget` and a `HEALTHCHECK` on `GET /healthz` (also declared in Compose).
- **Manual update (git):** `git pull && docker compose up -d --build`

### Registry compose (optional `.env` overrides)

[`docker-compose.registry.yml`](docker-compose.registry.yml) defaults to the same official image and host port **4122** (→ container **8080**) — **no `.env` required**. Set `REMI_IMAGE` or `REMI_PORT` in `.env` only if you fork the image or need another port.

Compose files **do not** include Watchtower or other auto-update sidecars; pull/rebuild when you want a new version, or add your own tooling.

Full instructions: **[docs/self-hosting.md](docs/self-hosting.md)**.

CI: **[.github/workflows/docker-publish.yml](.github/workflows/docker-publish.yml)** pushes `ghcr.io/renji61/remi-trip-planner` on `main` and on SemVer tags `v*.*.*`.

**Publishing / forks:** **[docs/publish-image.md](docs/publish-image.md)** (Actions permissions, PAT `workflow` scope, manual `docker push`).

---

## Development

- **Templates:** `web/templates/*.html` — parsed together; shared fragments live in `partials.html` where used.
- **Front-end:** `web/static/app.css`, `app.js`; bump `?v=` on `app.css` in templates when you need cache busts.
- **Migrations:** new installs run `migrations/001_init.sql`; existing DBs get additive `ALTER TABLE` statements in `internal/storage/sqlite/db.go`.
- **CI:** copy [docs/github-actions-ci.yml](docs/github-actions-ci.yml) to `.github/workflows/ci.yml` and push (use a Git credential with the **`workflow`** scope if GitHub rejects OAuth pushes to workflow files).

---

## Deployment & HTTPS

1. Point DNS at your server.
2. Adjust `deploy/Caddyfile` for your domain.
3. Run behind Caddy or another reverse proxy with TLS. See [deploy/remote_access.md](deploy/remote_access.md) for notes.

---

## PWA & offline

- The service worker caches core assets for faster repeat visits; it does **not** replace server data offline.
- For true offline editing, clients should use the **sync** direction described in [docs/sync_contract.md](docs/sync_contract.md) (work in progress).

---

## Sync API

| Method | Path | Role |
|--------|------|------|
| `GET` | `/api/v1/trips/{tripID}/changes` | Paginated-style change log for replication (`since` query supported as implemented). |
| `POST` | `/api/v1/trips/{tripID}/sync` | Prototype endpoint; request/response evolution documented in `docs/sync_contract.md`. |

---

## Backup & data

- **File:** copy `SQLITE_PATH` (e.g. `./data/trips.db` locally, or the Docker volume contents).
- **Uploads:** if you use attachments/images, back up `web/static/uploads/` (ignored by git).

---

## Security notes

- Use **strong passwords**, **HTTPS** in production, and **secure cookies** when exposing the app beyond localhost. Restrict who can reach the server (VPN, firewall, or authenticated reverse proxy).
- Keep **secrets** out of git (`.env` is gitignored). Use strong host permissions on the SQLite file.

---

## License & changelog

- **License:** [MIT](LICENSE).
- **History:** [CHANGELOG.md](CHANGELOG.md).
