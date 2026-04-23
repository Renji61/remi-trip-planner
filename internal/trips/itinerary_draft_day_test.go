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

func TestItineraryDayDateISO(t *testing.T) {
	iso, ok := ItineraryDayDateISO("2026-04-10", 1)
	if !ok || iso != "2026-04-10" {
		t.Fatalf("relative start: ok=%v iso=%q", ok, iso)
	}
	iso2, ok2 := ItineraryDayDateISO("2026-04-10", 3)
	if !ok2 || iso2 != "2026-04-12" {
		t.Fatalf("relative day3: ok=%v iso=%q", ok2, iso2)
	}
	iso3, ok3 := ItineraryDayDateISO("not-a-date", DraftItineraryDayNumberFromDate(time.Date(2026, 4, 18, 0, 0, 0, 0, time.Local)))
	if !ok3 || iso3 != "2026-04-18" {
		t.Fatalf("draft: ok=%v iso=%q", ok3, iso3)
	}
	_, bad := ItineraryDayDateISO("not-a-date", 2)
	if bad {
		t.Fatal("expected no ISO for unparseable start and relative day")
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
