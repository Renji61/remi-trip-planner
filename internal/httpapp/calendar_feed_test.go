package httpapp

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	ics "github.com/arran4/golang-ical"

	"remi-trip-planner/internal/storage/sqlite"
	"remi-trip-planner/internal/trips"
)

// TestCalendarFeedICS_HTTP validates the public feed endpoint used by Google Calendar and Apple Calendar
// "subscribe by URL" flows: correct secret returns a valid text/calendar body; wrong or missing secret is forbidden.
func TestCalendarFeedICS_HTTP(t *testing.T) {
	root := findModuleRoot(t)
	prevWd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(root); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(prevWd) })

	_, fname, _, _ := runtime.Caller(0)
	modRoot := filepath.Clean(filepath.Join(filepath.Dir(fname), "..", ".."))
	mig := filepath.Join(modRoot, "migrations", "001_init.sql")
	dbPath := filepath.Join(t.TempDir(), "cal_http.sqlite")
	db, err := sqlite.OpenAndMigrate(dbPath, mig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := sqlite.NewRepository(db)
	svc := trips.NewService(repo)
	ctx := context.Background()

	uid, err := repo.CreateUser(ctx, trips.User{
		Email: "cal@example.com", Username: "cal", DisplayName: "Cal", PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{
		Name: "Feed Test", OwnerUserID: uid, StartDate: "2026-08-01", EndDate: "2026-08-03",
	})
	if err != nil {
		t.Fatal(err)
	}
	plain, err := svc.RegenerateCalendarFeedToken(ctx, tripID, uid)
	if err != nil || plain == "" {
		t.Fatalf("token: %v", err)
	}

	h := NewRouter(Dependencies{TripService: svc, DB: db})
	feedPath := "/calendar/feed/" + tripID + ".ics"

	t.Run("missing key", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, feedPath, nil)
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status %d", rec.Code)
		}
	})

	t.Run("wrong key", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, feedPath+"?k=nope", nil)
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("status %d", rec.Code)
		}
	})

	t.Run("ok parses as calendar with cache headers", func(t *testing.T) {
		rec := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodGet, feedPath+"?k="+plain, nil)
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d body=%q", rec.Code, rec.Body.String())
		}
		ct := rec.Header().Get("Content-Type")
		if !strings.HasPrefix(ct, "text/calendar") {
			t.Fatalf("Content-Type: %q", ct)
		}
		cc := rec.Header().Get("Cache-Control")
		if !strings.Contains(cc, "max-age=300") {
			t.Fatalf("Cache-Control: %q", cc)
		}
		body, err := io.ReadAll(rec.Body)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := ics.ParseCalendar(bytes.NewReader(body)); err != nil {
			t.Fatalf("calendar body must parse like Google/Apple clients expect: %v", err)
		}
		raw := string(body)
		if !strings.Contains(raw, "METHOD:PUBLISH") {
			t.Fatal("expected METHOD:PUBLISH for subscribed calendars")
		}
	})
}
