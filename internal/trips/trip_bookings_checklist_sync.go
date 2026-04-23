package trips

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"github.com/google/uuid"
)

// syncTripBookingsChecklistForFlight keeps the Trip Bookings checklist in sync with flight booking status.
// Call with the latest row from the database (e.g. after add/update).
func (s *Service) syncTripBookingsChecklistForFlight(ctx context.Context, f Flight) error {
	f.BookingStatus = NormalizeBookingStatus(f.BookingStatus)
	title := BookFlightChecklistTitle(f)
	cat := TripBookingsChecklistCategory

	// Stale link: referenced checklist row was removed outside our flow.
	if f.TripBookingsChecklistItemID != "" {
		_, err := s.repo.GetChecklistItem(ctx, f.TripBookingsChecklistItemID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				f.TripBookingsChecklistItemID = ""
				if err := s.repo.UpdateFlightTripBookingsChecklistMeta(ctx, f.TripID, f.ID, "", f.TripBookingsChecklistDismissed); err != nil {
					return err
				}
				return s.syncTripBookingsChecklistForFlight(ctx, f)
			}
			return err
		}
	}

	// Done / Not required: keep checklist line in sync with the airline and strike it (done) so it shows with checklist strikethrough.
	if f.BookingStatus != BookingStatusToBeDone {
		if f.TripBookingsChecklistItemID != "" {
			ch, err := s.repo.GetChecklistItem(ctx, f.TripBookingsChecklistItemID)
			if err != nil {
				return err
			}
			needText := strings.TrimSpace(ch.Text) != title
			needCat := strings.TrimSpace(ch.Category) != cat
			if needText || needCat {
				ch.Text = title
				ch.Category = cat
				if err := s.repo.UpdateChecklistItem(ctx, ch); err != nil {
					return err
				}
			}
			if !ch.Done {
				if err := s.repo.ToggleChecklistItem(ctx, ch.ID, true); err != nil {
					return err
				}
			}
		}
		return nil
	}

	if f.TripBookingsChecklistDismissed {
		return nil
	}

	if f.TripBookingsChecklistItemID != "" {
		ch, err := s.repo.GetChecklistItem(ctx, f.TripBookingsChecklistItemID)
		if err != nil {
			return err
		}
		needText := strings.TrimSpace(ch.Text) != title
		needCat := strings.TrimSpace(ch.Category) != cat
		if needText || needCat {
			ch.Text = title
			ch.Category = cat
			if err := s.repo.UpdateChecklistItem(ctx, ch); err != nil {
				return err
			}
		}
		if ch.Done {
			if err := s.repo.ToggleChecklistItem(ctx, ch.ID, false); err != nil {
				return err
			}
		}
		return nil
	}

	// Create checklist task.
	itemID := uuid.NewString()
	item := ChecklistItem{
		ID:       itemID,
		TripID:   f.TripID,
		Category: cat,
		Text:     title,
	}
	if err := s.repo.AddChecklistItem(ctx, item); err != nil {
		return err
	}
	return s.repo.UpdateFlightTripBookingsChecklistMeta(ctx, f.TripID, f.ID, itemID, false)
}

func (s *Service) applyFlightBookingStatusFromTripBookingsChecklistToggle(ctx context.Context, tripID, checklistItemID string, done bool) error {
	flight, err := s.repo.GetFlightByTripBookingsChecklistItemID(ctx, tripID, checklistItemID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	want := BookingStatusToBeDone
	if done {
		want = BookingStatusDone
	}
	if NormalizeBookingStatus(flight.BookingStatus) == want {
		return nil
	}
	flight.BookingStatus = want
	flight.EnforceOptimisticLock = false
	return s.UpdateFlight(ctx, flight)
}

// applyTripBookingsChecklistTextToFlightIfLinked runs after a checklist edit: if this row is a linked Trip Bookings
// line, push the parsed airline name into the flight.
func (s *Service) applyTripBookingsChecklistTextToFlightIfLinked(ctx context.Context, updated ChecklistItem) error {
	if strings.TrimSpace(updated.Category) != TripBookingsChecklistCategory {
		return nil
	}
	airline, ok := ParseBookFlightChecklistTitle(updated.Text)
	if !ok {
		return nil
	}
	flight, err := s.repo.GetFlightByTripBookingsChecklistItemID(ctx, updated.TripID, updated.ID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil
		}
		return err
	}
	label := airlineLabelForBookChecklist(flight.FlightName)
	if strings.EqualFold(strings.TrimSpace(airline), label) {
		return nil
	}
	if strings.TrimSpace(flight.FlightName) == strings.TrimSpace(airline) {
		return nil
	}
	flight.FlightName = strings.TrimSpace(airline)
	flight.EnforceOptimisticLock = false
	return s.UpdateFlight(ctx, flight)
}
