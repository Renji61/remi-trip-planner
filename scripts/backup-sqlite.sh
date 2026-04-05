#!/usr/bin/env sh
# Opt-in SQLite backup using .backup (safe while the app is running).
# Usage: ./scripts/backup-sqlite.sh [path/to/trips.db] [output_directory]
# Schedule with cron on the host if desired.

set -eu
SRC="${1:-./data/trips.db}"
OUT_DIR="${2:-./backup}"
mkdir -p "$OUT_DIR"
TS=$(date -u +%Y%m%d-%H%M%S)
DST="$OUT_DIR/remi-trips-$TS.db"
sqlite3 "$SRC" ".backup '$DST'"
echo "Wrote $DST"
