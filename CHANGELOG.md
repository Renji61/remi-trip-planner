# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html) where practical.

## [Unreleased]

### Planned / follow-ups

- Richer conflict handling for sync clients beyond optimistic locking on selected entities.

## [1.49.0] - 2026-04-05

### Added

- **Auth audit logging:** structured `slog` events `login_success`, `login_failed` (extended), `logout`, and `password_changed` with `request_id`, trusted-proxy client IP, and truncated User-Agent (no passwords or login identifiers).
- **5xx error correlation:** custom recoverer and `writeInternalServerError` return a public `error_id` (HTML or JSON for API/JSON clients); server logs include `error_id`, `request_id`, path, method, and `user_id` when known. Panic logs attach the same `error_id` to stack traces. Authenticated and public **`httpapp`** handlers that previously used `http.Error(..., 500)` route through `writeInternalServerError` where applicable (including template render failures on trip pages).
- **Account data export:** `GET /profile/export` returns `application/json` attachment `remi-export-YYYYMMDD.json` (profile without password hash, user settings, redacted app settings, visible trips with persisted entities and document metadata).
- **Backup helpers:** `scripts/backup-sqlite.sh`, `scripts/backup-sqlite.ps1`, and optional `docker-compose.backup.yml` (Compose profile `backup`); README and self-hosting docs include `sqlite3 .backup` examples and uploads volume reminders.
- **Sync API:** `POST /api/v1/trips/{tripID}/sync` applies JSON `ops` (trip / itinerary_item / expense / checklist_item) with per-op results, `server_changes` delta, optional numeric `base_cursor` + `stale_base`; trip delete/archive require owner. `GET .../changes` JSON uses snake_case fields on change objects. **Conflict** results use code `conflict` when optimistic locking rejects a stale write.
- **Money in minor units:** expense amounts, group-expense splits, tab settlements, trip budget cap, itinerary estimated costs, and booking totals are stored as **integer minor units** (e.g. cents) in SQLite with additive migration from legacy `REAL` columns; UI and APIs continue to show decimal amounts where appropriate.
- **Optimistic concurrency:** `updated_at` and `expected_updated_at` on key entities (expenses, bookings, itinerary lines, etc.) so concurrent edits can return **409 Conflict** with JSON `code: conflict` for HTMX/async clients instead of silent last-write-wins.
- **Transactional bookings:** add/update/delete for lodging, vehicle rental, and flights runs in a **single DB transaction** with linked itinerary rows and expenses so partial failures roll back cleanly.
- **Live UI refresh:** Server-Sent Events (`GET /api/v1/trips/{tripID}/events`, `text/event-stream`) so open sessions can refresh lists without a full page reload (alongside HTMX/fetch patterns elsewhere).

### Changed

- **User-facing errors:** calmer copy for common server failures instead of a raw “500 Internal Server Error” where the unified error writer applies.
- **Trip Documents / invites:** client-side caching of generated invite links avoids repeated `POST /invite-link` when the members panel re-renders (prevents token rotation loops on document-heavy pages).
- **Docker / Compose:** image runs as a **non-root** user; optional **read-only root filesystem**, **`tmpfs` for `/tmp`**, **`no-new-privileges`**, and **`cap_drop: ALL`** on the default `docker-compose.yml` service (adjust if your environment needs extra caps).

### Security

- **CSRF** validation on state-changing requests; **session cookies** use **HttpOnly** and **SameSite=Strict** (and **Secure** when `REMI_ENV=production`).
- **Upload hardening:** stricter type/extension checks (e.g. SVG and risky types rejected where configured); size limits enforced server-side to match app settings.
- **Self-hosted update check** unchanged: still compares this build to **GitHub Releases** — publishing **`v1.49.0`** is what makes older installs show an update on **About** / `GET /api/about/update-check`.

### Notes for self-hosters

- **Update notification:** publish GitHub Release **`v1.49.0`** and the matching **GHCR** image tag if you pull from the registry, so instances on **1.48.0** or older detect the new version.
- **Database:** startup migrations add cent columns and `updated_at` fields; existing installs upgrade in place. **Rebuild** or **pull** the image and restart the container/process after backup.

## [1.48.0] - 2026-04-01

### Added

- **Trip Documents** (`/trips/{id}/documents`): one place to upload general files and browse **all** trip attachments with quick search, category filter, rename/delete for general uploads, and links back to stays, vehicle rentals, flights, or group expenses where relevant.
- **SQLite `trip_documents`** table (created on upgrade for existing databases) plus index; metadata ties files to trip sections.
- **App settings — max upload size per file (MB)** (default **5**): applies to Trip Documents and document/image fields on trip forms.
- **Upload validation** (`SaveValidatedUploadFromHeader` / profiles): blocks dangerous extensions, sniffs allowed content types (images, PDF, common Office formats for bookings, receipts), and enforces the configured size cap.
- **Docker Compose:** named volume **`remi-uploads`** → `/app/web/static/uploads` in **`docker-compose.yml`**, **`docker-compose.registry.yml`**, and **`docker-compose.install.yml`** so attachments survive container recreation.

### Changed

- **Geocoding:** if the first free-text lookup misses, **one retry** after normalizing the query (whitespace, `;` / `|` as comma separators).
- **Templates / UI:** shared **date, time, and datetime** field partials aligned with per-trip **DD/MM/YYYY** vs **MM/DD/YYYY**; broad trip-shell, dashboard, settings, and booking-page polish (including long-press / sheet patterns and upload affordances).

### Security

- **Stricter uploads:** executable/script extensions and content/type mismatches are rejected before files are stored.

### Notes for self-hosters

- **Update notification:** publish GitHub Release **`v1.48.0`** (and the matching **GHCR** image tag if you pull from the registry) so instances on **1.47.0** or older see an update on **About** / `GET /api/about/update-check`.
- **Docker upgrades:** new installs get the **`remi-uploads`** volume automatically. If you previously stored uploads only inside the container filesystem, **copy them into the volume** (or bind-mount) when adopting this compose so existing files are not lost.

## [1.47.0] - 2026-03-29

### Added

- **Maps API caching (server):** in-memory TTL caches for Google Geocoding, Places Autocomplete, Place Details, and Nominatim suggestions/geocode; caps on map size to limit memory.
- **Location API HTTP caching:** `Cache-Control: private` on successful `/api/location/geocode` and `/api/location/suggest` responses where applicable.
- **Client-side:** in-memory caches and in-flight deduplication for geocode and suggest calls from the browser.

### Changed

- **Google Maps (trip page):** light/dark map styling uses the **styled maps** path so the basemap updates immediately when the app theme changes (the Maps JS `colorScheme` option does not apply after map creation).
- **Group expenses (UI):** “Total on Tab” renamed to **Total Group Expense**; the same summary tile appears in the **desktop sidebar** above quick group-expense actions when that section is enabled.
- **Trip page map:** day chips to show/hide markers by itinerary day (Google Maps and Leaflet); custom itinerary markers (day-colored ring + kind icon) on Google Maps.
- **About page:** “What you can do with REMI Trip Planner” refreshed for map modes, group expenses, per-trip date format, geocoding cache note, and self-hosted update checks.

### Fixed

- **Itinerary stop edits:** saving a plain stop now **geocodes** the location and persists **latitude/longitude** in SQLite; between-stop distances and the **trip map** stay in sync after AJAX save (await connector geocoding before map refresh; `fetch` for fresh trip HTML uses **`cache: "no-store"`**).
- **Trip map after edit:** markers update by itinerary item id without a full page reload.
- **Inline itinerary edit:** **Edit / Delete** actions remain available after save (clear `.editing` when the form node is missing; re-apply desktop `details` open state; re-wire open buttons).

### Notes for self-hosters

- **Update notification:** publish GitHub Release **`v1.47.0`** (and the matching GHCR image tag if you pull from the registry) so instances on **1.46.0** or older see an update on About / `GET /api/about/update-check`.

## [1.46.0] - 2026-03-29

### Added

- **Per-trip calendar date format:** choose **DD/MM/YYYY** or **MM/DD/YYYY** in trip settings (alongside 12h/24h); used for expense dates, itinerary, flights, and other trip-scoped date displays.
- **Group expenses (tab):** store **departed participants** (who left the trip) so historical splits and settlements stay consistent; show **Left trip** in labels where relevant.
- **Desktop account menu:** shared **Profile / App settings / Log out** dropdown (circular initial trigger) on trip topbars and app-shell pages; **mobile dashboard** bottom nav includes **Profile** on those pages.

### Changed

- **Expenses naming & URLs:** trip subpages use **Expenses** at `/trips/{id}/expenses` and **Group expenses** at `/trips/{id}/group-expenses`; **301** redirects from legacy `/budget` and `/tab` paths.
- **Group expenses:** split math merges participant keys from stored expenses and settlements (including guests/collaborators) so balances stay correct when people leave; payer thumbs and participant labels respect departed keys.
- **Templates / context:** trip page group-expense rows receive full template context; dashboard shell merges supply **`CurrentUser`** where the account menu needs it.

### Notes for self-hosters

- **Update notification:** publish GitHub Release **`v1.46.0`** (and use the matching GHCR image tag if you pull from the registry) so instances on **1.45.0** or older see an update on About / `GET /api/about/update-check`.

## [1.45.0] - 2026-03-28

### Added

- **Dashboard navigation:** the sidebar and mobile bottom bar still show up to **two** trip shortcuts, but they now prefer **in-progress** trips first (by start date), then **upcoming** scheduled trips to fill any remaining slots (draft/archived/completed are excluded as before).
- **Trip details (mobile):** **Add to Checklist** in the floating action menu on the main trip page and on **Budget**, **Stay**, **Vehicle**, **Flights**, and **Trip settings** — same visibility rules as trip settings (**Reminder Checklist** section on + **Add to Checklist** sidebar widget not hidden). On the main trip page the action opens a **checklist sheet**; elsewhere it links to the trip with `?open=checklist` so the sheet opens there.

### Changed

- **Mobile “Trip sections” bottom bar** (trip shell): when many sections are enabled, the bar **scrolls horizontally** instead of shrinking labels with no way to reach the last tabs.
- **Trip settings (wide layout):** the right column (**Trip sections** + assist card) no longer uses a **nested scroll area**; it scrolls with the main page so the full list is reachable with normal document scrolling.

### Notes for self-hosters

- **Update notification:** the About page and `GET /api/about/update-check` compare this build’s version to **GitHub’s latest Release**. After upgrading, publish GitHub Release **`v1.45.0`** (and image tag if you use GHCR) so older instances detect the update.

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

[Unreleased]: https://github.com/Renji61/remi-trip-planner/compare/v1.49.0...HEAD
[1.49.0]: https://github.com/Renji61/remi-trip-planner/compare/v1.48.0...v1.49.0
[1.48.0]: https://github.com/Renji61/remi-trip-planner/compare/v1.47.0...v1.48.0
[1.47.0]: https://github.com/Renji61/remi-trip-planner/compare/v1.46.0...v1.47.0
[1.46.0]: https://github.com/Renji61/remi-trip-planner/compare/v1.45.0...v1.46.0
[1.45.0]: https://github.com/Renji61/remi-trip-planner/compare/v1.40.0...v1.45.0
[1.40.0]: https://github.com/Renji61/remi-trip-planner/compare/v1.3.0...v1.40.0
[1.3.0]: https://github.com/Renji61/remi-trip-planner/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/Renji61/remi-trip-planner/compare/v1.1.1...v1.2.0
[1.1.1]: https://github.com/Renji61/remi-trip-planner/compare/v1.1.0...v1.1.1
[1.1.0]: https://github.com/Renji61/remi-trip-planner/compare/5d7e105...v1.1.0
