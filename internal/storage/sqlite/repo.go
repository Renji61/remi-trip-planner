package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"remi-trip-planner/internal/trips"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) *Repository {
	return &Repository{db: db}
}

func sqliteBool(b bool) int {
	if b {
		return 1
	}
	return 0
}

const tripSelectCols = `id, name, description, start_date, end_date, cover_image_url, currency_name, currency_symbol, home_map_latitude, home_map_longitude, is_archived, owner_user_id,
		ui_show_stay, ui_show_vehicle, ui_show_flights, ui_show_spends, ui_show_itinerary, ui_show_checklist,
		ui_itinerary_expand, ui_spends_expand, ui_tab_expand, ui_time_format, ui_date_format,
		ui_label_stay, ui_label_vehicle, ui_label_flights, ui_label_spends, ui_label_group_expenses,
		ui_main_section_order, ui_sidebar_widget_order,
		ui_main_section_hidden, ui_sidebar_widget_hidden,
		ui_show_custom_links, ui_custom_sidebar_links,
		created_at, updated_at, budget_cap, ui_show_the_tab,
		tab_default_split_mode, tab_default_split_json, distance_unit`

func scanTrip(scan func(dest ...any) error) (trips.Trip, error) {
	var t trips.Trip
	var ss, sv, sf, sp, sit, sck, scl, stt int
	err := scan(
		&t.ID, &t.Name, &t.Description, &t.StartDate, &t.EndDate, &t.CoverImage, &t.CurrencyName, &t.CurrencySymbol, &t.HomeMapLatitude, &t.HomeMapLongitude, &t.IsArchived, &t.OwnerUserID,
		&ss, &sv, &sf, &sp, &sit, &sck,
		&t.UIItineraryExpand, &t.UISpendsExpand, &t.UITabExpand, &t.UITimeFormat, &t.UIDateFormat,
		&t.UILabelStay, &t.UILabelVehicle, &t.UILabelFlights, &t.UILabelSpends, &t.UILabelGroupExpenses,
		&t.UIMainSectionOrder, &t.UISidebarWidgetOrder,
		&t.UIMainSectionHidden, &t.UISidebarWidgetHidden,
		&scl, &t.UICustomSidebarLinks,
		&t.CreatedAt, &t.UpdatedAt,
		&t.BudgetCap, &stt,
		&t.TabDefaultSplitMode, &t.TabDefaultSplitJSON,
		&t.DistanceUnit,
	)
	if err != nil {
		return t, err
	}
	t.UIShowStay = ss != 0
	t.UIShowVehicle = sv != 0
	t.UIShowFlights = sf != 0
	t.UIShowSpends = sp != 0
	t.UIShowItinerary = sit != 0
	t.UIShowChecklist = sck != 0
	t.UIShowCustomLinks = scl != 0
	t.UIShowTheTab = stt != 0
	t.UIDateFormat = trips.NormalizeUIDateFormat(t.UIDateFormat)
	return t, nil
}

func (r *Repository) CreateTrip(ctx context.Context, t trips.Trip) (string, error) {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO trips (
			id, name, description, start_date, end_date, cover_image_url, currency_name, currency_symbol, home_map_latitude, home_map_longitude, is_archived, owner_user_id,
			ui_show_stay, ui_show_vehicle, ui_show_flights, ui_show_spends, ui_show_itinerary, ui_show_checklist,
			ui_itinerary_expand, ui_spends_expand, ui_tab_expand, ui_time_format, ui_date_format,
			ui_label_stay, ui_label_vehicle, ui_label_flights, ui_label_spends, ui_label_group_expenses,
			ui_main_section_order, ui_sidebar_widget_order,
			ui_main_section_hidden, ui_sidebar_widget_hidden,
			ui_show_custom_links, ui_custom_sidebar_links,
			budget_cap, ui_show_the_tab, tab_default_split_mode, tab_default_split_json, distance_unit,
			created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, 1, 1, 1, 1, 1, 'first', 'first', 'first', '12h', ?, '', '', '', '', '', '', '', '', '', 1, '', ?, 1, '', '', ?, ?, ?)`,
		t.ID, t.Name, t.Description, t.StartDate, t.EndDate, t.CoverImage, t.CurrencyName, t.CurrencySymbol, t.HomeMapLatitude, t.HomeMapLongitude, t.IsArchived, t.OwnerUserID,
		trips.NormalizeUIDateFormat(t.UIDateFormat),
		t.BudgetCap, t.DistanceUnit, now, now,
	)
	if err == nil {
		_ = r.logChange(ctx, t.ID, "trip", t.ID, "create", map[string]any{"name": t.Name})
	}
	return t.ID, err
}

func (r *Repository) ListTrips(ctx context.Context) ([]trips.Trip, error) {
	rows, err := r.db.QueryContext(ctx, `SELECT `+tripSelectCols+` FROM trips ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []trips.Trip{}
	for rows.Next() {
		t, err := scanTrip(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repository) GetTrip(ctx context.Context, tripID string) (trips.Trip, error) {
	row := r.db.QueryRowContext(ctx, `SELECT `+tripSelectCols+` FROM trips WHERE id = ?`, tripID)
	return scanTrip(row.Scan)
}

func (r *Repository) UpdateTrip(ctx context.Context, t trips.Trip) error {
	t.UIDateFormat = trips.NormalizeUIDateFormat(t.UIDateFormat)
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		UPDATE trips
		SET name = ?, description = ?, start_date = ?, end_date = ?, cover_image_url = ?, currency_name = ?, currency_symbol = ?,
		    home_map_latitude = ?, home_map_longitude = ?,
		    budget_cap = ?,
		    ui_show_stay = ?, ui_show_vehicle = ?, ui_show_flights = ?, ui_show_spends = ?, ui_show_itinerary = ?, ui_show_checklist = ?,
		    ui_show_the_tab = ?,
		    ui_itinerary_expand = ?, ui_spends_expand = ?, ui_tab_expand = ?, ui_time_format = ?, ui_date_format = ?,
		    ui_label_stay = ?, ui_label_vehicle = ?, ui_label_flights = ?, ui_label_spends = ?, ui_label_group_expenses = ?,
		    ui_main_section_order = ?, ui_sidebar_widget_order = ?,
		    ui_main_section_hidden = ?, ui_sidebar_widget_hidden = ?,
		    ui_show_custom_links = ?, ui_custom_sidebar_links = ?,
		    tab_default_split_mode = ?, tab_default_split_json = ?,
		    distance_unit = ?,
		    updated_at = ?
		WHERE id = ?`,
		t.Name, t.Description, t.StartDate, t.EndDate, t.CoverImage, t.CurrencyName, t.CurrencySymbol, t.HomeMapLatitude, t.HomeMapLongitude,
		t.BudgetCap,
		sqliteBool(t.UIShowStay), sqliteBool(t.UIShowVehicle), sqliteBool(t.UIShowFlights), sqliteBool(t.UIShowSpends),
		sqliteBool(t.UIShowItinerary), sqliteBool(t.UIShowChecklist),
		sqliteBool(t.UIShowTheTab),
		t.UIItineraryExpand, t.UISpendsExpand, t.UITabExpand, t.UITimeFormat, t.UIDateFormat,
		t.UILabelStay, t.UILabelVehicle, t.UILabelFlights, t.UILabelSpends, t.UILabelGroupExpenses,
		t.UIMainSectionOrder, t.UISidebarWidgetOrder,
		t.UIMainSectionHidden, t.UISidebarWidgetHidden,
		sqliteBool(t.UIShowCustomLinks), t.UICustomSidebarLinks,
		t.TabDefaultSplitMode, t.TabDefaultSplitJSON,
		t.DistanceUnit,
		now, t.ID,
	)
	if err == nil {
		_ = r.logChange(ctx, t.ID, "trip", t.ID, "update", map[string]any{
			"name": t.Name, "start_date": t.StartDate, "end_date": t.EndDate, "cover_image_url": t.CoverImage, "currency_name": t.CurrencyName, "currency_symbol": t.CurrencySymbol,
		})
	}
	return err
}

func (r *Repository) ArchiveTrip(ctx context.Context, tripID string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		UPDATE trips SET is_archived = TRUE, updated_at = ? WHERE id = ?`,
		now, tripID,
	)
	if err == nil {
		_ = r.logChange(ctx, tripID, "trip", tripID, "archive", map[string]any{"is_archived": true})
	}
	return err
}

func (r *Repository) DeleteTrip(ctx context.Context, tripID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM trips WHERE id = ?`, tripID)
	if err == nil {
		_ = r.logChange(ctx, tripID, "trip", tripID, "delete", map[string]any{})
	}
	return err
}

func (r *Repository) AddItineraryItem(ctx context.Context, item trips.ItineraryItem) error {
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO itinerary_items
		(id, trip_id, day_number, title, notes, location, latitude, longitude, est_cost, start_time, end_time, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.TripID, item.DayNumber, item.Title, item.Notes, item.Location, item.Latitude, item.Longitude, item.EstCost, item.StartTime, item.EndTime, now, now,
	)
	if err == nil {
		_ = r.logChange(ctx, item.TripID, "itinerary_item", item.ID, "create", map[string]any{
			"title": item.Title, "day_number": item.DayNumber,
		})
	}
	return err
}

func (r *Repository) UpdateItineraryItem(ctx context.Context, item trips.ItineraryItem) (int64, error) {
	now := time.Now().UTC()
	res, err := r.db.ExecContext(ctx, `
		UPDATE itinerary_items
		SET day_number = ?, title = ?, notes = ?, location = ?, est_cost = ?, start_time = ?, end_time = ?, updated_at = ?
		WHERE id = ? AND trip_id = ?`,
		item.DayNumber, item.Title, item.Notes, item.Location, item.EstCost, item.StartTime, item.EndTime, now,
		item.ID, item.TripID,
	)
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if n > 0 {
		_ = r.logChange(ctx, item.TripID, "itinerary_item", item.ID, "update", map[string]any{
			"title": item.Title, "day_number": item.DayNumber,
		})
	}
	return n, nil
}

func (r *Repository) DeleteItineraryItem(ctx context.Context, tripID, itemID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM itinerary_items WHERE id = ? AND trip_id = ?`, itemID, tripID)
	if err == nil {
		_ = r.logChange(ctx, tripID, "itinerary_item", itemID, "delete", map[string]any{})
	}
	return err
}

func (r *Repository) ListItineraryItems(ctx context.Context, tripID string) ([]trips.ItineraryItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, trip_id, day_number, title, notes, location, latitude, longitude, est_cost, start_time, end_time, created_at, updated_at
		FROM itinerary_items WHERE trip_id = ?
		ORDER BY day_number ASC, created_at ASC`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []trips.ItineraryItem{}
	for rows.Next() {
		var i trips.ItineraryItem
		if err := rows.Scan(&i.ID, &i.TripID, &i.DayNumber, &i.Title, &i.Notes, &i.Location, &i.Latitude, &i.Longitude, &i.EstCost, &i.StartTime, &i.EndTime, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *Repository) ListAllItineraryItems(ctx context.Context) ([]trips.ItineraryItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, trip_id, day_number, title, notes, location, latitude, longitude, est_cost, start_time, end_time, created_at, updated_at
		FROM itinerary_items
		ORDER BY trip_id, day_number ASC, start_time ASC, created_at ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []trips.ItineraryItem{}
	for rows.Next() {
		var i trips.ItineraryItem
		if err := rows.Scan(&i.ID, &i.TripID, &i.DayNumber, &i.Title, &i.Notes, &i.Location, &i.Latitude, &i.Longitude, &i.EstCost, &i.StartTime, &i.EndTime, &i.CreatedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *Repository) AddExpense(ctx context.Context, e trips.Expense) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if strings.TrimSpace(e.PaymentMethod) == "" {
		e.PaymentMethod = "Cash"
	}
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO expenses (id, trip_id, category, amount, notes, spent_on, payment_method, lodging_id, from_tab, receipt_path, title, paid_by, split_mode, split_json, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.TripID, e.Category, e.Amount, e.Notes, e.SpentOn, e.PaymentMethod, e.LodgingID, sqliteBool(e.FromTab), e.ReceiptPath,
		e.Title, e.PaidBy, e.SplitMode, e.SplitJSON, now,
	)
	if err == nil {
		_ = r.logChange(ctx, e.TripID, "expense", e.ID, "create", map[string]any{"amount": e.Amount, "category": e.Category})
		_ = r.syncTabExpenseFTS(ctx, e)
	}
	return err
}

func (r *Repository) UpdateExpense(ctx context.Context, e trips.Expense) error {
	if strings.TrimSpace(e.PaymentMethod) == "" {
		e.PaymentMethod = "Cash"
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE expenses
		SET category = ?, amount = ?, notes = ?, spent_on = ?, payment_method = ?, lodging_id = ?, from_tab = ?, receipt_path = ?,
		    title = ?, paid_by = ?, split_mode = ?, split_json = ?
		WHERE id = ? AND trip_id = ?`,
		e.Category, e.Amount, e.Notes, e.SpentOn, e.PaymentMethod, e.LodgingID, sqliteBool(e.FromTab), e.ReceiptPath,
		e.Title, e.PaidBy, e.SplitMode, e.SplitJSON, e.ID, e.TripID,
	)
	if err == nil {
		_ = r.logChange(ctx, e.TripID, "expense", e.ID, "update", map[string]any{
			"amount": e.Amount, "category": e.Category,
		})
		_ = r.syncTabExpenseFTS(ctx, e)
	}
	return err
}

func (r *Repository) DeleteExpense(ctx context.Context, tripID, expenseID string) error {
	_, _ = r.db.ExecContext(ctx, `DELETE FROM tab_expense_search WHERE expense_id = ?`, expenseID)
	_, err := r.db.ExecContext(ctx, `DELETE FROM expenses WHERE id = ? AND trip_id = ?`, expenseID, tripID)
	if err == nil {
		_ = r.logChange(ctx, tripID, "expense", expenseID, "delete", map[string]any{})
	}
	return err
}

func (r *Repository) GetExpense(ctx context.Context, tripID, expenseID string) (trips.Expense, error) {
	var e trips.Expense
	var ft int
	err := r.db.QueryRowContext(ctx, `
		SELECT id, trip_id, category, amount, notes, spent_on, payment_method, lodging_id, from_tab, receipt_path, title, paid_by, split_mode, split_json, created_at
		FROM expenses WHERE id = ? AND trip_id = ?`, expenseID, tripID,
	).Scan(&e.ID, &e.TripID, &e.Category, &e.Amount, &e.Notes, &e.SpentOn, &e.PaymentMethod, &e.LodgingID, &ft, &e.ReceiptPath,
		&e.Title, &e.PaidBy, &e.SplitMode, &e.SplitJSON, &e.CreatedAt)
	e.FromTab = ft != 0
	return e, err
}

func (r *Repository) GetExpenseByLodgingID(ctx context.Context, tripID, lodgingID string) (trips.Expense, error) {
	var e trips.Expense
	var ft int
	err := r.db.QueryRowContext(ctx, `
		SELECT id, trip_id, category, amount, notes, spent_on, payment_method, lodging_id, from_tab, receipt_path, title, paid_by, split_mode, split_json, created_at
		FROM expenses WHERE trip_id = ? AND lodging_id = ?`, tripID, lodgingID,
	).Scan(&e.ID, &e.TripID, &e.Category, &e.Amount, &e.Notes, &e.SpentOn, &e.PaymentMethod, &e.LodgingID, &ft, &e.ReceiptPath,
		&e.Title, &e.PaidBy, &e.SplitMode, &e.SplitJSON, &e.CreatedAt)
	e.FromTab = ft != 0
	return e, err
}

func (r *Repository) DeleteExpenseByLodgingID(ctx context.Context, tripID, lodgingID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM expenses WHERE trip_id = ? AND lodging_id = ?`, tripID, lodgingID)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		_ = r.logChange(ctx, tripID, "expense", lodgingID, "delete", map[string]any{"by_lodging": true})
	}
	return nil
}

func (r *Repository) ListExpenses(ctx context.Context, tripID string) ([]trips.Expense, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, trip_id, category, amount, notes, spent_on, payment_method, lodging_id, from_tab, receipt_path, title, paid_by, split_mode, split_json, created_at
		FROM expenses WHERE trip_id = ? ORDER BY created_at DESC`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []trips.Expense{}
	for rows.Next() {
		var e trips.Expense
		var ft int
		if err := rows.Scan(&e.ID, &e.TripID, &e.Category, &e.Amount, &e.Notes, &e.SpentOn, &e.PaymentMethod, &e.LodgingID, &ft, &e.ReceiptPath,
			&e.Title, &e.PaidBy, &e.SplitMode, &e.SplitJSON, &e.CreatedAt); err != nil {
			return nil, err
		}
		e.FromTab = ft != 0
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) SumExpensesByTrip(ctx context.Context) (map[string]float64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT trip_id, COALESCE(SUM(amount), 0) FROM expenses GROUP BY trip_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]float64)
	for rows.Next() {
		var tripID string
		var sum float64
		if err := rows.Scan(&tripID, &sum); err != nil {
			return nil, err
		}
		out[tripID] = sum
	}
	return out, rows.Err()
}

func (r *Repository) AddChecklistItem(ctx context.Context, item trips.ChecklistItem) error {
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO checklist_items (id, trip_id, category, text, done, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		item.ID, item.TripID, item.Category, item.Text, item.Done, now,
	)
	if err == nil {
		_ = r.logChange(ctx, item.TripID, "checklist_item", item.ID, "create", map[string]any{"text": item.Text, "category": item.Category})
	}
	return err
}

func (r *Repository) GetChecklistItem(ctx context.Context, itemID string) (trips.ChecklistItem, error) {
	var i trips.ChecklistItem
	err := r.db.QueryRowContext(ctx, `
		SELECT id, trip_id, category, text, done, created_at
		FROM checklist_items WHERE id = ?`, itemID).
		Scan(&i.ID, &i.TripID, &i.Category, &i.Text, &i.Done, &i.CreatedAt)
	return i, err
}

func (r *Repository) UpdateChecklistItem(ctx context.Context, item trips.ChecklistItem) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE checklist_items
		SET category = ?, text = ?
		WHERE id = ?`,
		item.Category, item.Text, item.ID,
	)
	if err == nil {
		_ = r.logChange(ctx, item.TripID, "checklist_item", item.ID, "update", map[string]any{
			"text": item.Text, "category": item.Category,
		})
	}
	return err
}

func (r *Repository) DeleteChecklistItem(ctx context.Context, itemID string) error {
	var tripID string
	_ = r.db.QueryRowContext(ctx, `SELECT trip_id FROM checklist_items WHERE id = ?`, itemID).Scan(&tripID)
	_, err := r.db.ExecContext(ctx, `DELETE FROM checklist_items WHERE id = ?`, itemID)
	if err == nil && tripID != "" {
		_ = r.logChange(ctx, tripID, "checklist_item", itemID, "delete", map[string]any{})
	}
	return err
}

func (r *Repository) ToggleChecklistItem(ctx context.Context, itemID string, done bool) error {
	var tripID string
	_ = r.db.QueryRowContext(ctx, `SELECT trip_id FROM checklist_items WHERE id = ?`, itemID).Scan(&tripID)
	res, err := r.db.ExecContext(ctx, `
		UPDATE checklist_items
		SET done = ?
		WHERE id = ? AND trip_id IN (SELECT id FROM trips WHERE is_archived = FALSE)`,
		done, itemID,
	)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return nil
	}
	if tripID != "" {
		_ = r.logChange(ctx, tripID, "checklist_item", itemID, "update", map[string]any{"done": done})
	}
	return nil
}

func (r *Repository) ListChecklistItems(ctx context.Context, tripID string) ([]trips.ChecklistItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, trip_id, category, text, done, created_at
		FROM checklist_items WHERE trip_id = ? ORDER BY created_at DESC`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []trips.ChecklistItem{}
	for rows.Next() {
		var i trips.ChecklistItem
		if err := rows.Scan(&i.ID, &i.TripID, &i.Category, &i.Text, &i.Done, &i.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *Repository) AddLodging(ctx context.Context, item trips.Lodging) error {
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO lodging_entries
		(id, trip_id, name, address, check_in_at, check_out_at, booking_confirmation, cost, notes, attachment_path, check_in_itinerary_id, check_out_itinerary_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.TripID, item.Name, item.Address, item.CheckInAt, item.CheckOutAt, item.BookingConfirmation, item.Cost, item.Notes, item.AttachmentPath,
		item.CheckInItineraryID, item.CheckOutItineraryID, now,
	)
	return err
}

func (r *Repository) GetLodging(ctx context.Context, tripID, lodgingID string) (trips.Lodging, error) {
	var l trips.Lodging
	err := r.db.QueryRowContext(ctx, `
		SELECT id, trip_id, name, address, check_in_at, check_out_at, booking_confirmation, cost, notes, attachment_path, check_in_itinerary_id, check_out_itinerary_id, created_at
		FROM lodging_entries WHERE id = ? AND trip_id = ?`,
		lodgingID, tripID,
	).Scan(&l.ID, &l.TripID, &l.Name, &l.Address, &l.CheckInAt, &l.CheckOutAt, &l.BookingConfirmation, &l.Cost, &l.Notes, &l.AttachmentPath,
		&l.CheckInItineraryID, &l.CheckOutItineraryID, &l.CreatedAt)
	return l, err
}

func (r *Repository) UpdateLodging(ctx context.Context, item trips.Lodging) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE lodging_entries
		SET name = ?, address = ?, check_in_at = ?, check_out_at = ?, booking_confirmation = ?, cost = ?, notes = ?, attachment_path = ?, check_in_itinerary_id = ?, check_out_itinerary_id = ?
		WHERE id = ? AND trip_id = ?`,
		item.Name, item.Address, item.CheckInAt, item.CheckOutAt, item.BookingConfirmation, item.Cost, item.Notes, item.AttachmentPath,
		item.CheckInItineraryID, item.CheckOutItineraryID,
		item.ID, item.TripID,
	)
	return err
}

func (r *Repository) DeleteLodging(ctx context.Context, tripID, lodgingID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM lodging_entries WHERE id = ? AND trip_id = ?`, lodgingID, tripID)
	return err
}

func (r *Repository) ListLodgings(ctx context.Context, tripID string) ([]trips.Lodging, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, trip_id, name, address, check_in_at, check_out_at, booking_confirmation, cost, notes, attachment_path, check_in_itinerary_id, check_out_itinerary_id, created_at
		FROM lodging_entries
		WHERE trip_id = ?
		ORDER BY check_in_at ASC, created_at ASC`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []trips.Lodging{}
	for rows.Next() {
		var l trips.Lodging
		if err := rows.Scan(&l.ID, &l.TripID, &l.Name, &l.Address, &l.CheckInAt, &l.CheckOutAt, &l.BookingConfirmation, &l.Cost, &l.Notes, &l.AttachmentPath,
			&l.CheckInItineraryID, &l.CheckOutItineraryID, &l.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *Repository) AddVehicleRental(ctx context.Context, item trips.VehicleRental) error {
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO vehicle_rentals
		(id, trip_id, pick_up_location, drop_off_location, vehicle_detail, pick_up_at, drop_off_at, booking_confirmation, notes, vehicle_image_path, cost, insurance_cost, pay_at_pick_up, pick_up_itinerary_id, drop_off_itinerary_id, rental_expense_id, insurance_expense_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.TripID, item.PickUpLocation, item.DropOffLocation, item.VehicleDetail, item.PickUpAt, item.DropOffAt, item.BookingConfirmation, item.Notes, item.VehicleImagePath, item.Cost, item.InsuranceCost, item.PayAtPickUp,
		item.PickUpItineraryID, item.DropOffItineraryID, item.RentalExpenseID, item.InsuranceExpenseID, now,
	)
	return err
}

func (r *Repository) GetVehicleRental(ctx context.Context, tripID, rentalID string) (trips.VehicleRental, error) {
	var v trips.VehicleRental
	err := r.db.QueryRowContext(ctx, `
		SELECT id, trip_id, pick_up_location, drop_off_location, vehicle_detail, pick_up_at, drop_off_at, booking_confirmation, notes, vehicle_image_path, cost, insurance_cost, pay_at_pick_up, pick_up_itinerary_id, drop_off_itinerary_id, rental_expense_id, insurance_expense_id, created_at
		FROM vehicle_rentals WHERE id = ? AND trip_id = ?`,
		rentalID, tripID,
	).Scan(&v.ID, &v.TripID, &v.PickUpLocation, &v.DropOffLocation, &v.VehicleDetail, &v.PickUpAt, &v.DropOffAt, &v.BookingConfirmation, &v.Notes, &v.VehicleImagePath, &v.Cost, &v.InsuranceCost, &v.PayAtPickUp,
		&v.PickUpItineraryID, &v.DropOffItineraryID, &v.RentalExpenseID, &v.InsuranceExpenseID, &v.CreatedAt)
	return v, err
}

func (r *Repository) UpdateVehicleRental(ctx context.Context, item trips.VehicleRental) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE vehicle_rentals
		SET pick_up_location = ?, drop_off_location = ?, vehicle_detail = ?, pick_up_at = ?, drop_off_at = ?, booking_confirmation = ?, notes = ?, vehicle_image_path = ?, cost = ?, insurance_cost = ?, pay_at_pick_up = ?, pick_up_itinerary_id = ?, drop_off_itinerary_id = ?, rental_expense_id = ?, insurance_expense_id = ?
		WHERE id = ? AND trip_id = ?`,
		item.PickUpLocation, item.DropOffLocation, item.VehicleDetail, item.PickUpAt, item.DropOffAt, item.BookingConfirmation, item.Notes, item.VehicleImagePath, item.Cost, item.InsuranceCost, item.PayAtPickUp,
		item.PickUpItineraryID, item.DropOffItineraryID, item.RentalExpenseID, item.InsuranceExpenseID,
		item.ID, item.TripID,
	)
	return err
}

func (r *Repository) DeleteVehicleRental(ctx context.Context, tripID, rentalID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM vehicle_rentals WHERE id = ? AND trip_id = ?`, rentalID, tripID)
	return err
}

func (r *Repository) ListVehicleRentals(ctx context.Context, tripID string) ([]trips.VehicleRental, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, trip_id, pick_up_location, drop_off_location, vehicle_detail, pick_up_at, drop_off_at, booking_confirmation, notes, vehicle_image_path, cost, insurance_cost, pay_at_pick_up, pick_up_itinerary_id, drop_off_itinerary_id, rental_expense_id, insurance_expense_id, created_at
		FROM vehicle_rentals
		WHERE trip_id = ?
		ORDER BY pick_up_at ASC, created_at ASC`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []trips.VehicleRental{}
	for rows.Next() {
		var v trips.VehicleRental
		if err := rows.Scan(&v.ID, &v.TripID, &v.PickUpLocation, &v.DropOffLocation, &v.VehicleDetail, &v.PickUpAt, &v.DropOffAt, &v.BookingConfirmation, &v.Notes, &v.VehicleImagePath, &v.Cost, &v.InsuranceCost, &v.PayAtPickUp,
			&v.PickUpItineraryID, &v.DropOffItineraryID, &v.RentalExpenseID, &v.InsuranceExpenseID, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) AddFlight(ctx context.Context, item trips.Flight) error {
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO flight_entries
		(id, trip_id, flight_name, flight_number, depart_airport, arrive_airport, depart_at, arrive_at, booking_confirmation, notes, document_path, cost, depart_itinerary_id, arrive_itinerary_id, expense_id, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.TripID, item.FlightName, item.FlightNumber, item.DepartAirport, item.ArriveAirport, item.DepartAt, item.ArriveAt, item.BookingConfirmation, item.Notes, item.DocumentPath, item.Cost, item.DepartItineraryID, item.ArriveItineraryID, item.ExpenseID, now,
	)
	return err
}

func (r *Repository) GetFlight(ctx context.Context, tripID, flightID string) (trips.Flight, error) {
	var f trips.Flight
	err := r.db.QueryRowContext(ctx, `
		SELECT id, trip_id, flight_name, flight_number, depart_airport, arrive_airport, depart_at, arrive_at, booking_confirmation, notes, document_path, cost, depart_itinerary_id, arrive_itinerary_id, expense_id, created_at
		FROM flight_entries WHERE id = ? AND trip_id = ?`,
		flightID, tripID,
	).Scan(
		&f.ID, &f.TripID, &f.FlightName, &f.FlightNumber, &f.DepartAirport, &f.ArriveAirport, &f.DepartAt, &f.ArriveAt, &f.BookingConfirmation, &f.Notes, &f.DocumentPath, &f.Cost, &f.DepartItineraryID, &f.ArriveItineraryID, &f.ExpenseID, &f.CreatedAt,
	)
	return f, err
}

func (r *Repository) UpdateFlight(ctx context.Context, item trips.Flight) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE flight_entries
		SET flight_name = ?, flight_number = ?, depart_airport = ?, arrive_airport = ?, depart_at = ?, arrive_at = ?, booking_confirmation = ?, notes = ?, document_path = ?, cost = ?, depart_itinerary_id = ?, arrive_itinerary_id = ?, expense_id = ?
		WHERE id = ? AND trip_id = ?`,
		item.FlightName, item.FlightNumber, item.DepartAirport, item.ArriveAirport, item.DepartAt, item.ArriveAt, item.BookingConfirmation, item.Notes, item.DocumentPath, item.Cost, item.DepartItineraryID, item.ArriveItineraryID, item.ExpenseID,
		item.ID, item.TripID,
	)
	return err
}

func (r *Repository) DeleteFlight(ctx context.Context, tripID, flightID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM flight_entries WHERE id = ? AND trip_id = ?`, flightID, tripID)
	return err
}

func (r *Repository) ListFlights(ctx context.Context, tripID string) ([]trips.Flight, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, trip_id, flight_name, flight_number, depart_airport, arrive_airport, depart_at, arrive_at, booking_confirmation, notes, document_path, cost, depart_itinerary_id, arrive_itinerary_id, expense_id, created_at
		FROM flight_entries
		WHERE trip_id = ?
		ORDER BY depart_at ASC, created_at ASC`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []trips.Flight{}
	for rows.Next() {
		var f trips.Flight
		if err := rows.Scan(
			&f.ID, &f.TripID, &f.FlightName, &f.FlightNumber, &f.DepartAirport, &f.ArriveAirport, &f.DepartAt, &f.ArriveAt, &f.BookingConfirmation, &f.Notes, &f.DocumentPath, &f.Cost, &f.DepartItineraryID, &f.ArriveItineraryID, &f.ExpenseID, &f.CreatedAt,
		); err != nil {
			return nil, err
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *Repository) ListChanges(ctx context.Context, tripID, since string) ([]trips.Change, error) {
	query := `
		SELECT id, trip_id, entity, entity_id, operation, changed_at, payload
		FROM change_log
		WHERE trip_id = ?`
	args := []any{tripID}
	if since != "" {
		query += " AND changed_at > ?"
		args = append(args, since)
	}
	query += " ORDER BY id ASC LIMIT 500"
	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []trips.Change{}
	for rows.Next() {
		var c trips.Change
		if err := rows.Scan(&c.ID, &c.TripID, &c.Entity, &c.EntityID, &c.Operation, &c.ChangedAt, &c.Payload); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) logChange(ctx context.Context, tripID, entity, entityID, operation string, payload map[string]any) error {
	b, _ := json.Marshal(payload)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO change_log (trip_id, entity, entity_id, operation, changed_at, payload)
		VALUES (?, ?, ?, ?, ?, ?)`,
		tripID, entity, entityID, operation, time.Now().UTC(), string(b),
	)
	return err
}

func (r *Repository) GetAppSettings(ctx context.Context) (trips.AppSettings, error) {
	var settings trips.AppSettings
	err := r.db.QueryRowContext(ctx, `
		SELECT app_title, default_currency_name, default_currency_symbol,
		       COALESCE(NULLIF(TRIM(map_default_place_label), ''), 'Tokyo'),
		       map_default_latitude, map_default_longitude, map_default_zoom, enable_location_lookup,
		       COALESCE(registration_enabled, 1),
		       COALESCE(theme_preference, 'system'), COALESCE(dashboard_trip_layout, 'grid'), COALESCE(dashboard_trip_sort, 'name'), COALESCE(dashboard_hero_background, 'default'),
		       COALESCE(NULLIF(TRIM(trip_dashboard_heading), ''), 'Trip Dashboard'),
		       COALESCE(TRIM(google_maps_api_key), ''),
		       COALESCE(NULLIF(TRIM(default_distance_unit), ''), 'km')
		FROM app_settings
		WHERE id = 1`).
		Scan(
			&settings.AppTitle,
			&settings.DefaultCurrencyName,
			&settings.DefaultCurrencySymbol,
			&settings.MapDefaultPlaceLabel,
			&settings.MapDefaultLatitude,
			&settings.MapDefaultLongitude,
			&settings.MapDefaultZoom,
			&settings.EnableLocationLookup,
			&settings.RegistrationEnabled,
			&settings.ThemePreference,
			&settings.DashboardTripLayout,
			&settings.DashboardTripSort,
			&settings.DashboardHeroBackground,
			&settings.TripDashboardHeading,
			&settings.GoogleMapsAPIKey,
			&settings.DefaultDistanceUnit,
		)
	return settings, err
}

func (r *Repository) SaveAppSettings(ctx context.Context, settings trips.AppSettings) error {
	regEn := 0
	if settings.RegistrationEnabled {
		regEn = 1
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO app_settings
			(id, app_title, default_currency_name, default_currency_symbol, map_default_place_label, map_default_latitude, map_default_longitude, map_default_zoom, enable_location_lookup,
			 registration_enabled, theme_preference, dashboard_trip_layout, dashboard_trip_sort, dashboard_hero_background, trip_dashboard_heading, google_maps_api_key, default_distance_unit, updated_at)
		VALUES
			(1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			app_title = excluded.app_title,
			default_currency_name = excluded.default_currency_name,
			default_currency_symbol = excluded.default_currency_symbol,
			map_default_place_label = excluded.map_default_place_label,
			map_default_latitude = excluded.map_default_latitude,
			map_default_longitude = excluded.map_default_longitude,
			map_default_zoom = excluded.map_default_zoom,
			enable_location_lookup = excluded.enable_location_lookup,
			registration_enabled = excluded.registration_enabled,
			theme_preference = excluded.theme_preference,
			dashboard_trip_layout = excluded.dashboard_trip_layout,
			dashboard_trip_sort = excluded.dashboard_trip_sort,
			dashboard_hero_background = excluded.dashboard_hero_background,
			trip_dashboard_heading = excluded.trip_dashboard_heading,
			google_maps_api_key = excluded.google_maps_api_key,
			default_distance_unit = excluded.default_distance_unit,
			updated_at = excluded.updated_at
	`,
		settings.AppTitle,
		settings.DefaultCurrencyName,
		settings.DefaultCurrencySymbol,
		settings.MapDefaultPlaceLabel,
		settings.MapDefaultLatitude,
		settings.MapDefaultLongitude,
		settings.MapDefaultZoom,
		settings.EnableLocationLookup,
		regEn,
		settings.ThemePreference,
		settings.DashboardTripLayout,
		settings.DashboardTripSort,
		settings.DashboardHeroBackground,
		settings.TripDashboardHeading,
		settings.GoogleMapsAPIKey,
		settings.DefaultDistanceUnit,
		time.Now().UTC(),
	)
	return err
}

func (r *Repository) GetTripDayLabels(ctx context.Context, tripID string) (map[int]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT day_number, label
		FROM trip_day_labels
		WHERE trip_id = ?`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[int]string{}
	for rows.Next() {
		var dayNumber int
		var label string
		if err := rows.Scan(&dayNumber, &label); err != nil {
			return nil, err
		}
		out[dayNumber] = label
	}
	return out, rows.Err()
}

func (r *Repository) SaveTripDayLabel(ctx context.Context, tripID string, dayNumber int, label string) error {
	trimmed := strings.TrimSpace(label)
	if trimmed == "" {
		_, err := r.db.ExecContext(ctx, `
			DELETE FROM trip_day_labels
			WHERE trip_id = ? AND day_number = ?`, tripID, dayNumber)
		return err
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO trip_day_labels (trip_id, day_number, label, updated_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(trip_id, day_number) DO UPDATE SET
			label = excluded.label,
			updated_at = excluded.updated_at`,
		tripID, dayNumber, trimmed, time.Now().UTC(),
	)
	return err
}

func (r *Repository) syncTabExpenseFTS(ctx context.Context, e trips.Expense) error {
	_, _ = r.db.ExecContext(ctx, `DELETE FROM tab_expense_search WHERE expense_id = ?`, e.ID)
	if !e.FromTab {
		return nil
	}
	var b strings.Builder
	b.WriteString(strings.ToLower(strings.TrimSpace(e.Title)))
	b.WriteByte(' ')
	b.WriteString(strings.ToLower(strings.TrimSpace(e.Notes)))
	b.WriteByte(' ')
	b.WriteString(strings.ToLower(strings.TrimSpace(e.Category)))
	b.WriteByte(' ')
	b.WriteString(strings.ToLower(strings.TrimSpace(e.PaymentMethod)))
	b.WriteByte(' ')
	fmt.Fprintf(&b, "%.2f", e.Amount)
	body := strings.TrimSpace(b.String())
	if body == "" {
		body = "expense"
	}
	_, err := r.db.ExecContext(ctx, `INSERT INTO tab_expense_search(trip_id, expense_id, body) VALUES(?,?,?)`, e.TripID, e.ID, body)
	return err
}

// SearchTabExpenseIDs returns expense IDs for Tab entries matching the query (FTS).
func (r *Repository) SearchTabExpenseIDs(ctx context.Context, tripID, query string) ([]string, error) {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil, nil
	}
	parts := regexp.MustCompile(`\s+`).Split(q, -1)
	nonAlnum := regexp.MustCompile(`[^\p{L}\p{N}]+`)
	var terms []string
	for _, t := range parts {
		t = strings.TrimSpace(t)
		if t == "" {
			continue
		}
		t = nonAlnum.ReplaceAllString(t, "")
		if t == "" {
			continue
		}
		terms = append(terms, t+`*`)
	}
	if len(terms) == 0 {
		return nil, nil
	}
	match := strings.Join(terms, ` AND `)
	rows, err := r.db.QueryContext(ctx,
		`SELECT expense_id FROM tab_expense_search WHERE trip_id = ? AND tab_expense_search MATCH ? ORDER BY rowid DESC`,
		tripID, match)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (r *Repository) ListTripGuests(ctx context.Context, tripID string) ([]trips.TripGuest, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, trip_id, display_name, created_at FROM trip_guests WHERE trip_id = ? ORDER BY display_name COLLATE NOCASE`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []trips.TripGuest
	for rows.Next() {
		var g trips.TripGuest
		if err := rows.Scan(&g.ID, &g.TripID, &g.DisplayName, &g.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

func (r *Repository) UpdateTripTabDefaults(ctx context.Context, tripID, mode, splitJSON string) error {
	_, err := r.db.ExecContext(ctx, `
		UPDATE trips SET tab_default_split_mode = ?, tab_default_split_json = ? WHERE id = ?`,
		strings.TrimSpace(mode), splitJSON, tripID)
	return err
}

func (r *Repository) AddTripGuest(ctx context.Context, g trips.TripGuest) error {
	if g.ID == "" {
		g.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO trip_guests (id, trip_id, display_name, created_at) VALUES (?, ?, ?, ?)`,
		g.ID, g.TripID, strings.TrimSpace(g.DisplayName), now,
	)
	return err
}

func (r *Repository) DeleteTripGuest(ctx context.Context, tripID, guestID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM trip_guests WHERE id = ? AND trip_id = ?`, guestID, tripID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) GetTripGuest(ctx context.Context, tripID, guestID string) (trips.TripGuest, error) {
	var g trips.TripGuest
	err := r.db.QueryRowContext(ctx, `
		SELECT id, trip_id, display_name, created_at FROM trip_guests WHERE id = ? AND trip_id = ?`,
		guestID, tripID,
	).Scan(&g.ID, &g.TripID, &g.DisplayName, &g.CreatedAt)
	return g, err
}

func (r *Repository) ListDepartedTabParticipants(ctx context.Context, tripID string) ([]trips.DepartedTabParticipant, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT trip_id, participant_key, display_name, left_at
		FROM trip_departed_tab_participants WHERE trip_id = ?
		ORDER BY display_name COLLATE NOCASE`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []trips.DepartedTabParticipant
	for rows.Next() {
		var d trips.DepartedTabParticipant
		var leftS string
		if err := rows.Scan(&d.TripID, &d.ParticipantKey, &d.DisplayName, &leftS); err != nil {
			return nil, err
		}
		d.LeftAt, _ = time.Parse(time.RFC3339, leftS)
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repository) UpsertDepartedTabParticipant(ctx context.Context, tripID, participantKey, displayName string) error {
	tripID = strings.TrimSpace(tripID)
	participantKey = strings.TrimSpace(participantKey)
	displayName = strings.TrimSpace(displayName)
	if tripID == "" || participantKey == "" {
		return errors.New("invalid departed tab participant")
	}
	if displayName == "" {
		displayName = participantKey
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO trip_departed_tab_participants (trip_id, participant_key, display_name, left_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(trip_id, participant_key) DO UPDATE SET
			display_name = excluded.display_name,
			left_at = excluded.left_at`,
		tripID, participantKey, displayName, now)
	return err
}

func (r *Repository) ClearDepartedTabParticipant(ctx context.Context, tripID, participantKey string) error {
	_, err := r.db.ExecContext(ctx, `
		DELETE FROM trip_departed_tab_participants WHERE trip_id = ? AND participant_key = ?`,
		strings.TrimSpace(tripID), strings.TrimSpace(participantKey))
	return err
}

func (r *Repository) ListTabSettlements(ctx context.Context, tripID string) ([]trips.TabSettlement, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, trip_id, payer_user_id, payee_user_id, amount, method, settled_on, notes, created_at
		FROM tab_settlements WHERE trip_id = ? ORDER BY created_at DESC`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []trips.TabSettlement
	for rows.Next() {
		var s trips.TabSettlement
		if err := rows.Scan(&s.ID, &s.TripID, &s.PayerUserID, &s.PayeeUserID, &s.Amount, &s.Method, &s.SettledOn, &s.Notes, &s.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, s)
	}
	return out, rows.Err()
}

func (r *Repository) AddTabSettlement(ctx context.Context, s trips.TabSettlement) error {
	if s.ID == "" {
		s.ID = uuid.NewString()
	}
	if strings.TrimSpace(s.Method) == "" {
		s.Method = "Cash"
	}
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tab_settlements (id, trip_id, payer_user_id, payee_user_id, amount, method, settled_on, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.TripID, s.PayerUserID, s.PayeeUserID, s.Amount, s.Method, s.SettledOn, s.Notes, now,
	)
	return err
}

func (r *Repository) UpdateTabSettlement(ctx context.Context, s trips.TabSettlement) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE tab_settlements SET payer_user_id = ?, payee_user_id = ?, amount = ?, method = ?, settled_on = ?, notes = ?
		WHERE id = ? AND trip_id = ?`,
		s.PayerUserID, s.PayeeUserID, s.Amount, s.Method, s.SettledOn, s.Notes, s.ID, s.TripID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) DeleteTabSettlement(ctx context.Context, tripID, settlementID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM tab_settlements WHERE id = ? AND trip_id = ?`, settlementID, tripID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}
