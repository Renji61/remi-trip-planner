package trips

import "time"

// GlobalKeepNote is a user-scoped note template (not tied to a trip).
type GlobalKeepNote struct {
	ID        string
	UserID    string
	Title     string
	Body      string
	Color     string
	DueAt     string
	Pinned    bool
	Archived  bool
	Trashed   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// GlobalChecklistTemplate is a reusable checklist (category + lines) for the signed-in user.
type GlobalChecklistTemplate struct {
	ID        string
	UserID    string
	Category  string
	DueAt     string
	Lines     []string
	Pinned    bool
	Archived  bool
	Trashed   bool
	CreatedAt time.Time
	UpdatedAt time.Time
}

// GlobalKeepImportKind identifies what was imported from the global library into a trip.
type GlobalKeepImportKind string

const (
	GlobalKeepImportNote      GlobalKeepImportKind = "note"
	GlobalKeepImportChecklist GlobalKeepImportKind = "checklist"
)
