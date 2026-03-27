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
	if _, err = db.Exec(`PRAGMA journal_mode=WAL; PRAGMA foreign_keys=ON; PRAGMA busy_timeout=5000;`); err != nil {
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
		`ALTER TABLE app_settings ADD COLUMN registration_enabled INTEGER NOT NULL DEFAULT 1`,
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
		`ALTER TABLE trips ADD COLUMN ui_show_itinerary INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE trips ADD COLUMN ui_show_checklist INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE trips ADD COLUMN ui_itinerary_expand TEXT NOT NULL DEFAULT 'first'`,
		`ALTER TABLE trips ADD COLUMN ui_spends_expand TEXT NOT NULL DEFAULT 'first'`,
		`ALTER TABLE trips ADD COLUMN ui_time_format TEXT NOT NULL DEFAULT '12h'`,
		`ALTER TABLE trips ADD COLUMN ui_label_stay TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE trips ADD COLUMN ui_label_vehicle TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE trips ADD COLUMN ui_label_flights TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE trips ADD COLUMN ui_label_spends TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE trips ADD COLUMN ui_main_section_order TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE trips ADD COLUMN ui_sidebar_widget_order TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE trips ADD COLUMN ui_show_custom_links INTEGER NOT NULL DEFAULT 1`,
		`ALTER TABLE trips ADD COLUMN ui_custom_sidebar_links TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE trips ADD COLUMN ui_main_section_hidden TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE trips ADD COLUMN ui_sidebar_widget_hidden TEXT NOT NULL DEFAULT ''`,
	} {
		if _, err = db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return nil, err
		}
	}
	if err := migrateAuthAndSharing(db); err != nil {
		return nil, err
	}
	return db, nil
}

func migrateAuthAndSharing(db *sql.DB) error {
	stmts := []string{
		`CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE COLLATE NOCASE,
			username TEXT NOT NULL UNIQUE COLLATE NOCASE,
			display_name TEXT NOT NULL DEFAULT '',
			password_hash TEXT NOT NULL,
			avatar_path TEXT NOT NULL DEFAULT '',
			email_verified_at TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			csrf_token TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_sessions_user ON sessions(user_id)`,
		`CREATE TABLE IF NOT EXISTS email_verify_tokens (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			token_hash TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_email_verify_user ON email_verify_tokens(user_id)`,
		`CREATE TABLE IF NOT EXISTS user_settings (
			user_id TEXT PRIMARY KEY,
			theme_preference TEXT NOT NULL DEFAULT 'system',
			dashboard_trip_layout TEXT NOT NULL DEFAULT 'grid',
			dashboard_trip_sort TEXT NOT NULL DEFAULT 'name',
			dashboard_hero_background TEXT NOT NULL DEFAULT 'default',
			trip_dashboard_heading TEXT NOT NULL DEFAULT 'Trip Dashboard',
			default_currency_name TEXT NOT NULL DEFAULT 'USD',
			default_currency_symbol TEXT NOT NULL DEFAULT '$',
			updated_at DATETIME NOT NULL,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE TABLE IF NOT EXISTS trip_members (
			trip_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			invited_by_user_id TEXT NOT NULL DEFAULT '',
			joined_at DATETIME NOT NULL,
			left_at DATETIME,
			PRIMARY KEY (trip_id, user_id),
			FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_trip_members_user ON trip_members(user_id)`,
		`CREATE TABLE IF NOT EXISTS trip_invites (
			id TEXT PRIMARY KEY,
			trip_id TEXT NOT NULL,
			email_normalized TEXT NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			invited_by_user_id TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL,
			accepted_at DATETIME,
			revoked_at DATETIME,
			FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE,
			FOREIGN KEY (invited_by_user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_trip_invites_trip ON trip_invites(trip_id)`,
		`CREATE TABLE IF NOT EXISTS trip_invite_links (
			id TEXT PRIMARY KEY,
			trip_id TEXT NOT NULL,
			token_hash TEXT NOT NULL UNIQUE,
			invited_by_user_id TEXT NOT NULL,
			expires_at DATETIME NOT NULL,
			created_at DATETIME NOT NULL,
			revoked_at DATETIME,
			FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE,
			FOREIGN KEY (invited_by_user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_trip_invite_links_trip ON trip_invite_links(trip_id)`,
		`CREATE TABLE IF NOT EXISTS trip_collaborator_dashboard (
			trip_id TEXT NOT NULL,
			user_id TEXT NOT NULL,
			hidden_archived INTEGER NOT NULL DEFAULT 0,
			PRIMARY KEY (trip_id, user_id),
			FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE,
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
		)`,
	}
	for _, s := range stmts {
		if _, err := db.Exec(s); err != nil {
			return err
		}
	}
	if _, err := db.Exec(`ALTER TABLE trips ADD COLUMN owner_user_id TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
	}
	if _, err := db.Exec(`CREATE INDEX IF NOT EXISTS idx_trips_owner ON trips(owner_user_id)`); err != nil {
		return err
	}
	return nil
}
