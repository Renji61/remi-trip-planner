package trips

import (
	"context"
	"errors"
	"strings"

	"github.com/google/uuid"
)

func (s *Service) ListGlobalKeepNotesByUser(ctx context.Context, userID string) ([]GlobalKeepNote, error) {
	return s.ListGlobalKeepNotesForKeepView(ctx, userID, KeepViewNotes)
}

func (s *Service) ListGlobalKeepNotesForKeepView(ctx context.Context, userID, view string) ([]GlobalKeepNote, error) {
	if userID == "" {
		return nil, errors.New("user id is required")
	}
	return s.repo.ListGlobalKeepNotesForKeepView(ctx, userID, NormalizeKeepView(view))
}

func (s *Service) AddGlobalKeepNote(ctx context.Context, n GlobalKeepNote) error {
	if n.UserID == "" {
		return errors.New("user id is required")
	}
	n.Title = strings.TrimSpace(n.Title)
	n.Body = strings.TrimSpace(n.Body)
	if n.Title == "" && n.Body == "" {
		return errors.New("note needs a title or body")
	}
	return s.repo.AddGlobalKeepNote(ctx, n)
}

func (s *Service) DeleteGlobalKeepNote(ctx context.Context, userID, noteID string) error {
	if userID == "" || noteID == "" {
		return errors.New("invalid delete")
	}
	return s.repo.DeleteGlobalKeepNote(ctx, userID, noteID)
}

func (s *Service) UpdateGlobalKeepNote(ctx context.Context, n GlobalKeepNote) error {
	if n.UserID == "" || n.ID == "" {
		return errors.New("invalid note update")
	}
	return s.repo.UpdateGlobalKeepNote(ctx, n)
}

func (s *Service) GetGlobalKeepNote(ctx context.Context, userID, noteID string) (GlobalKeepNote, error) {
	if userID == "" || noteID == "" {
		return GlobalKeepNote{}, errors.New("invalid get")
	}
	return s.repo.GetGlobalKeepNote(ctx, userID, noteID)
}

func (s *Service) ListGlobalChecklistTemplatesByUser(ctx context.Context, userID string) ([]GlobalChecklistTemplate, error) {
	return s.ListGlobalChecklistTemplatesForKeepView(ctx, userID, KeepViewNotes)
}

func (s *Service) ListGlobalChecklistTemplatesForKeepView(ctx context.Context, userID, view string) ([]GlobalChecklistTemplate, error) {
	if userID == "" {
		return nil, errors.New("user id is required")
	}
	return s.repo.ListGlobalChecklistTemplatesForKeepView(ctx, userID, NormalizeKeepView(view))
}

func (s *Service) AddGlobalChecklistTemplate(ctx context.Context, t GlobalChecklistTemplate, lines []string) error {
	if t.UserID == "" {
		return errors.New("user id is required")
	}
	var nonempty []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			nonempty = append(nonempty, line)
		}
	}
	if len(nonempty) == 0 {
		return errors.New("add at least one checklist line")
	}
	if strings.TrimSpace(t.Category) == "" {
		t.Category = "Packing List"
	}
	return s.repo.AddGlobalChecklistTemplate(ctx, t, nonempty)
}

func (s *Service) DeleteGlobalChecklistTemplate(ctx context.Context, userID, templateID string) error {
	if userID == "" || templateID == "" {
		return errors.New("invalid delete")
	}
	return s.repo.DeleteGlobalChecklistTemplate(ctx, userID, templateID)
}

func (s *Service) UpdateGlobalChecklistTemplate(ctx context.Context, t GlobalChecklistTemplate) error {
	if t.UserID == "" || t.ID == "" {
		return errors.New("invalid checklist template update")
	}
	return s.repo.UpdateGlobalChecklistTemplate(ctx, t)
}

func (s *Service) GetGlobalChecklistTemplate(ctx context.Context, userID, templateID string) (GlobalChecklistTemplate, error) {
	if userID == "" || templateID == "" {
		return GlobalChecklistTemplate{}, errors.New("invalid get")
	}
	return s.repo.GetGlobalChecklistTemplate(ctx, userID, templateID)
}

// ListGlobalKeepImportedIDs returns global note or checklist template IDs already imported into the trip.
func (s *Service) ListGlobalKeepImportedIDs(ctx context.Context, tripID string, kind GlobalKeepImportKind) ([]string, error) {
	if tripID == "" {
		return nil, errors.New("trip id is required")
	}
	return s.repo.ListGlobalKeepImportedIDs(ctx, tripID, kind)
}

// ImportGlobalKeepIntoTrip copies selected global notes and checklist templates into the trip.
// Skips IDs that are not owned by userID, already imported, or invalid. Returns how many distinct global items were newly imported.
func (s *Service) ImportGlobalKeepIntoTrip(ctx context.Context, userID, tripID string, noteIDs, checklistTemplateIDs []string) (int, error) {
	if userID == "" || tripID == "" {
		return 0, errors.New("user and trip are required")
	}
	trip, err := s.repo.GetTrip(ctx, tripID)
	if err != nil {
		return 0, err
	}
	if trip.IsArchived {
		return 0, errors.New("archived trips are read-only")
	}
	if !trip.SectionEnabledChecklist() {
		return 0, errors.New("notes and checklists are disabled for this trip")
	}

	noteSeen := map[string]struct{}{}
	var noteList []string
	for _, raw := range noteIDs {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := noteSeen[id]; ok {
			continue
		}
		noteSeen[id] = struct{}{}
		noteList = append(noteList, id)
	}
	chSeen := map[string]struct{}{}
	var chList []string
	for _, raw := range checklistTemplateIDs {
		id := strings.TrimSpace(raw)
		if id == "" {
			continue
		}
		if _, ok := chSeen[id]; ok {
			continue
		}
		chSeen[id] = struct{}{}
		chList = append(chList, id)
	}

	imported := 0

	for _, gid := range noteList {
		ok, err := s.repo.IsGlobalKeepImported(ctx, tripID, GlobalKeepImportNote, gid)
		if err != nil {
			return imported, err
		}
		if ok {
			continue
		}
		n, err := s.repo.GetGlobalKeepNote(ctx, userID, gid)
		if err != nil {
			continue
		}
		if n.Trashed || n.Archived {
			continue
		}
		if err := s.repo.AddTripNote(ctx, TripNote{
			ID:     uuid.NewString(),
			TripID: tripID,
			Title:  n.Title,
			Body:   n.Body,
			Color:  n.Color,
			DueAt:  n.DueAt,
		}); err != nil {
			return imported, err
		}
		if err := s.repo.RecordGlobalKeepImport(ctx, tripID, GlobalKeepImportNote, gid); err != nil {
			return imported, err
		}
		imported++
	}

	for _, gid := range chList {
		ok, err := s.repo.IsGlobalKeepImported(ctx, tripID, GlobalKeepImportChecklist, gid)
		if err != nil {
			return imported, err
		}
		if ok {
			continue
		}
		tpl, err := s.repo.GetGlobalChecklistTemplate(ctx, userID, gid)
		if err != nil {
			continue
		}
		if tpl.Trashed || tpl.Archived {
			continue
		}
		if len(tpl.Lines) == 0 {
			continue
		}
		category := strings.TrimSpace(tpl.Category)
		if category == "" {
			category = "Packing List"
		}
		dueAt := strings.TrimSpace(tpl.DueAt)
		for _, line := range tpl.Lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if err := s.repo.AddChecklistItem(ctx, ChecklistItem{
				TripID:   tripID,
				Category: category,
				Text:     line,
				DueAt:    dueAt,
			}); err != nil {
				return imported, err
			}
		}
		if err := s.repo.RecordGlobalKeepImport(ctx, tripID, GlobalKeepImportChecklist, gid); err != nil {
			return imported, err
		}
		imported++
	}

	return imported, nil
}
