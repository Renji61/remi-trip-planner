#!/bin/sh
set -e
# Fresh named volumes mount over /app/data and uploads with root ownership; the app runs as non-root.
# Fix ownership on every start so SQLite and uploads can write (idempotent).
chown -R remi:remi /app/data /app/web/static/uploads
exec su-exec remi "$@"
