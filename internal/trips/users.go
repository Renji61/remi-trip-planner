package trips

import "time"

// User is a full account record (internal / service layer).
type User struct {
	ID              string
	Email           string
	Username        string
	DisplayName     string
	PasswordHash    string
	AvatarPath      string
	EmailVerifiedAt time.Time // zero = not verified
	CreatedAt       time.Time
	UpdatedAt       time.Time
	// IsAdmin grants access to instance user management (first setup user is administrator).
	IsAdmin bool
}

// UserProfile is safe to expose in HTML (no password hash).
type UserProfile struct {
	ID          string
	Email       string
	Username    string
	DisplayName string
	AvatarPath  string
}

// PublicDisplayName returns display name, username, or email for UI.
func (p UserProfile) PublicDisplayName() string {
	if p.DisplayName != "" {
		return p.DisplayName
	}
	if p.Username != "" {
		return p.Username
	}
	return p.Email
}

// InitialForAvatar single letter or emoji fallback for avatars without image.
func (p UserProfile) InitialForAvatar() string {
	s := p.PublicDisplayName()
	if s == "" {
		return "?"
	}
	for _, r := range s {
		if r > ' ' {
			return string(r)
		}
	}
	return "?"
}

// Session ties a browser cookie to a user.
type Session struct {
	ID        string
	UserID    string
	TokenHash string
	CSRFToken string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// UserSettings mirrors dashboard/theme fields previously stored only in app_settings.
type UserSettings struct {
	UserID                  string
	ThemePreference         string
	DashboardTripLayout     string
	DashboardTripSort       string
	DashboardHeroBackground string
	TripDashboardHeading    string
	DefaultCurrencyName     string
	DefaultCurrencySymbol   string
	// DistanceUnit: km | mi per user; empty uses app default_distance_unit.
	DistanceUnit string
	UpdatedAt    time.Time
}

// TripInvite is a pending collaboration invite.
type TripInvite struct {
	ID              string
	TripID          string
	EmailNormalized string
	InvitedByUserID string
	ExpiresAt       time.Time
	CreatedAt       time.Time
	AcceptedAt      *time.Time
	RevokedAt       *time.Time
}

// TripInviteLookup is a pending invite loaded by token (for accept flow).
// IsLinkInvite is true for shareable trip links (any signed-in user may join).
type TripInviteLookup struct {
	ID              string
	TripID          string
	EmailNormalized string
	InvitedByUserID string
	ExpiresAt       time.Time
	IsLinkInvite    bool
}

// TripInvitePending is an outstanding email invite shown to the trip owner on the trip page.
type TripInvitePending struct {
	ID    string
	Email string
}
