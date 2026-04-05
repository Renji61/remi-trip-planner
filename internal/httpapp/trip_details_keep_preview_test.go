package httpapp

import (
	"testing"
	"time"

	"remi-trip-planner/internal/trips"
)

func TestCategoryPinnedForTripDetails(t *testing.T) {
	if !categoryPinnedForTripDetails("Packing List", []string{"Lists"}) {
		t.Fatal("Keep pin Lists should match trip Packing List bucket")
	}
	if !categoryPinnedForTripDetails("Travel Documents", []string{"Travel Documents"}) {
		t.Fatal("exact category match")
	}
	if categoryPinnedForTripDetails("Other", []string{"Travel Documents"}) {
		t.Fatal("no false positive")
	}
}

func TestBuildTripDetailsKeepPreviewPinnedVsFallback(t *testing.T) {
	t0 := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	t1 := t0.Add(time.Hour)
	t2 := t0.Add(2 * time.Hour)

	notes := []trips.TripNote{
		{ID: "a", Title: "Old", UpdatedAt: t0},
		{ID: "b", Title: "Mid", UpdatedAt: t1},
		{ID: "c", Title: "New", UpdatedAt: t2},
	}
	groups := []checklistCategoryGroup{
		{Category: "X", Items: []trips.ChecklistItem{{ID: "i1", Text: "x", UpdatedAt: t0}}},
	}

	out := buildTripDetailsKeepPreview(notes, groups, nil)
	if len(out) != 3 {
		t.Fatalf("fallback want 3 cards, got %d", len(out))
	}
	if out[0].Kind != "note" || out[0].Note.ID != "c" {
		t.Fatalf("first should be newest note, got %+v", out[0])
	}

	pinnedNotes := []trips.TripNote{
		{ID: "p", Title: "Pinned", Pinned: true, UpdatedAt: t0},
		{ID: "u", Title: "Unpinned", UpdatedAt: t2},
	}
	out2 := buildTripDetailsKeepPreview(pinnedNotes, groups, nil)
	if len(out2) != 1 || out2[0].Note.ID != "p" {
		t.Fatalf("pinned mode: want only pinned note, got %+v", out2)
	}

	out3 := buildTripDetailsKeepPreview(notes, groups, []string{"X"})
	if len(out3) != 1 || out3[0].Kind != "checklist" {
		t.Fatalf("pinned category only: got %+v", out3)
	}
}
