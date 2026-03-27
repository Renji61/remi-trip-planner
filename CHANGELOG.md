# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html) where practical.

## [Unreleased]

### Planned / follow-ups

- Flesh out `POST /api/v1/trips/{tripID}/sync` request handling per [docs/sync_contract.md](docs/sync_contract.md).
- Richer conflict handling beyond last-write-wins for sync clients.

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
- **Cursor rules** under `.cursor/rules/` for local rebuild/restart on port 8051 (optional for contributors using Cursor).

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

- README expanded for Docker vs local Go, ports (`8051:8080` in Compose), and environment variables.
- Trip page template pipeline hardened (buffered render + error handling) to avoid truncated HTML on template errors.

### Security

- No authentication layer in this release — deploy behind a private network, VPN, or reverse proxy auth if exposed to the internet.
- Do not commit `.env` files or production databases; `data/` and uploads are gitignored by default.

[Unreleased]: https://github.com/Renji61/remi-trip-planner/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/Renji61/remi-trip-planner/compare/5d7e105...v1.1.0
