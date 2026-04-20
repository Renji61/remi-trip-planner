package trips_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"remi-trip-planner/internal/storage/sqlite"
	"remi-trip-planner/internal/trips"
)

func TestImportGlobalKeepIntoTrip(t *testing.T) {
	_, fname, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(fname), "..", ".."))
	mig := filepath.Join(root, "migrations", "001_init.sql")

	db, err := sqlite.OpenAndMigrate(filepath.Join(t.TempDir(), "imp.sqlite"), mig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := sqlite.NewRepository(db)
	svc := trips.NewService(repo)
	ctx := context.Background()

	uid, err := repo.CreateUser(ctx, trips.User{
		Email: "im@example.com", Username: "im", DisplayName: "IM", PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{Name: "Trip", OwnerUserID: uid, UIShowChecklist: true})
	if err != nil {
		t.Fatal(err)
	}

	if err := repo.AddGlobalKeepNote(ctx, trips.GlobalKeepNote{UserID: uid, Title: "G", Body: "b"}); err != nil {
		t.Fatal(err)
	}
	gNotes, _ := repo.ListGlobalKeepNotesByUser(ctx, uid)
	noteID := gNotes[0].ID

	if err := repo.AddGlobalChecklistTemplate(ctx, trips.GlobalChecklistTemplate{UserID: uid, Category: "C"}, []string{"one"}); err != nil {
		t.Fatal(err)
	}
	gCh, _ := repo.ListGlobalChecklistTemplatesByUser(ctx, uid)
	chID := gCh[0].ID

	n, err := svc.ImportGlobalKeepIntoTrip(ctx, uid, tripID, []string{noteID}, []string{chID})
	if err != nil || n != 2 {
		t.Fatalf("import n=%d err=%v", n, err)
	}
	tripNotes, err := repo.ListTripNotesForKeepView(ctx, tripID, trips.KeepViewNotes)
	if err != nil || len(tripNotes) != 1 || tripNotes[0].Title != "G" {
		t.Fatalf("trip notes %+v err %v", tripNotes, err)
	}
	items, err := repo.ListChecklistItemsForKeepView(ctx, tripID, trips.KeepViewNotes)
	if err != nil || len(items) != 1 || items[0].Text != "one" {
		t.Fatalf("items %+v err %v", items, err)
	}

	n2, err := svc.ImportGlobalKeepIntoTrip(ctx, uid, tripID, []string{noteID}, []string{chID})
	if err != nil || n2 != 0 {
		t.Fatalf("reimport n=%d err=%v", n2, err)
	}
}
