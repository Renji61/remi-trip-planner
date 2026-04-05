package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"

	"remi-trip-planner/internal/trips"
)

func (r *Repository) AddTripNote(ctx context.Context, n trips.TripNote) error {
	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO trip_notes (id, trip_id, title, body, color, pinned, archived, trashed, due_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		n.ID, n.TripID, strings.TrimSpace(n.Title), strings.TrimSpace(n.Body), strings.TrimSpace(n.Color),
		sqliteBool(n.Pinned), sqliteBool(n.Archived), sqliteBool(n.Trashed), strings.TrimSpace(n.DueAt), now, now,
	)
	if err == nil {
		_ = r.logChange(ctx, n.TripID, "trip_note", n.ID, "create", map[string]any{
			"title": n.Title, "body": n.Body, "color": n.Color,
			"pinned": n.Pinned, "archived": n.Archived, "trashed": n.Trashed, "due_at": strings.TrimSpace(n.DueAt),
		})
	}
	return err
}

func (r *Repository) GetTripNote(ctx context.Context, noteID string) (trips.TripNote, error) {
	var n trips.TripNote
	var pinned, archived, trashed int64
	err := r.db.QueryRowContext(ctx, `
		SELECT id, trip_id, title, body, color, pinned, archived, trashed, due_at, created_at, updated_at
		FROM trip_notes WHERE id = ?`, noteID).
		Scan(&n.ID, &n.TripID, &n.Title, &n.Body, &n.Color, &pinned, &archived, &trashed, &n.DueAt, &n.CreatedAt, &n.UpdatedAt)
	n.Pinned = scanBoolFromInt(pinned)
	n.Archived = scanBoolFromInt(archived)
	n.Trashed = scanBoolFromInt(trashed)
	return n, err
}

func (r *Repository) UpdateTripNote(ctx context.Context, n trips.TripNote) error {
	now := time.Now().UTC()
	res, err := r.db.ExecContext(ctx, `
		UPDATE trip_notes
		SET title = ?, body = ?, color = ?, pinned = ?, archived = ?, trashed = ?, due_at = ?, updated_at = ?
		WHERE id = ? AND trip_id = ?`,
		strings.TrimSpace(n.Title), strings.TrimSpace(n.Body), strings.TrimSpace(n.Color),
		sqliteBool(n.Pinned), sqliteBool(n.Archived), sqliteBool(n.Trashed), strings.TrimSpace(n.DueAt), now, n.ID, n.TripID,
	)
	if err != nil {
		return err
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		return sql.ErrNoRows
	}
	_ = r.logChange(ctx, n.TripID, "trip_note", n.ID, "update", map[string]any{
		"title": n.Title, "body": n.Body, "color": n.Color,
		"pinned": n.Pinned, "archived": n.Archived, "trashed": n.Trashed, "due_at": strings.TrimSpace(n.DueAt),
	})
	return nil
}

func (r *Repository) DeleteTripNote(ctx context.Context, noteID string) error {
	var tripID string
	_ = r.db.QueryRowContext(ctx, `SELECT trip_id FROM trip_notes WHERE id = ?`, noteID).Scan(&tripID)
	_, err := r.db.ExecContext(ctx, `DELETE FROM trip_notes WHERE id = ?`, noteID)
	if err == nil && tripID != "" {
		_ = r.logChange(ctx, tripID, "trip_note", noteID, "delete", map[string]any{})
	}
	return err
}

func (r *Repository) ListTripNotesForKeepView(ctx context.Context, tripID, view string) ([]trips.TripNote, error) {
	view = strings.TrimSpace(strings.ToLower(view))
	var q string
	switch view {
	case trips.KeepViewNotes:
		q = `SELECT id, trip_id, title, body, color, pinned, archived, trashed, due_at, created_at, updated_at
			FROM trip_notes WHERE trip_id = ? AND archived = 0 AND trashed = 0
			ORDER BY pinned DESC, updated_at DESC`
	case trips.KeepViewReminders:
		q = `SELECT id, trip_id, title, body, color, pinned, archived, trashed, due_at, created_at, updated_at
			FROM trip_notes WHERE trip_id = ? AND archived = 0 AND trashed = 0
			AND TRIM(COALESCE(due_at, '')) != ''
			ORDER BY pinned DESC, due_at ASC, updated_at DESC`
	case trips.KeepViewArchive:
		q = `SELECT id, trip_id, title, body, color, pinned, archived, trashed, due_at, created_at, updated_at
			FROM trip_notes WHERE trip_id = ? AND archived = 1 AND trashed = 0
			ORDER BY updated_at DESC`
	case trips.KeepViewTrash:
		q = `SELECT id, trip_id, title, body, color, pinned, archived, trashed, due_at, created_at, updated_at
			FROM trip_notes WHERE trip_id = ? AND trashed = 1
			ORDER BY updated_at DESC`
	default:
		q = `SELECT id, trip_id, title, body, color, pinned, archived, trashed, due_at, created_at, updated_at
			FROM trip_notes WHERE trip_id = ? AND archived = 0 AND trashed = 0
			ORDER BY pinned DESC, updated_at DESC`
	}
	rows, err := r.db.QueryContext(ctx, q, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTripNotes(rows)
}

// ListTripNotesForExport returns every note row for the trip (including archived and trashed) for account backup JSON.
func (r *Repository) ListTripNotesForExport(ctx context.Context, tripID string) ([]trips.TripNote, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, trip_id, title, body, color, pinned, archived, trashed, due_at, created_at, updated_at
		FROM trip_notes WHERE trip_id = ?
		ORDER BY updated_at DESC`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTripNotes(rows)
}

func scanTripNotes(rows *sql.Rows) ([]trips.TripNote, error) {
	var out []trips.TripNote
	for rows.Next() {
		var n trips.TripNote
		var pinned, archived, trashed int64
		if err := rows.Scan(&n.ID, &n.TripID, &n.Title, &n.Body, &n.Color, &pinned, &archived, &trashed, &n.DueAt, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		n.Pinned = scanBoolFromInt(pinned)
		n.Archived = scanBoolFromInt(archived)
		n.Trashed = scanBoolFromInt(trashed)
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *Repository) ListChecklistItemsForKeepView(ctx context.Context, tripID, view string) ([]trips.ChecklistItem, error) {
	view = strings.TrimSpace(strings.ToLower(view))
	var q string
	switch view {
	case trips.KeepViewNotes:
		q = `SELECT id, trip_id, category, text, done, due_at, created_at, updated_at, archived, trashed
			FROM checklist_items WHERE trip_id = ? AND archived = 0 AND trashed = 0
			ORDER BY created_at DESC`
	case trips.KeepViewReminders:
		q = `SELECT id, trip_id, category, text, done, due_at, created_at, updated_at, archived, trashed
			FROM checklist_items WHERE trip_id = ? AND archived = 0 AND trashed = 0
			AND TRIM(COALESCE(due_at, '')) != ''
			ORDER BY due_at ASC, created_at DESC`
	case trips.KeepViewArchive:
		q = `SELECT id, trip_id, category, text, done, due_at, created_at, updated_at, archived, trashed
			FROM checklist_items WHERE trip_id = ? AND archived = 1 AND trashed = 0
			ORDER BY created_at DESC`
	case trips.KeepViewTrash:
		q = `SELECT id, trip_id, category, text, done, due_at, created_at, updated_at, archived, trashed
			FROM checklist_items WHERE trip_id = ? AND trashed = 1
			ORDER BY created_at DESC`
	default:
		q = `SELECT id, trip_id, category, text, done, due_at, created_at, updated_at, archived, trashed
			FROM checklist_items WHERE trip_id = ? AND archived = 0 AND trashed = 0
			ORDER BY created_at DESC`
	}
	rows, err := r.db.QueryContext(ctx, q, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []trips.ChecklistItem
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

func (r *Repository) ListPinnedChecklistCategories(ctx context.Context, tripID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT category FROM trip_checklist_category_pins WHERE trip_id = ? ORDER BY category ASC`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var c string
		if err := rows.Scan(&c); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repository) SetChecklistCategoryPinned(ctx context.Context, tripID, category string, pinned bool) error {
	if pinned {
		res, err := r.db.ExecContext(ctx, `
			INSERT INTO trip_checklist_category_pins (trip_id, category) VALUES (?, ?)
			ON CONFLICT(trip_id, category) DO NOTHING`, tripID, category)
		if err != nil {
			return err
		}
		if n, _ := res.RowsAffected(); n > 0 {
			_ = r.logChange(ctx, tripID, "checklist_category_pin", category, "update", map[string]any{
				"category": category, "pinned": true,
			})
		}
		return nil
	}
	res, err := r.db.ExecContext(ctx, `
		DELETE FROM trip_checklist_category_pins WHERE trip_id = ? AND category = ?`, tripID, category)
	if err != nil {
		return err
	}
	if n, _ := res.RowsAffected(); n > 0 {
		_ = r.logChange(ctx, tripID, "checklist_category_pin", category, "update", map[string]any{
			"category": category, "pinned": false,
		})
	}
	return nil
}
