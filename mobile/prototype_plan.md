# Native App Prototype Path

## Recommended First Native Build

- Framework: Flutter
- Local database: Drift/SQLite
- Sync worker: periodic background sync with exponential backoff

## Shared Data Model

- `Trip`
- `ItineraryItem`
- `Expense`
- `ChecklistItem`
- `ChangeCursor`
- `QueuedOp`

## MVP Native Screens

- Trip list
- Trip details (itinerary, expenses, checklist)
- Offline queue status

## Sync Behavior

- All writes go to local DB first.
- Write operation is pushed to `QueuedOp`.
- Sync worker sends queued ops to `/api/v1/trips/{tripID}/sync`.
- Worker pulls server changes from `/changes?since=cursor`.

## Acceptance Criteria

- User can create itinerary/expense/checklist items without internet.
- Changes sync automatically when network returns.
