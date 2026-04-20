package httpapp

import (
	"net/http/httptest"
	"strings"
	"testing"

	"remi-trip-planner/internal/trips"
)

func TestParseItineraryDayAndTimesFromForm_StartOnly(t *testing.T) {
	t.Parallel()
	trip := trips.Trip{StartDate: "2026-04-01", EndDate: "2026-04-10"}
	body := "start_at=2026-04-05T14%3A30&end_at="
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := req.ParseForm(); err != nil {
		t.Fatal(err)
	}
	day, startHM, endHM, err := parseItineraryDayAndTimesFromForm(req, trip)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if day != 5 || startHM != "14:30" || endHM != "" {
		t.Fatalf("got day=%d start=%q end=%q", day, startHM, endHM)
	}
}

func TestParseItineraryDayAndTimesFromForm_StartAndEnd(t *testing.T) {
	t.Parallel()
	trip := trips.Trip{StartDate: "2026-04-01", EndDate: "2026-04-10"}
	body := "start_at=2026-04-05T09%3A00&end_at=2026-04-05T11%3A30"
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := req.ParseForm(); err != nil {
		t.Fatal(err)
	}
	day, startHM, endHM, err := parseItineraryDayAndTimesFromForm(req, trip)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if day != 5 || startHM != "09:00" || endHM != "11:30" {
		t.Fatalf("got day=%d start=%q end=%q", day, startHM, endHM)
	}
}

func TestParseItineraryDayAndTimesFromForm_EndBeforeStart(t *testing.T) {
	t.Parallel()
	trip := trips.Trip{StartDate: "2026-04-01", EndDate: "2026-04-10"}
	body := "start_at=2026-04-05T12%3A00&end_at=2026-04-05T11%3A00"
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := req.ParseForm(); err != nil {
		t.Fatal(err)
	}
	_, _, _, err := parseItineraryDayAndTimesFromForm(req, trip)
	if err == nil || !strings.Contains(err.Error(), "end time") {
		t.Fatalf("want end time error, got %v", err)
	}
}

func TestParseItineraryDayAndTimesFromForm_EndWithoutStart(t *testing.T) {
	t.Parallel()
	trip := trips.Trip{StartDate: "2026-04-01", EndDate: "2026-04-10"}
	body := "end_at=2026-04-05T11%3A00"
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := req.ParseForm(); err != nil {
		t.Fatal(err)
	}
	_, _, _, err := parseItineraryDayAndTimesFromForm(req, trip)
	if err == nil || !strings.Contains(err.Error(), "start date and time is required") {
		t.Fatalf("want start required error, got %v", err)
	}
}

func TestParseCommuteLegTimesFromForm_OvernightNextCalendarDay(t *testing.T) {
	t.Parallel()
	trip := trips.Trip{StartDate: "2026-04-01", EndDate: "2026-04-10"}
	body := "start_at=2026-04-05T23%3A00&end_at=2026-04-06T07%3A30"
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := req.ParseForm(); err != nil {
		t.Fatal(err)
	}
	startHM, endHM, off, err := parseCommuteLegTimesFromForm(req, trip)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if startHM != "23:00" || endHM != "07:30" || off != 1 {
		t.Fatalf("got start=%q end=%q off=%d", startHM, endHM, off)
	}
}

func TestParseCommuteLegTimesFromForm_EndNotAfterStart(t *testing.T) {
	t.Parallel()
	trip := trips.Trip{StartDate: "2026-04-01", EndDate: "2026-04-10"}
	body := "start_at=2026-04-05T12%3A00&end_at=2026-04-05T11%3A00"
	req := httptest.NewRequest("POST", "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	if err := req.ParseForm(); err != nil {
		t.Fatal(err)
	}
	_, _, _, err := parseCommuteLegTimesFromForm(req, trip)
	if err == nil || !strings.Contains(err.Error(), "end must be after start") {
		t.Fatalf("want end after start error, got %v", err)
	}
}
