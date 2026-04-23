package trips

import (
	"strings"
	"testing"
)

func TestMergeOpeningHoursUserInput(t *testing.T) {
	base := `{"date":"2026-04-26","open_mins":480,"close_mins":990}`
	got := MergeOpeningHoursUserInput(base, "  Custom  ")
	if !strings.Contains(got, `"user_opening_hours":"Custom"`) {
		t.Fatalf("expected user_opening_hours in JSON, got %q", got)
	}
	cleared := MergeOpeningHoursUserInput(got, "   ")
	if strings.Contains(cleared, "user_opening_hours") {
		t.Fatalf("expected user_opening_hours removed, got %q", cleared)
	}
}

func TestOpeningHoursCardPrimary_userOverride(t *testing.T) {
	trip := Trip{UITimeFormat: "12h"}
	j := `{"user_opening_hours":"Call ahead","open_mins":480,"close_mins":990}`
	item := ItineraryItem{VenueHoursJSON: j}
	if g := item.OpeningHoursCardPrimary(trip); g != "Call ahead" {
		t.Fatalf("got %q", g)
	}
}

func TestOpeningHoursCardPrimary_range24h(t *testing.T) {
	trip := Trip{UITimeFormat: "24h"}
	j := `{"open_mins":510,"close_mins":1050,"status":"open"}`
	item := ItineraryItem{VenueHoursJSON: j}
	if g := item.OpeningHoursCardPrimary(trip); g != "08:30 – 17:30" {
		t.Fatalf("got %q want 08:30 – 17:30", g)
	}
}

func TestOpeningHoursCardPrimary_legacyOpenSummary(t *testing.T) {
	trip := Trip{UITimeFormat: "12h"}
	j := `{"summary":"🟢 Open: 8:00 AM – 4:30 PM"}`
	item := ItineraryItem{VenueHoursJSON: j}
	if g := item.OpeningHoursCardPrimary(trip); g != "8:00 AM – 4:30 PM" {
		t.Fatalf("got %q", g)
	}
}

func TestOpeningHoursCardPrimary_unavailable(t *testing.T) {
	trip := Trip{}
	j := `{"summary":"🔴 No hours listed for this place."}`
	item := ItineraryItem{VenueHoursJSON: j}
	if g := item.OpeningHoursCardPrimary(trip); g != openingHoursUnavailable {
		t.Fatalf("got %q", g)
	}
}

func TestOpeningHoursCardPrimary_emptyJSON(t *testing.T) {
	trip := Trip{}
	item := ItineraryItem{VenueHoursJSON: ""}
	if g := item.OpeningHoursCardPrimary(trip); g != openingHoursUnavailable {
		t.Fatalf("got %q", g)
	}
}
