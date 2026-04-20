package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"remi-trip-planner/internal/trips"
)

func (r *Repository) listGlobalKeepNotesForView(ctx context.Context, userID, view string) ([]trips.GlobalKeepNote, error) {
	view = strings.TrimSpace(strings.ToLower(view))
	var q string
	switch view {
	case trips.KeepViewArchive:
		q = `SELECT id, user_id, title, body, color, due_at, pinned, archived, trashed, created_at, updated_at
			FROM global_keep_notes WHERE user_id = ? AND archived = 1 AND trashed = 0
			ORDER BY updated_at DESC`
	case trips.KeepViewTrash:
		q = `SELECT id, user_id, title, body, color, due_at, pinned, archived, trashed, created_at, updated_at
			FROM global_keep_notes WHERE user_id = ? AND trashed = 1
			ORDER BY updated_at DESC`
	default:
		q = `SELECT id, user_id, title, body, color, due_at, pinned, archived, trashed, created_at, updated_at
			FROM global_keep_notes WHERE user_id = ? AND archived = 0 AND trashed = 0
			ORDER BY pinned DESC, updated_at DESC`
	}
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []trips.GlobalKeepNote
	for rows.Next() {
		var n trips.GlobalKeepNote
		var pinned, archived, trashed int64
		if err := rows.Scan(&n.ID, &n.UserID, &n.Title, &n.Body, &n.Color, &n.DueAt, &pinned, &archived, &trashed, &n.CreatedAt, &n.UpdatedAt); err != nil {
			return nil, err
		}
		n.Pinned = scanBoolFromInt(pinned)
		n.Archived = scanBoolFromInt(archived)
		n.Trashed = scanBoolFromInt(trashed)
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *Repository) ListGlobalKeepNotesByUser(ctx context.Context, userID string) ([]trips.GlobalKeepNote, error) {
	return r.listGlobalKeepNotesForView(ctx, userID, trips.KeepViewNotes)
}

func (r *Repository) ListGlobalKeepNotesForKeepView(ctx context.Context, userID, view string) ([]trips.GlobalKeepNote, error) {
	return r.listGlobalKeepNotesForView(ctx, userID, view)
}

func (r *Repository) AddGlobalKeepNote(ctx context.Context, n trips.GlobalKeepNote) error {
	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO global_keep_notes (id, user_id, title, body, color, due_at, pinned, archived, trashed, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		n.ID, n.UserID, strings.TrimSpace(n.Title), strings.TrimSpace(n.Body), strings.TrimSpace(n.Color),
		strings.TrimSpace(n.DueAt), sqliteBool(n.Pinned), sqliteBool(n.Archived), sqliteBool(n.Trashed), now, now,
	)
	return err
}

func (r *Repository) GetGlobalKeepNote(ctx context.Context, userID, noteID string) (trips.GlobalKeepNote, error) {
	var n trips.GlobalKeepNote
	var pinned, archived, trashed int64
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, title, body, color, due_at, pinned, archived, trashed, created_at, updated_at
		FROM global_keep_notes WHERE id = ? AND user_id = ?`, noteID, userID).
		Scan(&n.ID, &n.UserID, &n.Title, &n.Body, &n.Color, &n.DueAt, &pinned, &archived, &trashed, &n.CreatedAt, &n.UpdatedAt)
	n.Pinned = scanBoolFromInt(pinned)
	n.Archived = scanBoolFromInt(archived)
	n.Trashed = scanBoolFromInt(trashed)
	return n, err
}

func (r *Repository) UpdateGlobalKeepNote(ctx context.Context, n trips.GlobalKeepNote) error {
	now := time.Now().UTC()
	res, err := r.db.ExecContext(ctx, `
		UPDATE global_keep_notes
		SET title = ?, body = ?, color = ?, due_at = ?, pinned = ?, archived = ?, trashed = ?, updated_at = ?
		WHERE id = ? AND user_id = ?`,
		strings.TrimSpace(n.Title), strings.TrimSpace(n.Body), strings.TrimSpace(n.Color), strings.TrimSpace(n.DueAt),
		sqliteBool(n.Pinned), sqliteBool(n.Archived), sqliteBool(n.Trashed), now, n.ID, n.UserID,
	)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) DeleteGlobalKeepNote(ctx context.Context, userID, noteID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM global_keep_notes WHERE id = ? AND user_id = ?`, noteID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	_, _ = r.db.ExecContext(ctx, `DELETE FROM trip_global_keep_imports WHERE kind = ? AND global_id = ?`, string(trips.GlobalKeepImportNote), noteID)
	return nil
}

func (r *Repository) listGlobalChecklistTemplatesForView(ctx context.Context, userID, view string) ([]trips.GlobalChecklistTemplate, error) {
	view = strings.TrimSpace(strings.ToLower(view))
	var q string
	switch view {
	case trips.KeepViewArchive:
		q = `SELECT id, user_id, category, due_at, pinned, archived, trashed, created_at, updated_at
			FROM global_checklist_templates WHERE user_id = ? AND archived = 1 AND trashed = 0
			ORDER BY updated_at DESC`
	case trips.KeepViewTrash:
		q = `SELECT id, user_id, category, due_at, pinned, archived, trashed, created_at, updated_at
			FROM global_checklist_templates WHERE user_id = ? AND trashed = 1
			ORDER BY updated_at DESC`
	default:
		q = `SELECT id, user_id, category, due_at, pinned, archived, trashed, created_at, updated_at
			FROM global_checklist_templates WHERE user_id = ? AND archived = 0 AND trashed = 0
			ORDER BY pinned DESC, updated_at DESC`
	}
	rows, err := r.db.QueryContext(ctx, q, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	byID := make(map[string]trips.GlobalChecklistTemplate)
	for rows.Next() {
		var t trips.GlobalChecklistTemplate
		var pinned, archived, trashed int64
		if err := rows.Scan(&t.ID, &t.UserID, &t.Category, &t.DueAt, &pinned, &archived, &trashed, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, err
		}
		t.Pinned = scanBoolFromInt(pinned)
		t.Archived = scanBoolFromInt(archived)
		t.Trashed = scanBoolFromInt(trashed)
		ids = append(ids, t.ID)
		byID[t.ID] = t
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, nil
	}
	ph := make([]string, len(ids))
	args := make([]any, len(ids))
	for i, id := range ids {
		ph[i] = "?"
		args[i] = id
	}
	lineRows, err := r.db.QueryContext(ctx,
		`SELECT template_id, text FROM global_checklist_template_lines
		 WHERE template_id IN (`+strings.Join(ph, ",")+`)
		 ORDER BY template_id, sort_order ASC`,
		args...,
	)
	if err != nil {
		return nil, err
	}
	defer lineRows.Close()
	for lineRows.Next() {
		var tid, text string
		if err := lineRows.Scan(&tid, &text); err != nil {
			return nil, err
		}
		t := byID[tid]
		t.Lines = append(t.Lines, text)
		byID[tid] = t
	}
	if err := lineRows.Err(); err != nil {
		return nil, err
	}
	out := make([]trips.GlobalChecklistTemplate, 0, len(ids))
	for _, id := range ids {
		out = append(out, byID[id])
	}
	return out, nil
}

func (r *Repository) ListGlobalChecklistTemplatesByUser(ctx context.Context, userID string) ([]trips.GlobalChecklistTemplate, error) {
	return r.listGlobalChecklistTemplatesForView(ctx, userID, trips.KeepViewNotes)
}

func (r *Repository) ListGlobalChecklistTemplatesForKeepView(ctx context.Context, userID, view string) ([]trips.GlobalChecklistTemplate, error) {
	return r.listGlobalChecklistTemplatesForView(ctx, userID, view)
}

func (r *Repository) AddGlobalChecklistTemplate(ctx context.Context, t trips.GlobalChecklistTemplate, lines []string) error {
	if t.ID == "" {
		t.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `
		INSERT INTO global_checklist_templates (id, user_id, category, due_at, pinned, archived, trashed, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.UserID, strings.TrimSpace(t.Category), strings.TrimSpace(t.DueAt),
		sqliteBool(t.Pinned), sqliteBool(t.Archived), sqliteBool(t.Trashed), now, now,
	)
	if err != nil {
		return err
	}
	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lid := uuid.NewString()
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO global_checklist_template_lines (id, template_id, sort_order, text) VALUES (?, ?, ?, ?)`,
			lid, t.ID, i, line,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) GetGlobalChecklistTemplate(ctx context.Context, userID, templateID string) (trips.GlobalChecklistTemplate, error) {
	var t trips.GlobalChecklistTemplate
	var pinned, archived, trashed int64
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, category, due_at, pinned, archived, trashed, created_at, updated_at
		FROM global_checklist_templates WHERE id = ? AND user_id = ?`, templateID, userID).
		Scan(&t.ID, &t.UserID, &t.Category, &t.DueAt, &pinned, &archived, &trashed, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return t, err
	}
	t.Pinned = scanBoolFromInt(pinned)
	t.Archived = scanBoolFromInt(archived)
	t.Trashed = scanBoolFromInt(trashed)
	rows, err := r.db.QueryContext(ctx, `
		SELECT text FROM global_checklist_template_lines WHERE template_id = ? ORDER BY sort_order ASC`, templateID)
	if err != nil {
		return t, err
	}
	defer rows.Close()
	for rows.Next() {
		var text string
		if err := rows.Scan(&text); err != nil {
			return t, err
		}
		t.Lines = append(t.Lines, text)
	}
	return t, rows.Err()
}

func (r *Repository) UpdateGlobalChecklistTemplate(ctx context.Context, t trips.GlobalChecklistTemplate) error {
	now := time.Now().UTC()
	res, err := r.db.ExecContext(ctx, `
		UPDATE global_checklist_templates
		SET category = ?, due_at = ?, pinned = ?, archived = ?, trashed = ?, updated_at = ?
		WHERE id = ? AND user_id = ?`,
		strings.TrimSpace(t.Category), strings.TrimSpace(t.DueAt),
		sqliteBool(t.Pinned), sqliteBool(t.Archived), sqliteBool(t.Trashed), now, t.ID, t.UserID,
	)
	if err != nil {
		return err
	}
	aff, _ := res.RowsAffected()
	if aff == 0 {
		return sql.ErrNoRows
	}
	return nil
}

func (r *Repository) DeleteGlobalChecklistTemplate(ctx context.Context, userID, templateID string) error {
	res, err := r.db.ExecContext(ctx, `DELETE FROM global_checklist_templates WHERE id = ? AND user_id = ?`, templateID, userID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return sql.ErrNoRows
	}
	_, _ = r.db.ExecContext(ctx, `DELETE FROM trip_global_keep_imports WHERE kind = ? AND global_id = ?`, string(trips.GlobalKeepImportChecklist), templateID)
	return nil
}

func (r *Repository) IsGlobalKeepImported(ctx context.Context, tripID string, kind trips.GlobalKeepImportKind, globalID string) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT 1 FROM trip_global_keep_imports WHERE trip_id = ? AND kind = ? AND global_id = ? LIMIT 1`,
		tripID, string(kind), globalID).Scan(&n)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (r *Repository) RecordGlobalKeepImport(ctx context.Context, tripID string, kind trips.GlobalKeepImportKind, globalID string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO trip_global_keep_imports (trip_id, kind, global_id, imported_at) VALUES (?, ?, ?, ?)`,
		tripID, string(kind), globalID, now,
	)
	return err
}

func (r *Repository) ListGlobalKeepImportedIDs(ctx context.Context, tripID string, kind trips.GlobalKeepImportKind) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT global_id FROM trip_global_keep_imports WHERE trip_id = ? AND kind = ?`, tripID, string(kind))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		out = append(out, id)
	}
	sort.Strings(out)
	return out, rows.Err()
}
