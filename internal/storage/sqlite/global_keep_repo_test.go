package sqlite_test

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"remi-trip-planner/internal/storage/sqlite"
	"remi-trip-planner/internal/trips"
)

func TestGlobalKeepNotesAndImport(t *testing.T) {
	_, fname, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(fname), "..", "..", ".."))
	mig := filepath.Join(root, "migrations", "001_init.sql")

	db, err := sqlite.OpenAndMigrate(filepath.Join(t.TempDir(), "gk.sqlite"), mig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := sqlite.NewRepository(db)
	ctx := context.Background()

	uid, err := repo.CreateUser(ctx, trips.User{
		Email: "gk@example.com", Username: "gk", DisplayName: "GK", PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{Name: "T", OwnerUserID: uid})
	if err != nil {
		t.Fatal(err)
	}

	if err := repo.AddGlobalKeepNote(ctx, trips.GlobalKeepNote{UserID: uid, Title: "Lib", Body: "x", Color: "mint"}); err != nil {
		t.Fatal(err)
	}
	notes, err := repo.ListGlobalKeepNotesByUser(ctx, uid)
	if err != nil || len(notes) != 1 || notes[0].Title != "Lib" {
		t.Fatalf("notes %+v err %v", notes, err)
	}
	gid := notes[0].ID

	ok, err := repo.IsGlobalKeepImported(ctx, tripID, trips.GlobalKeepImportNote, gid)
	if err != nil || ok {
		t.Fatalf("imported before %v %v", ok, err)
	}
	if err := repo.RecordGlobalKeepImport(ctx, tripID, trips.GlobalKeepImportNote, gid); err != nil {
		t.Fatal(err)
	}
	ok, err = repo.IsGlobalKeepImported(ctx, tripID, trips.GlobalKeepImportNote, gid)
	if err != nil || !ok {
		t.Fatalf("imported after %v %v", ok, err)
	}
	ids, err := repo.ListGlobalKeepImportedIDs(ctx, tripID, trips.GlobalKeepImportNote)
	if err != nil || len(ids) != 1 || ids[0] != gid {
		t.Fatalf("ids %+v err %v", ids, err)
	}

	if err := repo.DeleteGlobalKeepNote(ctx, uid, gid); err != nil {
		t.Fatal(err)
	}
	notes, err = repo.ListGlobalKeepNotesByUser(ctx, uid)
	if err != nil || len(notes) != 0 {
		t.Fatalf("after delete %+v err %v", notes, err)
	}
}

func TestGlobalChecklistTemplateAndImport(t *testing.T) {
	_, fname, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(fname), "..", "..", ".."))
	mig := filepath.Join(root, "migrations", "001_init.sql")

	db, err := sqlite.OpenAndMigrate(filepath.Join(t.TempDir(), "gkc.sqlite"), mig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := sqlite.NewRepository(db)
	ctx := context.Background()

	uid, err := repo.CreateUser(ctx, trips.User{
		Email: "gkc@example.com", Username: "gkc", DisplayName: "GKC", PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{Name: "T2", OwnerUserID: uid})
	if err != nil {
		t.Fatal(err)
	}

	if err := repo.AddGlobalChecklistTemplate(ctx, trips.GlobalChecklistTemplate{UserID: uid, Category: "Pack"}, []string{"a", "b"}); err != nil {
		t.Fatal(err)
	}
	tpls, err := repo.ListGlobalChecklistTemplatesByUser(ctx, uid)
	if err != nil || len(tpls) != 1 || len(tpls[0].Lines) != 2 {
		t.Fatalf("tpls %+v err %v", tpls, err)
	}
	tplID := tpls[0].ID

	if err := repo.RecordGlobalKeepImport(ctx, tripID, trips.GlobalKeepImportChecklist, tplID); err != nil {
		t.Fatal(err)
	}
	ok, err := repo.IsGlobalKeepImported(ctx, tripID, trips.GlobalKeepImportChecklist, tplID)
	if err != nil || !ok {
		t.Fatalf("chk import %v %v", ok, err)
	}

	if err := repo.DeleteGlobalChecklistTemplate(ctx, uid, tplID); err != nil {
		t.Fatal(err)
	}
	tpls, err = repo.ListGlobalChecklistTemplatesByUser(ctx, uid)
	if err != nil || len(tpls) != 0 {
		t.Fatalf("after del %+v err %v", tpls, err)
	}
}

func TestGlobalKeepNoteArchiveTrashViews(t *testing.T) {
	_, fname, _, _ := runtime.Caller(0)
	root := filepath.Clean(filepath.Join(filepath.Dir(fname), "..", "..", ".."))
	mig := filepath.Join(root, "migrations", "001_init.sql")

	db, err := sqlite.OpenAndMigrate(filepath.Join(t.TempDir(), "gk-views.sqlite"), mig)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	repo := sqlite.NewRepository(db)
	ctx := context.Background()

	uid, err := repo.CreateUser(ctx, trips.User{
		Email: "gkv@example.com", Username: "gkv", DisplayName: "GKV", PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.AddGlobalKeepNote(ctx, trips.GlobalKeepNote{UserID: uid, Title: "A", Body: ""}); err != nil {
		t.Fatal(err)
	}
	notes, err := repo.ListGlobalKeepNotesByUser(ctx, uid)
	if err != nil || len(notes) != 1 {
		t.Fatalf("notes %+v err %v", notes, err)
	}
	n := notes[0]
	n.Archived, n.Trashed = true, false
	if err := repo.UpdateGlobalKeepNote(ctx, n); err != nil {
		t.Fatal(err)
	}
	active, err := repo.ListGlobalKeepNotesByUser(ctx, uid)
	if err != nil || len(active) != 0 {
		t.Fatalf("active after archive %+v err %v", active, err)
	}
	arch, err := repo.ListGlobalKeepNotesForKeepView(ctx, uid, trips.KeepViewArchive)
	if err != nil || len(arch) != 1 {
		t.Fatalf("archive view %+v err %v", arch, err)
	}
	n = arch[0]
	n.Archived, n.Trashed = false, true
	if err := repo.UpdateGlobalKeepNote(ctx, n); err != nil {
		t.Fatal(err)
	}
	arch, err = repo.ListGlobalKeepNotesForKeepView(ctx, uid, trips.KeepViewArchive)
	if err != nil || len(arch) != 0 {
		t.Fatalf("archive empty %+v err %v", arch, err)
	}
	tr, err := repo.ListGlobalKeepNotesForKeepView(ctx, uid, trips.KeepViewTrash)
	if err != nil || len(tr) != 1 {
		t.Fatalf("trash %+v err %v", tr, err)
	}
}
