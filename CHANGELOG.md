# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html) where practical.

## [Unreleased]

### Planned / follow-ups

- Flesh out `POST /api/v1/trips/{tripID}/sync` request handling per [docs/sync_contract.md](docs/sync_contract.md).
- Richer conflict handling beyond last-write-wins for sync clients.

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
