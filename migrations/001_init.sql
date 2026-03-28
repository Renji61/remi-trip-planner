CREATE TABLE IF NOT EXISTS trips (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  description TEXT NOT NULL DEFAULT '',
  start_date TEXT NOT NULL DEFAULT '',
  end_date TEXT NOT NULL DEFAULT '',
  cover_image_url TEXT NOT NULL DEFAULT '',
  currency_name TEXT NOT NULL DEFAULT 'USD',
  currency_symbol TEXT NOT NULL DEFAULT '$',
  home_map_latitude REAL NOT NULL DEFAULT 0,
  home_map_longitude REAL NOT NULL DEFAULT 0,
  is_archived BOOLEAN NOT NULL DEFAULT FALSE,
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL
);

CREATE TABLE IF NOT EXISTS itinerary_items (
  id TEXT PRIMARY KEY,
  trip_id TEXT NOT NULL,
  day_number INTEGER NOT NULL DEFAULT 1,
  title TEXT NOT NULL,
  notes TEXT NOT NULL DEFAULT '',
  location TEXT NOT NULL DEFAULT '',
  latitude REAL NOT NULL DEFAULT 0,
  longitude REAL NOT NULL DEFAULT 0,
  est_cost REAL NOT NULL DEFAULT 0,
  start_time TEXT NOT NULL DEFAULT '',
  end_time TEXT NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL,
  updated_at DATETIME NOT NULL,
  FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS expenses (
  id TEXT PRIMARY KEY,
  trip_id TEXT NOT NULL,
  category TEXT NOT NULL DEFAULT 'general',
  amount REAL NOT NULL DEFAULT 0,
  notes TEXT NOT NULL DEFAULT '',
  spent_on TEXT NOT NULL DEFAULT '',
  payment_method TEXT NOT NULL DEFAULT 'Cash',
  lodging_id TEXT NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL,
  FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS checklist_items (
  id TEXT PRIMARY KEY,
  trip_id TEXT NOT NULL,
  category TEXT NOT NULL DEFAULT 'Packing List',
  text TEXT NOT NULL,
  done BOOLEAN NOT NULL DEFAULT FALSE,
  created_at DATETIME NOT NULL,
  FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS change_log (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  trip_id TEXT NOT NULL,
  entity TEXT NOT NULL,
  entity_id TEXT NOT NULL,
  operation TEXT NOT NULL,
  changed_at DATETIME NOT NULL,
  payload TEXT NOT NULL DEFAULT '{}',
  FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS app_settings (
  id INTEGER PRIMARY KEY CHECK (id = 1),
  app_title TEXT NOT NULL DEFAULT 'REMI Trip Planner',
  default_currency_name TEXT NOT NULL DEFAULT 'USD',
  default_currency_symbol TEXT NOT NULL DEFAULT '$',
  map_default_place_label TEXT NOT NULL DEFAULT 'Tokyo',
  map_default_latitude REAL NOT NULL DEFAULT 35.6762,
  map_default_longitude REAL NOT NULL DEFAULT 139.6503,
  map_default_zoom INTEGER NOT NULL DEFAULT 6,
  enable_location_lookup BOOLEAN NOT NULL DEFAULT TRUE,
  updated_at DATETIME NOT NULL
);

-- Omit map_default_place_label so this INSERT still works on legacy app_settings rows before db.go adds that column; new installs get DEFAULT 'Tokyo' from CREATE TABLE.
INSERT OR IGNORE INTO app_settings
  (id, app_title, default_currency_name, default_currency_symbol, map_default_latitude, map_default_longitude, map_default_zoom, enable_location_lookup, updated_at)
VALUES
  (1, 'REMI Trip Planner', 'USD', '$', 35.6762, 139.6503, 6, TRUE, CURRENT_TIMESTAMP);

CREATE TABLE IF NOT EXISTS lodging_entries (
  id TEXT PRIMARY KEY,
  trip_id TEXT NOT NULL,
  name TEXT NOT NULL,
  address TEXT NOT NULL DEFAULT '',
  check_in_at TEXT NOT NULL DEFAULT '',
  check_out_at TEXT NOT NULL DEFAULT '',
  booking_confirmation TEXT NOT NULL DEFAULT '',
  cost REAL NOT NULL DEFAULT 0,
  notes TEXT NOT NULL DEFAULT '',
  attachment_path TEXT NOT NULL DEFAULT '',
  check_in_itinerary_id TEXT NOT NULL DEFAULT '',
  check_out_itinerary_id TEXT NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL,
  FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS vehicle_rentals (
  id TEXT PRIMARY KEY,
  trip_id TEXT NOT NULL,
  pick_up_location TEXT NOT NULL DEFAULT '',
  vehicle_detail TEXT NOT NULL DEFAULT '',
  pick_up_at TEXT NOT NULL DEFAULT '',
  drop_off_at TEXT NOT NULL DEFAULT '',
  booking_confirmation TEXT NOT NULL DEFAULT '',
  notes TEXT NOT NULL DEFAULT '',
  vehicle_image_path TEXT NOT NULL DEFAULT '',
  cost REAL NOT NULL DEFAULT 0,
  insurance_cost REAL NOT NULL DEFAULT 0,
  pay_at_pick_up BOOLEAN NOT NULL DEFAULT FALSE,
  pick_up_itinerary_id TEXT NOT NULL DEFAULT '',
  drop_off_itinerary_id TEXT NOT NULL DEFAULT '',
  rental_expense_id TEXT NOT NULL DEFAULT '',
  insurance_expense_id TEXT NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL,
  FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS flight_entries (
  id TEXT PRIMARY KEY,
  trip_id TEXT NOT NULL,
  flight_name TEXT NOT NULL DEFAULT '',
  flight_number TEXT NOT NULL DEFAULT '',
  depart_airport TEXT NOT NULL DEFAULT '',
  arrive_airport TEXT NOT NULL DEFAULT '',
  depart_at TEXT NOT NULL DEFAULT '',
  arrive_at TEXT NOT NULL DEFAULT '',
  booking_confirmation TEXT NOT NULL DEFAULT '',
  notes TEXT NOT NULL DEFAULT '',
  document_path TEXT NOT NULL DEFAULT '',
  cost REAL NOT NULL DEFAULT 0,
  depart_itinerary_id TEXT NOT NULL DEFAULT '',
  arrive_itinerary_id TEXT NOT NULL DEFAULT '',
  expense_id TEXT NOT NULL DEFAULT '',
  created_at DATETIME NOT NULL,
  FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
);

CREATE TABLE IF NOT EXISTS trip_day_labels (
  trip_id TEXT NOT NULL,
  day_number INTEGER NOT NULL,
  label TEXT NOT NULL DEFAULT '',
  updated_at DATETIME NOT NULL,
  PRIMARY KEY (trip_id, day_number),
  FOREIGN KEY (trip_id) REFERENCES trips(id) ON DELETE CASCADE
);
