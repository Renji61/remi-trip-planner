package sqlite_test

import (
	"context"
	"encoding/json"
	"path/filepath"
	"runtime"
	"testing"

	"remi-trip-planner/internal/storage/sqlite"
	"remi-trip-planner/internal/trips"
)

func TestApplyTripSyncOpsTripRename(t *testing.T) {
	_, fname, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(fname), "..", "..", ".."))
	mig := filepath.Join(root, "migrations", "001_init.sql")

	dbPath := filepath.Join(t.TempDir(), "sync.sqlite")
	db, err := sqlite.OpenAndMigrate(dbPath, mig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := sqlite.NewRepository(db)
	svc := trips.NewService(repo)
	ctx := context.Background()

	ownerID, err := repo.CreateUser(ctx, trips.User{
		Email: "o@example.com", Username: "owner1", DisplayName: "O",
		PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{Name: "Before", OwnerUserID: ownerID})
	if err != nil {
		t.Fatal(err)
	}

	payload, _ := json.Marshal(map[string]string{"name": "After"})
	req := trips.SyncApplyRequest{
		ClientID: "c1",
		Ops: []trips.SyncOpInput{
			{Entity: "trip", Operation: "update", Payload: payload},
		},
	}
	res, err := svc.ApplyTripSyncOps(ctx, tripID, trips.TripAccess{TripID: tripID, IsOwner: true, CanManage: true}, req)
	if err != nil {
		t.Fatal(err)
	}
	if res.AppliedCount != 1 || res.Status != "accepted" {
		t.Fatalf("res %+v", res)
	}
	got, err := repo.GetTrip(ctx, tripID)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "After" {
		t.Fatalf("name %q", got.Name)
	}
	if len(res.ServerChanges) < 1 {
		t.Fatal("expected change_log delta")
	}
}

func TestApplyTripSyncOpsDeleteRequiresOwner(t *testing.T) {
	_, fname, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(fname), "..", "..", ".."))
	mig := filepath.Join(root, "migrations", "001_init.sql")

	dbPath := filepath.Join(t.TempDir(), "sync2.sqlite")
	db, err := sqlite.OpenAndMigrate(dbPath, mig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := sqlite.NewRepository(db)
	svc := trips.NewService(repo)
	ctx := context.Background()

	ownerID, err := repo.CreateUser(ctx, trips.User{
		Email: "a@example.com", Username: "a", DisplayName: "A", PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{Name: "T", OwnerUserID: ownerID})
	if err != nil {
		t.Fatal(err)
	}

	req := trips.SyncApplyRequest{
		Ops: []trips.SyncOpInput{{Entity: "trip", Operation: "delete", Payload: json.RawMessage(`{}`)}},
	}
	res, err := svc.ApplyTripSyncOps(ctx, tripID, trips.TripAccess{TripID: tripID, IsOwner: false}, req)
	if err != nil {
		t.Fatal(err)
	}
	if res.AppliedCount != 0 || res.Status != "rejected" {
		t.Fatalf("res %+v", res)
	}
	if len(res.Results) != 1 || res.Results[0].OK {
		t.Fatalf("results %+v", res.Results)
	}
	if _, err := repo.GetTrip(ctx, tripID); err != nil {
		t.Fatal("trip should still exist")
	}
}

func TestApplyTripSyncOpsTripNoteAndChecklistFlags(t *testing.T) {
	_, fname, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(fname), "..", "..", ".."))
	mig := filepath.Join(root, "migrations", "001_init.sql")

	dbPath := filepath.Join(t.TempDir(), "sync_notes.sqlite")
	db, err := sqlite.OpenAndMigrate(dbPath, mig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := sqlite.NewRepository(db)
	svc := trips.NewService(repo)
	ctx := context.Background()

	ownerID, err := repo.CreateUser(ctx, trips.User{
		Email: "n@example.com", Username: "n1", DisplayName: "N", PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{Name: "N", OwnerUserID: ownerID})
	if err != nil {
		t.Fatal(err)
	}
	acc := trips.TripAccess{TripID: tripID, IsOwner: true, CanManage: true}

	noteID := "note-sync-1"
	createNote, _ := json.Marshal(map[string]any{"id": noteID, "title": "Hello", "body": "World", "pinned": true})
	res, err := svc.ApplyTripSyncOps(ctx, tripID, acc, trips.SyncApplyRequest{
		Ops: []trips.SyncOpInput{{Entity: "trip_note", Operation: "create", Payload: createNote}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.AppliedCount != 1 || res.Status != "accepted" {
		t.Fatalf("create note res %+v", res)
	}
	notes, err := repo.ListTripNotesForExport(ctx, tripID)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 1 || notes[0].Title != "Hello" || !notes[0].Pinned {
		t.Fatalf("notes %+v", notes)
	}

	updNote, _ := json.Marshal(map[string]string{"title": "Hi"})
	res2, err := svc.ApplyTripSyncOps(ctx, tripID, acc, trips.SyncApplyRequest{
		Ops: []trips.SyncOpInput{{Entity: "trip_note", Operation: "update", EntityID: noteID, Payload: updNote}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res2.AppliedCount != 1 {
		t.Fatalf("update note %+v", res2)
	}
	n2, err := repo.GetTripNote(ctx, noteID)
	if err != nil || n2.Title != "Hi" || n2.Body != "World" {
		t.Fatalf("note after update %+v err %v", n2, err)
	}

	res3, err := svc.ApplyTripSyncOps(ctx, tripID, acc, trips.SyncApplyRequest{
		Ops: []trips.SyncOpInput{{Entity: "trip_note", Operation: "delete", EntityID: noteID, Payload: json.RawMessage(`{}`)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res3.AppliedCount != 1 {
		t.Fatalf("delete note %+v", res3)
	}
	notes, err = repo.ListTripNotesForExport(ctx, tripID)
	if err != nil {
		t.Fatal(err)
	}
	if len(notes) != 0 {
		t.Fatalf("expected no notes, got %d", len(notes))
	}

	chPayload, _ := json.Marshal(map[string]any{"text": "archived line", "category": "Packing List", "archived": true})
	res4, err := svc.ApplyTripSyncOps(ctx, tripID, acc, trips.SyncApplyRequest{
		Ops: []trips.SyncOpInput{{Entity: "checklist_item", Operation: "create", Payload: chPayload}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res4.AppliedCount != 1 {
		t.Fatalf("checklist create %+v", res4)
	}
	items, err := repo.ListChecklistItems(ctx, tripID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 0 {
		t.Fatalf("archived item should not appear in active list, got %+v", items)
	}
	archived, err := repo.ListChecklistItemsForKeepView(ctx, tripID, trips.KeepViewArchive)
	if err != nil {
		t.Fatal(err)
	}
	if len(archived) != 1 || !archived[0].Archived || archived[0].Trashed {
		t.Fatalf("archive view %+v", archived)
	}
}

func TestApplyTripSyncOpsTripNoteUnknownOperation(t *testing.T) {
	_, fname, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(fname), "..", "..", ".."))
	mig := filepath.Join(root, "migrations", "001_init.sql")

	db, err := sqlite.OpenAndMigrate(filepath.Join(t.TempDir(), "sync_bad.sqlite"), mig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := sqlite.NewRepository(db)
	svc := trips.NewService(repo)
	ctx := context.Background()

	ownerID, err := repo.CreateUser(ctx, trips.User{
		Email: "bad@example.com", Username: "bad", DisplayName: "B", PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{Name: "T", OwnerUserID: ownerID})
	if err != nil {
		t.Fatal(err)
	}
	acc := trips.TripAccess{TripID: tripID, IsOwner: true, CanManage: true}

	res, err := svc.ApplyTripSyncOps(ctx, tripID, acc, trips.SyncApplyRequest{
		Ops: []trips.SyncOpInput{{Entity: "trip_note", Operation: "merge", Payload: json.RawMessage(`{}`)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.AppliedCount != 0 || res.Status != "rejected" {
		t.Fatalf("expected rejected op, got %+v", res)
	}
	if len(res.Results) != 1 || res.Results[0].OK || res.Results[0].Error == "" {
		t.Fatalf("results %+v", res.Results)
	}
}
