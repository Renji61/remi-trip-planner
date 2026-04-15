package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"

	"remi-trip-planner/internal/trips"

	"github.com/google/uuid"
)

type Repository struct {
	db *sql.DB
	tx *sql.Tx
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

func scanBoolFromInt(v int64) bool {
	return v != 0
}

const tripSelectCols = `id, name, description, start_date, end_date, cover_image_url, currency_name, currency_symbol, home_map_latitude, home_map_longitude, home_map_place_label, is_archived, owner_user_id,
		ui_show_stay, ui_show_vehicle, ui_show_flights, ui_show_spends, ui_show_itinerary, ui_show_checklist,
		ui_itinerary_expand, ui_spends_expand, ui_tab_expand, ui_time_format, ui_date_format,
		ui_label_stay, ui_label_vehicle, ui_label_flights, ui_label_spends, ui_label_group_expenses,
		ui_main_section_order, ui_sidebar_widget_order,
		ui_main_section_hidden, ui_sidebar_widget_hidden,
		ui_show_custom_links, ui_custom_sidebar_links,
		created_at, updated_at, budget_cap_cents, ui_show_the_tab, ui_show_documents, ui_collaboration_enabled,
		tab_default_split_mode, tab_default_split_json, distance_unit, setup_complete`

func scanTrip(scan func(dest ...any) error) (trips.Trip, error) {
	var t trips.Trip
	var ss, sv, sf, sp, sit, sck, scl, stt, sdoc, scoll, setupDone int
	err := scan(
		&t.ID, &t.Name, &t.Description, &t.StartDate, &t.EndDate, &t.CoverImage, &t.CurrencyName, &t.CurrencySymbol, &t.HomeMapLatitude, &t.HomeMapLongitude, &t.HomeMapPlaceLabel, &t.IsArchived, &t.OwnerUserID,
		&ss, &sv, &sf, &sp, &sit, &sck,
		&t.UIItineraryExpand, &t.UISpendsExpand, &t.UITabExpand, &t.UITimeFormat, &t.UIDateFormat,
		&t.UILabelStay, &t.UILabelVehicle, &t.UILabelFlights, &t.UILabelSpends, &t.UILabelGroupExpenses,
		&t.UIMainSectionOrder, &t.UISidebarWidgetOrder,
		&t.UIMainSectionHidden, &t.UISidebarWidgetHidden,
		&scl, &t.UICustomSidebarLinks,
		&t.CreatedAt, &t.UpdatedAt,
		&t.BudgetCapCents, &stt, &sdoc, &scoll,
		&t.TabDefaultSplitMode, &t.TabDefaultSplitJSON,
		&t.DistanceUnit,
		&setupDone,
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
	t.UIShowDocuments = sdoc != 0
	t.UICollaborationEnabled = scoll != 0
	t.SetupComplete = setupDone != 0
	t.UIDateFormat = trips.NormalizeTripUIDateStorage(t.UIDateFormat)
	trips.SetTripBudgetCapCents(&t, t.BudgetCapCents)
	return t, nil
}

func (r *Repository) CreateTrip(ctx context.Context, t trips.Trip) (string, error) {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	if t.BudgetCapCents == 0 && t.BudgetCap != 0 {
		t.BudgetCapCents = trips.MoneyToCentsFloat(t.BudgetCap)
	}
	trips.SetTripBudgetCapCents(&t, t.BudgetCapCents)
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO trips (
			id, name, description, start_date, end_date, cover_image_url, currency_name, currency_symbol, home_map_latitude, home_map_longitude, home_map_place_label, is_archived, owner_user_id,
			ui_show_stay, ui_show_vehicle, ui_show_flights, ui_show_spends, ui_show_itinerary, ui_show_checklist,
			ui_itinerary_expand, ui_spends_expand, ui_tab_expand, ui_time_format, ui_date_format,
			ui_label_stay, ui_label_vehicle, ui_label_flights, ui_label_spends, ui_label_group_expenses,
			ui_main_section_order, ui_sidebar_widget_order,
			ui_main_section_hidden, ui_sidebar_widget_hidden,
			ui_show_custom_links, ui_custom_sidebar_links,
			budget_cap, budget_cap_cents, ui_show_the_tab, ui_show_documents, ui_collaboration_enabled, tab_default_split_mode, tab_default_split_json, distance_unit, setup_complete,
			created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, 1, 1, 1, 1, 1, 'first', 'first', 'first', '12h', ?, '', '', '', '', '', '', '', '', '', 1, '', ?, ?, 1, 1, 1, '', '', ?, ?, ?, ?)`,
		t.ID, t.Name, t.Description, t.StartDate, t.EndDate, t.CoverImage, t.CurrencyName, t.CurrencySymbol, t.HomeMapLatitude, t.HomeMapLongitude, t.HomeMapPlaceLabel, t.IsArchived, t.OwnerUserID,
		trips.NormalizeTripUIDateStorage(t.UIDateFormat),
		t.BudgetCap, t.BudgetCapCents, t.DistanceUnit, sqliteBool(t.SetupComplete), now, now,
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
	row := r.queryRowContext(ctx, `SELECT `+tripSelectCols+` FROM trips WHERE id = ?`, tripID)
	return scanTrip(row.Scan)
}

func (r *Repository) UpdateTrip(ctx context.Context, t trips.Trip) error {
	t.UIDateFormat = trips.NormalizeTripUIDateStorage(t.UIDateFormat)
	if t.BudgetCapCents == 0 && t.BudgetCap != 0 {
		t.BudgetCapCents = trips.MoneyToCentsFloat(t.BudgetCap)
	}
	trips.SetTripBudgetCapCents(&t, t.BudgetCapCents)
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		UPDATE trips
		SET name = ?, description = ?, start_date = ?, end_date = ?, cover_image_url = ?, currency_name = ?, currency_symbol = ?,
		    home_map_latitude = ?, home_map_longitude = ?, home_map_place_label = ?,
		    budget_cap = ?, budget_cap_cents = ?,
		    ui_show_stay = ?, ui_show_vehicle = ?, ui_show_flights = ?, ui_show_spends = ?, ui_show_itinerary = ?, ui_show_checklist = ?,
		    ui_show_the_tab = ?,
		    ui_show_documents = ?,
		    ui_collaboration_enabled = ?,
		    ui_itinerary_expand = ?, ui_spends_expand = ?, ui_tab_expand = ?, ui_time_format = ?, ui_date_format = ?,
		    ui_label_stay = ?, ui_label_vehicle = ?, ui_label_flights = ?, ui_label_spends = ?, ui_label_group_expenses = ?,
		    ui_main_section_order = ?, ui_sidebar_widget_order = ?,
		    ui_main_section_hidden = ?, ui_sidebar_widget_hidden = ?,
		    ui_show_custom_links = ?, ui_custom_sidebar_links = ?,
		    tab_default_split_mode = ?, tab_default_split_json = ?,
		    distance_unit = ?, setup_complete = ?,
		    updated_at = ?
		WHERE id = ?`,
		t.Name, t.Description, t.StartDate, t.EndDate, t.CoverImage, t.CurrencyName, t.CurrencySymbol, t.HomeMapLatitude, t.HomeMapLongitude, t.HomeMapPlaceLabel,
		t.BudgetCap, t.BudgetCapCents,
		sqliteBool(t.UIShowStay), sqliteBool(t.UIShowVehicle), sqliteBool(t.UIShowFlights), sqliteBool(t.UIShowSpends),
		sqliteBool(t.UIShowItinerary), sqliteBool(t.UIShowChecklist),
		sqliteBool(t.UIShowTheTab),
		sqliteBool(t.UIShowDocuments),
		sqliteBool(t.UICollaborationEnabled),
		t.UIItineraryExpand, t.UISpendsExpand, t.UITabExpand, t.UITimeFormat, t.UIDateFormat,
		t.UILabelStay, t.UILabelVehicle, t.UILabelFlights, t.UILabelSpends, t.UILabelGroupExpenses,
		t.UIMainSectionOrder, t.UISidebarWidgetOrder,
		t.UIMainSectionHidden, t.UISidebarWidgetHidden,
		sqliteBool(t.UIShowCustomLinks), t.UICustomSidebarLinks,
		t.TabDefaultSplitMode, t.TabDefaultSplitJSON,
		t.DistanceUnit,
		sqliteBool(t.SetupComplete),
		now, t.ID,
	)
	if err == nil {
		_ = r.logChange(ctx, t.ID, "trip", t.ID, "update", map[string]any{
			"name": t.Name, "start_date": t.StartDate, "end_date": t.EndDate, "cover_image_url": t.CoverImage, "currency_name": t.CurrencyName, "currency_symbol": t.CurrencySymbol,
			"setup_complete": t.SetupComplete,
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

func (r *Repository) nextItinerarySortOrder(ctx context.Context, tripID string, dayNumber int) (int, error) {
	row := r.db.QueryRowContext(ctx, `SELECT COALESCE(MAX(sort_order), 0) FROM itinerary_items WHERE trip_id = ? AND day_number = ?`, tripID, dayNumber)
	var m int64
	if err := row.Scan(&m); err != nil {
		return 0, err
	}
	next := int(m) + 100000
	if next < 100000 {
		return 100000, nil
	}
	return next, nil
}

// NextItinerarySortOrder returns a sort_order value after the current max for that trip day.
func (r *Repository) NextItinerarySortOrder(ctx context.Context, tripID string, dayNumber int) (int, error) {
	return r.nextItinerarySortOrder(ctx, tripID, dayNumber)
}

func (r *Repository) AddItineraryItem(ctx context.Context, item trips.ItineraryItem) error {
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.EstCostCents == 0 && item.EstCost != 0 {
		item.EstCostCents = trips.MoneyToCentsFloat(item.EstCost)
	}
	trips.SetItineraryEstCostCents(&item, item.EstCostCents)
	kind := strings.TrimSpace(item.ItemKind)
	if kind != trips.ItineraryItemKindCommute {
		kind = trips.ItineraryItemKindStop
	}
	if item.SortOrder == 0 {
		so, err := r.nextItinerarySortOrder(ctx, item.TripID, item.DayNumber)
		if err != nil {
			return err
		}
		item.SortOrder = so
	}
	now := time.Now().UTC()
	_, err := r.execContext(ctx, `
		INSERT INTO itinerary_items
		(id, trip_id, day_number, title, notes, location, image_path, latitude, longitude, est_cost, est_cost_cents, start_time, end_time, item_kind, commute_from_item_id, commute_to_item_id, transport_mode, sort_order, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.TripID, item.DayNumber, item.Title, item.Notes, item.Location, item.ImagePath, item.Latitude, item.Longitude, item.EstCost, item.EstCostCents, item.StartTime, item.EndTime, kind, item.CommuteFromItemID, item.CommuteToItemID, item.TransportMode, item.SortOrder, now, now,
	)
	if err == nil {
		item.CreatedAt = now
		item.UpdatedAt = now
		_ = r.logChange(ctx, item.TripID, "itinerary_item", item.ID, "create", map[string]any{
			"title": item.Title, "day_number": item.DayNumber,
		})
	}
	return err
}

func (r *Repository) UpdateItineraryItem(ctx context.Context, item trips.ItineraryItem) (int64, error) {
	if item.EstCostCents == 0 && item.EstCost != 0 {
		item.EstCostCents = trips.MoneyToCentsFloat(item.EstCost)
	}
	trips.SetItineraryEstCostCents(&item, item.EstCostCents)
	now := time.Now().UTC()
	var (
		res sql.Result
		err error
	)
	kind := strings.TrimSpace(item.ItemKind)
	if kind != trips.ItineraryItemKindCommute {
		kind = trips.ItineraryItemKindStop
	}
	if item.EnforceOptimisticLock {
		res, err = r.execContext(ctx, `
			UPDATE itinerary_items
			SET day_number = ?, title = ?, notes = ?, location = ?, image_path = ?, latitude = ?, longitude = ?, est_cost = ?, est_cost_cents = ?, start_time = ?, end_time = ?, item_kind = ?, commute_from_item_id = ?, commute_to_item_id = ?, transport_mode = ?, sort_order = ?, updated_at = ?
			WHERE id = ? AND trip_id = ? AND updated_at = ?`,
			item.DayNumber, item.Title, item.Notes, item.Location, item.ImagePath, item.Latitude, item.Longitude, item.EstCost, item.EstCostCents, item.StartTime, item.EndTime, kind, item.CommuteFromItemID, item.CommuteToItemID, item.TransportMode, item.SortOrder, now,
			item.ID, item.TripID, item.ExpectedUpdatedAt,
		)
	} else {
		res, err = r.execContext(ctx, `
			UPDATE itinerary_items
			SET day_number = ?, title = ?, notes = ?, location = ?, image_path = ?, latitude = ?, longitude = ?, est_cost = ?, est_cost_cents = ?, start_time = ?, end_time = ?, item_kind = ?, commute_from_item_id = ?, commute_to_item_id = ?, transport_mode = ?, sort_order = ?, updated_at = ?
			WHERE id = ? AND trip_id = ?`,
			item.DayNumber, item.Title, item.Notes, item.Location, item.ImagePath, item.Latitude, item.Longitude, item.EstCost, item.EstCostCents, item.StartTime, item.EndTime, kind, item.CommuteFromItemID, item.CommuteToItemID, item.TransportMode, item.SortOrder, now,
			item.ID, item.TripID,
		)
	}
	if err != nil {
		return 0, err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return 0, err
	}
	if n > 0 {
		item.UpdatedAt = now
		_ = r.logChange(ctx, item.TripID, "itinerary_item", item.ID, "update", map[string]any{
			"title": item.Title, "day_number": item.DayNumber,
		})
	}
	return n, nil
}

func (r *Repository) DeleteItineraryItem(ctx context.Context, tripID, itemID string) error {
	_, err := r.execContext(ctx, `DELETE FROM itinerary_items WHERE id = ? AND trip_id = ?`, itemID, tripID)
	if err == nil {
		_ = r.logChange(ctx, tripID, "itinerary_item", itemID, "delete", map[string]any{})
	}
	return err
}

func (r *Repository) ListItineraryItems(ctx context.Context, tripID string) ([]trips.ItineraryItem, error) {
	rows, err := r.queryContext(ctx, `
		SELECT id, trip_id, day_number, title, notes, location, image_path, latitude, longitude, est_cost_cents, start_time, end_time, created_at, updated_at, item_kind, commute_from_item_id, commute_to_item_id, transport_mode, sort_order
		FROM itinerary_items WHERE trip_id = ?
		ORDER BY day_number ASC, sort_order ASC, created_at ASC, id ASC`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []trips.ItineraryItem{}
	for rows.Next() {
		var i trips.ItineraryItem
		if err := rows.Scan(&i.ID, &i.TripID, &i.DayNumber, &i.Title, &i.Notes, &i.Location, &i.ImagePath, &i.Latitude, &i.Longitude, &i.EstCostCents, &i.StartTime, &i.EndTime, &i.CreatedAt, &i.UpdatedAt, &i.ItemKind, &i.CommuteFromItemID, &i.CommuteToItemID, &i.TransportMode, &i.SortOrder); err != nil {
			return nil, err
		}
		trips.SetItineraryEstCostCents(&i, i.EstCostCents)
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *Repository) ListAllItineraryItems(ctx context.Context) ([]trips.ItineraryItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, trip_id, day_number, title, notes, location, image_path, latitude, longitude, est_cost_cents, start_time, end_time, created_at, updated_at, item_kind, commute_from_item_id, commute_to_item_id, transport_mode, sort_order
		FROM itinerary_items
		ORDER BY trip_id, day_number ASC, sort_order ASC, created_at ASC, id ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []trips.ItineraryItem{}
	for rows.Next() {
		var i trips.ItineraryItem
		if err := rows.Scan(&i.ID, &i.TripID, &i.DayNumber, &i.Title, &i.Notes, &i.Location, &i.ImagePath, &i.Latitude, &i.Longitude, &i.EstCostCents, &i.StartTime, &i.EndTime, &i.CreatedAt, &i.UpdatedAt, &i.ItemKind, &i.CommuteFromItemID, &i.CommuteToItemID, &i.TransportMode, &i.SortOrder); err != nil {
			return nil, err
		}
		trips.SetItineraryEstCostCents(&i, i.EstCostCents)
		out = append(out, i)
	}
	return out, rows.Err()
}

// ClearCommuteRefsForDeletedItem blanks commute neighbor IDs on other rows that pointed at a deleted itinerary item.
func (r *Repository) ClearCommuteRefsForDeletedItem(ctx context.Context, tripID, deletedItemID string) error {
	now := time.Now().UTC()
	_, err := r.execContext(ctx, `UPDATE itinerary_items SET commute_from_item_id = '', updated_at = ? WHERE trip_id = ? AND item_kind = ? AND commute_from_item_id = ?`,
		now, tripID, trips.ItineraryItemKindCommute, deletedItemID)
	if err != nil {
		return err
	}
	_, err = r.execContext(ctx, `UPDATE itinerary_items SET commute_to_item_id = '', updated_at = ? WHERE trip_id = ? AND item_kind = ? AND commute_to_item_id = ?`,
		now, tripID, trips.ItineraryItemKindCommute, deletedItemID)
	return err
}

// RebalanceItineraryDaySortOrders rewrites sort_order for one trip day to evenly spaced values (after inserts collide).
func (r *Repository) RebalanceItineraryDaySortOrders(ctx context.Context, tripID string, dayNumber int) error {
	rows, err := r.queryContext(ctx, `SELECT id FROM itinerary_items WHERE trip_id = ? AND day_number = ? ORDER BY sort_order ASC, created_at ASC, id ASC`, tripID, dayNumber)
	if err != nil {
		return err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}
	now := time.Now().UTC()
	for i, id := range ids {
		if _, err := r.execContext(ctx, `UPDATE itinerary_items SET sort_order = ?, updated_at = ? WHERE id = ? AND trip_id = ?`, (i+1)*100000, now, id, tripID); err != nil {
			return err
		}
	}
	return nil
}

func (r *Repository) AddExpense(ctx context.Context, e trips.Expense) error {
	if e.ID == "" {
		e.ID = uuid.NewString()
	}
	if strings.TrimSpace(e.PaymentMethod) == "" {
		e.PaymentMethod = "Cash"
	}
	if e.AmountCents == 0 && e.Amount != 0 {
		e.AmountCents = trips.MoneyToCentsFloat(e.Amount)
	}
	trips.SetExpenseAmountCents(&e, e.AmountCents)
	now := time.Now().UTC()
	_, err := r.execContext(ctx, `
		INSERT INTO expenses (id, trip_id, category, amount, amount_cents, notes, spent_on, payment_method, lodging_id, from_tab, receipt_path, title, paid_by, split_mode, split_json, due_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.TripID, e.Category, e.Amount, e.AmountCents, e.Notes, e.SpentOn, e.PaymentMethod, e.LodgingID, sqliteBool(e.FromTab), e.ReceiptPath,
		e.Title, e.PaidBy, e.SplitMode, e.SplitJSON, strings.TrimSpace(e.DueAt), now, now,
	)
	if err == nil {
		e.CreatedAt = now
		e.UpdatedAt = now
		_ = r.logChange(ctx, e.TripID, "expense", e.ID, "create", map[string]any{"amount": e.Amount, "category": e.Category})
		_ = r.syncTabExpenseFTS(ctx, e)
	}
	return err
}

func (r *Repository) UpdateExpense(ctx context.Context, e trips.Expense) error {
	if strings.TrimSpace(e.PaymentMethod) == "" {
		e.PaymentMethod = "Cash"
	}
	if e.AmountCents == 0 && e.Amount != 0 {
		e.AmountCents = trips.MoneyToCentsFloat(e.Amount)
	}
	trips.SetExpenseAmountCents(&e, e.AmountCents)
	now := time.Now().UTC()
	var (
		res sql.Result
		err error
	)
	if e.EnforceOptimisticLock {
		res, err = r.execContext(ctx, `
			UPDATE expenses
			SET category = ?, amount = ?, amount_cents = ?, notes = ?, spent_on = ?, payment_method = ?, lodging_id = ?, from_tab = ?, receipt_path = ?,
			    title = ?, paid_by = ?, split_mode = ?, split_json = ?, due_at = ?, updated_at = ?
			WHERE id = ? AND trip_id = ? AND updated_at = ?`,
			e.Category, e.Amount, e.AmountCents, e.Notes, e.SpentOn, e.PaymentMethod, e.LodgingID, sqliteBool(e.FromTab), e.ReceiptPath,
			e.Title, e.PaidBy, e.SplitMode, e.SplitJSON, strings.TrimSpace(e.DueAt), now, e.ID, e.TripID, e.ExpectedUpdatedAt,
		)
	} else {
		res, err = r.execContext(ctx, `
			UPDATE expenses
			SET category = ?, amount = ?, amount_cents = ?, notes = ?, spent_on = ?, payment_method = ?, lodging_id = ?, from_tab = ?, receipt_path = ?,
			    title = ?, paid_by = ?, split_mode = ?, split_json = ?, due_at = ?, updated_at = ?
			WHERE id = ? AND trip_id = ?`,
			e.Category, e.Amount, e.AmountCents, e.Notes, e.SpentOn, e.PaymentMethod, e.LodgingID, sqliteBool(e.FromTab), e.ReceiptPath,
			e.Title, e.PaidBy, e.SplitMode, e.SplitJSON, strings.TrimSpace(e.DueAt), now, e.ID, e.TripID,
		)
	}
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		current, getErr := r.GetExpense(ctx, e.TripID, e.ID)
		if getErr == nil {
			return &trips.ConflictError{
				Resource:        "expense",
				Message:         "Someone else updated this expense a moment ago. Reopen it to review the latest values, then try again.",
				LatestUpdatedAt: current.UpdatedAt,
			}
		}
		if errors.Is(getErr, sql.ErrNoRows) {
			return sql.ErrNoRows
		}
		return getErr
	}
	e.UpdatedAt = now
	_ = r.logChange(ctx, e.TripID, "expense", e.ID, "update", map[string]any{
		"amount": e.Amount, "category": e.Category,
	})
	_ = r.syncTabExpenseFTS(ctx, e)
	return err
}

func (r *Repository) DeleteExpense(ctx context.Context, tripID, expenseID string) error {
	_, _ = r.execContext(ctx, `DELETE FROM tab_expense_search WHERE expense_id = ?`, expenseID)
	_, err := r.execContext(ctx, `DELETE FROM expenses WHERE id = ? AND trip_id = ?`, expenseID, tripID)
	if err == nil {
		_ = r.logChange(ctx, tripID, "expense", expenseID, "delete", map[string]any{})
	}
	return err
}

func (r *Repository) GetExpense(ctx context.Context, tripID, expenseID string) (trips.Expense, error) {
	var e trips.Expense
	var ft int
	err := r.queryRowContext(ctx, `
		SELECT id, trip_id, category, amount_cents, notes, spent_on, payment_method, lodging_id, from_tab, receipt_path, title, paid_by, split_mode, split_json, due_at, created_at, updated_at
		FROM expenses WHERE id = ? AND trip_id = ?`, expenseID, tripID,
	).Scan(&e.ID, &e.TripID, &e.Category, &e.AmountCents, &e.Notes, &e.SpentOn, &e.PaymentMethod, &e.LodgingID, &ft, &e.ReceiptPath,
		&e.Title, &e.PaidBy, &e.SplitMode, &e.SplitJSON, &e.DueAt, &e.CreatedAt, &e.UpdatedAt)
	e.FromTab = ft != 0
	trips.SetExpenseAmountCents(&e, e.AmountCents)
	return e, err
}

func (r *Repository) GetExpenseByLodgingID(ctx context.Context, tripID, lodgingID string) (trips.Expense, error) {
	var e trips.Expense
	var ft int
	err := r.queryRowContext(ctx, `
		SELECT id, trip_id, category, amount_cents, notes, spent_on, payment_method, lodging_id, from_tab, receipt_path, title, paid_by, split_mode, split_json, due_at, created_at, updated_at
		FROM expenses WHERE trip_id = ? AND lodging_id = ?`, tripID, lodgingID,
	).Scan(&e.ID, &e.TripID, &e.Category, &e.AmountCents, &e.Notes, &e.SpentOn, &e.PaymentMethod, &e.LodgingID, &ft, &e.ReceiptPath,
		&e.Title, &e.PaidBy, &e.SplitMode, &e.SplitJSON, &e.DueAt, &e.CreatedAt, &e.UpdatedAt)
	e.FromTab = ft != 0
	trips.SetExpenseAmountCents(&e, e.AmountCents)
	return e, err
}

func (r *Repository) DeleteExpenseByLodgingID(ctx context.Context, tripID, lodgingID string) error {
	res, err := r.execContext(ctx, `DELETE FROM expenses WHERE trip_id = ? AND lodging_id = ?`, tripID, lodgingID)
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
		SELECT id, trip_id, category, amount_cents, notes, spent_on, payment_method, lodging_id, from_tab, receipt_path, title, paid_by, split_mode, split_json, due_at, created_at, updated_at
		FROM expenses WHERE trip_id = ? ORDER BY created_at DESC`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []trips.Expense{}
	for rows.Next() {
		var e trips.Expense
		var ft int
		if err := rows.Scan(&e.ID, &e.TripID, &e.Category, &e.AmountCents, &e.Notes, &e.SpentOn, &e.PaymentMethod, &e.LodgingID, &ft, &e.ReceiptPath,
			&e.Title, &e.PaidBy, &e.SplitMode, &e.SplitJSON, &e.DueAt, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		e.FromTab = ft != 0
		trips.SetExpenseAmountCents(&e, e.AmountCents)
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) SumExpensesByTrip(ctx context.Context) (map[string]float64, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT trip_id, COALESCE(SUM(amount_cents), 0) FROM expenses GROUP BY trip_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := make(map[string]float64)
	for rows.Next() {
		var tripID string
		var sumCents int64
		if err := rows.Scan(&tripID, &sumCents); err != nil {
			return nil, err
		}
		out[tripID] = trips.MoneyFromCents(sumCents)
	}
	return out, rows.Err()
}

func (r *Repository) AddChecklistItem(ctx context.Context, item trips.ChecklistItem) error {
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO checklist_items (id, trip_id, category, text, done, due_at, created_at, updated_at, archived, trashed)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.TripID, item.Category, item.Text, item.Done, strings.TrimSpace(item.DueAt), now, now,
		sqliteBool(item.Archived), sqliteBool(item.Trashed),
	)
	if err == nil {
		pl := map[string]any{
			"text": item.Text, "category": item.Category, "done": item.Done,
			"archived": item.Archived, "trashed": item.Trashed,
		}
		if strings.TrimSpace(item.DueAt) != "" {
			pl["due_at"] = strings.TrimSpace(item.DueAt)
		}
		_ = r.logChange(ctx, item.TripID, "checklist_item", item.ID, "create", pl)
	}
	return err
}

func (r *Repository) GetChecklistItem(ctx context.Context, itemID string) (trips.ChecklistItem, error) {
	var i trips.ChecklistItem
	var archived, trashed int64
	err := r.db.QueryRowContext(ctx, `
		SELECT id, trip_id, category, text, done, due_at, created_at, updated_at, archived, trashed
		FROM checklist_items WHERE id = ?`, itemID).
		Scan(&i.ID, &i.TripID, &i.Category, &i.Text, &i.Done, &i.DueAt, &i.CreatedAt, &i.UpdatedAt, &archived, &trashed)
	i.Archived = scanBoolFromInt(archived)
	i.Trashed = scanBoolFromInt(trashed)
	return i, err
}

func (r *Repository) UpdateChecklistItem(ctx context.Context, item trips.ChecklistItem) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		UPDATE checklist_items
		SET category = ?, text = ?, due_at = ?, archived = ?, trashed = ?, updated_at = ?
		WHERE id = ?`,
		item.Category, item.Text, strings.TrimSpace(item.DueAt),
		sqliteBool(item.Archived), sqliteBool(item.Trashed), now, item.ID,
	)
	if err == nil {
		_ = r.logChange(ctx, item.TripID, "checklist_item", item.ID, "update", map[string]any{
			"text": item.Text, "category": item.Category, "due_at": strings.TrimSpace(item.DueAt),
			"done": item.Done, "archived": item.Archived, "trashed": item.Trashed,
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
	now := time.Now().UTC()
	res, err := r.db.ExecContext(ctx, `
		UPDATE checklist_items
		SET done = ?, updated_at = ?
		WHERE id = ? AND trip_id IN (SELECT id FROM trips WHERE is_archived = FALSE)`,
		done, now, itemID,
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
		SELECT id, trip_id, category, text, done, due_at, created_at, updated_at, archived, trashed
		FROM checklist_items
		WHERE trip_id = ? AND archived = 0 AND trashed = 0
		ORDER BY created_at DESC`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []trips.ChecklistItem{}
	for rows.Next() {
		var i trips.ChecklistItem
		var archived, trashed int64
		if err := rows.Scan(&i.ID, &i.TripID, &i.Category, &i.Text, &i.Done, &i.DueAt, &i.CreatedAt, &i.UpdatedAt, &archived, &trashed); err != nil {
			return nil, err
		}
		i.Archived = scanBoolFromInt(archived)
		i.Trashed = scanBoolFromInt(trashed)
		out = append(out, i)
	}
	return out, rows.Err()
}

func (r *Repository) AddLodging(ctx context.Context, item trips.Lodging) error {
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.CostCents == 0 && item.Cost != 0 {
		item.CostCents = trips.MoneyToCentsFloat(item.Cost)
	}
	trips.SetLodgingCostCents(&item, item.CostCents)
	now := time.Now().UTC()
	_, err := r.execContext(ctx, `
		INSERT INTO lodging_entries
		(id, trip_id, name, address, check_in_at, check_out_at, booking_confirmation, cost, cost_cents, notes, attachment_path, image_path, check_in_itinerary_id, check_out_itinerary_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.TripID, item.Name, item.Address, item.CheckInAt, item.CheckOutAt, item.BookingConfirmation, item.Cost, item.CostCents, item.Notes, item.AttachmentPath, item.ImagePath,
		item.CheckInItineraryID, item.CheckOutItineraryID, now, now,
	)
	if err == nil {
		item.CreatedAt = now
		item.UpdatedAt = now
		_ = r.logChange(ctx, item.TripID, "lodging", item.ID, "create", map[string]any{
			"name": item.Name,
		})
	}
	return err
}

func (r *Repository) GetLodging(ctx context.Context, tripID, lodgingID string) (trips.Lodging, error) {
	var l trips.Lodging
	err := r.queryRowContext(ctx, `
		SELECT id, trip_id, name, address, check_in_at, check_out_at, booking_confirmation, cost_cents, notes, attachment_path, image_path, check_in_itinerary_id, check_out_itinerary_id, created_at, updated_at
		FROM lodging_entries WHERE id = ? AND trip_id = ?`,
		lodgingID, tripID,
	).Scan(&l.ID, &l.TripID, &l.Name, &l.Address, &l.CheckInAt, &l.CheckOutAt, &l.BookingConfirmation, &l.CostCents, &l.Notes, &l.AttachmentPath, &l.ImagePath,
		&l.CheckInItineraryID, &l.CheckOutItineraryID, &l.CreatedAt, &l.UpdatedAt)
	trips.SetLodgingCostCents(&l, l.CostCents)
	return l, err
}

func (r *Repository) UpdateLodging(ctx context.Context, item trips.Lodging) error {
	if item.CostCents == 0 && item.Cost != 0 {
		item.CostCents = trips.MoneyToCentsFloat(item.Cost)
	}
	trips.SetLodgingCostCents(&item, item.CostCents)
	now := time.Now().UTC()
	var (
		res sql.Result
		err error
	)
	if item.EnforceOptimisticLock {
		res, err = r.execContext(ctx, `
			UPDATE lodging_entries
			SET name = ?, address = ?, check_in_at = ?, check_out_at = ?, booking_confirmation = ?, cost = ?, cost_cents = ?, notes = ?, attachment_path = ?, image_path = ?, check_in_itinerary_id = ?, check_out_itinerary_id = ?, updated_at = ?
			WHERE id = ? AND trip_id = ? AND updated_at = ?`,
			item.Name, item.Address, item.CheckInAt, item.CheckOutAt, item.BookingConfirmation, item.Cost, item.CostCents, item.Notes, item.AttachmentPath, item.ImagePath,
			item.CheckInItineraryID, item.CheckOutItineraryID, now,
			item.ID, item.TripID, item.ExpectedUpdatedAt,
		)
	} else {
		res, err = r.execContext(ctx, `
			UPDATE lodging_entries
			SET name = ?, address = ?, check_in_at = ?, check_out_at = ?, booking_confirmation = ?, cost = ?, cost_cents = ?, notes = ?, attachment_path = ?, image_path = ?, check_in_itinerary_id = ?, check_out_itinerary_id = ?, updated_at = ?
			WHERE id = ? AND trip_id = ?`,
			item.Name, item.Address, item.CheckInAt, item.CheckOutAt, item.BookingConfirmation, item.Cost, item.CostCents, item.Notes, item.AttachmentPath, item.ImagePath,
			item.CheckInItineraryID, item.CheckOutItineraryID, now,
			item.ID, item.TripID,
		)
	}
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		current, getErr := r.GetLodging(ctx, item.TripID, item.ID)
		if getErr == nil {
			return &trips.ConflictError{
				Resource:        "lodging",
				Message:         "Someone else updated this accommodation a moment ago. Reopen it to review the latest details, then try again.",
				LatestUpdatedAt: current.UpdatedAt,
			}
		}
		if errors.Is(getErr, sql.ErrNoRows) {
			return sql.ErrNoRows
		}
		return getErr
	}
	item.UpdatedAt = now
	if err == nil {
		_ = r.logChange(ctx, item.TripID, "lodging", item.ID, "update", map[string]any{
			"name": item.Name,
		})
	}
	return err
}

func (r *Repository) DeleteLodging(ctx context.Context, tripID, lodgingID string) error {
	_, err := r.execContext(ctx, `DELETE FROM lodging_entries WHERE id = ? AND trip_id = ?`, lodgingID, tripID)
	if err == nil {
		_ = r.logChange(ctx, tripID, "lodging", lodgingID, "delete", map[string]any{})
	}
	return err
}

func (r *Repository) ListLodgings(ctx context.Context, tripID string) ([]trips.Lodging, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, trip_id, name, address, check_in_at, check_out_at, booking_confirmation, cost_cents, notes, attachment_path, image_path, check_in_itinerary_id, check_out_itinerary_id, created_at, updated_at
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
		if err := rows.Scan(&l.ID, &l.TripID, &l.Name, &l.Address, &l.CheckInAt, &l.CheckOutAt, &l.BookingConfirmation, &l.CostCents, &l.Notes, &l.AttachmentPath, &l.ImagePath,
			&l.CheckInItineraryID, &l.CheckOutItineraryID, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		trips.SetLodgingCostCents(&l, l.CostCents)
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *Repository) AddVehicleRental(ctx context.Context, item trips.VehicleRental) error {
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.CostCents == 0 && item.Cost != 0 {
		item.CostCents = trips.MoneyToCentsFloat(item.Cost)
	}
	if item.InsuranceCostCents == 0 && item.InsuranceCost != 0 {
		item.InsuranceCostCents = trips.MoneyToCentsFloat(item.InsuranceCost)
	}
	trips.SetVehicleCostCents(&item, item.CostCents)
	trips.SetVehicleInsuranceCostCents(&item, item.InsuranceCostCents)
	now := time.Now().UTC()
	_, err := r.execContext(ctx, `
		INSERT INTO vehicle_rentals
		(id, trip_id, pick_up_location, drop_off_location, vehicle_detail, pick_up_at, drop_off_at, booking_confirmation, notes, attachment_path, vehicle_image_path, cost, cost_cents, insurance_cost, insurance_cost_cents, pay_at_pick_up, pick_up_itinerary_id, drop_off_itinerary_id, rental_expense_id, insurance_expense_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.TripID, item.PickUpLocation, item.DropOffLocation, item.VehicleDetail, item.PickUpAt, item.DropOffAt, item.BookingConfirmation, item.Notes, item.AttachmentPath, item.VehicleImagePath, item.Cost, item.CostCents, item.InsuranceCost, item.InsuranceCostCents, item.PayAtPickUp,
		item.PickUpItineraryID, item.DropOffItineraryID, item.RentalExpenseID, item.InsuranceExpenseID, now, now,
	)
	if err == nil {
		item.CreatedAt = now
		item.UpdatedAt = now
		_ = r.logChange(ctx, item.TripID, "vehicle_rental", item.ID, "create", map[string]any{
			"vehicle_detail":   item.VehicleDetail,
			"pick_up_location": item.PickUpLocation,
		})
	}
	return err
}

func (r *Repository) GetVehicleRental(ctx context.Context, tripID, rentalID string) (trips.VehicleRental, error) {
	var v trips.VehicleRental
	err := r.queryRowContext(ctx, `
		SELECT id, trip_id, pick_up_location, drop_off_location, vehicle_detail, pick_up_at, drop_off_at, booking_confirmation, notes, attachment_path, vehicle_image_path, cost_cents, insurance_cost_cents, pay_at_pick_up, pick_up_itinerary_id, drop_off_itinerary_id, rental_expense_id, insurance_expense_id, created_at, updated_at
		FROM vehicle_rentals WHERE id = ? AND trip_id = ?`,
		rentalID, tripID,
	).Scan(&v.ID, &v.TripID, &v.PickUpLocation, &v.DropOffLocation, &v.VehicleDetail, &v.PickUpAt, &v.DropOffAt, &v.BookingConfirmation, &v.Notes, &v.AttachmentPath, &v.VehicleImagePath, &v.CostCents, &v.InsuranceCostCents, &v.PayAtPickUp,
		&v.PickUpItineraryID, &v.DropOffItineraryID, &v.RentalExpenseID, &v.InsuranceExpenseID, &v.CreatedAt, &v.UpdatedAt)
	trips.SetVehicleCostCents(&v, v.CostCents)
	trips.SetVehicleInsuranceCostCents(&v, v.InsuranceCostCents)
	return v, err
}

func (r *Repository) UpdateVehicleRental(ctx context.Context, item trips.VehicleRental) error {
	if item.CostCents == 0 && item.Cost != 0 {
		item.CostCents = trips.MoneyToCentsFloat(item.Cost)
	}
	if item.InsuranceCostCents == 0 && item.InsuranceCost != 0 {
		item.InsuranceCostCents = trips.MoneyToCentsFloat(item.InsuranceCost)
	}
	trips.SetVehicleCostCents(&item, item.CostCents)
	trips.SetVehicleInsuranceCostCents(&item, item.InsuranceCostCents)
	now := time.Now().UTC()
	var (
		res sql.Result
		err error
	)
	if item.EnforceOptimisticLock {
		res, err = r.execContext(ctx, `
			UPDATE vehicle_rentals
			SET pick_up_location = ?, drop_off_location = ?, vehicle_detail = ?, pick_up_at = ?, drop_off_at = ?, booking_confirmation = ?, notes = ?, attachment_path = ?, vehicle_image_path = ?, cost = ?, cost_cents = ?, insurance_cost = ?, insurance_cost_cents = ?, pay_at_pick_up = ?, pick_up_itinerary_id = ?, drop_off_itinerary_id = ?, rental_expense_id = ?, insurance_expense_id = ?, updated_at = ?
			WHERE id = ? AND trip_id = ? AND updated_at = ?`,
			item.PickUpLocation, item.DropOffLocation, item.VehicleDetail, item.PickUpAt, item.DropOffAt, item.BookingConfirmation, item.Notes, item.AttachmentPath, item.VehicleImagePath, item.Cost, item.CostCents, item.InsuranceCost, item.InsuranceCostCents, item.PayAtPickUp,
			item.PickUpItineraryID, item.DropOffItineraryID, item.RentalExpenseID, item.InsuranceExpenseID, now,
			item.ID, item.TripID, item.ExpectedUpdatedAt,
		)
	} else {
		res, err = r.execContext(ctx, `
			UPDATE vehicle_rentals
			SET pick_up_location = ?, drop_off_location = ?, vehicle_detail = ?, pick_up_at = ?, drop_off_at = ?, booking_confirmation = ?, notes = ?, attachment_path = ?, vehicle_image_path = ?, cost = ?, cost_cents = ?, insurance_cost = ?, insurance_cost_cents = ?, pay_at_pick_up = ?, pick_up_itinerary_id = ?, drop_off_itinerary_id = ?, rental_expense_id = ?, insurance_expense_id = ?, updated_at = ?
			WHERE id = ? AND trip_id = ?`,
			item.PickUpLocation, item.DropOffLocation, item.VehicleDetail, item.PickUpAt, item.DropOffAt, item.BookingConfirmation, item.Notes, item.AttachmentPath, item.VehicleImagePath, item.Cost, item.CostCents, item.InsuranceCost, item.InsuranceCostCents, item.PayAtPickUp,
			item.PickUpItineraryID, item.DropOffItineraryID, item.RentalExpenseID, item.InsuranceExpenseID, now,
			item.ID, item.TripID,
		)
	}
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		current, getErr := r.GetVehicleRental(ctx, item.TripID, item.ID)
		if getErr == nil {
			return &trips.ConflictError{
				Resource:        "vehicle_rental",
				Message:         "Someone else updated this vehicle rental a moment ago. Reopen it to review the latest details, then try again.",
				LatestUpdatedAt: current.UpdatedAt,
			}
		}
		if errors.Is(getErr, sql.ErrNoRows) {
			return sql.ErrNoRows
		}
		return getErr
	}
	item.UpdatedAt = now
	if err == nil {
		_ = r.logChange(ctx, item.TripID, "vehicle_rental", item.ID, "update", map[string]any{
			"vehicle_detail":   item.VehicleDetail,
			"pick_up_location": item.PickUpLocation,
		})
	}
	return err
}

func (r *Repository) DeleteVehicleRental(ctx context.Context, tripID, rentalID string) error {
	_, err := r.execContext(ctx, `DELETE FROM vehicle_rentals WHERE id = ? AND trip_id = ?`, rentalID, tripID)
	if err == nil {
		_ = r.logChange(ctx, tripID, "vehicle_rental", rentalID, "delete", map[string]any{})
	}
	return err
}

func (r *Repository) ListVehicleRentals(ctx context.Context, tripID string) ([]trips.VehicleRental, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, trip_id, pick_up_location, drop_off_location, vehicle_detail, pick_up_at, drop_off_at, booking_confirmation, notes, attachment_path, vehicle_image_path, cost_cents, insurance_cost_cents, pay_at_pick_up, pick_up_itinerary_id, drop_off_itinerary_id, rental_expense_id, insurance_expense_id, created_at, updated_at
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
		if err := rows.Scan(&v.ID, &v.TripID, &v.PickUpLocation, &v.DropOffLocation, &v.VehicleDetail, &v.PickUpAt, &v.DropOffAt, &v.BookingConfirmation, &v.Notes, &v.AttachmentPath, &v.VehicleImagePath, &v.CostCents, &v.InsuranceCostCents, &v.PayAtPickUp,
			&v.PickUpItineraryID, &v.DropOffItineraryID, &v.RentalExpenseID, &v.InsuranceExpenseID, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		trips.SetVehicleCostCents(&v, v.CostCents)
		trips.SetVehicleInsuranceCostCents(&v, v.InsuranceCostCents)
		out = append(out, v)
	}
	return out, rows.Err()
}

func (r *Repository) AddFlight(ctx context.Context, item trips.Flight) error {
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	if item.CostCents == 0 && item.Cost != 0 {
		item.CostCents = trips.MoneyToCentsFloat(item.Cost)
	}
	trips.SetFlightCostCents(&item, item.CostCents)
	now := time.Now().UTC()
	_, err := r.execContext(ctx, `
		INSERT INTO flight_entries
		(id, trip_id, flight_name, flight_number, depart_airport, arrive_airport, depart_at, arrive_at, booking_confirmation, notes, document_path, image_path, cost, cost_cents, depart_itinerary_id, arrive_itinerary_id, expense_id, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.TripID, item.FlightName, item.FlightNumber, item.DepartAirport, item.ArriveAirport, item.DepartAt, item.ArriveAt, item.BookingConfirmation, item.Notes, item.DocumentPath, item.ImagePath, item.Cost, item.CostCents, item.DepartItineraryID, item.ArriveItineraryID, item.ExpenseID, now, now,
	)
	if err == nil {
		item.CreatedAt = now
		item.UpdatedAt = now
		_ = r.logChange(ctx, item.TripID, "flight", item.ID, "create", map[string]any{
			"flight_name":   item.FlightName,
			"flight_number": item.FlightNumber,
		})
	}
	return err
}

func (r *Repository) GetFlight(ctx context.Context, tripID, flightID string) (trips.Flight, error) {
	var f trips.Flight
	err := r.queryRowContext(ctx, `
		SELECT id, trip_id, flight_name, flight_number, depart_airport, arrive_airport, depart_at, arrive_at, booking_confirmation, notes, document_path, image_path, cost_cents, depart_itinerary_id, arrive_itinerary_id, expense_id, created_at, updated_at
		FROM flight_entries WHERE id = ? AND trip_id = ?`,
		flightID, tripID,
	).Scan(
		&f.ID, &f.TripID, &f.FlightName, &f.FlightNumber, &f.DepartAirport, &f.ArriveAirport, &f.DepartAt, &f.ArriveAt, &f.BookingConfirmation, &f.Notes, &f.DocumentPath, &f.ImagePath, &f.CostCents, &f.DepartItineraryID, &f.ArriveItineraryID, &f.ExpenseID, &f.CreatedAt, &f.UpdatedAt,
	)
	trips.SetFlightCostCents(&f, f.CostCents)
	return f, err
}

func (r *Repository) UpdateFlight(ctx context.Context, item trips.Flight) error {
	if item.CostCents == 0 && item.Cost != 0 {
		item.CostCents = trips.MoneyToCentsFloat(item.Cost)
	}
	trips.SetFlightCostCents(&item, item.CostCents)
	now := time.Now().UTC()
	var (
		res sql.Result
		err error
	)
	if item.EnforceOptimisticLock {
		res, err = r.execContext(ctx, `
			UPDATE flight_entries
			SET flight_name = ?, flight_number = ?, depart_airport = ?, arrive_airport = ?, depart_at = ?, arrive_at = ?, booking_confirmation = ?, notes = ?, document_path = ?, image_path = ?, cost = ?, cost_cents = ?, depart_itinerary_id = ?, arrive_itinerary_id = ?, expense_id = ?, updated_at = ?
			WHERE id = ? AND trip_id = ? AND updated_at = ?`,
			item.FlightName, item.FlightNumber, item.DepartAirport, item.ArriveAirport, item.DepartAt, item.ArriveAt, item.BookingConfirmation, item.Notes, item.DocumentPath, item.ImagePath, item.Cost, item.CostCents, item.DepartItineraryID, item.ArriveItineraryID, item.ExpenseID, now,
			item.ID, item.TripID, item.ExpectedUpdatedAt,
		)
	} else {
		res, err = r.execContext(ctx, `
			UPDATE flight_entries
			SET flight_name = ?, flight_number = ?, depart_airport = ?, arrive_airport = ?, depart_at = ?, arrive_at = ?, booking_confirmation = ?, notes = ?, document_path = ?, image_path = ?, cost = ?, cost_cents = ?, depart_itinerary_id = ?, arrive_itinerary_id = ?, expense_id = ?, updated_at = ?
			WHERE id = ? AND trip_id = ?`,
			item.FlightName, item.FlightNumber, item.DepartAirport, item.ArriveAirport, item.DepartAt, item.ArriveAt, item.BookingConfirmation, item.Notes, item.DocumentPath, item.ImagePath, item.Cost, item.CostCents, item.DepartItineraryID, item.ArriveItineraryID, item.ExpenseID, now,
			item.ID, item.TripID,
		)
	}
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		current, getErr := r.GetFlight(ctx, item.TripID, item.ID)
		if getErr == nil {
			return &trips.ConflictError{
				Resource:        "flight",
				Message:         "Someone else updated this flight a moment ago. Reopen it to review the latest details, then try again.",
				LatestUpdatedAt: current.UpdatedAt,
			}
		}
		if errors.Is(getErr, sql.ErrNoRows) {
			return sql.ErrNoRows
		}
		return getErr
	}
	item.UpdatedAt = now
	if err == nil {
		_ = r.logChange(ctx, item.TripID, "flight", item.ID, "update", map[string]any{
			"flight_name":   item.FlightName,
			"flight_number": item.FlightNumber,
		})
	}
	return err
}

func (r *Repository) DeleteFlight(ctx context.Context, tripID, flightID string) error {
	_, err := r.execContext(ctx, `DELETE FROM flight_entries WHERE id = ? AND trip_id = ?`, flightID, tripID)
	if err == nil {
		_ = r.logChange(ctx, tripID, "flight", flightID, "delete", map[string]any{})
	}
	return err
}

func (r *Repository) ListFlights(ctx context.Context, tripID string) ([]trips.Flight, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, trip_id, flight_name, flight_number, depart_airport, arrive_airport, depart_at, arrive_at, booking_confirmation, notes, document_path, image_path, cost_cents, depart_itinerary_id, arrive_itinerary_id, expense_id, created_at, updated_at
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
			&f.ID, &f.TripID, &f.FlightName, &f.FlightNumber, &f.DepartAirport, &f.ArriveAirport, &f.DepartAt, &f.ArriveAt, &f.BookingConfirmation, &f.Notes, &f.DocumentPath, &f.ImagePath, &f.CostCents, &f.DepartItineraryID, &f.ArriveItineraryID, &f.ExpenseID, &f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			return nil, err
		}
		trips.SetFlightCostCents(&f, f.CostCents)
		out = append(out, f)
	}
	return out, rows.Err()
}

func (r *Repository) ListTripDocuments(ctx context.Context, tripID string) ([]trips.TripDocument, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, trip_id, section, category, item_name, file_name, display_name, file_path, file_size, uploaded_at
		FROM trip_documents
		WHERE trip_id = ?
		ORDER BY uploaded_at DESC, id DESC`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []trips.TripDocument{}
	for rows.Next() {
		var d trips.TripDocument
		if err := rows.Scan(&d.ID, &d.TripID, &d.Section, &d.Category, &d.ItemName, &d.FileName, &d.DisplayName, &d.FilePath, &d.FileSize, &d.UploadedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repository) AddTripDocument(ctx context.Context, d trips.TripDocument) error {
	if d.ID == "" {
		d.ID = uuid.NewString()
	}
	if d.UploadedAt.IsZero() {
		d.UploadedAt = time.Now().UTC()
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO trip_documents
		(id, trip_id, section, category, item_name, file_name, display_name, file_path, file_size, uploaded_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		d.ID, d.TripID, d.Section, d.Category, d.ItemName, d.FileName, d.DisplayName, d.FilePath, d.FileSize, d.UploadedAt,
	)
	if err == nil {
		_ = r.logChange(ctx, d.TripID, "trip_document", d.ID, "create", map[string]any{
			"section":      d.Section,
			"display_name": d.DisplayName,
		})
	}
	return err
}

func (r *Repository) UpdateTripDocumentDisplayName(ctx context.Context, tripID, documentID, displayName string) error {
	res, err := r.db.ExecContext(ctx, `
		UPDATE trip_documents SET display_name = ? WHERE id = ? AND trip_id = ? AND section = 'general'`,
		displayName, documentID, tripID)
	if err != nil {
		return err
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	_ = r.logChange(ctx, tripID, "trip_document", documentID, "update", map[string]any{
		"display_name": displayName,
	})
	return nil
}

func (r *Repository) DeleteTripDocument(ctx context.Context, tripID, documentID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM trip_documents WHERE id = ? AND trip_id = ?`, documentID, tripID)
	if err == nil {
		_ = r.logChange(ctx, tripID, "trip_document", documentID, "delete", map[string]any{})
	}
	return err
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

func (r *Repository) ListChangesAfterID(ctx context.Context, tripID string, afterID int64) ([]trips.Change, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, trip_id, entity, entity_id, operation, changed_at, payload
		FROM change_log
		WHERE trip_id = ? AND id > ?
		ORDER BY id ASC LIMIT 500`, tripID, afterID)
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

func (r *Repository) LatestChangeLogID(ctx context.Context, tripID string) (int64, error) {
	var id sql.NullInt64
	err := r.db.QueryRowContext(ctx, `SELECT MAX(id) FROM change_log WHERE trip_id = ?`, tripID).Scan(&id)
	if err != nil {
		return 0, err
	}
	if !id.Valid {
		return 0, nil
	}
	return id.Int64, nil
}

func (r *Repository) logChange(ctx context.Context, tripID, entity, entityID, operation string, payload map[string]any) error {
	b, _ := json.Marshal(payload)
	_, err := r.execContext(ctx, `
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
		       COALESCE(TRIM(google_maps_map_id), ''),
		       COALESCE(NULLIF(TRIM(default_distance_unit), ''), 'km'),
		       COALESCE(max_upload_file_size_mb, 5),
		       COALESCE(NULLIF(TRIM(default_ui_date_format), ''), 'dmy')
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
			&settings.GoogleMapsMapID,
			&settings.DefaultDistanceUnit,
			&settings.MaxUploadFileSizeMB,
			&settings.DefaultUIDateFormat,
		)
	if settings.DefaultUIDateFormat != "" {
		settings.DefaultUIDateFormat = trips.NormalizeUIDateFormat(settings.DefaultUIDateFormat)
	}
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
			 registration_enabled, theme_preference, dashboard_trip_layout, dashboard_trip_sort, dashboard_hero_background, trip_dashboard_heading, google_maps_api_key, google_maps_map_id, default_distance_unit, max_upload_file_size_mb, default_ui_date_format, updated_at)
		VALUES
			(1, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
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
			google_maps_map_id = excluded.google_maps_map_id,
			default_distance_unit = excluded.default_distance_unit,
			max_upload_file_size_mb = excluded.max_upload_file_size_mb,
			default_ui_date_format = excluded.default_ui_date_format,
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
		settings.GoogleMapsMapID,
		settings.DefaultDistanceUnit,
		settings.MaxUploadFileSizeMB,
		trips.NormalizeUIDateFormat(settings.DefaultUIDateFormat),
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
		if err == nil {
			_ = r.logChange(ctx, tripID, "trip_day_label", strconv.Itoa(dayNumber), "delete", map[string]any{})
		}
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
	if err == nil {
		_ = r.logChange(ctx, tripID, "trip_day_label", strconv.Itoa(dayNumber), "update", map[string]any{
			"label": trimmed,
		})
	}
	return err
}

func (r *Repository) syncTabExpenseFTS(ctx context.Context, e trips.Expense) error {
	_, _ = r.execContext(ctx, `DELETE FROM tab_expense_search WHERE expense_id = ?`, e.ID)
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
	_, err := r.execContext(ctx, `INSERT INTO tab_expense_search(trip_id, expense_id, body) VALUES(?,?,?)`, e.TripID, e.ID, body)
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
	if err == nil {
		_ = r.logChange(ctx, tripID, "trip_tab_defaults", tripID, "update", map[string]any{
			"split_mode": strings.TrimSpace(mode),
		})
	}
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
	if err == nil {
		_ = r.logChange(ctx, g.TripID, "trip_guest", g.ID, "create", map[string]any{
			"display_name": strings.TrimSpace(g.DisplayName),
		})
	}
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
	_ = r.logChange(ctx, tripID, "trip_guest", guestID, "delete", map[string]any{})
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
		SELECT id, trip_id, payer_user_id, payee_user_id, amount_cents, method, settled_on, notes, created_at
		FROM tab_settlements WHERE trip_id = ? ORDER BY created_at DESC`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []trips.TabSettlement
	for rows.Next() {
		var s trips.TabSettlement
		if err := rows.Scan(&s.ID, &s.TripID, &s.PayerUserID, &s.PayeeUserID, &s.AmountCents, &s.Method, &s.SettledOn, &s.Notes, &s.CreatedAt); err != nil {
			return nil, err
		}
		trips.SetTabSettlementAmountCents(&s, s.AmountCents)
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
	if s.AmountCents == 0 && s.Amount != 0 {
		s.AmountCents = trips.MoneyToCentsFloat(s.Amount)
	}
	trips.SetTabSettlementAmountCents(&s, s.AmountCents)
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO tab_settlements (id, trip_id, payer_user_id, payee_user_id, amount, amount_cents, method, settled_on, notes, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ID, s.TripID, s.PayerUserID, s.PayeeUserID, s.Amount, s.AmountCents, s.Method, s.SettledOn, s.Notes, now,
	)
	if err == nil {
		_ = r.logChange(ctx, s.TripID, "tab_settlement", s.ID, "create", map[string]any{
			"amount": s.Amount,
		})
	}
	return err
}

func (r *Repository) UpdateTabSettlement(ctx context.Context, s trips.TabSettlement) error {
	if s.AmountCents == 0 && s.Amount != 0 {
		s.AmountCents = trips.MoneyToCentsFloat(s.Amount)
	}
	trips.SetTabSettlementAmountCents(&s, s.AmountCents)
	res, err := r.db.ExecContext(ctx, `
		UPDATE tab_settlements SET payer_user_id = ?, payee_user_id = ?, amount = ?, amount_cents = ?, method = ?, settled_on = ?, notes = ?
		WHERE id = ? AND trip_id = ?`,
		s.PayerUserID, s.PayeeUserID, s.Amount, s.AmountCents, s.Method, s.SettledOn, s.Notes, s.ID, s.TripID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	_ = r.logChange(ctx, s.TripID, "tab_settlement", s.ID, "update", map[string]any{
		"amount": s.Amount,
	})
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
	_ = r.logChange(ctx, tripID, "tab_settlement", settlementID, "delete", map[string]any{})
	return nil
}
