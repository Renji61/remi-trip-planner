package sqlite_test

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"remi-trip-planner/internal/storage/sqlite"
	"remi-trip-planner/internal/trips"
)

func TestTripNotesRepoExportAndKeepViews(t *testing.T) {
	_, fname, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(fname), "..", "..", ".."))
	mig := filepath.Join(root, "migrations", "001_init.sql")

	db, err := sqlite.OpenAndMigrate(filepath.Join(t.TempDir(), "notes.sqlite"), mig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := sqlite.NewRepository(db)
	ctx := context.Background()

	uid, err := repo.CreateUser(ctx, trips.User{
		Email: "tn@example.com", Username: "tn", DisplayName: "TN", PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{Name: "N", OwnerUserID: uid})
	if err != nil {
		t.Fatal(err)
	}

	active := trips.TripNote{
		ID: "n-active", TripID: tripID, Title: "A", Body: "b",
		Color: "sand", Pinned: true,
	}
	arch := trips.TripNote{
		ID: "n-arch", TripID: tripID, Title: "Arch", Archived: true,
	}
	tr := trips.TripNote{
		ID: "n-trash", TripID: tripID, Title: "T", Trashed: true,
	}
	for _, n := range []trips.TripNote{active, arch, tr} {
		if err := repo.AddTripNote(ctx, n); err != nil {
			t.Fatal(err)
		}
	}

	all, err := repo.ListTripNotesForExport(ctx, tripID)
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("export list len %d", len(all))
	}

	main, err := repo.ListTripNotesForKeepView(ctx, tripID, trips.KeepViewNotes)
	if err != nil {
		t.Fatal(err)
	}
	if len(main) != 1 || main[0].ID != "n-active" {
		t.Fatalf("notes view %+v", main)
	}
	arL, err := repo.ListTripNotesForKeepView(ctx, tripID, trips.KeepViewArchive)
	if err != nil || len(arL) != 1 || arL[0].ID != "n-arch" {
		t.Fatalf("archive %+v err %v", arL, err)
	}
	trL, err := repo.ListTripNotesForKeepView(ctx, tripID, trips.KeepViewTrash)
	if err != nil || len(trL) != 1 || trL[0].ID != "n-trash" {
		t.Fatalf("trash %+v err %v", trL, err)
	}

	active.Title = "Updated"
	if err := repo.UpdateTripNote(ctx, active); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetTripNote(ctx, "n-active")
	if err != nil || got.Title != "Updated" {
		t.Fatalf("get after update %+v %v", got, err)
	}

	if err := repo.DeleteTripNote(ctx, "n-trash"); err != nil {
		t.Fatal(err)
	}
	all, _ = repo.ListTripNotesForExport(ctx, tripID)
	if len(all) != 2 {
		t.Fatalf("after delete len %d", len(all))
	}
}

func TestChecklistActiveListExcludesArchived(t *testing.T) {
	_, fname, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(fname), "..", "..", ".."))
	mig := filepath.Join(root, "migrations", "001_init.sql")

	db, err := sqlite.OpenAndMigrate(filepath.Join(t.TempDir(), "cl.sqlite"), mig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := sqlite.NewRepository(db)
	ctx := context.Background()

	uid, err := repo.CreateUser(ctx, trips.User{
		Email: "cl@example.com", Username: "cl", DisplayName: "CL", PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{Name: "T", OwnerUserID: uid})
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	active := trips.ChecklistItem{ID: "c1", TripID: tripID, Category: "L", Text: "a", CreatedAt: now}
	arch := trips.ChecklistItem{ID: "c2", TripID: tripID, Category: "L", Text: "b", Archived: true, CreatedAt: now}
	for _, it := range []trips.ChecklistItem{active, arch} {
		if err := repo.AddChecklistItem(ctx, it); err != nil {
			t.Fatal(err)
		}
	}

	list, err := repo.ListChecklistItems(ctx, tripID)
	if err != nil || len(list) != 1 || list[0].ID != "c1" {
		t.Fatalf("active only %+v", list)
	}
	archL, err := repo.ListChecklistItemsForKeepView(ctx, tripID, trips.KeepViewArchive)
	if err != nil || len(archL) != 1 || archL[0].ID != "c2" {
		t.Fatalf("archive view %+v", archL)
	}
}

func TestTripNoteServiceDeleteSoftVersusHard(t *testing.T) {
	_, fname, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(fname), "..", "..", ".."))
	mig := filepath.Join(root, "migrations", "001_init.sql")

	db, err := sqlite.OpenAndMigrate(filepath.Join(t.TempDir(), "del.sqlite"), mig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := sqlite.NewRepository(db)
	svc := trips.NewService(repo)
	ctx := context.Background()

	uid, err := repo.CreateUser(ctx, trips.User{
		Email: "d@example.com", Username: "d", DisplayName: "D", PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{Name: "T", OwnerUserID: uid})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.AddTripNote(ctx, trips.TripNote{ID: "nd", TripID: tripID, Title: "x"}); err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteTripNote(ctx, tripID, "nd"); err == nil {
		t.Fatal("DeleteTripNote should require trashed")
	}
	if err := svc.DeleteTripNoteHard(ctx, tripID, "nd"); err != nil {
		t.Fatalf("DeleteTripNoteHard: %v", err)
	}
	_, err = repo.GetTripNote(ctx, "nd")
	if !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("want ErrNoRows, got %v", err)
	}
}

func TestSetChecklistCategoryPinnedWritesChangeLog(t *testing.T) {
	_, fname, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(fname), "..", "..", ".."))
	mig := filepath.Join(root, "migrations", "001_init.sql")

	db, err := sqlite.OpenAndMigrate(filepath.Join(t.TempDir(), "pin.sqlite"), mig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := sqlite.NewRepository(db)
	ctx := context.Background()

	uid, err := repo.CreateUser(ctx, trips.User{
		Email: "pin@example.com", Username: "pin", DisplayName: "P", PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{Name: "T", OwnerUserID: uid})
	if err != nil {
		t.Fatal(err)
	}
	cat := trips.NormalizeKeepChecklistCategory("Packing List")
	if err := repo.SetChecklistCategoryPinned(ctx, tripID, cat, true); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM change_log WHERE trip_id = ? AND entity = 'checklist_category_pin'`, tripID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 1 {
		t.Fatalf("change_log rows for pin: %d", n)
	}
	if err := repo.SetChecklistCategoryPinned(ctx, tripID, cat, false); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRowContext(ctx, `SELECT COUNT(*) FROM change_log WHERE trip_id = ? AND entity = 'checklist_category_pin'`, tripID).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("change_log rows after unpin: %d", n)
	}
}

func TestUpdateTripNotePersistsDueAt(t *testing.T) {
	_, fname, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(fname), "..", "..", ".."))
	mig := filepath.Join(root, "migrations", "001_init.sql")

	db, err := sqlite.OpenAndMigrate(filepath.Join(t.TempDir(), "due.sqlite"), mig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := sqlite.NewRepository(db)
	ctx := context.Background()

	uid, err := repo.CreateUser(ctx, trips.User{
		Email: "due@example.com", Username: "due", DisplayName: "D", PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{Name: "T", OwnerUserID: uid})
	if err != nil {
		t.Fatal(err)
	}
	n := trips.TripNote{ID: "n-due", TripID: tripID, Title: "T", Body: "B"}
	if err := repo.AddTripNote(ctx, n); err != nil {
		t.Fatal(err)
	}
	n.DueAt = "2026-06-15"
	if err := repo.UpdateTripNote(ctx, n); err != nil {
		t.Fatal(err)
	}
	got, err := repo.GetTripNote(ctx, "n-due")
	if err != nil {
		t.Fatal(err)
	}
	if got.DueAt != "2026-06-15" {
		t.Fatalf("due_at = %q", got.DueAt)
	}
}
