package trips_test

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"remi-trip-planner/internal/storage/sqlite"
	"remi-trip-planner/internal/trips"
)

func TestAddItineraryCommute_DuplicateRejected(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := sqlite.OpenAndMigrate(dbPath, filepath.Join("..", "..", "migrations", "001_init.sql"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repo := sqlite.NewRepository(db)
	svc := trips.NewService(repo)

	tripID, err := repo.CreateTrip(ctx, trips.Trip{Name: "T", StartDate: "2026-06-01", EndDate: "2026-06-03", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	a := trips.ItineraryItem{TripID: tripID, DayNumber: 1, Title: "A", StartTime: "09:00", EndTime: "10:00"}
	b := trips.ItineraryItem{TripID: tripID, DayNumber: 1, Title: "B", StartTime: "11:00", EndTime: "12:00"}
	if err := svc.AddItineraryItem(ctx, a); err != nil {
		t.Fatal(err)
	}
	if err := svc.AddItineraryItem(ctx, b); err != nil {
		t.Fatal(err)
	}
	items, _ := repo.ListItineraryItems(ctx, tripID)
	var idA, idB string
	for _, it := range items {
		if it.Title == "A" {
			idA = it.ID
		}
		if it.Title == "B" {
			idB = it.ID
		}
	}
	if idA == "" || idB == "" {
		t.Fatal("missing item ids")
	}
	err = svc.AddItineraryCommute(ctx, idA, idB, trips.ItineraryItem{TripID: tripID, Title: "Leg", TransportMode: "transit"})
	if err != nil {
		t.Fatal(err)
	}
	err = svc.AddItineraryCommute(ctx, idA, idB, trips.ItineraryItem{TripID: tripID, Title: "Dup"})
	if err == nil {
		t.Fatal("expected duplicate error")
	}
}

func TestDeleteItineraryItem_ClearsCommuteLinks(t *testing.T) {
	ctx := context.Background()
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "test.db")
	db, err := sqlite.OpenAndMigrate(dbPath, filepath.Join("..", "..", "migrations", "001_init.sql"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	repo := sqlite.NewRepository(db)
	svc := trips.NewService(repo)

	tripID, err := repo.CreateTrip(ctx, trips.Trip{Name: "T", StartDate: "2026-06-01", EndDate: "2026-06-03", CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC()})
	if err != nil {
		t.Fatal(err)
	}
	a := trips.ItineraryItem{TripID: tripID, DayNumber: 1, Title: "A", StartTime: "09:00", EndTime: "10:00"}
	b := trips.ItineraryItem{TripID: tripID, DayNumber: 1, Title: "B", StartTime: "11:00", EndTime: "12:00"}
	_ = svc.AddItineraryItem(ctx, a)
	_ = svc.AddItineraryItem(ctx, b)
	items, _ := repo.ListItineraryItems(ctx, tripID)
	var idA, idB string
	for _, it := range items {
		if it.Title == "A" {
			idA = it.ID
		}
		if it.Title == "B" {
			idB = it.ID
		}
	}
	_ = svc.AddItineraryCommute(ctx, idA, idB, trips.ItineraryItem{TripID: tripID, Title: "Leg"})
	items, _ = repo.ListItineraryItems(ctx, tripID)
	var commuteID string
	for _, it := range items {
		if trips.NormalizeItineraryItemKind(it.ItemKind) == trips.ItineraryItemKindCommute {
			commuteID = it.ID
			break
		}
	}
	if commuteID == "" {
		t.Fatal("no commute row")
	}
	if err := svc.DeleteItineraryItem(ctx, tripID, idA); err != nil {
		t.Fatal(err)
	}
	items, _ = repo.ListItineraryItems(ctx, tripID)
	for _, it := range items {
		if it.ID == commuteID {
			if it.CommuteFromItemID != "" {
				t.Fatalf("expected from cleared, got %q", it.CommuteFromItemID)
			}
		}
	}
}
