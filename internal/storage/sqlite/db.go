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
	if _, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS trip_day_labels (
			trip_id TEXT NOT NULL,
			day_number INTEGER NOT NULL,
			label TEXT NOT NULL DEFAULT '',
			updated_at DATETIME NOT NULL,
			PRIMARY KEY (trip_id, day_number),
			FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
		)`); err != nil {
		return nil, err
	}
	for _, stmt := range []string{
		`ALTER TABLE app_settings ADD COLUMN theme_preference TEXT NOT NULL DEFAULT 'system'`,
		`ALTER TABLE app_settings ADD COLUMN dashboard_trip_layout TEXT NOT NULL DEFAULT 'grid'`,
		`ALTER TABLE app_settings ADD COLUMN dashboard_trip_sort TEXT NOT NULL DEFAULT 'name'`,
		`ALTER TABLE app_settings ADD COLUMN dashboard_hero_background TEXT NOT NULL DEFAULT 'default'`,
		`ALTER TABLE app_settings ADD COLUMN trip_dashboard_heading TEXT NOT NULL DEFAULT 'Trip Dashboard'`,
	} {
		if _, err = db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return nil, err
		}
	}
	for _, stmt := range []string{
		`ALTER TABLE trips ADD COLUMN ui_show_stay INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE trips ADD COLUMN ui_show_vehicle INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE trips ADD COLUMN ui_show_flights INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE trips ADD COLUMN ui_show_spends INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE trips ADD COLUMN ui_itinerary_expand TEXT NOT NULL DEFAULT 'first'`,
		`ALTER TABLE trips ADD COLUMN ui_spends_expand TEXT NOT NULL DEFAULT 'first'`,
		`ALTER TABLE trips ADD COLUMN ui_time_format TEXT NOT NULL DEFAULT '12h'`,
		`ALTER TABLE trips ADD COLUMN ui_label_stay TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE trips ADD COLUMN ui_label_vehicle TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE trips ADD COLUMN ui_label_flights TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE trips ADD COLUMN ui_label_spends TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE trips ADD COLUMN ui_main_section_order TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE trips ADD COLUMN ui_sidebar_widget_order TEXT NOT NULL DEFAULT ''`,
	} {
		if _, err = db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return nil, err
		}
	}
	return db, nil
}
