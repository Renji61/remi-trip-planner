package sqlite

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	"remi-trip-planner/internal/trips"

	"github.com/google/uuid"
)

func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}

// --- Users ---

func (r *Repository) CountUsers(ctx context.Context) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&n)
	return n, err
}

func (r *Repository) CreateUser(ctx context.Context, u trips.User) (string, error) {
	if u.ID == "" {
		u.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	var verified string
	if !u.EmailVerifiedAt.IsZero() {
		verified = u.EmailVerifiedAt.UTC().Format(time.RFC3339)
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO users (id, email, username, display_name, password_hash, avatar_path, email_verified_at, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		u.ID, strings.TrimSpace(u.Email), strings.TrimSpace(u.Username), strings.TrimSpace(u.DisplayName),
		u.PasswordHash, strings.TrimSpace(u.AvatarPath), verified, now, now,
	)
	return u.ID, err
}

func (r *Repository) UpdateUser(ctx context.Context, u trips.User) error {
	now := time.Now().UTC()
	var verified string
	if !u.EmailVerifiedAt.IsZero() {
		verified = u.EmailVerifiedAt.UTC().Format(time.RFC3339)
	}
	_, err := r.db.ExecContext(ctx, `
		UPDATE users SET email = ?, username = ?, display_name = ?, password_hash = ?, avatar_path = ?, email_verified_at = ?, updated_at = ?
		WHERE id = ?`,
		strings.TrimSpace(u.Email), strings.TrimSpace(u.Username), strings.TrimSpace(u.DisplayName),
		u.PasswordHash, strings.TrimSpace(u.AvatarPath), verified, now, u.ID,
	)
	return err
}

func scanUser(row *sql.Row) (trips.User, error) {
	var u trips.User
	var verified sql.NullString
	err := row.Scan(&u.ID, &u.Email, &u.Username, &u.DisplayName, &u.PasswordHash, &u.AvatarPath, &verified, &u.CreatedAt, &u.UpdatedAt)
	if err != nil {
		return u, err
	}
	if verified.Valid && verified.String != "" {
		u.EmailVerifiedAt, _ = time.Parse(time.RFC3339, verified.String)
	}
	return u, nil
}

func (r *Repository) GetUserByID(ctx context.Context, id string) (trips.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, email, username, display_name, password_hash, avatar_path, email_verified_at, created_at, updated_at FROM users WHERE id = ?`, id)
	return scanUser(row)
}

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (trips.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, email, username, display_name, password_hash, avatar_path, email_verified_at, created_at, updated_at FROM users WHERE email = ? COLLATE NOCASE`,
		strings.TrimSpace(email))
	return scanUser(row)
}

func (r *Repository) GetUserByUsername(ctx context.Context, username string) (trips.User, error) {
	row := r.db.QueryRowContext(ctx, `
		SELECT id, email, username, display_name, password_hash, avatar_path, email_verified_at, created_at, updated_at FROM users WHERE username = ? COLLATE NOCASE`,
		strings.TrimSpace(username))
	return scanUser(row)
}

func (r *Repository) EmailExists(ctx context.Context, email string, excludeUserID string) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE email = ? COLLATE NOCASE AND id != ?`,
		strings.TrimSpace(email), excludeUserID).Scan(&n)
	return n > 0, err
}

func (r *Repository) UsernameExists(ctx context.Context, username string, excludeUserID string) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users WHERE username = ? COLLATE NOCASE AND id != ?`,
		strings.TrimSpace(username), excludeUserID).Scan(&n)
	return n > 0, err
}

func (r *Repository) AssignOrphanTripsToUser(ctx context.Context, userID string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE trips SET owner_user_id = ? WHERE owner_user_id = '' OR owner_user_id IS NULL`, userID)
	return err
}

// --- Sessions ---

func (r *Repository) CreateSession(ctx context.Context, userID, tokenRaw, csrf string, ttl time.Duration) (sessionID string, err error) {
	id := uuid.NewString()
	now := time.Now().UTC()
	exp := now.Add(ttl)
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO sessions (id, user_id, token_hash, csrf_token, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, userID, hashToken(tokenRaw), csrf, exp.Format(time.RFC3339), now.Format(time.RFC3339),
	)
	return id, err
}

func (r *Repository) GetSessionByTokenHash(ctx context.Context, tokenRaw string) (trips.Session, error) {
	var s trips.Session
	var exp, created string
	err := r.db.QueryRowContext(ctx, `
		SELECT id, user_id, token_hash, csrf_token, expires_at, created_at FROM sessions WHERE token_hash = ?`, hashToken(tokenRaw)).
		Scan(&s.ID, &s.UserID, &s.TokenHash, &s.CSRFToken, &exp, &created)
	if err != nil {
		return s, err
	}
	s.ExpiresAt, _ = time.Parse(time.RFC3339, exp)
	s.CreatedAt, _ = time.Parse(time.RFC3339, created)
	return s, nil
}

func (r *Repository) DeleteSession(ctx context.Context, sessionID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE id = ?`, sessionID)
	return err
}

func (r *Repository) DeleteSessionByTokenRaw(ctx context.Context, tokenRaw string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE token_hash = ?`, hashToken(tokenRaw))
	return err
}

func (r *Repository) DeleteExpiredSessions(ctx context.Context) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM sessions WHERE expires_at < ?`, time.Now().UTC().Format(time.RFC3339))
	return err
}

// --- Email verification ---

func (r *Repository) ReplaceEmailVerifyToken(ctx context.Context, userID, tokenRaw string, ttl time.Duration) error {
	_, _ = r.db.ExecContext(ctx, `DELETE FROM email_verify_tokens WHERE user_id = ?`, userID)
	id := uuid.NewString()
	now := time.Now().UTC()
	exp := now.Add(ttl)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO email_verify_tokens (id, user_id, token_hash, expires_at, created_at) VALUES (?, ?, ?, ?, ?)`,
		id, userID, hashToken(tokenRaw), exp.Format(time.RFC3339), now.Format(time.RFC3339))
	return err
}

func (r *Repository) ConsumeEmailVerifyToken(ctx context.Context, tokenRaw string) (userID string, err error) {
	th := hashToken(tokenRaw)
	var uid string
	var exp string
	err = r.db.QueryRowContext(ctx, `
		SELECT user_id, expires_at FROM email_verify_tokens WHERE token_hash = ?`, th).Scan(&uid, &exp)
	if err != nil {
		return "", err
	}
	texp, _ := time.Parse(time.RFC3339, exp)
	if time.Now().UTC().After(texp) {
		_, _ = r.db.ExecContext(ctx, `DELETE FROM email_verify_tokens WHERE token_hash = ?`, th)
		return "", sql.ErrNoRows
	}
	_, err = r.db.ExecContext(ctx, `DELETE FROM email_verify_tokens WHERE token_hash = ?`, th)
	if err != nil {
		return "", err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	_, err = r.db.ExecContext(ctx, `UPDATE users SET email_verified_at = ?, updated_at = ? WHERE id = ?`, now, now, uid)
	return uid, err
}

// --- User settings ---

func (r *Repository) GetUserSettings(ctx context.Context, userID string) (trips.UserSettings, error) {
	var s trips.UserSettings
	var updated string
	err := r.db.QueryRowContext(ctx, `
		SELECT user_id, theme_preference, dashboard_trip_layout, dashboard_trip_sort, dashboard_hero_background,
			trip_dashboard_heading, default_currency_name, default_currency_symbol, COALESCE(distance_unit, ''), updated_at
		FROM user_settings WHERE user_id = ?`, userID).
		Scan(&s.UserID, &s.ThemePreference, &s.DashboardTripLayout, &s.DashboardTripSort, &s.DashboardHeroBackground,
			&s.TripDashboardHeading, &s.DefaultCurrencyName, &s.DefaultCurrencySymbol, &s.DistanceUnit, &updated)
	s.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
	return s, err
}

func (r *Repository) SaveUserSettings(ctx context.Context, s trips.UserSettings) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO user_settings (user_id, theme_preference, dashboard_trip_layout, dashboard_trip_sort, dashboard_hero_background,
			trip_dashboard_heading, default_currency_name, default_currency_symbol, distance_unit, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(user_id) DO UPDATE SET
			theme_preference = excluded.theme_preference,
			dashboard_trip_layout = excluded.dashboard_trip_layout,
			dashboard_trip_sort = excluded.dashboard_trip_sort,
			dashboard_hero_background = excluded.dashboard_hero_background,
			trip_dashboard_heading = excluded.trip_dashboard_heading,
			default_currency_name = excluded.default_currency_name,
			default_currency_symbol = excluded.default_currency_symbol,
			distance_unit = excluded.distance_unit,
			updated_at = excluded.updated_at`,
		s.UserID, s.ThemePreference, s.DashboardTripLayout, s.DashboardTripSort, s.DashboardHeroBackground,
		s.TripDashboardHeading, s.DefaultCurrencyName, s.DefaultCurrencySymbol, s.DistanceUnit, now,
	)
	return err
}

func (r *Repository) SeedUserSettingsFromAppDefaults(ctx context.Context, userID string, app trips.AppSettings) error {
	s := trips.UserSettings{
		UserID:                  userID,
		ThemePreference:         app.ThemePreference,
		DashboardTripLayout:     app.DashboardTripLayout,
		DashboardTripSort:       app.DashboardTripSort,
		DashboardHeroBackground: app.DashboardHeroBackground,
		TripDashboardHeading:    app.TripDashboardHeading,
		DefaultCurrencyName:     app.DefaultCurrencyName,
		DefaultCurrencySymbol:   app.DefaultCurrencySymbol,
	}
	return r.SaveUserSettings(ctx, s)
}

// --- Trip visibility & collaboration ---

func (r *Repository) ListVisibleTripsForUser(ctx context.Context, userID string) ([]trips.Trip, error) {
	q := `
SELECT ` + tripSelectCols + ` FROM trips t
WHERE (
	t.owner_user_id = ?
) OR (
	EXISTS (SELECT 1 FROM trip_members m WHERE m.trip_id = t.id AND m.user_id = ? AND m.left_at IS NULL)
	AND NOT (
		t.is_archived = 1 AND EXISTS (
			SELECT 1 FROM trip_collaborator_dashboard d
			WHERE d.trip_id = t.id AND d.user_id = ? AND d.hidden_archived = 1
		)
	)
)
ORDER BY t.created_at DESC`
	rows, err := r.db.QueryContext(ctx, q, userID, userID, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []trips.Trip
	for rows.Next() {
		t, err := scanTrip(rows.Scan)
		if err != nil {
			return nil, err
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

func (r *Repository) IsTripOwner(ctx context.Context, tripID, userID string) (bool, error) {
	var owner string
	err := r.db.QueryRowContext(ctx, `SELECT owner_user_id FROM trips WHERE id = ?`, tripID).Scan(&owner)
	if err != nil {
		return false, err
	}
	return owner == userID, nil
}

func (r *Repository) IsActiveCollaborator(ctx context.Context, tripID, userID string) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM trip_members WHERE trip_id = ? AND user_id = ? AND left_at IS NULL`, tripID, userID).Scan(&n)
	return n > 0, err
}

func (r *Repository) AddTripMember(ctx context.Context, tripID, userID, invitedBy string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO trip_members (trip_id, user_id, invited_by_user_id, joined_at, left_at)
		VALUES (?, ?, ?, ?, NULL)
		ON CONFLICT(trip_id, user_id) DO UPDATE SET left_at = NULL, invited_by_user_id = excluded.invited_by_user_id, joined_at = excluded.joined_at`,
		tripID, userID, invitedBy, now)
	return err
}

func (r *Repository) MarkTripMemberLeft(ctx context.Context, tripID, userID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `UPDATE trip_members SET left_at = ? WHERE trip_id = ? AND user_id = ?`, now, tripID, userID)
	return err
}

func (r *Repository) RevokeAllCollaborators(ctx context.Context, tripID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `UPDATE trip_members SET left_at = ? WHERE trip_id = ? AND left_at IS NULL`, now, tripID)
	return err
}

func (r *Repository) ListActiveTripMemberUserIDs(ctx context.Context, tripID string) ([]string, error) {
	rows, err := r.db.QueryContext(ctx, `
		SELECT user_id FROM trip_members WHERE trip_id = ? AND left_at IS NULL`, tripID)
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
	return out, rows.Err()
}

func (r *Repository) SetTripArchivedHiddenForUser(ctx context.Context, tripID, userID string, hidden bool) error {
	if !hidden {
		_, err := r.db.ExecContext(ctx, `DELETE FROM trip_collaborator_dashboard WHERE trip_id = ? AND user_id = ?`, tripID, userID)
		return err
	}
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO trip_collaborator_dashboard (trip_id, user_id, hidden_archived) VALUES (?, ?, 1)
		ON CONFLICT(trip_id, user_id) DO UPDATE SET hidden_archived = 1`, tripID, userID)
	return err
}

func (r *Repository) IsArchivedTripHiddenOnDashboard(ctx context.Context, tripID, userID string) (bool, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM trip_collaborator_dashboard
		WHERE trip_id = ? AND user_id = ? AND hidden_archived = 1`, tripID, userID).Scan(&n)
	return n > 0, err
}

func (r *Repository) CountActiveCollaborators(ctx context.Context, tripID string) (int, error) {
	var n int
	err := r.db.QueryRowContext(ctx, `
		SELECT COUNT(*) FROM trip_members WHERE trip_id = ? AND left_at IS NULL`, tripID).Scan(&n)
	return n, err
}

func (r *Repository) getUserProfileByID(ctx context.Context, id string) (trips.UserProfile, error) {
	var p trips.UserProfile
	err := r.db.QueryRowContext(ctx, `SELECT id, email, username, display_name, avatar_path FROM users WHERE id = ?`, id).
		Scan(&p.ID, &p.Email, &p.Username, &p.DisplayName, &p.AvatarPath)
	return p, err
}

func (r *Repository) ListTripPartyProfiles(ctx context.Context, tripID string) ([]trips.UserProfile, error) {
	var ownerID string
	if err := r.db.QueryRowContext(ctx, `SELECT owner_user_id FROM trips WHERE id = ?`, tripID).Scan(&ownerID); err != nil {
		return nil, err
	}
	var out []trips.UserProfile
	if ownerID != "" {
		if o, err := r.getUserProfileByID(ctx, ownerID); err == nil {
			out = append(out, o)
		}
	}
	rows, err := r.db.QueryContext(ctx, `
		SELECT u.id, u.email, u.username, u.display_name, u.avatar_path FROM trip_members m
		JOIN users u ON u.id = m.user_id
		WHERE m.trip_id = ? AND m.left_at IS NULL AND u.id != ?
		ORDER BY COALESCE(NULLIF(TRIM(u.display_name), ''), u.username), u.email`, tripID, ownerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var p trips.UserProfile
		if err := rows.Scan(&p.ID, &p.Email, &p.Username, &p.DisplayName, &p.AvatarPath); err != nil {
			return nil, err
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// --- Invites ---

func (r *Repository) CreateTripInvite(ctx context.Context, inv trips.TripInvite, tokenRaw string) error {
	if inv.ID == "" {
		inv.ID = uuid.NewString()
	}
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO trip_invites (id, trip_id, email_normalized, token_hash, invited_by_user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		inv.ID, inv.TripID, strings.ToLower(strings.TrimSpace(inv.EmailNormalized)), hashToken(tokenRaw),
		inv.InvitedByUserID, inv.ExpiresAt.UTC().Format(time.RFC3339), now.Format(time.RFC3339))
	return err
}

func (r *Repository) getEmailTripInviteByToken(ctx context.Context, tokenRaw string) (trips.TripInviteLookup, error) {
	var row trips.TripInviteLookup
	var expS string
	var acc, rev sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT id, trip_id, email_normalized, invited_by_user_id, expires_at, accepted_at, revoked_at
		FROM trip_invites WHERE token_hash = ?`, hashToken(tokenRaw)).
		Scan(&row.ID, &row.TripID, &row.EmailNormalized, &row.InvitedByUserID, &expS, &acc, &rev)
	if err != nil {
		return row, err
	}
	row.ExpiresAt, _ = time.Parse(time.RFC3339, expS)
	if rev.Valid && rev.String != "" {
		return row, sql.ErrNoRows
	}
	if acc.Valid && acc.String != "" {
		return row, sql.ErrNoRows
	}
	if time.Now().UTC().After(row.ExpiresAt) {
		return row, sql.ErrNoRows
	}
	return row, nil
}

func (r *Repository) getLinkTripInviteByToken(ctx context.Context, tokenRaw string) (trips.TripInviteLookup, error) {
	var row trips.TripInviteLookup
	var expS string
	var rev sql.NullString
	err := r.db.QueryRowContext(ctx, `
		SELECT id, trip_id, invited_by_user_id, expires_at, revoked_at
		FROM trip_invite_links WHERE token_hash = ?`, hashToken(tokenRaw)).
		Scan(&row.ID, &row.TripID, &row.InvitedByUserID, &expS, &rev)
	if err != nil {
		return row, err
	}
	row.EmailNormalized = ""
	row.IsLinkInvite = true
	row.ExpiresAt, _ = time.Parse(time.RFC3339, expS)
	if rev.Valid && rev.String != "" {
		return row, sql.ErrNoRows
	}
	if time.Now().UTC().After(row.ExpiresAt) {
		return row, sql.ErrNoRows
	}
	return row, nil
}

func (r *Repository) GetTripInviteByTokenRaw(ctx context.Context, tokenRaw string) (trips.TripInviteLookup, error) {
	row, err := r.getEmailTripInviteByToken(ctx, tokenRaw)
	if err == nil {
		return row, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return row, err
	}
	return r.getLinkTripInviteByToken(ctx, tokenRaw)
}

func (r *Repository) MarkTripInviteAccepted(ctx context.Context, inviteID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `UPDATE trip_invites SET accepted_at = ? WHERE id = ?`, now, inviteID)
	return err
}

func (r *Repository) RevokePendingTripInvites(ctx context.Context, tripID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		UPDATE trip_invites SET revoked_at = ? WHERE trip_id = ? AND accepted_at IS NULL AND revoked_at IS NULL`, now, tripID)
	return err
}

func (r *Repository) ListPendingTripInvitesForTrip(ctx context.Context, tripID string) ([]trips.TripInvitePending, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	rows, err := r.db.QueryContext(ctx, `
		SELECT id, email_normalized FROM trip_invites
		WHERE trip_id = ?
			AND accepted_at IS NULL
			AND revoked_at IS NULL
			AND expires_at > ?`, tripID, now)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []trips.TripInvitePending
	for rows.Next() {
		var row trips.TripInvitePending
		if err := rows.Scan(&row.ID, &row.Email); err != nil {
			return nil, err
		}
		out = append(out, row)
	}
	return out, rows.Err()
}

func (r *Repository) RevokeTripInviteForTrip(ctx context.Context, tripID, inviteID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	res, err := r.db.ExecContext(ctx, `
		UPDATE trip_invites SET revoked_at = ?
		WHERE id = ? AND trip_id = ? AND accepted_at IS NULL AND revoked_at IS NULL`,
		now, inviteID, tripID)
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
	return nil
}

func (r *Repository) CreateTripInviteLink(ctx context.Context, tripID, invitedByUserID, tokenRaw string, expiresAt time.Time) error {
	id := uuid.NewString()
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		INSERT INTO trip_invite_links (id, trip_id, token_hash, invited_by_user_id, expires_at, created_at)
		VALUES (?, ?, ?, ?, ?, ?)`,
		id, tripID, hashToken(tokenRaw), invitedByUserID, expiresAt.UTC().Format(time.RFC3339), now)
	return err
}

func (r *Repository) RevokeAllTripInviteLinksForTrip(ctx context.Context, tripID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	_, err := r.db.ExecContext(ctx, `
		UPDATE trip_invite_links SET revoked_at = ?
		WHERE trip_id = ? AND revoked_at IS NULL`, now, tripID)
	return err
}
