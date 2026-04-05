package httpapp

import (
	"testing"
	"time"

	"remi-trip-planner/internal/trips"
)

func TestTripNotesReturnURL(t *testing.T) {
	if got := tripNotesReturnURL("abc", trips.KeepViewNotes); got != "/trips/abc/notes" {
		t.Fatalf("default view: %q", got)
	}
	if got := tripNotesReturnURL("abc", trips.KeepViewArchive); got != "/trips/abc/notes?view=archive" {
		t.Fatalf("archive: %q", got)
	}
}

func TestGroupChecklistForKeep(t *testing.T) {
	t1 := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)
	t2 := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC)
	items := []trips.ChecklistItem{
		{ID: "a", Category: "B", Text: "1", CreatedAt: t1},
		{ID: "b", Category: "A", Text: "2", CreatedAt: t2},
		{ID: "c", Category: "", Text: "3", CreatedAt: t2},
	}
	groups := groupChecklistForKeep(items)
	if len(groups) != 3 {
		t.Fatalf("want 3 groups (order of first appearance), got %d %+v", len(groups), groups)
	}
	if groups[0].Category != "B" || len(groups[0].Items) != 1 {
		t.Fatalf("first group: %+v", groups[0])
	}
	if groups[1].Category != "A" {
		t.Fatalf("second group category %q", groups[1].Category)
	}
	if groups[2].Category != "Lists" {
		t.Fatalf("empty category -> Lists, got %q", groups[2].Category)
	}
}

func TestKeepMatchesQuery(t *testing.T) {
	n := trips.TripNote{Title: "Passport", Body: "visa"}
	items := []trips.ChecklistItem{{Text: "socks", Category: "Packing List"}}
	if !keepMatchesQuery("", n, items, "X") {
		t.Fatal("empty query matches")
	}
	if !keepMatchesQuery("pass", n, items, "") {
		t.Fatal("title match")
	}
	if !keepMatchesQuery("visa", n, items, "") {
		t.Fatal("body match")
	}
	if !keepMatchesQuery("packing", n, items, "Packing List") {
		t.Fatal("category match on checklist card")
	}
	if !keepMatchesQuery("socks", n, items, "Other") {
		t.Fatal("item text match")
	}
	if keepMatchesQuery("nomatch", n, items, "Cat") {
		t.Fatal("should not match")
	}
}

func TestBuildKeepMasonryPinnedAndSort(t *testing.T) {
	older := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	newer := time.Date(2026, 2, 1, 12, 0, 0, 0, time.UTC)
	notes := []trips.TripNote{
		{ID: "n1", Title: "Old", UpdatedAt: older, Pinned: false},
		{ID: "n2", Title: "New", UpdatedAt: newer, Pinned: false},
		{ID: "n3", Title: "Pin", UpdatedAt: older, Pinned: true},
	}
	ch := []trips.ChecklistItem{{ID: "c1", Category: "C", Text: "x", CreatedAt: newer, UpdatedAt: newer}}
	cards := buildKeepMasonry(trips.KeepViewNotes, notes, ch, "", nil)
	if len(cards) != 4 {
		t.Fatalf("want 4 cards (notes + checklist group), got %d", len(cards))
	}
	if cards[0].Kind != "note" || cards[0].Note.ID != "n3" {
		t.Fatalf("pinned note first: %+v", cards[0])
	}
	if cards[1].Kind != "note" || cards[1].Note.ID != "n2" {
		t.Fatalf("newer unpinned note before checklist tie: %+v", cards[1])
	}
	if cards[2].Kind != "checklist" || cards[2].Category != "C" {
		t.Fatalf("checklist after newer note (stable tie): %+v", cards[2])
	}
	if cards[3].Kind != "note" || cards[3].Note.ID != "n1" {
		t.Fatalf("older note last: %+v", cards[3])
	}
	grid := buildKeepChecklistGroupsForGrid(trips.KeepViewNotes, ch, "")
	if len(grid) != 1 || grid[0].Category != "C" {
		t.Fatalf("checklist grid: %+v", grid)
	}
}

func TestBuildKeepMasonrySearchFilters(t *testing.T) {
	notes := []trips.TripNote{{ID: "n1", Title: "Alpha", Body: "x"}}
	ch := []trips.ChecklistItem{{ID: "c1", Category: "Beta", Text: "gamma"}}
	cards := buildKeepMasonry(trips.KeepViewNotes, notes, ch, "alpha", nil)
	if len(cards) != 1 || cards[0].Kind != "note" {
		t.Fatalf("expected only note: %+v", cards)
	}
	grid := buildKeepChecklistGroupsForGrid(trips.KeepViewNotes, ch, "gamma")
	if len(grid) != 1 || grid[0].Category != "Beta" {
		t.Fatalf("expected checklist group only: %+v", grid)
	}
	cards = buildKeepMasonry(trips.KeepViewNotes, notes, ch, "gamma", nil)
	if len(cards) != 1 || cards[0].Kind != "checklist" {
		t.Fatalf("expected one checklist card: %+v", cards)
	}
}

func TestReminderSingletonGroups(t *testing.T) {
	items := []trips.ChecklistItem{
		{ID: "1", Category: "A", Text: "a"},
		{ID: "2", Category: "B", Text: "b"},
	}
	g := reminderSingletonGroups(items)
	if len(g) != 2 || len(g[0].Items) != 1 || len(g[1].Items) != 1 {
		t.Fatalf("%+v", g)
	}
}
