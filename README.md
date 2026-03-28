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
| Language & HTTP | Go 1.23, [chi](https://github.com/go-chi/chi) router |
| UI | HTML templates, [HTMX](https://htmx.org/) for partial updates and forms |
| Data | SQLite ([modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite)), WAL mode |
| Maps | [Leaflet](https://leafletjs.com/) + OpenStreetMap / Nominatim (optional geocoding) |
| Container | Multi-stage Dockerfile, Alpine runtime |
| Optional TLS | [Caddy](https://caddyserver.com/) example in `deploy/` |

---

## Features

### Trips & dashboard

- Create trips from the home page; view **active**, **draft**, and **archived** groups.
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

- App title, default currency, map defaults, theme (light / dark / system), location lookup, dashboard presentation options — via **Settings** and quick theme POST from the trip shell.

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
docs/                # Sync contract, CI workflow template (github-actions-ci.yml), etc.
```

---

## Requirements

- **Go 1.23+** (see `go.mod`) for local builds.
- **Docker** / **Docker Compose** optional, for containerized runs.

---

## Configuration

Environment variables (all optional except as noted):

| Variable | Default | Purpose |
|----------|---------|---------|
| `APP_ADDR` | `:8080` | HTTP listen address (inside container use `:8080`; map host port in Compose). |
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

Open [http://localhost:8080](http://localhost:8080).

Use another port:

```bash
# Unix / Git Bash
APP_ADDR=:8051 go run ./cmd/server
```

```powershell
# Windows PowerShell
$env:APP_ADDR=":8051"; go run ./cmd/server
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

### Quick start (build from clone)

```bash
cp .env.example .env   # optional: REMI_PORT=8051 maps host → container 8080
docker compose up -d --build
```

Open [http://localhost:8080](http://localhost:8080) (or your `REMI_PORT`).

- **Data:** named volume **`remi-data`** → `/app/data/trips.db` in the container.
- **Health:** image includes `wget` and a `HEALTHCHECK` on `GET /healthz` (also declared in Compose).
- **Manual update (git):** `git pull && docker compose up -d --build`
- **Install without git:** use `docker-compose.registry.yml` + `REMI_IMAGE=ghcr.io/<owner>/remi-trip-planner:latest` in `.env`.
- **Auto-updates (registry installs):** `docker-compose.registry.yml` includes an optional Watchtower profile — see [docs/self-hosting.md](docs/self-hosting.md). The default `docker-compose.yml` is build-from-git only (manual rebuild) so Watchtower does not pull a random `remi-trip-planner` image from Docker Hub.

Full instructions, GHCR publishing, backups, and Watchtower notes: **[docs/self-hosting.md](docs/self-hosting.md)**.

The workflow **[.github/workflows/docker-publish.yml](.github/workflows/docker-publish.yml)** pushes to `ghcr.io/<lowercase-owner>/remi-trip-planner` on pushes to `main` and on SemVer tags `v*.*.*`.

**Publishing for others:** only you (or your CI) can push to your registry — see **[docs/publish-image.md](docs/publish-image.md)** (GitHub Actions, manual `docker push`, and making GHCR packages **public** for homelab `docker pull` without login).

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

- This release has **no built-in user accounts**. Do not expose the app to the public internet without **network isolation**, **VPN**, or **proxy authentication**.
- Keep **secrets** out of git (`.env` is gitignored). Use strong host permissions on the SQLite file.

---

## License & changelog

- **License:** [MIT](LICENSE).
- **History:** [CHANGELOG.md](CHANGELOG.md).
