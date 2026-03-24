package sqlite

import (
	"context"
	"database/sql"
	"encoding/json"
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

func (r *Repository) CreateTrip(ctx context.Context, t trips.Trip) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO trips (id, name, description, start_date, end_date, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.Name, t.Description, t.StartDate, t.EndDate, now, now,
	)
	if err == nil {
		_ = r.logChange(ctx, t.ID, "trip", t.ID, "create", map[string]any{"name": t.Name})
	}
	return err
}

func (r *Repository) ListTrips(ctx context.Context) ([]trips.Trip, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, name, description, start_date, end_date, created_at, updated_at
		FROM trips ORDER BY created_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []trips.Trip{}
	for rows.Next() {
		var t trips.Trip
		if err := rows.Scan(&t.ID, &t.Name, &t.Description, &t.StartDate, &t.EndDate, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repository) GetTrip(ctx context.Context, tripID string) (trips.Trip, error) {
	var t trips.Trip
	err := r.db.QueryRowContext(ctx, `
		SELECT id, name, description, start_date, end_date, created_at, updated_at
		FROM trips WHERE id = ?`, tripID).
		Scan(&t.ID, &t.Name, &t.Description, &t.StartDate, &t.EndDate, &t.CreatedAt, &t.UpdatedAt)
	return t, err
}

func (r *Repository) AddItineraryItem(ctx context.Context, item trips.ItineraryItem) error {
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO itinerary_items
		(id, trip_id, day_number, order_index, title, notes, location, latitude, longitude, est_cost, start_time, end_time, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.TripID, item.DayNumber, item.OrderIndex, item.Title, item.Notes, item.Location, item.Latitude, item.Longitude, item.EstCost, item.StartTime, item.EndTime, now, now,
	)
	if err == nil {
		_ = r.logChange(ctx, item.TripID, "itinerary_item", item.ID, "create", map[string]any{
			"title": item.Title, "day_number": item.DayNumber,
		})
	}
	return err
}

func (r *Repository) ListItineraryItems(ctx context.Context, tripID string) ([]trips.ItineraryItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, trip_id, day_number, order_index, title, notes, location, latitude, longitude, est_cost, start_time, end_time, created_at, updated_at
		FROM itinerary_items WHERE trip_id = ?
		ORDER BY day_number ASC, order_index ASC, created_at ASC`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []trips.ItineraryItem{}
	for rows.Next() {
		var i trips.ItineraryItem
		if err := rows.Scan(&i.ID, &i.TripID, &i.DayNumber, &i.OrderIndex, &i.Title, &i.Notes, &i.Location, &i.Latitude, &i.Longitude, &i.EstCost, &i.StartTime, &i.EndTime, &i.CreatedAt, &i.UpdatedAt); err != nil {
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
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO expenses (id, trip_id, category, amount, notes, spent_on, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		e.ID, e.TripID, e.Category, e.Amount, e.Notes, e.SpentOn, now,
	)
	if err == nil {
		_ = r.logChange(ctx, e.TripID, "expense", e.ID, "create", map[string]any{"amount": e.Amount, "category": e.Category})
	}
	return err
}

func (r *Repository) ListExpenses(ctx context.Context, tripID string) ([]trips.Expense, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, trip_id, category, amount, notes, spent_on, created_at
		FROM expenses WHERE trip_id = ? ORDER BY created_at DESC`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []trips.Expense{}
	for rows.Next() {
		var e trips.Expense
		if err := rows.Scan(&e.ID, &e.TripID, &e.Category, &e.Amount, &e.Notes, &e.SpentOn, &e.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

func (r *Repository) AddChecklistItem(ctx context.Context, item trips.ChecklistItem) error {
	if item.ID == "" {
		item.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO checklist_items (id, trip_id, text, done, created_at)
		VALUES (?, ?, ?, ?, ?)`,
		item.ID, item.TripID, item.Text, item.Done, now,
	)
	if err == nil {
		_ = r.logChange(ctx, item.TripID, "checklist_item", item.ID, "create", map[string]any{"text": item.Text})
	}
	return err
}

func (r *Repository) ToggleChecklistItem(ctx context.Context, itemID string, done bool) error {
	var tripID string
	_ = r.db.QueryRowContext(ctx, `SELECT trip_id FROM checklist_items WHERE id = ?`, itemID).Scan(&tripID)
	_, err := r.db.ExecContext(ctx, `UPDATE checklist_items SET done = ? WHERE id = ?`, done, itemID)
	if err == nil && tripID != "" {
		_ = r.logChange(ctx, tripID, "checklist_item", itemID, "update", map[string]any{"done": done})
	}
	return err
}

func (r *Repository) ListChecklistItems(ctx context.Context, tripID string) ([]trips.ChecklistItem, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, trip_id, text, done, created_at
		FROM checklist_items WHERE trip_id = ? ORDER BY created_at DESC`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []trips.ChecklistItem{}
	for rows.Next() {
		var i trips.ChecklistItem
		if err := rows.Scan(&i.ID, &i.TripID, &i.Text, &i.Done, &i.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, i)
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
