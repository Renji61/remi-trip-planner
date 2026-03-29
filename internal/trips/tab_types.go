package trips

import (
	"strings"
	"time"
)

// TripGuest is a named person on a trip who can appear in Tab splits but has no login.
type TripGuest struct {
	ID          string
	TripID      string
	DisplayName string
	CreatedAt   time.Time
}

// DepartedTabParticipant records someone removed from the trip party or guest list
// so Tab math and settlement UIs can still resolve their participant keys.
type DepartedTabParticipant struct {
	TripID         string
	ParticipantKey string
	DisplayName    string
	LeftAt         time.Time
}

// IsGuestKey is true when ParticipantKey refers to a trip guest (for templates / UI).
func (d DepartedTabParticipant) IsGuestKey() bool {
	kind, _, ok := ParseParticipantKey(d.ParticipantKey)
	return ok && kind == "guest"
}

// RowLabel is the short display name before the "(Left trip)" suffix.
func (d DepartedTabParticipant) RowLabel() string {
	s := strings.TrimSpace(d.DisplayName)
	if s == "" {
		return strings.TrimSpace(d.ParticipantKey)
	}
	return s
}

// TabSettlement records a payment toward settling Tab balances.
// PayerUserID and PayeeUserID are participant keys: "user:{id}" or "guest:{id}" (legacy rows may store a bare user UUID, treated as a member).
type TabSettlement struct {
	ID          string
	TripID      string
	PayerUserID string
	PayeeUserID string
	Amount      float64
	Method      string // Cash, Bank Transfer, etc.
	SettledOn   string // date YYYY-MM-DD
	Notes       string
	CreatedAt   time.Time
}
