package httpapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTripNotesPageCopy validates notes/checklists copy on /notes and shared composer
// templates used from the trip page (sidebar, mobile sheet).
func TestTripNotesPageCopy(t *testing.T) {
	root := findModuleRoot(t)

	notesB, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip_notes.html"))
	if err != nil {
		t.Fatal(err)
	}
	notes := string(notesB)

	composerB, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip_keep_composer_tabbed.html"))
	if err != nil {
		t.Fatal(err)
	}
	composer := string(composerB)

	boardB, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip_keep_notes_board_inner.html"))
	if err != nil {
		t.Fatal(err)
	}
	board := string(boardB)

	previewB, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip_keep_details_preview_inner.html"))
	if err != nil {
		t.Fatal(err)
	}
	preview := string(previewB)

	tripB, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip.html"))
	if err != nil {
		t.Fatal(err)
	}
	trip := string(tripB)

	fabB, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip_mobile_fab_links.html"))
	if err != nil {
		t.Fatal(err)
	}
	fab := string(fabB)

	notesWant := []string{
		"Capture notes and checklists for your trip in one place.",
		"Import Notes &amp; Checklists",
		"/notes/import",
		"Notes and checklist items with a due date will follow your notification settings.",
		"Archived items are hidden until restored.",
		"Deleted items are permanently removed when purged.",
	}
	for _, s := range notesWant {
		if !strings.Contains(notes, s) {
			t.Errorf("trip_notes.html missing %q", s)
		}
	}
	notesAvoid := []string{
		"Capture ideas and packing lines in one calm grid",
		"Notifications still follow your reminder settings.",
		"Archived cards stay out of your main grid until you restore them.",
		"Trash is deleted permanently when you purge an item.",
	}
	for _, s := range notesAvoid {
		if strings.Contains(notes, s) {
			t.Errorf("trip_notes.html should not contain %q", s)
		}
	}

	composerWant := []string{
		"Create a note or a checklist.",
		">Add Note</button>",
		">Add Checklist</button>",
		"Title is optional. Add notes, links, or reminders.",
		`placeholder="Note Title (e.g. Hidden gems in Tokyo)"`,
		`placeholder="Write your notes, links, or reminders..."`,
		"Set Reminder (optional)",
		">Save Note</button>",
		`<option value="" disabled selected>Select a category</option>`,
		`data-checklist-add-btn>Add Item</button>`,
		"No items added yet.",
		`type="submit">Add Checklist</button>`,
	}
	for _, s := range composerWant {
		if !strings.Contains(composer, s) {
			t.Errorf("trip_keep_composer_tabbed.html missing %q", s)
		}
	}
	composerAvoid := []string{
		"Save a note or create a batch of checklist items.",
		"Save a Note",
		"Save a Checklist",
		"Title optional. Body supports short plans, links, or reminders.",
		"Hidden Gems in Tokyo",
		"Write details, links, or reminders here...",
		"Set Reminder (Optional)",
		">Save note</button>",
		`<option value="" disabled selected>Select category</option>`,
		`data-checklist-add-btn>Add</button>`,
		"Save Checklist",
	}
	for _, s := range composerAvoid {
		if strings.Contains(composer, s) {
			t.Errorf("trip_keep_composer_tabbed.html should not contain %q", s)
		}
	}

	for _, s := range []string{
		`placeholder="Note Title (e.g. Hidden gems in Tokyo)"`,
		`placeholder="Write your notes, links, or reminders..."`,
		"Set Reminder (optional)",
		"Nothing here yet",
	} {
		if !strings.Contains(board, s) {
			t.Errorf("trip_keep_notes_board_inner.html missing %q", s)
		}
	}
	if strings.Contains(board, "Nothing in this view yet.") || strings.Contains(board, "Hidden Gems in Tokyo") {
		t.Error("trip_keep_notes_board_inner.html should use updated empty state and title placeholder")
	}
	if strings.Contains(board, "Set Reminder (Optional)") {
		t.Error("trip_keep_notes_board_inner.html should use Set Reminder (optional)")
	}

	for _, s := range []string{
		`placeholder="Note Title (e.g. Hidden gems in Tokyo)"`,
		`placeholder="Write your notes, links, or reminders..."`,
		"Set Reminder (optional)",
	} {
		if !strings.Contains(preview, s) {
			t.Errorf("trip_keep_details_preview_inner.html missing %q", s)
		}
	}
	if strings.Contains(preview, "Hidden Gems in Tokyo") || strings.Contains(preview, "Set Reminder (Optional)") {
		t.Error("trip_keep_details_preview_inner.html should match updated note edit copy")
	}

	if !strings.Contains(trip, `template "tripKeepComposerTabbed"`) {
		t.Error("trip.html should embed tripKeepComposerTabbed for notes/checklist")
	}
	if !strings.Contains(fab, "Add Note / Checklist") {
		t.Error("trip_mobile_fab_links.html should keep Add Note / Checklist shortcut label")
	}
}
