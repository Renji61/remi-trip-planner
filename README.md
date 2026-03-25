# REMI Trip Planner

A lightweight, Dockerized, Wanderlog-style trip planner focused on low-end hardware performance.

## Stack

- Go + Chi + HTMX (SSR)
- SQLite (WAL mode)
- Leaflet + OpenStreetMap
- Docker + Compose + Caddy (HTTPS)

## Features Implemented

- Trip creation and trip listing
- Day-based itinerary items with locations and notes
- Map pins + route polyline for itinerary coordinates
- Expense tracking and total trip budget view
- Packing/checklist with done/undo state
- PWA base (manifest + service worker)
- Sync-ready API endpoints:
  - `GET /api/v1/trips/{tripID}/changes`
  - `POST /api/v1/trips/{tripID}/sync`

## Run Locally

```bash
docker compose up --build
```

Open `http://localhost:8051`.

## Remote Access

1. Point your domain DNS to your server.
2. Update `deploy/Caddyfile` with your real domain.
3. Start stack with Docker Compose. Caddy automatically provisions HTTPS certs.

## Offline/Native Path

- Current PWA caches core assets.
- Sync endpoints are in place for mobile/native clients.
- Recommended native app path: Flutter or React Native using sync API.

## Backup

SQLite DB path in container: `/app/data/trips.db` (persisted in Docker volume `trip_data`).
