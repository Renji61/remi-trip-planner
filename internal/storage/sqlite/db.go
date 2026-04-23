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
		`ALTER TABLE app_settings ADD COLUMN google_maps_api_key TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE app_settings ADD COLUMN default_distance_unit TEXT NOT NULL DEFAULT 'km'`,
		`ALTER TABLE app_settings ADD COLUMN max_upload_file_size_mb INTEGER NOT NULL DEFAULT 5`,
		`ALTER TABLE trips ADD COLUMN distance_unit TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE vehicle_rentals ADD COLUMN drop_off_location TEXT NOT NULL DEFAULT ''`,
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
		`ALTER TABLE trips ADD COLUMN ui_tab_expand TEXT NOT NULL DEFAULT 'first'`,
		`ALTER TABLE trips ADD COLUMN ui_time_format TEXT NOT NULL DEFAULT '12h'`,
		`ALTER TABLE trips ADD COLUMN ui_label_stay TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE trips ADD COLUMN ui_label_vehicle TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE trips ADD COLUMN ui_label_flights TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE trips ADD COLUMN ui_label_spends TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE trips ADD COLUMN ui_label_group_expenses TEXT NOT NULL DEFAULT ''`,
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
	if _, err = db.Exec(`ALTER TABLE user_settings ADD COLUMN distance_unit TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	for _, stmt := range []string{
		`ALTER TABLE trips ADD COLUMN home_map_latitude REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE trips ADD COLUMN home_map_longitude REAL NOT NULL DEFAULT 0`,
		`ALTER TABLE trips ADD COLUMN home_map_place_label TEXT NOT NULL DEFAULT ''`,
	} {
		if _, err = db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return nil, err
		}
	}
	if _, err = db.Exec(`ALTER TABLE app_settings ADD COLUMN map_default_place_label TEXT NOT NULL DEFAULT 'Tokyo'`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE app_settings ADD COLUMN default_ui_date_format TEXT NOT NULL DEFAULT 'dmy'`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE app_settings ADD COLUMN google_maps_map_id TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE trips ADD COLUMN budget_cap REAL NOT NULL DEFAULT 0`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE trips ADD COLUMN budget_cap_cents INTEGER NOT NULL DEFAULT 0`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`UPDATE trips SET budget_cap_cents = CAST(ROUND(COALESCE(budget_cap, 0) * 100.0) AS INTEGER) WHERE budget_cap_cents = 0 AND ABS(COALESCE(budget_cap, 0)) > 0`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE trips ADD COLUMN ui_show_the_tab INTEGER NOT NULL DEFAULT 1`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE trips ADD COLUMN ui_show_documents INTEGER NOT NULL DEFAULT 1`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE trips ADD COLUMN ui_collaboration_enabled INTEGER NOT NULL DEFAULT 1`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE expenses ADD COLUMN from_tab INTEGER NOT NULL DEFAULT 0`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE expenses ADD COLUMN receipt_path TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE expenses ADD COLUMN amount_cents INTEGER NOT NULL DEFAULT 0`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE expenses ADD COLUMN updated_at DATETIME NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`UPDATE expenses SET amount_cents = CAST(ROUND(COALESCE(amount, 0) * 100.0) AS INTEGER) WHERE amount_cents = 0 AND ABS(COALESCE(amount, 0)) > 0`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`UPDATE expenses SET updated_at = created_at WHERE TRIM(COALESCE(updated_at, '')) = ''`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`UPDATE vehicle_rentals SET pay_at_pick_up = 0`); err != nil {
		return nil, err
	}
	for _, stmt := range []string{
		`ALTER TABLE trips ADD COLUMN tab_default_split_mode TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE trips ADD COLUMN tab_default_split_json TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE expenses ADD COLUMN title TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE expenses ADD COLUMN paid_by TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE expenses ADD COLUMN split_mode TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE expenses ADD COLUMN split_json TEXT NOT NULL DEFAULT ''`,
	} {
		if _, err = db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return nil, err
		}
	}
	if _, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS trip_guests (
			id TEXT PRIMARY KEY,
			trip_id TEXT NOT NULL,
			display_name TEXT NOT NULL,
			created_at DATETIME NOT NULL,
			FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
		)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_trip_guests_trip ON trip_guests(trip_id)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS tab_settlements (
			id TEXT PRIMARY KEY,
			trip_id TEXT NOT NULL,
			payer_user_id TEXT NOT NULL,
			payee_user_id TEXT NOT NULL,
			amount REAL NOT NULL,
			method TEXT NOT NULL DEFAULT 'Cash',
			settled_on TEXT NOT NULL DEFAULT '',
			notes TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
		)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_tab_settlements_trip ON tab_settlements(trip_id)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE tab_settlements ADD COLUMN amount_cents INTEGER NOT NULL DEFAULT 0`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`UPDATE tab_settlements SET amount_cents = CAST(ROUND(COALESCE(amount, 0) * 100.0) AS INTEGER) WHERE amount_cents = 0 AND ABS(COALESCE(amount, 0)) > 0`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS trip_departed_tab_participants (
			trip_id TEXT NOT NULL,
			participant_key TEXT NOT NULL,
			display_name TEXT NOT NULL,
			left_at DATETIME NOT NULL,
			PRIMARY KEY (trip_id, participant_key),
			FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
		)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_departed_tab_trip ON trip_departed_tab_participants(trip_id)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`CREATE VIRTUAL TABLE IF NOT EXISTS tab_expense_search USING fts5(
			trip_id UNINDEXED,
			expense_id UNINDEXED,
			body
		)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE trips ADD COLUMN ui_date_format TEXT NOT NULL DEFAULT 'dmy'`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	for _, stmt := range []string{
		`ALTER TABLE lodging_entries ADD COLUMN image_path TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE vehicle_rentals ADD COLUMN attachment_path TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE flight_entries ADD COLUMN image_path TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE itinerary_items ADD COLUMN image_path TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE itinerary_items ADD COLUMN est_cost_cents INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE lodging_entries ADD COLUMN cost_cents INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE vehicle_rentals ADD COLUMN cost_cents INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE vehicle_rentals ADD COLUMN insurance_cost_cents INTEGER NOT NULL DEFAULT 0`,
		`ALTER TABLE flight_entries ADD COLUMN cost_cents INTEGER NOT NULL DEFAULT 0`,
	} {
		if _, err = db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return nil, err
		}
	}
	if _, err = db.Exec(`UPDATE itinerary_items SET est_cost_cents = CAST(ROUND(COALESCE(est_cost, 0) * 100.0) AS INTEGER) WHERE est_cost_cents = 0 AND ABS(COALESCE(est_cost, 0)) > 0`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`UPDATE lodging_entries SET cost_cents = CAST(ROUND(COALESCE(cost, 0) * 100.0) AS INTEGER) WHERE cost_cents = 0 AND ABS(COALESCE(cost, 0)) > 0`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`UPDATE vehicle_rentals SET cost_cents = CAST(ROUND(COALESCE(cost, 0) * 100.0) AS INTEGER) WHERE cost_cents = 0 AND ABS(COALESCE(cost, 0)) > 0`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`UPDATE vehicle_rentals SET insurance_cost_cents = CAST(ROUND(COALESCE(insurance_cost, 0) * 100.0) AS INTEGER) WHERE insurance_cost_cents = 0 AND ABS(COALESCE(insurance_cost, 0)) > 0`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`UPDATE flight_entries SET cost_cents = CAST(ROUND(COALESCE(cost, 0) * 100.0) AS INTEGER) WHERE cost_cents = 0 AND ABS(COALESCE(cost, 0)) > 0`); err != nil {
		return nil, err
	}
	for _, stmt := range []string{
		`ALTER TABLE lodging_entries ADD COLUMN updated_at DATETIME NOT NULL DEFAULT ''`,
		`ALTER TABLE vehicle_rentals ADD COLUMN updated_at DATETIME NOT NULL DEFAULT ''`,
		`ALTER TABLE flight_entries ADD COLUMN updated_at DATETIME NOT NULL DEFAULT ''`,
	} {
		if _, err = db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return nil, err
		}
	}
	if _, err = db.Exec(`UPDATE lodging_entries SET updated_at = created_at WHERE TRIM(COALESCE(updated_at, '')) = ''`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`UPDATE vehicle_rentals SET updated_at = created_at WHERE TRIM(COALESCE(updated_at, '')) = ''`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`UPDATE flight_entries SET updated_at = created_at WHERE TRIM(COALESCE(updated_at, '')) = ''`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE flight_entries ADD COLUMN booking_status TEXT NOT NULL DEFAULT 'to_be_done'`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE lodging_entries ADD COLUMN booking_status TEXT NOT NULL DEFAULT 'to_be_done'`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE vehicle_rentals ADD COLUMN booking_status TEXT NOT NULL DEFAULT 'to_be_done'`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE flight_entries ADD COLUMN trip_bookings_checklist_item_id TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE flight_entries ADD COLUMN trip_bookings_checklist_dismissed INTEGER NOT NULL DEFAULT 0`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS trip_documents (
			id TEXT PRIMARY KEY,
			trip_id TEXT NOT NULL,
			section TEXT NOT NULL DEFAULT '',
			category TEXT NOT NULL DEFAULT '',
			item_name TEXT NOT NULL DEFAULT '',
			file_name TEXT NOT NULL DEFAULT '',
			file_path TEXT NOT NULL DEFAULT '',
			file_size INTEGER NOT NULL DEFAULT 0,
			uploaded_at DATETIME NOT NULL,
			FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
		)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_trip_documents_trip ON trip_documents(trip_id, uploaded_at DESC)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE trip_documents ADD COLUMN display_name TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE users ADD COLUMN is_admin INTEGER NOT NULL DEFAULT 0`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	// Promote the earliest-created account if no administrator exists (existing databases pre-roles).
	if _, err = db.Exec(`
		UPDATE users SET is_admin = 1
		WHERE id = (SELECT id FROM users ORDER BY datetime(created_at) ASC LIMIT 1)
		  AND (SELECT COUNT(*) FROM users WHERE is_admin = 1) = 0`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE expenses ADD COLUMN due_at TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE checklist_items ADD COLUMN due_at TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS app_notifications (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			trip_id TEXT NOT NULL,
			title TEXT NOT NULL,
			body TEXT NOT NULL DEFAULT '',
			href TEXT NOT NULL DEFAULT '',
			kind TEXT NOT NULL DEFAULT '',
			dedupe_key TEXT NOT NULL,
			read_at TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			UNIQUE(user_id, dedupe_key),
			FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
			FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
		)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_app_notifications_user ON app_notifications(user_id, created_at DESC)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_app_notifications_unread ON app_notifications(user_id, read_at)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS itinerary_custom_reminders (
			id TEXT PRIMARY KEY,
			trip_id TEXT NOT NULL,
			itinerary_item_id TEXT NOT NULL,
			minutes_before_start INTEGER NOT NULL,
			label TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			UNIQUE(itinerary_item_id, minutes_before_start, label),
			FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE,
			FOREIGN KEY (itinerary_item_id) REFERENCES itinerary_items(id) ON DELETE CASCADE
		)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_itin_custom_reminders_item ON itinerary_custom_reminders(itinerary_item_id)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS trip_calendar_feed_tokens (
			trip_id TEXT PRIMARY KEY,
			token_hash TEXT NOT NULL,
			created_by_user_id TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
		)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE checklist_items ADD COLUMN archived INTEGER NOT NULL DEFAULT 0`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE checklist_items ADD COLUMN trashed INTEGER NOT NULL DEFAULT 0`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE checklist_items ADD COLUMN updated_at DATETIME`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`UPDATE checklist_items SET updated_at = created_at WHERE updated_at IS NULL`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS trip_notes (
			id TEXT PRIMARY KEY,
			trip_id TEXT NOT NULL,
			title TEXT NOT NULL DEFAULT '',
			body TEXT NOT NULL DEFAULT '',
			color TEXT NOT NULL DEFAULT '',
			pinned INTEGER NOT NULL DEFAULT 0,
			archived INTEGER NOT NULL DEFAULT 0,
			trashed INTEGER NOT NULL DEFAULT 0,
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL,
			FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
		)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_trip_notes_trip_updated ON trip_notes(trip_id, pinned DESC, updated_at DESC)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE trip_notes ADD COLUMN due_at TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS trip_checklist_category_pins (
			trip_id TEXT NOT NULL,
			category TEXT NOT NULL,
			PRIMARY KEY (trip_id, category),
			FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
		)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE itinerary_items ADD COLUMN item_kind TEXT NOT NULL DEFAULT 'stop'`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE itinerary_items ADD COLUMN commute_from_item_id TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE itinerary_items ADD COLUMN commute_to_item_id TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE itinerary_items ADD COLUMN transport_mode TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE itinerary_items ADD COLUMN sort_order INTEGER NOT NULL DEFAULT 0`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if err := backfillItinerarySortOrderIfNeeded(db); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE trips ADD COLUMN setup_complete INTEGER NOT NULL DEFAULT 0`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE itinerary_items ADD COLUMN google_place_id TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE itinerary_items ADD COLUMN venue_hours_json TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE itinerary_items ADD COLUMN time_needed TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE itinerary_items ADD COLUMN commute_end_day_offset INTEGER NOT NULL DEFAULT 0`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS global_keep_notes (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			title TEXT NOT NULL DEFAULT '',
			body TEXT NOT NULL DEFAULT '',
			color TEXT NOT NULL DEFAULT '',
			due_at TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_global_keep_notes_user ON global_keep_notes(user_id, updated_at DESC)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE global_keep_notes ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE global_keep_notes ADD COLUMN archived INTEGER NOT NULL DEFAULT 0`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE global_keep_notes ADD COLUMN trashed INTEGER NOT NULL DEFAULT 0`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS global_checklist_templates (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			category TEXT NOT NULL DEFAULT '',
			due_at TEXT NOT NULL DEFAULT '',
			created_at DATETIME NOT NULL,
			updated_at DATETIME NOT NULL
		)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_global_checklist_templates_user ON global_checklist_templates(user_id, updated_at DESC)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE global_checklist_templates ADD COLUMN pinned INTEGER NOT NULL DEFAULT 0`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE global_checklist_templates ADD COLUMN archived INTEGER NOT NULL DEFAULT 0`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE global_checklist_templates ADD COLUMN trashed INTEGER NOT NULL DEFAULT 0`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS global_checklist_template_lines (
			id TEXT PRIMARY KEY,
			template_id TEXT NOT NULL,
			sort_order INTEGER NOT NULL DEFAULT 0,
			text TEXT NOT NULL DEFAULT '',
			FOREIGN KEY (template_id) REFERENCES global_checklist_templates(id) ON DELETE CASCADE
		)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_global_checklist_lines_tpl ON global_checklist_template_lines(template_id, sort_order)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS trip_global_keep_imports (
			trip_id TEXT NOT NULL,
			kind TEXT NOT NULL CHECK(kind IN ('note','checklist')),
			global_id TEXT NOT NULL,
			imported_at DATETIME NOT NULL,
			PRIMARY KEY (trip_id, kind, global_id),
			FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
		)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_trip_global_keep_imports_global ON trip_global_keep_imports(kind, global_id)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS airports (
			iata_code TEXT NOT NULL PRIMARY KEY,
			icao_code TEXT NOT NULL DEFAULT '',
			name TEXT NOT NULL DEFAULT '',
			city TEXT NOT NULL DEFAULT '',
			country TEXT NOT NULL DEFAULT '',
			country_code TEXT NOT NULL DEFAULT '',
			timezone TEXT NOT NULL DEFAULT '',
			last_updated DATETIME NOT NULL
		)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_airports_name ON airports(name)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_airports_city ON airports(city)`); err != nil {
		return nil, err
	}
	for _, stmt := range []string{
		`ALTER TABLE app_settings ADD COLUMN amadeus_client_id TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE app_settings ADD COLUMN amadeus_client_secret TEXT NOT NULL DEFAULT ''`,
	} {
		if _, err = db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return nil, err
		}
	}
	for _, stmt := range []string{
		`ALTER TABLE flight_entries ADD COLUMN depart_airport_iata TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE flight_entries ADD COLUMN arrive_airport_iata TEXT NOT NULL DEFAULT ''`,
	} {
		if _, err = db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return nil, err
		}
	}
	for _, stmt := range []string{
		`ALTER TABLE app_settings ADD COLUMN airlabs_api_key TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE airports ADD COLUMN icao_code TEXT NOT NULL DEFAULT ''`,
		`ALTER TABLE airports ADD COLUMN country_code TEXT NOT NULL DEFAULT ''`,
	} {
		if _, err = db.Exec(stmt); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
			return nil, err
		}
	}
	if _, err = db.Exec(`UPDATE airports SET country_code = country WHERE TRIM(COALESCE(country_code, '')) = '' AND TRIM(COALESCE(country, '')) != ''`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`ALTER TABLE app_settings ADD COLUMN openweather_api_key TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return nil, err
	}
	if _, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS weather_cache (
			id TEXT NOT NULL PRIMARY KEY,
			trip_id TEXT NOT NULL,
			itinerary_item_id TEXT NOT NULL,
			cache_date TEXT NOT NULL,
			latitude REAL NOT NULL,
			longitude REAL NOT NULL,
			payload_json TEXT NOT NULL,
			fetched_at DATETIME NOT NULL,
			UNIQUE(itinerary_item_id, cache_date),
			FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
		)`); err != nil {
		return nil, err
	}
	if _, err = db.Exec(`CREATE INDEX IF NOT EXISTS idx_weather_cache_trip ON weather_cache(trip_id)`); err != nil {
		return nil, err
	}
	return db, nil
}

// backfillItinerarySortOrderIfNeeded assigns sort_order spacing for legacy rows (all zero) once.
func backfillItinerarySortOrderIfNeeded(db *sql.DB) error {
	var maxSo sql.NullInt64
	if err := db.QueryRow(`SELECT MAX(sort_order) FROM itinerary_items`).Scan(&maxSo); err != nil {
		return err
	}
	if maxSo.Valid && maxSo.Int64 > 0 {
		return nil
	}
	rows, err := db.Query(`SELECT id, trip_id, day_number FROM itinerary_items ORDER BY trip_id, day_number, created_at ASC, id ASC`)
	if err != nil {
		return err
	}
	defer rows.Close()
	type dayKey struct {
		tripID string
		day    int
	}
	var ids []string
	var cur dayKey
	first := true
	flush := func() error {
		for i, id := range ids {
			if _, err := db.Exec(`UPDATE itinerary_items SET sort_order = ? WHERE id = ?`, (i+1)*100000, id); err != nil {
				return err
			}
		}
		ids = nil
		return nil
	}
	for rows.Next() {
		var id, tripID string
		var dayNum int
		if err := rows.Scan(&id, &tripID, &dayNum); err != nil {
			return err
		}
		k := dayKey{tripID: tripID, day: dayNum}
		if !first && k != cur {
			if err := flush(); err != nil {
				return err
			}
		}
		first = false
		cur = k
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	return flush()
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
			distance_unit TEXT NOT NULL DEFAULT '',
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
