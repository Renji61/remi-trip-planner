package trips_test

import (
	"bytes"
	"context"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	ics "github.com/arran4/golang-ical"

	"remi-trip-planner/internal/storage/sqlite"
	"remi-trip-planner/internal/trips"
)

func testDB(t *testing.T) (*sqlite.Repository, func()) {
	t.Helper()
	_, fname, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(fname), "..", ".."))
	mig := filepath.Join(root, "migrations", "001_init.sql")
	dbPath := filepath.Join(t.TempDir(), "cal.sqlite")
	db, err := sqlite.OpenAndMigrate(dbPath, mig)
	if err != nil {
		t.Fatal(err)
	}
	return sqlite.NewRepository(db), func() { _ = db.Close() }
}

func TestCalendarFeedTokenHash_isDeterministicSHA256Hex(t *testing.T) {
	plain := "deadbeef01"
	h := trips.CalendarFeedTokenHash(plain)
	if h != trips.CalendarFeedTokenHash(plain) {
		t.Fatal("same input must yield same hash")
	}
	if len(h) != 64 {
		t.Fatalf("sha256 hex length: got %d", len(h))
	}
	for _, c := range h {
		if c >= '0' && c <= '9' || c >= 'a' && c <= 'f' {
			continue
		}
		t.Fatalf("non-lowercase hex in output: %q", h)
	}
}

func TestRegenerateCalendarFeedToken_andMatches(t *testing.T) {
	repo, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	svc := trips.NewService(repo)

	uid, err := repo.CreateUser(ctx, trips.User{
		Email: "u@example.com", Username: "u", DisplayName: "U", PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{
		Name: "Paris", OwnerUserID: uid, StartDate: "2026-06-01", EndDate: "2026-06-07",
	})
	if err != nil {
		t.Fatal(err)
	}

	plain, err := svc.RegenerateCalendarFeedToken(ctx, tripID, uid)
	if err != nil || plain == "" {
		t.Fatalf("regenerate: %v plain=%q", err, plain)
	}
	if !svc.CalendarFeedMatches(ctx, tripID, plain) {
		t.Fatal("expected token to match after regenerate")
	}
	if svc.CalendarFeedMatches(ctx, tripID, plain+"x") {
		t.Fatal("wrong token must not match")
	}

	ok, err := svc.HasCalendarFeedToken(ctx, tripID)
	if err != nil || !ok {
		t.Fatalf("HasCalendarFeedToken: ok=%v err=%v", ok, err)
	}
}

func TestBuildTripICSBytes_publishCalendarParsesWithItineraryEvent(t *testing.T) {
	repo, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	svc := trips.NewService(repo)

	uid, err := repo.CreateUser(ctx, trips.User{
		Email: "o@example.com", Username: "o", DisplayName: "O", PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{
		Name: "Tokyo Trip", OwnerUserID: uid, StartDate: "2026-07-10", EndDate: "2026-07-12",
	})
	if err != nil {
		t.Fatal(err)
	}
	itemID := "it-1"
	err = repo.AddItineraryItem(ctx, trips.ItineraryItem{
		ID: itemID, TripID: tripID, DayNumber: 1, Title: "Museum visit",
		Location: "Ueno", StartTime: "10:30", EndTime: "12:00",
	})
	if err != nil {
		t.Fatal(err)
	}

	raw, err := svc.BuildTripICSBytes(ctx, tripID)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "METHOD:PUBLISH") || !strings.Contains(s, "BEGIN:VCALENDAR") {
		t.Fatalf("expected PUBLISH calendar, got snippet: %q", truncate(s, 200))
	}
	if !strings.Contains(s, "Tokyo Trip") {
		t.Fatalf("expected trip name in calendar: %q", truncate(s, 300))
	}

	cal, err := ics.ParseCalendar(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	events := cal.Events()
	if len(events) != 1 {
		t.Fatalf("want 1 VEVENT, got %d", len(events))
	}
	sum := events[0].GetProperty(ics.ComponentPropertySummary)
	if sum == nil || !strings.Contains(sum.Value, "Museum visit") {
		t.Fatalf("summary missing: %+v", sum)
	}
	st, err := events[0].GetStartAt()
	if err != nil {
		t.Fatal(err)
	}
	st = st.In(time.Local)
	if st.Year() != 2026 || st.Month() != time.July || st.Day() != 10 {
		t.Fatalf("start date: %v", st)
	}
	if st.Hour() != 10 || st.Minute() != 30 {
		t.Fatalf("start time: %v", st)
	}
}

func TestBuildTripICSBytes_emitsTZIDWhenREMI_ICS_TIMEZONE(t *testing.T) {
	t.Setenv("REMI_ICS_TIMEZONE", "Europe/Berlin")
	repo, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	svc := trips.NewService(repo)

	uid, err := repo.CreateUser(ctx, trips.User{
		Email: "tz@example.com", Username: "tz", DisplayName: "TZ", PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{
		Name: "Berlin labels", OwnerUserID: uid, StartDate: "2026-07-10", EndDate: "2026-07-12",
	})
	if err != nil {
		t.Fatal(err)
	}
	err = repo.AddItineraryItem(ctx, trips.ItineraryItem{
		ID: "it-tz", TripID: tripID, DayNumber: 1, Title: "Museum",
		Location: "Mitte", StartTime: "10:30", EndTime: "12:00",
	})
	if err != nil {
		t.Fatal(err)
	}

	raw, err := svc.BuildTripICSBytes(ctx, tripID)
	if err != nil {
		t.Fatal(err)
	}
	s := string(raw)
	if !strings.Contains(s, "TZID=Europe/Berlin") {
		t.Fatalf("expected TZID=Europe/Berlin in feed, snippet: %q", truncate(s, 400))
	}
	if !strings.Contains(s, "X-WR-TIMEZONE:Europe/Berlin") {
		t.Fatalf("expected X-WR-TIMEZONE for Google Calendar, snippet: %q", truncate(s, 400))
	}
	if strings.Contains(s, "DTSTART:20260710T103000Z") {
		t.Fatal("did not expect plain UTC Z DTSTART when REMI_ICS_TIMEZONE is set")
	}
	cal, err := ics.ParseCalendar(bytes.NewReader(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(cal.Events()) != 1 {
		t.Fatalf("events: %d", len(cal.Events()))
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
