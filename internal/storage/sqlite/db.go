package sqlite

import (
	"database/sql"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

func OpenAndMigrate(dbPath, migrationFile string) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	if _, err = db.Exec(`PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON;`); err != nil {
		return nil, err
	}
	migrationSQL, err := os.ReadFile(migrationFile)
	if err != nil {
		return nil, err
	}
	if _, err = db.Exec(string(migrationSQL)); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE trips ADD COLUMN cover_image_url TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE trips ADD COLUMN currency_name TEXT NOT NULL DEFAULT 'USD'`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE trips ADD COLUMN currency_symbol TEXT NOT NULL DEFAULT '$'`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE trips ADD COLUMN is_archived BOOLEAN NOT NULL DEFAULT FALSE`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE lodging_entries ADD COLUMN check_in_itinerary_id TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE lodging_entries ADD COLUMN check_out_itinerary_id TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE expenses ADD COLUMN lodging_id TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE expenses ADD COLUMN payment_method TEXT NOT NULL DEFAULT 'Cash'`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE vehicle_rentals ADD COLUMN vehicle_detail TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE vehicle_rentals ADD COLUMN vehicle_image_path TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE checklist_items ADD COLUMN category TEXT NOT NULL DEFAULT 'Packing List'`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`UPDATE checklist_items SET category = 'Packing List' WHERE TRIM(COALESCE(category, '')) = ''`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`UPDATE expenses SET category = 'Accommodation' WHERE category = 'Hotels & Lodging'`); err != nil {
		return nil, err
	}
	return db, nil
}
