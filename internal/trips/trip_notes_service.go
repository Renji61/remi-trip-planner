package trips

import (
	"context"
	"errors"
	"strings"
)

// NormalizeKeepView maps query values to a known keep sidebar view.
func NormalizeKeepView(v string) string {
	switch strings.TrimSpace(strings.ToLower(v)) {
	case KeepViewReminders, KeepViewArchive, KeepViewTrash:
		return strings.TrimSpace(strings.ToLower(v))
	default:
		return KeepViewNotes
	}
}

// NormalizeKeepChecklistCategory matches grouping on the Notes & lists board (empty → "Lists").
func NormalizeKeepChecklistCategory(cat string) string {
	c := strings.TrimSpace(cat)
	if c == "" {
		return "Lists"
	}
	return c
}

func (s *Service) ListTripNotesForKeepView(ctx context.Context, tripID, view string) ([]TripNote, error) {
	if tripID == "" {
		return nil, errors.New("trip id is required")
	}
	return s.repo.ListTripNotesForKeepView(ctx, tripID, NormalizeKeepView(view))
}

func (s *Service) ListChecklistItemsForKeepView(ctx context.Context, tripID, view string) ([]ChecklistItem, error) {
	if tripID == "" {
		return nil, errors.New("trip id is required")
	}
	return s.repo.ListChecklistItemsForKeepView(ctx, tripID, NormalizeKeepView(view))
}

func (s *Service) AddTripNote(ctx context.Context, n TripNote) error {
	if n.TripID == "" {
		return errors.New("trip id is required")
	}
	trip, err := s.repo.GetTrip(ctx, n.TripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	n.Title = strings.TrimSpace(n.Title)
	n.Body = strings.TrimSpace(n.Body)
	n.Color = strings.TrimSpace(n.Color)
	n.DueAt = strings.TrimSpace(n.DueAt)
	return s.repo.AddTripNote(ctx, n)
}

func (s *Service) GetTripNote(ctx context.Context, noteID string) (TripNote, error) {
	return s.repo.GetTripNote(ctx, noteID)
}

func (s *Service) UpdateTripNote(ctx context.Context, n TripNote) error {
	if n.ID == "" || n.TripID == "" {
		return errors.New("invalid note")
	}
	existing, err := s.repo.GetTripNote(ctx, n.ID)
	if err != nil {
		return err
	}
	if existing.TripID != n.TripID {
		return errors.New("note not in this trip")
	}
	trip, err := s.repo.GetTrip(ctx, n.TripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	n.Title = strings.TrimSpace(n.Title)
	n.Body = strings.TrimSpace(n.Body)
	n.Color = strings.TrimSpace(n.Color)
	n.DueAt = strings.TrimSpace(n.DueAt)
	n.CreatedAt = existing.CreatedAt
	return s.repo.UpdateTripNote(ctx, n)
}

// DeleteTripNoteHard removes a note regardless of trash state (used by sync delete).
func (s *Service) DeleteTripNoteHard(ctx context.Context, tripID, noteID string) error {
	if tripID == "" || noteID == "" {
		return errors.New("invalid note")
	}
	n, err := s.repo.GetTripNote(ctx, noteID)
	if err != nil {
		return err
	}
	if n.TripID != tripID {
		return errors.New("note not in this trip")
	}
	trip, err := s.repo.GetTrip(ctx, tripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	return s.repo.DeleteTripNote(ctx, noteID)
}

func (s *Service) DeleteTripNote(ctx context.Context, tripID, noteID string) error {
	if tripID == "" || noteID == "" {
		return errors.New("invalid note")
	}
	n, err := s.repo.GetTripNote(ctx, noteID)
	if err != nil {
		return err
	}
	if n.TripID != tripID {
		return errors.New("note not in this trip")
	}
	if !n.Trashed {
		return errors.New("only notes in trash can be deleted permanently")
	}
	trip, err := s.repo.GetTrip(ctx, tripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	return s.repo.DeleteTripNote(ctx, noteID)
}

func (s *Service) ListPinnedChecklistCategories(ctx context.Context, tripID string) ([]string, error) {
	if tripID == "" {
		return nil, errors.New("trip id is required")
	}
	return s.repo.ListPinnedChecklistCategories(ctx, tripID)
}

func (s *Service) SetChecklistCategoryPinned(ctx context.Context, tripID, category string, pinned bool) error {
	if tripID == "" {
		return errors.New("trip id is required")
	}
	trip, err := s.repo.GetTrip(ctx, tripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	cat := NormalizeKeepChecklistCategory(category)
	return s.repo.SetChecklistCategoryPinned(ctx, tripID, cat, pinned)
}

func (s *Service) UpdateChecklistItemKeepFlags(ctx context.Context, tripID, itemID string, archived, trashed bool) error {
	if tripID == "" || itemID == "" {
		return errors.New("invalid checklist item")
	}
	existing, err := s.repo.GetChecklistItem(ctx, itemID)
	if err != nil {
		return err
	}
	if existing.TripID != tripID {
		return errors.New("checklist item not in this trip")
	}
	trip, err := s.repo.GetTrip(ctx, tripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	existing.Archived = archived
	existing.Trashed = trashed
	return s.UpdateChecklistItem(ctx, existing)
}
