# Sync Contract (Prototype)

## Objective

Enable local-first clients (web PWA, Android, iOS) to:
- read trip updates from server
- queue local changes while offline
- sync when online

## Endpoints

- `GET /api/v1/trips/{tripID}/changes?since=<rfc3339>`
  - Returns ordered server-side `change_log` records.
- `POST /api/v1/trips/{tripID}/sync`
  - Prototype response implemented.
  - Intended request body:
    - `client_id` (string)
    - `base_cursor` (string timestamp or numeric change id)
    - `ops` (array of create/update/delete operations)

## Change Record

Each change entry contains:
- `id` (monotonic integer cursor)
- `trip_id`
- `entity` (`trip`, `itinerary_item`, `expense`, `checklist_item`)
- `entity_id`
- `operation` (`create`, `update`, `delete`)
- `changed_at`
- `payload` (JSON blob)

## Conflict Strategy

Initial strategy: **last-write-wins** by server timestamp.

Later improvements:
- per-field merge for text fields
- conflict buckets for concurrent position edits in itinerary ordering

## Client Flow (Offline-Capable)

1. Client stores local operations queue.
2. On reconnect, client sends queued operations to `sync`.
3. Server applies operations and appends change log entries.
4. Client fetches `changes?since=lastCursor`.
5. Client updates local store and advances cursor.
