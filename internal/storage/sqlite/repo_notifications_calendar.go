package sqlite

import (
	"context"
	"database/sql"
	"strings"
	"time"

	"github.com/google/uuid"

	"remi-trip-planner/internal/trips"
)

func (r *Repository) InsertAppNotification(ctx context.Context, n trips.AppNotification) (inserted bool, err error) {
	if n.ID == "" {
		n.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	nowS := now.Format(time.RFC3339)
	readS := ""
	if !n.ReadAt.IsZero() {
		readS = n.ReadAt.UTC().Format(time.RFC3339)
	}
	res, err := r.db.ExecContext(ctx, `
		INSERT OR IGNORE INTO app_notifications (id, user_id, trip_id, title, body, href, kind, dedupe_key, read_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		n.ID, n.UserID, n.TripID, n.Title, n.Body, n.Href, n.Kind, n.DedupeKey, readS, nowS,
	)
	if err != nil {
		return false, err
	}
	aff, _ := res.RowsAffected()
	return aff > 0, nil
}

func (r *Repository) CountUnreadAppNotifications(ctx context.Context, userID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM app_notifications
		WHERE user_id = ? AND TRIM(COALESCE(read_at, '')) = ''`, userID).Scan(&n)
	return n, err
}

func (r *Repository) ListAppNotifications(ctx context.Context, userID string, limit int) ([]trips.AppNotification, error) {
	if limit <= 0 {
		limit = 50
	}
	if limit > 200 {
		limit = 200
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, trip_id, title, body, href, kind, dedupe_key, read_at, created_at
		FROM app_notifications WHERE user_id = ? ORDER BY datetime(created_at) DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []trips.AppNotification
	for rows.Next() {
		var n trips.AppNotification
		var readAt, created sql.NullString
		if err := rows.Scan(&n.ID, &n.UserID, &n.TripID, &n.Title, &n.Body, &n.Href, &n.Kind, &n.DedupeKey, &readAt, &created); err != nil {
			return nil, err
		}
		if readAt.Valid && readAt.String != "" {
			n.ReadAt, _ = time.Parse(time.RFC3339, readAt.String)
		}
		if created.Valid {
			n.CreatedAt, _ = time.Parse(time.RFC3339, created.String)
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *Repository) MarkAppNotificationRead(ctx context.Context, userID, notificationID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		UPDATE app_notifications SET read_at = ? WHERE id = ? AND user_id = ? AND TRIM(COALESCE(read_at, '')) = ''`,
		now, notificationID, userID)
	return err
}

func (r *Repository) MarkAllAppNotificationsRead(ctx context.Context, userID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		UPDATE app_notifications SET read_at = ? WHERE user_id = ? AND TRIM(COALESCE(read_at, '')) = ''`,
		now, userID)
	return err
}

func (r *Repository) DeleteAppNotificationsForTrip(ctx context.Context, tripID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM app_notifications WHERE trip_id = ?`, tripID)
	return err
}

func (r *Repository) ListUnreadAppNotifications(ctx context.Context, userID string, limit int) ([]trips.AppNotification, error) {
	if limit <= 0 {
		limit = 200
	}
	if limit > 2000 {
		limit = 2000
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, user_id, trip_id, title, body, href, kind, dedupe_key, read_at, created_at
		FROM app_notifications
		WHERE user_id = ? AND TRIM(COALESCE(read_at, '')) = ''
		ORDER BY datetime(created_at) DESC LIMIT ?`, userID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []trips.AppNotification
	for rows.Next() {
		var n trips.AppNotification
		var readAt, created sql.NullString
		if err := rows.Scan(&n.ID, &n.UserID, &n.TripID, &n.Title, &n.Body, &n.Href, &n.Kind, &n.DedupeKey, &readAt, &created); err != nil {
			return nil, err
		}
		if readAt.Valid && readAt.String != "" {
			n.ReadAt, _ = time.Parse(time.RFC3339, readAt.String)
		}
		if created.Valid {
			n.CreatedAt, _ = time.Parse(time.RFC3339, created.String)
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

func (r *Repository) ListItineraryCustomRemindersByTrip(ctx context.Context, tripID string) ([]trips.ItineraryCustomReminder, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, trip_id, itinerary_item_id, minutes_before_start, label, created_at
		FROM itinerary_custom_reminders WHERE trip_id = ? ORDER BY itinerary_item_id, minutes_before_start`, tripID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []trips.ItineraryCustomReminder
	for rows.Next() {
		var m trips.ItineraryCustomReminder
		var created string
		if err := rows.Scan(&m.ID, &m.TripID, &m.ItineraryItemID, &m.MinutesBeforeStart, &m.Label, &created); err != nil {
			return nil, err
		}
		m.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Repository) ReplaceItineraryItemCustomReminders(ctx context.Context, tripID, itineraryItemID string, rows []trips.ItineraryCustomReminder) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `DELETE FROM itinerary_custom_reminders WHERE trip_id = ? AND itinerary_item_id = ?`, tripID, itineraryItemID)
	if err != nil {
		return err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	for _, row := range rows {
		id := row.ID
		if id == "" {
			id = uuid.NewString()
		}
		mb := row.MinutesBeforeStart
		if mb < 0 || mb > 365*24*60 {
			continue
		}
		label := strings.TrimSpace(row.Label)
		if label == "" {
			label = "Reminder"
		}
		_, err = tx.ExecContext(ctx, `
			INSERT INTO itinerary_custom_reminders (id, trip_id, itinerary_item_id, minutes_before_start, label, created_at)
			VALUES (?, ?, ?, ?, ?, ?)`, id, tripID, itineraryItemID, mb, label, now)
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repository) ListTripIDsForReminderScan(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT id FROM trips WHERE is_archived = 0 AND end_date >= date('now', '-120 day')`)
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

func (r *Repository) GetCalendarFeedTokenHash(ctx context.Context, tripID string) (string, bool, error) {
	var h string
	err := r.db.QueryRowContext(ctx, `SELECT token_hash FROM trip_calendar_feed_tokens WHERE trip_id = ?`, tripID).Scan(&h)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return h, true, nil
}

func (r *Repository) UpsertCalendarFeedToken(ctx context.Context, tripID, tokenHash, createdByUserID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO trip_calendar_feed_tokens (trip_id, token_hash, created_by_user_id, created_at)
		VALUES (?, ?, ?, ?)
		ON CONFLICT(trip_id) DO UPDATE SET token_hash = excluded.token_hash, created_by_user_id = excluded.created_by_user_id, created_at = excluded.created_at`,
		tripID, tokenHash, strings.TrimSpace(createdByUserID), now)
	return err
}

func (r *Repository) DeleteCalendarFeedToken(ctx context.Context, tripID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM trip_calendar_feed_tokens WHERE trip_id = ?`, tripID)
	return err
}
