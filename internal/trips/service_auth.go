package trips

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

// ErrWrongCurrentPassword is returned when UpdateUserPassword receives an invalid current password.
var ErrWrongCurrentPassword = errors.New("Current password is incorrect.")

const (
	sessionTTL     = 30 * 24 * time.Hour
	inviteTTL      = 7 * 24 * time.Hour
	emailVerifyTTL = 72 * time.Hour
	bcryptCost     = bcrypt.DefaultCost
	minPasswordLen = 8
	minUsernameLen = 2
)

func (s *Service) CountUsers(ctx context.Context) (int, error) {
	return s.repo.CountUsers(ctx)
}

func randomTokenRaw() string {
	b := make([]byte, 32)
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}

func HashPassword(plain string) (string, error) {
	if len(plain) < minPasswordLen {
		return "", fmt.Errorf("password must be at least %d characters", minPasswordLen)
	}
	h, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", err
	}
	return string(h), nil
}

func CheckPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

func normalizeUsername(s string) string {
	return strings.TrimSpace(strings.ToLower(s))
}

func validateUsername(s string) error {
	s = strings.TrimSpace(s)
	if len([]rune(s)) < minUsernameLen {
		return fmt.Errorf("username must be at least %d characters", minUsernameLen)
	}
	for _, r := range s {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '-' {
			continue
		}
		return errors.New("username may only contain letters, digits, hyphen, and underscore")
	}
	return nil
}

// RegisterFirstUser creates the initial admin (email pre-verified) and attaches orphan trips.
func (s *Service) RegisterFirstUser(ctx context.Context, email, username, displayName, password string) (user User, sessionToken, csrf string, err error) {
	n, err := s.repo.CountUsers(ctx)
	if err != nil {
		return user, "", "", err
	}
	if n > 0 {
		return user, "", "", errors.New("setup already completed")
	}
	if err := validateUsername(username); err != nil {
		return user, "", "", err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return user, "", "", err
	}
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return user, "", "", errors.New("email is required")
	}
	now := time.Now().UTC()
	user = User{
		Email:           email,
		Username:        normalizeUsername(username),
		DisplayName:     strings.TrimSpace(displayName),
		PasswordHash:    hash,
		EmailVerifiedAt: now,
		IsAdmin:         true,
	}
	user.ID, err = s.repo.CreateUser(ctx, user)
	if err != nil {
		return user, "", "", err
	}
	app, err := s.repo.GetAppSettings(ctx)
	if err != nil {
		return user, "", "", err
	}
	if err := s.repo.SeedUserSettingsFromAppDefaults(ctx, user.ID, app); err != nil {
		return user, "", "", err
	}
	if err := s.repo.AssignOrphanTripsToUser(ctx, user.ID); err != nil {
		return user, "", "", err
	}
	sessionToken = randomTokenRaw()
	csrf = randomTokenRaw()
	_, err = s.repo.CreateSession(ctx, user.ID, sessionToken, csrf, sessionTTL)
	return user, sessionToken, csrf, err
}

// RegisterUser creates an additional account when RegistrationEnabled is true (not for the first user; use setup).
func (s *Service) RegisterUser(ctx context.Context, email, username, displayName, password string) (user User, sessionToken, csrf string, err error) {
	app, err := s.repo.GetAppSettings(ctx)
	if err != nil {
		return user, "", "", err
	}
	if !app.RegistrationEnabled {
		return user, "", "", errors.New("registration is disabled on this server")
	}
	n, err := s.repo.CountUsers(ctx)
	if err != nil {
		return user, "", "", err
	}
	if n == 0 {
		return user, "", "", errors.New("complete initial setup first")
	}
	if err := validateUsername(username); err != nil {
		return user, "", "", err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return user, "", "", err
	}
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return user, "", "", errors.New("email is required")
	}
	un := normalizeUsername(username)
	if ok, e := s.repo.EmailExists(ctx, email, ""); e != nil {
		return user, "", "", e
	} else if ok {
		return user, "", "", errors.New("that email is already registered")
	}
	if ok, e := s.repo.UsernameExists(ctx, un, ""); e != nil {
		return user, "", "", e
	} else if ok {
		return user, "", "", errors.New("that username is already taken")
	}
	user = User{
		Email:        email,
		Username:     un,
		DisplayName:  strings.TrimSpace(displayName),
		PasswordHash: hash,
	}
	user.ID, err = s.repo.CreateUser(ctx, user)
	if err != nil {
		return user, "", "", err
	}
	if err := s.repo.SeedUserSettingsFromAppDefaults(ctx, user.ID, app); err != nil {
		return user, "", "", err
	}
	sessionToken = randomTokenRaw()
	csrf = randomTokenRaw()
	if _, err = s.repo.CreateSession(ctx, user.ID, sessionToken, csrf, sessionTTL); err != nil {
		return user, "", "", err
	}
	_, _ = s.IssueEmailVerificationToken(ctx, user.ID)
	return user, sessionToken, csrf, nil
}

// LoginWithIdentifier accepts email or username.
func (s *Service) LoginWithIdentifier(ctx context.Context, identifier, password string) (user User, sessionToken, csrf string, err error) {
	id := strings.TrimSpace(identifier)
	var u User
	var e error
	if strings.Contains(id, "@") {
		u, e = s.repo.GetUserByEmail(ctx, id)
	} else {
		u, e = s.repo.GetUserByUsername(ctx, id)
	}
	if e != nil {
		if errors.Is(e, sql.ErrNoRows) {
			return user, "", "", errors.New("invalid credentials")
		}
		return user, "", "", e
	}
	if !CheckPassword(u.PasswordHash, password) {
		return user, "", "", errors.New("invalid credentials")
	}
	_ = s.repo.DeleteExpiredSessions(ctx)
	sessionToken = randomTokenRaw()
	csrf = randomTokenRaw()
	_, err = s.repo.CreateSession(ctx, u.ID, sessionToken, csrf, sessionTTL)
	return u, sessionToken, csrf, err
}

func (s *Service) Logout(ctx context.Context, sessionTokenRaw string) error {
	if sessionTokenRaw == "" {
		return nil
	}
	return s.repo.DeleteSessionByTokenRaw(ctx, sessionTokenRaw)
}

// SessionUser returns the user for a valid session token, or sql.ErrNoRows.
func (s *Service) SessionUser(ctx context.Context, sessionTokenRaw string) (User, Session, error) {
	if sessionTokenRaw == "" {
		return User{}, Session{}, sql.ErrNoRows
	}
	sess, err := s.repo.GetSessionByTokenHash(ctx, sessionTokenRaw)
	if err != nil {
		return User{}, Session{}, err
	}
	if time.Now().UTC().After(sess.ExpiresAt) {
		_ = s.repo.DeleteSession(ctx, sess.ID)
		return User{}, Session{}, sql.ErrNoRows
	}
	u, err := s.repo.GetUserByID(ctx, sess.UserID)
	return u, sess, err
}

// MergedSettingsForUI combines instance app settings (map, etc.) with per-user dashboard preferences.
func (s *Service) MergedSettingsForUI(ctx context.Context, userID string) (AppSettings, error) {
	app, err := s.repo.GetAppSettings(ctx)
	if err != nil {
		return app, err
	}
	if userID == "" {
		app.DashboardHeroBackground = CanonicalDashboardHeroBackground(app.DashboardHeroBackground)
		return app, nil
	}
	us, err := s.repo.GetUserSettings(ctx, userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			app.DashboardHeroBackground = CanonicalDashboardHeroBackground(app.DashboardHeroBackground)
			return app, nil
		}
		return app, err
	}
	app.ThemePreference = us.ThemePreference
	app.DashboardTripLayout = us.DashboardTripLayout
	app.DashboardTripSort = us.DashboardTripSort
	app.DashboardHeroBackground = CanonicalDashboardHeroBackground(us.DashboardHeroBackground)
	app.TripDashboardHeading = us.TripDashboardHeading
	app.DefaultCurrencyName = us.DefaultCurrencyName
	app.DefaultCurrencySymbol = us.DefaultCurrencySymbol
	app.UserDistanceUnit = us.DistanceUnit
	return app, nil
}

func (s *Service) SaveUserUISettings(ctx context.Context, userID string, us UserSettings) error {
	us.UserID = userID
	return s.repo.SaveUserSettings(ctx, us)
}

func (s *Service) EnsureUserSettings(ctx context.Context, userID string) error {
	_, err := s.repo.GetUserSettings(ctx, userID)
	if err == nil {
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	app, err := s.repo.GetAppSettings(ctx)
	if err != nil {
		return err
	}
	return s.repo.SeedUserSettingsFromAppDefaults(ctx, userID, app)
}

// ListVisibleTrips returns trips the user may see (visibility rules).
func (s *Service) ListVisibleTrips(ctx context.Context, userID string) ([]Trip, error) {
	if userID == "" {
		return nil, ErrAuthRequired
	}
	return s.repo.ListVisibleTripsForUser(ctx, userID)
}

func (s *Service) TripAccess(ctx context.Context, tripID, userID string) (TripAccess, error) {
	return ResolveTripAccess(ctx, s.repo, tripID, userID)
}

func (s *Service) GetUserByID(ctx context.Context, id string) (User, error) {
	return s.repo.GetUserByID(ctx, id)
}

func (s *Service) UpdateUserProfile(ctx context.Context, userID, email, username, displayName string) (User, error) {
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return u, err
	}
	email = strings.TrimSpace(strings.ToLower(email))
	if err := validateUsername(username); err != nil {
		return u, err
	}
	un := normalizeUsername(username)
	if ok, _ := s.repo.EmailExists(ctx, email, userID); ok {
		return u, errors.New("that email is already in use")
	}
	if ok, _ := s.repo.UsernameExists(ctx, un, userID); ok {
		return u, errors.New("that username is already in use")
	}
	oldEmail := u.Email
	u.Email = email
	u.Username = un
	u.DisplayName = strings.TrimSpace(displayName)
	if oldEmail != email {
		u.EmailVerifiedAt = time.Time{}
	}
	if err := s.repo.UpdateUser(ctx, u); err != nil {
		return u, err
	}
	return s.repo.GetUserByID(ctx, userID)
}

func (s *Service) UpdateUserPassword(ctx context.Context, userID, currentPassword, newPassword string) error {
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if !CheckPassword(u.PasswordHash, currentPassword) {
		return ErrWrongCurrentPassword
	}
	h, err := HashPassword(newPassword)
	if err != nil {
		return err
	}
	u.PasswordHash = h
	return s.repo.UpdateUser(ctx, u)
}

func (s *Service) SetUserAvatarPath(ctx context.Context, userID, path string) error {
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	u.AvatarPath = strings.TrimSpace(path)
	return s.repo.UpdateUser(ctx, u)
}

// IssueEmailVerificationToken stores a token; in dev log the link (no SMTP in this build).
func (s *Service) IssueEmailVerificationToken(ctx context.Context, userID string) (rawToken string, err error) {
	rawToken = randomTokenRaw()
	if err := s.repo.ReplaceEmailVerifyToken(ctx, userID, rawToken, emailVerifyTTL); err != nil {
		return "", err
	}
	return rawToken, nil
}

func (s *Service) VerifyEmailToken(ctx context.Context, raw string) error {
	_, err := s.repo.ConsumeEmailVerifyToken(ctx, raw)
	return err
}

// ListUsersForManagement returns all users with password hashes cleared (admin only).
func (s *Service) ListUsersForManagement(ctx context.Context, actorID string) ([]User, error) {
	if actorID == "" {
		return nil, ErrAuthRequired
	}
	actor, err := s.repo.GetUserByID(ctx, actorID)
	if err != nil {
		return nil, err
	}
	if !actor.IsAdmin {
		return nil, ErrAdminRequired
	}
	list, err := s.repo.ListAllUsers(ctx)
	if err != nil {
		return nil, err
	}
	for i := range list {
		list[i].PasswordHash = ""
	}
	return list, nil
}

// SetUserAdministrator promotes or demotes a user (admin only; cannot remove the last administrator).
func (s *Service) SetUserAdministrator(ctx context.Context, actorID, targetUserID string, makeAdmin bool) error {
	if actorID == "" || targetUserID == "" {
		return ErrAuthRequired
	}
	actor, err := s.repo.GetUserByID(ctx, actorID)
	if err != nil {
		return err
	}
	if !actor.IsAdmin {
		return ErrAdminRequired
	}
	if _, err := s.repo.GetUserByID(ctx, targetUserID); err != nil {
		return err
	}
	if !makeAdmin {
		tgt, err := s.repo.GetUserByID(ctx, targetUserID)
		if err != nil {
			return err
		}
		if !tgt.IsAdmin {
			return nil
		}
		n, err := s.repo.CountAdmins(ctx)
		if err != nil {
			return err
		}
		if n <= 1 {
			return errors.New("cannot remove the last administrator")
		}
	}
	return s.repo.SetUserIsAdmin(ctx, targetUserID, makeAdmin)
}

func (s *Service) TripParty(ctx context.Context, tripID string) ([]UserProfile, error) {
	return s.repo.ListTripPartyProfiles(ctx, tripID)
}

func (s *Service) TripCollaboratorCount(ctx context.Context, tripID string) (int, error) {
	return s.repo.CountActiveCollaborators(ctx, tripID)
}

// InviteCollaboratorByEmail adds an existing user to the trip, or creates a pending email invite.
// When the user already has an account, it returns addedExistingUser=true (no email invite row).
func (s *Service) InviteCollaboratorByEmail(ctx context.Context, tripID, ownerUserID, email string) (addedExistingUser bool, err error) {
	acc, err := s.TripAccess(ctx, tripID, ownerUserID)
	if err != nil {
		return false, err
	}
	if !acc.IsOwner {
		return false, ErrTripAccessDenied
	}
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" {
		return false, errors.New("email is required")
	}
	u, e := s.repo.GetUserByEmail(ctx, email)
	if e == nil && u.ID != "" {
		if u.ID == ownerUserID {
			return false, errors.New("you are already the owner")
		}
		if err := s.repo.AddTripMember(ctx, tripID, u.ID, ownerUserID); err != nil {
			return false, err
		}
		_ = s.repo.ClearDepartedTabParticipant(ctx, tripID, ParticipantKeyUser(u.ID))
		return true, nil
	}
	if !errors.Is(e, sql.ErrNoRows) {
		return false, e
	}
	rawToken := randomTokenRaw()
	inv := TripInvite{
		TripID:          tripID,
		EmailNormalized: email,
		InvitedByUserID: ownerUserID,
		ExpiresAt:       time.Now().UTC().Add(inviteTTL),
	}
	if err := s.repo.CreateTripInvite(ctx, inv, rawToken); err != nil {
		return false, err
	}
	return false, nil
}

// ListPendingTripInvitesForTrip returns pending email invites for owners; empty for non-owners.
func (s *Service) ListPendingTripInvitesForTrip(ctx context.Context, tripID, actorID string) ([]TripInvitePending, error) {
	acc, err := s.TripAccess(ctx, tripID, actorID)
	if err != nil {
		return nil, err
	}
	if !acc.IsOwner {
		return []TripInvitePending{}, nil
	}
	return s.repo.ListPendingTripInvitesForTrip(ctx, tripID)
}

// OwnerRemoveTripMember removes an active collaborator (not the trip owner).
func (s *Service) OwnerRemoveTripMember(ctx context.Context, tripID, actorID, targetUserID string) error {
	acc, err := s.TripAccess(ctx, tripID, actorID)
	if err != nil {
		return err
	}
	if !acc.IsOwner {
		return ErrTripAccessDenied
	}
	t, err := s.repo.GetTrip(ctx, tripID)
	if err != nil {
		return err
	}
	if targetUserID == t.OwnerUserID {
		return errors.New("cannot remove the trip owner")
	}
	return s.repo.MarkTripMemberLeft(ctx, tripID, targetUserID)
}

// OwnerRevokeTripInvite revokes a single pending invite.
func (s *Service) OwnerRevokeTripInvite(ctx context.Context, tripID, actorID, inviteID string) error {
	acc, err := s.TripAccess(ctx, tripID, actorID)
	if err != nil {
		return err
	}
	if !acc.IsOwner {
		return ErrTripAccessDenied
	}
	return s.repo.RevokeTripInviteForTrip(ctx, tripID, inviteID)
}

func (s *Service) PreviewTripInvite(ctx context.Context, rawToken string) (TripInviteLookup, error) {
	return s.repo.GetTripInviteByTokenRaw(ctx, rawToken)
}

func (s *Service) AcceptTripInvite(ctx context.Context, userID, rawToken string) error {
	inv, err := s.repo.GetTripInviteByTokenRaw(ctx, rawToken)
	if err != nil {
		return err
	}
	t, err := s.repo.GetTrip(ctx, inv.TripID)
	if err != nil {
		return err
	}
	if t.OwnerUserID == userID {
		return errors.New("you already own this trip")
	}
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return err
	}
	if !inv.IsLinkInvite {
		if strings.ToLower(strings.TrimSpace(u.Email)) != inv.EmailNormalized {
			return errors.New("sign in with the invited email to accept")
		}
	}
	if err := s.repo.AddTripMember(ctx, inv.TripID, userID, inv.InvitedByUserID); err != nil {
		return err
	}
	_ = s.repo.ClearDepartedTabParticipant(ctx, inv.TripID, ParticipantKeyUser(userID))
	if inv.IsLinkInvite {
		return nil
	}
	return s.repo.MarkTripInviteAccepted(ctx, inv.ID)
}

// CreateTripInviteLink rotates any previous link for the trip and returns a new secret token (multi-use until rotated or revoked).
func (s *Service) CreateTripInviteLink(ctx context.Context, tripID, ownerUserID string) (rawToken string, err error) {
	acc, err := s.TripAccess(ctx, tripID, ownerUserID)
	if err != nil {
		return "", err
	}
	if !acc.IsOwner {
		return "", ErrTripAccessDenied
	}
	if err := s.repo.RevokeAllTripInviteLinksForTrip(ctx, tripID); err != nil {
		return "", err
	}
	rawToken = randomTokenRaw()
	expires := time.Now().UTC().Add(inviteTTL)
	if err := s.repo.CreateTripInviteLink(ctx, tripID, ownerUserID, rawToken, expires); err != nil {
		return "", err
	}
	return rawToken, nil
}

func (s *Service) LeaveTrip(ctx context.Context, tripID, userID string) error {
	acc, err := s.TripAccess(ctx, tripID, userID)
	if err != nil {
		return err
	}
	if acc.IsOwner {
		return errors.New("owner cannot leave; transfer ownership is not supported")
	}
	return s.markCollaboratorLeftRecordTab(ctx, tripID, userID)
}

func (s *Service) markCollaboratorLeftRecordTab(ctx context.Context, tripID, userID string) error {
	display := ParticipantKeyUser(userID)
	if u, err := s.repo.GetUserByID(ctx, userID); err == nil {
		display = userProfileDisplayName(u)
	}
	_ = s.repo.UpsertDepartedTabParticipant(ctx, tripID, ParticipantKeyUser(userID), display)
	return s.repo.MarkTripMemberLeft(ctx, tripID, userID)
}

func (s *Service) StopSharingTrip(ctx context.Context, tripID, ownerUserID string) error {
	ok, err := s.repo.IsTripOwner(ctx, tripID, ownerUserID)
	if err != nil {
		return err
	}
	if !ok {
		return ErrTripAccessDenied
	}
	t, err := s.repo.GetTrip(ctx, tripID)
	if err != nil {
		return err
	}
	memberIDs, err := s.repo.ListActiveTripMemberUserIDs(ctx, tripID)
	if err != nil {
		return err
	}
	for _, id := range memberIDs {
		if id == t.OwnerUserID {
			continue
		}
		display := ParticipantKeyUser(id)
		if u, e := s.repo.GetUserByID(ctx, id); e == nil {
			display = userProfileDisplayName(u)
		}
		_ = s.repo.UpsertDepartedTabParticipant(ctx, tripID, ParticipantKeyUser(id), display)
	}
	if err := s.repo.RevokeAllCollaborators(ctx, tripID); err != nil {
		return err
	}
	if err := s.repo.RevokePendingTripInvites(ctx, tripID); err != nil {
		return err
	}
	return s.repo.RevokeAllTripInviteLinksForTrip(ctx, tripID)
}

func (s *Service) SetArchivedTripHidden(ctx context.Context, tripID, userID, actorUserID string, hidden bool) error {
	acc, err := s.TripAccess(ctx, tripID, actorUserID)
	if err != nil {
		return err
	}
	if acc.IsOwner {
		return errors.New("owner does not use hide; archive state is shared")
	}
	if userID != actorUserID {
		return ErrTripAccessDenied
	}
	t, err := s.repo.GetTrip(ctx, tripID)
	if err != nil {
		return err
	}
	if !t.IsArchived {
		return errors.New("hide is only for archived shared trips")
	}
	return s.repo.SetTripArchivedHiddenForUser(ctx, tripID, userID, hidden)
}

// IsArchivedTripHiddenOnDashboard is true when a collaborator chose to hide this archived trip from their dashboard only.
func (s *Service) IsArchivedTripHiddenOnDashboard(ctx context.Context, tripID, userID string) (bool, error) {
	return s.repo.IsArchivedTripHiddenOnDashboard(ctx, tripID, userID)
}
