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
- **Practical trip workflow** — itinerary on a map, expenses vs budget, stays, rentals, flights, and packing-style checklists in one place.

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
- **Dashboard sidebar & mobile bottom bar:** up to **two** shortcuts to trips that are **in progress** or **upcoming** (in that priority order), in addition to **My Trip**, **Profile**, and **Settings**.
- **Dashboard customization:** grid vs list cards, sort order, hero background style, heading text (app settings).
- Per-trip: name, description, dates, **cover image URL**, **currency**, archive/delete.

### Itinerary & map

- **Day-grouped** stops with titles, locations, notes, optional cost and times.
- **Per-day descriptions** (labels) editable inline on the trip page.
- **Interactive map** (Leaflet + OpenStreetMap by default, or **Google Maps** when an API key is set) with markers, optional **day filters**, and travel hints between stops; editing a stop **updates stored coordinates** and map pins after save.
- **Search** across itinerary text from the trip header.
- **Geocoding** can be disabled globally (app settings) for privacy or rate-limit reasons.
- **Caching:** server and browser caches reduce repeat geocoding and place-suggestion traffic for the same queries.

### Expenses & group expenses

- Manual **expenses** (category, amount, date, payment method, notes) at `/trips/{id}/expenses`; **group expenses** at `/trips/{id}/group-expenses` with equal, exact, percent, and share-based splits; **301** redirects from legacy `/budget` and `/tab` URLs.
- Amounts are persisted as **integer minor units** (e.g. cents) in SQLite for accurate split math; the UI still uses familiar decimal entry.
- **Optimistic concurrency** on many trip-scoped edits: if someone else saved first, the UI can receive **409 Conflict** with a structured error instead of silently overwriting.
- **Budget summary** on the trip page (budgeted vs spent), including **Total group expense** on mobile and in the desktop sidebar when group expenses are enabled; dedicated expenses subpage with transactions and export.
- **Quick expense** entry from the trip sidebar (when the expenses section is enabled).
- Some expenses are **linked** to stay, vehicle, or flight bookings and edited from those flows.
- **Departed participants** on group expenses keep historical splits and settlements consistent when collaborators leave; labels show **Left trip** where applicable.

### Stay, vehicle, flights

- Full **accommodation**, **vehicle rental**, and **flights** sections with forms, attachments/images/documents, and links to **itinerary stops** and **expenses** where designed.
- **Add/update/delete** flows for those bookings run in **transactions** with linked itinerary lines (and related cleanup) so you do not get half-applied saves if something fails mid-way.

### Trip documents & uploads

- **Trip Documents** at `/trips/{id}/documents`: upload general files, search and filter the full attachment list (including files from stays, rentals, flights, and group expenses), rename or delete general uploads, and jump to the source booking or expense where applicable.
- **App settings:** **Max upload size per file (MB)** (default 5) applies to Trip Documents and attachment fields across trip forms.
- **Docker:** Compose files mount a **`remi-uploads`** volume at `/app/web/static/uploads` so uploads persist across container rebuilds.

### Checklist

- **Categorized** reminder items; mark done/undo; add from the trip page (including multi-item draft list).
- **Mobile:** **Add to Checklist** appears in the trip **FAB** menu when checklist + sidebar widget visibility allow it (same as trip settings); opens the checklist sheet on the main trip page or via `?open=checklist` from subpages.

### Trip page layout & personalization

- Toggle visibility of **Stay**, **Vehicle**, **Flights**, and **Expenses** (and related nav/widgets).
- Rename section labels for nav and headings.
- Control **default expanded** state for itinerary and expense day groups.
- **12h / 24h** clock and **DD/MM/YYYY** vs **MM/DD/YYYY** calendar date format per trip.
- **Reorder** main column sections: Trip Map, Itinerary, Expenses, Reminder Checklist, Stay, Vehicle, Flights (hero and trip edit panel stay at the top).
- **Reorder** right-sidebar widgets: Add New Stop, Total Budgeted Cost, Quick expenses, Add to Checklist (wide layouts; budget/quick respect the expenses section toggle).
- **Mobile:** the bottom **Trip sections** navigation **scrolls horizontally** when many sections are on, so every tab stays reachable.

### App-wide settings

- App title, default currency, **default map location** (place search with short name stored; Tokyo fallback), map zoom, theme (light / dark / system), location lookup, dashboard presentation options — via **Settings** and quick theme POST from the trip shell.
- **Desktop:** shared **account** dropdown (profile initial, **Profile**, **App settings**, **Log out**) on trip topbars and other app-shell pages for consistent navigation.

### Account export & privacy

- **Profile → Your data:** `GET /profile/export` (session cookie, same-site browser navigation) downloads **`remi-export-YYYYMMDD.json`**: safe profile fields (no password hash), your user settings, app settings with **secrets redacted** (e.g. Google Maps API key), and every trip you can see (itinerary, expenses, checklist, stays, vehicles, flights, group/tab settlements, guests, departed participants, trip document metadata including stored paths — not file bytes). **CSRF:** export uses **GET** so it is not tied to form CSRF tokens; it only works for an authenticated session from the same site (mitigate cross-site download by keeping cookies `SameSite`/`Secure` in production as configured).

### About & updates

- **About** page with installed version, bundled changelog excerpt for that version, and **check for updates** for self-hosted installs: the server compares your build to the **newest stable SemVer** found from **GitHub Releases and git tags** (so tag-only publishes still count). Publish **`v*.*.*`** on GitHub for update prompts; **`ahead_of_published`** in the JSON covers preview builds newer than the latest tag.

### PWA

- **Manifest** and **service worker** for add-to-home-screen style use and basic static caching (see [PWA & offline](#pwa--offline)).

### Sync (local-first / native clients)

- **Read** server change history: `GET /api/v1/trips/{tripID}/changes?since=…` (session cookie, trip access).
- **Live stream:** `GET /api/v1/trips/{tripID}/events` streams **`text/event-stream`** (change batches) for push-style refresh while a trip is open in the browser.
- **Write batch:** `POST /api/v1/trips/{tripID}/sync` with JSON body (`client_id`, optional numeric `base_cursor`, `ops[]`) — applies create/update/delete (and trip `archive`) for `trip`, `itinerary_item`, `expense`, `checklist_item`; returns per-op results and new `change_log` rows; conflicts surface as **`conflict`** in per-op results when optimistic locking applies. See [docs/sync_contract.md](docs/sync_contract.md).

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
| `APP_ADDR` | `127.0.0.1:4122` | HTTP listen address for **native** runs (loopback only; not reachable from other machines). Set `:4122` to listen on all interfaces. Inside Docker the image uses `:8080`; map host port in Compose (default **4122**). |
| `SQLITE_PATH` | `./data/trips.db` | SQLite database file path. |
| `REMI_ROOT` | _(unset)_ | Absolute path to repo root if the process cwd is not the module directory. |
| `REMI_ENV` | _(unset)_ | Set to **`production`** behind HTTPS: **JSON logs** to stdout, **`Secure` session cookies**, **HSTS** and **CSP-Report-Only** headers, stricter browser defaults. Omit or use any other value for local dev (text logs to stderr). |
| `REMI_TRUSTED_PROXIES` | _(unset)_ | Comma-separated **trusted proxy IPs or CIDRs** (e.g. `127.0.0.1,10.0.0.0/8`). When the **direct** client IP matches, `X-Forwarded-For` is used for **client IP** (rate limits and access logs). Leave unset if the app is not behind a reverse proxy (avoids spoofing). |
| `REMI_RATE_LIMIT_AUTH_RPM` | `40` | Per-IP requests per minute for sensitive routes: `POST /login`, `/register`, `/setup`; `GET /verify-email` (with token); `POST /invites/accept`; `POST /profile/resend-verify`. |
| `REMI_RATE_LIMIT_AUTH_BURST` | `12` | Burst allowance for the same limiter. |
| `REMI_HSTS_MAX_AGE` | `31536000` | `Strict-Transport-Security` **max-age** (seconds) when `REMI_ENV=production`. Set to `0` to disable HSTS. |
| `REMI_HEALTHZ_DB` | _(unset)_ | If `1` / `true`, `GET /healthz` also runs **`SELECT 1`** against SQLite (returns **503** if the DB is unreachable). Default health check stays cheap (no DB). |

Inside Docker, the image sets `APP_ADDR=:8080` and `SQLITE_PATH=/app/data/trips.db`.

**Production example (Compose / systemd):** set `REMI_ENV=production`, terminate **TLS** at Caddy/nginx/Cloudflare, and set `REMI_TRUSTED_PROXIES` to your proxy’s **egress** IP(s) toward the app so `X-Forwarded-For` is trusted.

### SQLite concurrency

- SQLite uses **WAL** mode: many **readers** and **one writer** at a time per database file. For internet-facing multi-user use, run **one app instance** (one process/container) writing to a given `SQLITE_PATH` unless you know what you’re doing; **do not** point multiple replicas at the same file on network storage without understanding locking.
- **Backup:** copy `SQLITE_PATH` and `web/static/uploads/` (or your Docker volumes) on a schedule; test restores.

---

## Run locally

### With Go (no Docker)

From the directory that contains `go.mod` (the module root):

```bash
mkdir -p data
go run ./cmd/server
```

Open [http://127.0.0.1:4122](http://127.0.0.1:4122) (or [http://localhost:4122](http://localhost:4122) if your system resolves `localhost` to IPv4).

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
Version pins: `ghcr.io/renji61/remi-trip-planner:v1.49.4` (and other SemVer tags published by CI).

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
- **Uploads:** named volume **`remi-uploads`** → `/app/web/static/uploads` (same as registry/install compose variants).
- **Health:** image includes `wget` and a `HEALTHCHECK` on `GET /healthz` (also declared in Compose).
- **Hardening (default compose):** app user is non-root; service may use **read-only** root, **`tmpfs`** on `/tmp`, and dropped capabilities — override only if you need extra privileges.
- **Volume permissions:** the image **`docker-entrypoint.sh`** runs as root on each start, **`chown`s** **`remi-data`** / **`remi-uploads`** mount points for user **`remi`**, then starts the app with **`su-exec`** (so Compose shows one running service). Images **before** this entrypoint may need a one-off manual **`chown`** on those paths.
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

- **SQLite (recommended online backup):** use the SQLite shell so the app can keep running safely (writes a consistent snapshot). Example (adjust paths):

  ```bash
  mkdir -p ./backup
  TS=$(date -u +%Y%m%d-%H%M%S)
  sqlite3 ./data/trips.db ".backup ./backup/remi-trips-${TS}.db"
  ```

  On Windows (PowerShell), if `sqlite3` is on your `PATH`:

  ```powershell
  New-Item -ItemType Directory -Force -Path .\backup | Out-Null
  $ts = (Get-Date).ToUniversalTime().ToString("yyyyMMdd-HHmmss")
  sqlite3 .\data\trips.db ".backup '$(Resolve-Path .\backup)\remi-trips-$ts.db'"
  ```

  Store backups on a **different disk or host** than the live database when possible.

- **Raw file copy:** you can also copy `SQLITE_PATH` while the app is stopped, or copy from the Docker **`remi-data`** volume (see [docs/self-hosting.md](docs/self-hosting.md)).
- **Uploads:** back up **`web/static/uploads/`** together with the DB (or the Docker **`remi-uploads`** volume / bind mount). Attachments are not inside the SQLite file.
- **Opt-in scripts:** [scripts/backup-sqlite.sh](scripts/backup-sqlite.sh) and [scripts/backup-sqlite.ps1](scripts/backup-sqlite.ps1); optional [docker-compose.backup.yml](docker-compose.backup.yml) (Compose **profile `backup`**) — wire to **cron** or **Task Scheduler** yourself; nothing runs automatically.

---

## Security notes

- Use **strong passwords**, **HTTPS** in production, and set **`REMI_ENV=production`** so session cookies are **`Secure`** and HSTS/CSP-report-only headers apply. Restrict who can reach the server (VPN, firewall, or authenticated reverse proxy).
- Configure **`REMI_TRUSTED_PROXIES`** when behind a reverse proxy so rate limits and logs see the real client IP; never trust `X-Forwarded-For` from the open internet without a trusted hop.
- **Auth audit (structured logs):** successful login, failed login, logout, and password change emit **`slog`** events with **`request_id`**, **client IP** (after trusted-proxy handling), and a **truncated `User-Agent`** — never the password or login identifier.
- **5xx correlation:** panics and internal server failures use **`writeInternalServerError`**, returning a short public **`error_id`** (HTML page or JSON for `/api/*` / `Accept: application/json`); server logs include the same id plus **`request_id`**, path, method, and **`user_id`** when authenticated.
- **Auth:** registration errors use a **generic** message when the email/username may already exist, to reduce enumeration.
- **CSP (report-only):** in production the app sends `Content-Security-Policy-Report-Only`. Tune `connect-src` / `script-src` in `internal/httpapp/routes.go` if your deployment adds other CDNs or APIs, then consider enforcing CSP (drop `Report-Only`) once stable.
- Keep **secrets** out of git (`.env` is gitignored). Use strong host permissions on the SQLite file.

---

## License & changelog

- **License:** [MIT](LICENSE).
- **History:** [CHANGELOG.md](CHANGELOG.md).
