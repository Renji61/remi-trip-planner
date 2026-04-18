package trips_test

import (
	"context"
	"testing"
	"time"

	"remi-trip-planner/internal/trips"
)

func TestUpdateTrip_migratesDraftItineraryDayNumbersWhenSettingTripDates(t *testing.T) {
	repo, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	svc := trips.NewService(repo)

	ownerID, err := repo.CreateUser(ctx, trips.User{
		Email:        "m@example.com",
		Username:     "draft_migrate",
		DisplayName:  "M",
		PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{
		Name:        "Draft Trip",
		OwnerUserID: ownerID,
		StartDate:   "",
		EndDate:     "",
	})
	if err != nil {
		t.Fatal(err)
	}

	draftDay := trips.DraftItineraryDayNumberFromDate(time.Date(2026, 6, 15, 0, 0, 0, 0, time.Local))
	stopID := "stop-draft-1"
	if err := repo.AddItineraryItem(ctx, trips.ItineraryItem{
		ID:        stopID,
		TripID:    tripID,
		DayNumber: draftDay,
		Title:     "Museum",
	}); err != nil {
		t.Fatal(err)
	}
	if err := repo.SaveTripDayLabel(ctx, tripID, draftDay, "Kyoto day"); err != nil {
		t.Fatal(err)
	}

	tr, err := repo.GetTrip(ctx, tripID)
	if err != nil {
		t.Fatal(err)
	}
	tr.StartDate = "2026-06-01"
	tr.EndDate = "2026-06-30"
	if err := svc.UpdateTrip(ctx, tr); err != nil {
		t.Fatal(err)
	}

	items, err := repo.ListItineraryItems(ctx, tripID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("items: %d", len(items))
	}
	if items[0].DayNumber != 15 {
		t.Fatalf("day_number want 15 got %d", items[0].DayNumber)
	}

	labels, err := repo.GetTripDayLabels(ctx, tripID)
	if err != nil {
		t.Fatal(err)
	}
	if labels[draftDay] != "" {
		t.Fatalf("old draft label key should be gone, got %q", labels[draftDay])
	}
	if labels[15] != "Kyoto day" {
		t.Fatalf("label on day 15: got %#v", labels)
	}
}

func TestUpdateTrip_leavesRelativeDayNumbersWhenSettingTripDates(t *testing.T) {
	repo, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	svc := trips.NewService(repo)

	ownerID, err := repo.CreateUser(ctx, trips.User{
		Email:        "m2@example.com",
		Username:     "draft_migrate2",
		DisplayName:  "M",
		PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{
		Name:        "Draft Trip 2",
		OwnerUserID: ownerID,
		StartDate:   "",
		EndDate:     "",
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.AddItineraryItem(ctx, trips.ItineraryItem{
		ID:        "legacy-1",
		TripID:    tripID,
		DayNumber: 2,
		Title:     "Odd legacy row",
	}); err != nil {
		t.Fatal(err)
	}

	tr, err := repo.GetTrip(ctx, tripID)
	if err != nil {
		t.Fatal(err)
	}
	tr.StartDate = "2026-06-01"
	tr.EndDate = "2026-06-10"
	if err := svc.UpdateTrip(ctx, tr); err != nil {
		t.Fatal(err)
	}
	items, err := repo.ListItineraryItems(ctx, tripID)
	if err != nil {
		t.Fatal(err)
	}
	if items[0].DayNumber != 2 {
		t.Fatalf("want day 2 unchanged, got %d", items[0].DayNumber)
	}
}
