package trips

import (
	"testing"
	"time"
)

func TestDraftItineraryDayRoundTrip(t *testing.T) {
	d := time.Date(2026, 4, 18, 0, 0, 0, 0, time.Local)
	n := DraftItineraryDayNumberFromDate(d)
	if n != 20260418 {
		t.Fatalf("encode: got %d want 20260418", n)
	}
	iso, ok := CalendarDateFromDraftItineraryDayNumber(n)
	if !ok || iso != "2026-04-18" {
		t.Fatalf("decode: ok=%v iso=%q", ok, iso)
	}
}

func TestCalendarDateFromDraftItineraryDayNumberRejectsRelative(t *testing.T) {
	_, ok := CalendarDateFromDraftItineraryDayNumber(3)
	if ok {
		t.Fatal("expected relative day 3 to be rejected")
	}
}

func TestIsDraftTripForDateBounds(t *testing.T) {
	if !IsDraftTripForDateBounds("", "") {
		t.Fatal("empty bounds should be draft")
	}
	if !IsDraftTripForDateBounds("2026-01-01", "") {
		t.Fatal("missing end should be draft")
	}
	if IsDraftTripForDateBounds("2026-01-01", "2026-01-10") {
		t.Fatal("valid bounds should not be draft")
	}
	if !IsDraftTripForDateBounds("not-a-date", "2026-01-10") {
		t.Fatal("invalid start should be draft")
	}
}

func TestRelativeDayNumberInTripWindow(t *testing.T) {
	d := time.Date(2026, 6, 15, 12, 0, 0, 0, time.Local)
	n, err := RelativeDayNumberInTripWindow("2026-06-01", "2026-06-30", d)
	if err != nil || n != 15 {
		t.Fatalf("got %d err=%v want 15", n, err)
	}
	early := time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local)
	n2, err := RelativeDayNumberInTripWindow("2026-06-01", "2026-06-30", early)
	if err != nil || n2 != 1 {
		t.Fatalf("clamp early: got %d err=%v want 1", n2, err)
	}
	late := time.Date(2026, 12, 31, 0, 0, 0, 0, time.Local)
	n3, err := RelativeDayNumberInTripWindow("2026-06-01", "2026-06-30", late)
	if err != nil || n3 != 30 {
		t.Fatalf("clamp late: got %d err=%v want 30", n3, err)
	}
}
