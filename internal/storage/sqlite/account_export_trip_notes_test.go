package sqlite_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"remi-trip-planner/internal/storage/sqlite"
	"remi-trip-planner/internal/trips"
)

func TestBuildAccountExportIncludesTripNotes(t *testing.T) {
	_, fname, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(fname), "..", "..", ".."))
	mig := filepath.Join(root, "migrations", "001_init.sql")

	db, err := sqlite.OpenAndMigrate(filepath.Join(t.TempDir(), "export.sqlite"), mig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := sqlite.NewRepository(db)
	svc := trips.NewService(repo)
	ctx := context.Background()

	userID, err := repo.CreateUser(ctx, trips.User{
		Email: "e@example.com", Username: "ex", DisplayName: "E", PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	app, err := repo.GetAppSettings(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.SeedUserSettingsFromAppDefaults(ctx, userID, app); err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{Name: "Trip", OwnerUserID: userID})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.AddTripNote(ctx, trips.TripNote{
		ID: "note-1", TripID: tripID, Title: "Backup", Body: "text", Color: "mist",
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.AddTripNote(ctx, trips.TripNote{
		ID: "note-2", TripID: tripID, Title: "Trashed", Trashed: true,
	}); err != nil {
		t.Fatal(err)
	}

	exp, err := svc.BuildAccountExport(ctx, userID)
	if err != nil {
		t.Fatal(err)
	}
	if exp.ExportVersion == "" {
		t.Fatal("missing export version")
	}
	if len(exp.Trips) != 1 {
		t.Fatalf("trips len %d", len(exp.Trips))
	}
	pack := exp.Trips[0]
	if len(pack.TripNotes) != 2 {
		t.Fatalf("trip_notes len %d, want 2", len(pack.TripNotes))
	}
	seen := map[string]bool{}
	for _, n := range pack.TripNotes {
		seen[n.ID] = true
		if n.TripID != tripID {
			t.Fatalf("wrong trip_id on note %s", n.ID)
		}
	}
	if !seen["note-1"] || !seen["note-2"] {
		t.Fatalf("missing note ids: %+v", seen)
	}
}
