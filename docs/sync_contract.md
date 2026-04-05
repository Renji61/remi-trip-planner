# Sync Contract (Prototype)

## Objective

Enable local-first clients (web PWA, Android, iOS) to:
- read trip updates from server
- queue local changes while offline
- sync when online

## Endpoints

### `GET /api/v1/trips/{tripID}/changes?since=<rfc3339>`

- Requires session cookie and trip access (same as trip pages).
- Returns ordered server-side `change_log` records for that trip.
- Query `since`: optional RFC3339 timestamp; only rows with `changed_at` strictly after this value are returned (up to 500).
- Response JSON: `{ "changes": [ ... ] }` — each change has snake_case fields: `id`, `trip_id`, `entity`, `entity_id`, `operation`, `changed_at`, `payload` (JSON string).

### `POST /api/v1/trips/{tripID}/sync`

- Requires session cookie and trip access.
- **Content-Type:** `application/json`
- Applies up to **200** operations per request. Failures are **per-op**; the batch continues. Each successful mutation appends to `change_log` (same as server UI actions).

#### Request body

| Field | Type | Description |
|--------|------|-------------|
| `client_id` | string | Optional client/instance id for your own logging. |
| `base_cursor` | string | Optional **numeric** change-log cursor (digits only). If set and less than the server’s latest `change_log.id` at the start of the request, the response includes `stale_base: true` (informational; ops are still applied — last-write-wins). |
| `ops` | array | Operation objects (see below). |

#### Operation object

| Field | Type | Description |
|--------|------|-------------|
| `entity` | string | `trip`, `itinerary_item`, `expense`, `checklist_item`, or `trip_note`. |
| `entity_id` | string | Target id for update/delete (and optional for create). |
| `operation` | string | `create`, `update`, `delete`; for trips also `archive`. |
| `payload` | object | Entity-specific JSON (required for create/update). |

#### Entity notes

- **`trip`**
  - `update`: JSON patch with snake_case keys matching persisted fields, e.g. `name`, `description`, `start_date`, `end_date`, `cover_image`, `currency_name`, `currency_symbol`, `budget_cap`, map center fields, `ui_show_*`, section order strings, etc. Same rules as server `UpdateTrip` (e.g. archived trips stay read-only).
  - `archive` / `delete`: **trip owner only**; collaborators receive an error for that op.
  - `create`: not allowed on this URL (trip already exists).
- **`itinerary_item`**
  - `create`: `title` required; optional `id` (or use `entity_id`); optional `day_number`, `notes`, `location`, coordinates, times, `est_cost`, `image_path`.
  - `update`: `entity_id` or `payload.id`; payload fields override the stored item (title required non-empty when updating title).
  - `delete`: `entity_id` required.
- **`expense`**
  - `create`: `amount` ≥ 0; `category` defaults to `Miscellaneous`; optional `id` / `entity_id`; optional tab fields (`from_tab`, `split_mode`, `split_json`, `paid_by`, `title`, etc.).
  - `update`: `entity_id` or `payload.id`. Payload may include **partial** fields; only keys present are merged (same constraints as web, e.g. lodging-linked expenses cannot be edited as standalone expenses).
  - `delete`: `entity_id` required.
- **`checklist_item`**
  - `create`: `text` required; optional `category`, `id`, `entity_id`, `done`, `due_at` (YYYY-MM-DD), **`archived`**, **`trashed`** (booleans; default false). Use these flags when restoring from backup or mirroring Keep archive/trash state.
  - `update`: `entity_id` or `payload.id`; optional `text`, `category`, `due_at`, **`done`** (when `done` changes, server runs the same path as the web toggle), **`archived`**, **`trashed`**. Omit a field to leave it unchanged.
  - `delete`: `entity_id` required (hard delete, same as trip-page checklist delete).
- **`trip_note`**
  - `create`: optional `id` / `entity_id`; optional `title`, `body`, `color` (defaults to `default`), `pinned`, **`archived`**, **`trashed`** (all booleans default false). Empty title and body are allowed.
  - `update`: `entity_id` or `payload.id` required; any of `title`, `body`, `color`, `pinned`, **`archived`**, **`trashed`** may be sent as **nullable JSON fields**—only keys present are applied (set string fields to `""` explicitly if your client represents clears that way; booleans use `true`/`false`).
  - `delete`: `entity_id` required (**hard delete**, same as purge from Trash on the Notes page; does not require the note to be trashed first).

#### Response JSON

| Field | Type | Description |
|--------|------|-------------|
| `status` | string | `accepted` (all ops ok), `partial`, `rejected` (all failed), or `accepted` with `applied_count` 0 when `ops` is empty. |
| `trip_id` | string | |
| `client_id` | string | Echo when sent. |
| `applied_count` | int | Number of successful ops. |
| `stale_base` | bool | True when `base_cursor` was numeric and behind latest server change id at request start. |
| `latest_change_id` | int64 | Max `change_log.id` for this trip after applying. |
| `results` | array | `{ "index", "ok", "error"? }` per input op index. |
| `server_changes` | array | New `change_log` rows created **during this request** (same shape as `GET .../changes`). |

## Change Record

Each change entry contains:
- `id` (monotonic integer cursor)
- `trip_id`
- `entity` (`trip`, `itinerary_item`, `expense`, `checklist_item`, `trip_note`)
- `entity_id`
- `operation` (`create`, `update`, `delete`, `archive`)
- `changed_at`
- `payload` (JSON string)

## Conflict Strategy

Initial strategy: **last-write-wins** by server timestamp (operations applied in order; server validation may reject an op).

Later improvements:
- per-field merge for text fields
- conflict buckets for concurrent position edits in itinerary ordering

## Client Flow (Offline-Capable)

1. Client stores local operations queue.
2. On reconnect, client sends queued operations to `sync`.
3. Server applies operations and appends change log entries.
4. Client fetches `changes?since=lastCursor` (timestamp) or tracks numeric `id` from responses.
5. Client updates local store and advances cursor.
