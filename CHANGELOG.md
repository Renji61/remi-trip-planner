# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html) where practical.

## [Unreleased]

### Planned / follow-ups

- Flesh out `POST /api/v1/trips/{tripID}/sync` request handling per [docs/sync_contract.md](docs/sync_contract.md).
- Richer conflict handling beyond last-write-wins for sync clients.

## [1.40.0] - 2026-03-28

### Added

- **Site settings — default map location:** single **Default location** field with place suggestions (same pattern as the dashboard); stores a **short place name** and coordinates. **Tokyo** is the default when the field is cleared or when a new trip is created without picking a place on the dashboard (trip map uses app defaults).
- **Database:** `map_default_place_label` on `app_settings` (migrated on startup); `INSERT` in base migration kept compatible with existing databases before the new column exists.
- **Mobile account menu:** **Profile** as the first action; **App settings** / **Trip settings** and **Log out** with clearer layout, icons, and separator.

### Changed

- **Default ports:** native `go run` / `APP_ADDR` default **`:4122`**; Docker Compose default host mapping **`4122:8080`** (`REMI_PORT` default **4122**; container still listens on **8080** inside the image).
- **Docker Compose:** **Watchtower** (and related labels/env) **removed** from all compose files — add your own auto-update tooling if desired.

### Removed

- Watchtower service and `com.centurylinklabs.watchtower.enable` labels from `docker-compose.yml`, `docker-compose.registry.yml`, and `docker-compose.install.yml`.

### Notes for self-hosters

- **Update notification:** the About page and `GET /api/about/update-check` compare the running build to **GitHub Releases**. Publish tag **`v1.40.0`** (and the corresponding image tag if you use GHCR) so instances on older versions see a newer release.

## [1.3.0] - 2026-03-28

### Added

- **About** page (`/about`): installed version, optional **Check for updates** (and automatic check on load via GitHub Releases), release notes from the bundled changelog for this version, and a plain-language **feature** list.
- **Dashboard navigation:** **Discover** and **Spends** removed from the sidebar; **About**, **Profile**, and **Logout** use icons; up to **two in-progress trips** (by trip dates, not archived) appear under **My Trip** and in the **mobile bottom bar** in place of the old Explore/Spends slots.
- **API:** `GET /api/about/update-check` (authenticated) returns current vs latest release metadata for the UI.

### Changed

- **Support** renamed to **About** (sidebar and trip sidebars link to `/about`).
- **Trip** sidebars: **Logout** uses a proper POST form with CSRF; **About** includes an icon.

## [1.2.0] - 2026-03-28

### Added

- **Public container image** on GitHub Container Registry: `ghcr.io/renji61/remi-trip-planner:latest` (and SemVer tags from `v*.*.*` via CI).
- **`docker-compose.install.yml`** — single-file homelab install with **no `.env` required** (literal image and port).
- **`docker-compose.registry.yml`** — same stack with **optional** `.env` overrides (`REMI_IMAGE` defaults to the official image; host port default later standardized to **4122** in v1.40.0).
- **`docker-compose.yml`** — build from clone; healthchecks via `wget`.
- **`.github/workflows/docker-publish.yml`** — build and push to GHCR on `main` and SemVer tags.
- **Docs:** [docs/self-hosting.md](docs/self-hosting.md), [docs/publish-image.md](docs/publish-image.md); **`.env.example`**, **`.dockerignore`**; **`scripts/publish-ghcr.ps1`** / **`scripts/publish-ghcr.sh`**.

### Changed

- **Dockerfile:** Alpine runtime includes **`wget`**, **`HEALTHCHECK`** on `GET /healthz`.
- **README:** Docker & self-hosting section aligned with GHCR install paths.

### Notes for installers

- Default GitHub Actions build targets **linux/amd64**; ARM homelabs may need `platform:` in Compose or multi-arch builds (see [docs/publish-image.md](docs/publish-image.md)).

## [1.1.1] - 2026-03-28

### Fixed

- **Docker image build:** builder stage now uses **Go 1.25** (`golang:1.25-alpine`) to match `go 1.25.0` in `go.mod`, fixing `go mod download` failing under the previous Go 1.23 image.
- **Dockerfile:** copy **`go.sum`** together with `go.mod` before `go mod download` for correct module checksum resolution.

## [1.1.0] - 2026-03-27

Multi-user accounts, trip collaboration, richer trip layout controls, and an interactive itinerary map. Changes since [5d7e105](https://github.com/Renji61/remi-trip-planner/commit/5d7e105) (v1.0.0 line on `main`).

### Upgrade notes

- **Authentication is now required** for the web UI: first-time **setup**, **register**, and **login** replace the anonymous-only flow from v1.0.0. SQLite databases are migrated on startup; use **HTTPS** and **secure cookies** when exposing the server to a network.

### Added

#### Accounts & access

- **User registration**, **login**, **logout**, and **session** handling with CSRF on POST routes.
- **First-run setup** and **site settings** integration for an initial admin-style bootstrap path.
- **Profile** page: display name, avatar path, **password change**, **email verification** request/resend.
- **Trip access** model: owner/collaborator checks on trip routes; **invite** flows (invite link / email-oriented handlers) and **trip members** UI (`trip_members_panel` partial).
- **Invite accept** page for pending collaborators.
- **`cmd/resetpassword`:** CLI to set a user password against the configured SQLite database (operators / recovery).

#### Trip UI & layout

- **Trip settings** page for per-trip options (layout, labels, visibility) aligned with the main trip shell.
- **Trip sidebar navigation** partial for consistent section links.
- **Custom sidebar links** (per trip): ordered list of extra shortcuts in the trip rail.
- **Main section hide/show** and **layout order** persisted per trip (extends v1 layout ordering); templates and services updated for `UIMainSectionHidden` / `UISidebarWidgetHidden` and related order fields.
- **Shared confirm dialog** partial for destructive actions (`app_confirm_dialog`).

#### Trip map (Leaflet)

- **Per-itinerary-day marker colors** (distinct palette; ordinal mapping across days present on the map).
- **Marker kinds:** hotel (stays), flight (airports), car (vehicle pick-up), place (generic stops).
- **Day legend** as **toggle buttons**: show/hide markers per day; map **zooms and pans** to fit **only visible** markers (falls back to card default center when none selected).
- **No connecting polylines** between stops; markers are independent.

#### HTTP & static assets

- **`Cache-Control: no-cache, must-revalidate`** on `/static/*.js`, `/static/*.css`, and **`/sw.js`** so template query-string bumps and SW updates are picked up reliably.
- **Service worker** cache revision bumped (`remi-trip-planner-v15`); install precache list unchanged in behavior (network-first for `/static/`).

#### Operations & developer experience

- **Server cwd:** if the binary lives under `bin/`, the process **`chdir`s to the parent** of the executable directory so `web/templates` and `web/static` resolve when running `bin/remi-server.exe`.
- **`.air.toml`** and **`scripts/run-dev.cmd`** for live reload workflows; **`scripts/deploy-docker.ps1`** helper; **`scripts/dev-watch.ps1`** updates.

#### Other

- **Budget formatting** helpers and tests (`budget_format.go`).
- **Google Maps search URL** helper for itinerary map links (`maps_url.go`).

### Changed

- **`internal/httpapp/routes.go`:** large routing expansion (auth, profile, invites, trip settings, static handler wrapper).
- **SQLite repository and schema migration path** extended for users, sessions, and trip access (see `repo_auth.go`, `db.go`, `repo.go`).
- **Trip and user services** split/extended (`service_auth.go`, `access.go`, `users.go`, `defaults_reset.go`).
- **Templates:** new login, register, setup, profile, verify email, invite accept, trip settings, members panel, sidebar nav; trip/home/settings/partials and entity pages adjusted for auth shell and layout flags.
- **`go.mod` / `go.sum`:** dependency updates for the above.
- **`.gitignore`:** ignore backup files (`*~`); existing rules for `bin/`, `.env`, uploads unchanged.

### Security

- **Sessions and passwords** are now part of the threat model: run **HTTPS** in production, configure **secure cookies** for your host, and restrict database file permissions.
- **Email verification** and **invites** should use trustworthy mail delivery and short-lived tokens where applicable (configure outbound mail / base URL per deployment).
- Prior v1.0.0 guidance still applies: do not commit `.env` or production databases; keep `data/` and uploads out of git.

## [1.0.0] - 2026-03-26

First public release: self-hosted trip planner with SQLite, SSR UI, optional Docker deployment, and sync-oriented API stubs.

### Added

#### Core & data

- **SQLite** persistence with WAL mode, foreign keys, and incremental **schema migrations** applied at startup (`migrations/001_init.sql` plus `ALTER` steps in code for existing databases).
- **Change log** table and **API** to list changes for sync clients: `GET /api/v1/trips/{tripID}/changes`.
- **Sync prototype** endpoint: `POST /api/v1/trips/{tripID}/sync` (contract described in [docs/sync_contract.md](docs/sync_contract.md)).

#### Trips & dashboard

- **Home / dashboard:** trip cards (grid and list layouts), draft vs active vs archived groupings, travel-style stats, configurable hero and headings.
- **Trip CRUD:** create from dashboard; per-trip name, description, start/end dates, cover image URL, currency name/symbol, archive and delete.
- **Archived trips:** read-only UI with archive/delete still available where applicable.

#### Itinerary

- **Day-grouped itinerary** on the trip page with optional **per-day labels** (inline save).
- **Stops** with title, location, notes, estimated cost, start/end times, optional geocoding (Nominatim) when enabled in app settings.
- **Map card** (Leaflet + OpenStreetMap): default center/zoom from app settings; pins and route from itinerary coordinates.
- **Itinerary search** (client-side filter) on the trip page.
- Inline **edit / delete** for stops; **mobile** long-press sheet for actions where supported.

#### Spends & budget

- **Expenses** with category, amount, date, payment method, notes; some categories tied to lodging / vehicle / flight bookings.
- **Trip page:** spends section grouped by day, **budget tile** (total budgeted vs spent, progress); mobile budget placement inside spends card on narrow viewports.
- **Quick expense** form on desktop sidebar (order configurable).
- **Budget page** per trip: transactions view, export, HTMX-powered rows where used.

#### Stay, vehicle, flights

- Dedicated **accommodation**, **vehicle rental**, and **flights** pages with full CRUD.
- **File uploads** for booking attachments, vehicle images, flight documents (served from `web/static/uploads/`; path ignored by git).
- **Itinerary integration:** check-in/out, pick-up/drop-off, depart/arrive rows linked to bookings; synced **expense** lines where applicable.

#### Checklist

- **Reminder checklist** with categories; done/undo toggle; add from trip page (single or batch JSON); edit/delete per item.

#### Trip page UI & personalization

- **Section visibility:** show or hide Stay, Vehicle, Flights, Spends (and related nav, budget, quick expense).
- **Labels:** optional custom nav/titles for Stay, Vehicle, Flights, Spends.
- **Defaults:** first / all / none for expanded itinerary days and spend days on the trip page.
- **Time format:** 12h vs 24h for trip-scoped datetime display.
- **Layout order (per trip):** user-defined sequence for main blocks — Trip Map, Itinerary, Spends, Reminder Checklist, Stay, Vehicle, Flights (hero/edit panel stay fixed above).
- **Sidebar order (per trip):** user-defined sequence for Add New Stop, Total Budgeted Cost, Quick Spends, Add to Checklist (desktop/tablet right column; budget/quick hidden when Spends is off).

#### App settings

- Global **settings** page: app title, default currency, map defaults (lat/lng/zoom), location lookup toggle, **theme** preference (light/dark/system) with quick POST toggle from trip shell.
- **Dashboard:** trip card layout (grid/list), sort order, hero background pattern, trip dashboard heading text.

#### PWA & static assets

- **Web app manifest** and **service worker** for installability and basic offline asset caching.
- Static **CSS/JS** pipeline: Manrope/Inter, Material Symbols, Leaflet from CDN on relevant pages.

#### Operations & developer experience

- **Dockerfile** (multi-stage Go 1.23 build, Alpine runtime) and **docker-compose** with named volume for `/app/data`.
- **Optional Caddy** example for HTTPS in `deploy/`.
- **Server** resolves module root via `REMI_ROOT`, executable directory, cwd, parent-folder scan (only `module remi-trip-planner`), or walking up to `go.mod` — so templates and static files load reliably.
- **`cmd/dbpeek`:** small CLI to inspect trip rows in SQLite (development aid).
- **CI workflow template** in `docs/github-actions-ci.yml` (`gofmt -l`, `go vet ./...`, `go test ./...`) — copy to `.github/workflows/` when ready; OAuth pushes without `workflow` scope may be rejected for workflow files.
- **Repository:** `.gitattributes` (line endings), `.gitignore` for `data/`, uploads, env files, binaries, coverage.

#### Legal & docs

- **MIT License** (`LICENSE`).
- **README** with stack, features, run instructions, deployment pointers.
- **This changelog.**

### Changed

- README expanded for Docker vs local Go, ports, and environment variables.
- Trip page template pipeline hardened (buffered render + error handling) to avoid truncated HTML on template errors.

### Security

- No authentication layer in this release — deploy behind a private network, VPN, or reverse proxy auth if exposed to the internet.
- Do not commit `.env` files or production databases; `data/` and uploads are gitignored by default.

[Unreleased]: https://github.com/Renji61/remi-trip-planner/compare/v1.40.0...HEAD
[1.40.0]: https://github.com/Renji61/remi-trip-planner/compare/v1.3.0...v1.40.0
[1.3.0]: https://github.com/Renji61/remi-trip-planner/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/Renji61/remi-trip-planner/compare/v1.1.1...v1.2.0
[1.1.1]: https://github.com/Renji61/remi-trip-planner/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/Renji61/remi-trip-planner/compare/5d7e105...v1.1.0
