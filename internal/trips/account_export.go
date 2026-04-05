package trips

import (
	"context"
	"fmt"
	"strings"
	"time"
)

const accountExportVersion = "1.1"

// AccountExport is a portable JSON snapshot for the authenticated user (no password hashes).
type AccountExport struct {
	ExportVersion string                  `json:"export_version"`
	ExportedAt    time.Time               `json:"exported_at"`
	User          AccountExportUser       `json:"user"`
	UserSettings  UserSettings            `json:"user_settings"`
	AppSettings   AppSettings             `json:"app_settings"`
	Trips         []AccountExportTripPack `json:"trips"`
}

// AccountExportUser mirrors safe profile fields only.
type AccountExportUser struct {
	ID              string    `json:"id"`
	Email           string    `json:"email"`
	Username        string    `json:"username"`
	DisplayName     string    `json:"display_name"`
	AvatarPath      string    `json:"avatar_path"`
	EmailVerifiedAt time.Time `json:"email_verified_at,omitempty"`
}

// AccountExportTripPack bundles one trip the user can access.
type AccountExportTripPack struct {
	Details                   TripDetails              `json:"trip_details"`
	Documents                 []TripDocument           `json:"trip_documents"`
	TripNotes                 []TripNote               `json:"trip_notes"`
	GroupExpenseSettlements   []TabSettlement          `json:"group_expense_settlements"`
	TripGuests                []TripGuest              `json:"trip_guests"`
	DepartedGroupParticipants []DepartedTabParticipant `json:"departed_group_expense_participants"`
}

// RedactAppSettingsForExport returns a copy safe to embed in user downloads (secrets masked).
func RedactAppSettingsForExport(a AppSettings) AppSettings {
	if strings.TrimSpace(a.GoogleMapsAPIKey) != "" {
		a.GoogleMapsAPIKey = "[REDACTED]"
	}
	return a
}

// BuildAccountExport aggregates profile, merged-visible settings, and all trips the user may see.
func (s *Service) BuildAccountExport(ctx context.Context, userID string) (AccountExport, error) {
	var out AccountExport
	if strings.TrimSpace(userID) == "" {
		return out, fmt.Errorf("user id required")
	}
	u, err := s.repo.GetUserByID(ctx, userID)
	if err != nil {
		return out, err
	}
	us, err := s.repo.GetUserSettings(ctx, userID)
	if err != nil {
		return out, err
	}
	app, err := s.repo.GetAppSettings(ctx)
	if err != nil {
		return out, err
	}
	tripList, err := s.repo.ListVisibleTripsForUser(ctx, userID)
	if err != nil {
		return out, err
	}
	out = AccountExport{
		ExportVersion: accountExportVersion,
		ExportedAt:    time.Now().UTC(),
		User: AccountExportUser{
			ID:              u.ID,
			Email:           u.Email,
			Username:        u.Username,
			DisplayName:     u.DisplayName,
			AvatarPath:      u.AvatarPath,
			EmailVerifiedAt: u.EmailVerifiedAt,
		},
		UserSettings: us,
		AppSettings:  RedactAppSettingsForExport(app),
		Trips:        make([]AccountExportTripPack, 0, len(tripList)),
	}
	for _, t := range tripList {
		details, err := s.GetTripDetailsVisible(ctx, t.ID, userID)
		if err != nil {
			return out, err
		}
		docs, err := s.repo.ListTripDocuments(ctx, t.ID)
		if err != nil {
			return out, err
		}
		notes, err := s.repo.ListTripNotesForExport(ctx, t.ID)
		if err != nil {
			return out, err
		}
		settlements, err := s.repo.ListTabSettlements(ctx, t.ID)
		if err != nil {
			return out, err
		}
		guests, err := s.repo.ListTripGuests(ctx, t.ID)
		if err != nil {
			return out, err
		}
		departed, err := s.repo.ListDepartedTabParticipants(ctx, t.ID)
		if err != nil {
			return out, err
		}
		out.Trips = append(out.Trips, AccountExportTripPack{
			Details:                   details,
			Documents:                 docs,
			TripNotes:                 notes,
			GroupExpenseSettlements:   settlements,
			TripGuests:                guests,
			DepartedGroupParticipants: departed,
		})
	}
	return out, nil
}
