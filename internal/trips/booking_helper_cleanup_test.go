package trips_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"remi-trip-planner/internal/trips"
)

type failingItineraryDeleteRepo struct {
	trips.Repository
	targetID string
}

func (r failingItineraryDeleteRepo) DeleteItineraryItem(ctx context.Context, tripID, itemID string) error {
	if itemID == r.targetID {
		return errors.New("injected itinerary delete failure")
	}
	return r.Repository.DeleteItineraryItem(ctx, tripID, itemID)
}

func (r failingItineraryDeleteRepo) RunInTx(ctx context.Context, fn func(trips.Repository) error) error {
	txRepo, ok := r.Repository.(txCapableRepo)
	if !ok {
		return errors.New("repository does not support transactions")
	}
	return txRepo.RunInTx(ctx, func(repo trips.Repository) error {
		return fn(failingItineraryDeleteRepo{Repository: repo, targetID: r.targetID})
	})
}

func TestUpdateLodgingWithItineraryRollsBackOnDuplicateCleanupFailure(t *testing.T) {
	repo, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	svc := trips.NewService(repo)

	ownerID, err := repo.CreateUser(ctx, trips.User{
		Email:        "owner-cleanup@example.com",
		Username:     "owner_cleanup_atomic",
		DisplayName:  "Owner",
		PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{
		Name:        "Cleanup Atomic",
		OwnerUserID: ownerID,
		StartDate:   "2026-06-01",
		EndDate:     "2026-06-07",
	})
	if err != nil {
		t.Fatal(err)
	}

	lodgingID := "lodging-cleanup-1"
	checkInID := "lodging-checkin-1"
	checkOutID := "lodging-checkout-1"
	if err := svc.AddLodgingWithItinerary(ctx,
		trips.Lodging{
			ID:                  lodgingID,
			TripID:              tripID,
			Name:                "Aman Tokyo",
			Address:             "Tokyo",
			CheckInAt:           "2026-06-02T15:00",
			CheckOutAt:          "2026-06-04T11:00",
			BookingConfirmation: "ABC123",
			CostCents:           45000,
			Notes:               "Original notes",
			CheckInItineraryID:  checkInID,
			CheckOutItineraryID: checkOutID,
		},
		trips.ItineraryItem{
			ID:           checkInID,
			TripID:       tripID,
			DayNumber:    2,
			Title:        trips.AccommodationItineraryCheckInTitle("Aman Tokyo"),
			Location:     "Tokyo",
			EstCostCents: 45000,
			StartTime:    "15:00",
			EndTime:      "15:00",
		},
		trips.ItineraryItem{
			ID:           checkOutID,
			TripID:       tripID,
			DayNumber:    4,
			Title:        trips.AccommodationItineraryCheckOutTitle("Aman Tokyo"),
			Location:     "Tokyo",
			EstCostCents: 45000,
			StartTime:    "11:00",
			EndTime:      "11:00",
		},
	); err != nil {
		t.Fatal(err)
	}

	strayID := "lodging-stray-1"
	if err := repo.AddItineraryItem(ctx, trips.ItineraryItem{
		ID:           strayID,
		TripID:       tripID,
		DayNumber:    2,
		Title:        trips.AccommodationItineraryCheckInTitle("Aman Tokyo"),
		Location:     "Tokyo old duplicate",
		EstCostCents: 1,
		StartTime:    "14:00",
		EndTime:      "14:00",
	}); err != nil {
		t.Fatal(err)
	}

	beforeLodging, err := repo.GetLodging(ctx, tripID, lodgingID)
	if err != nil {
		t.Fatal(err)
	}
	beforeItems, err := repo.ListItineraryItems(ctx, tripID)
	if err != nil {
		t.Fatal(err)
	}

	failingSvc := trips.NewService(failingItineraryDeleteRepo{Repository: repo, targetID: strayID})
	err = failingSvc.UpdateLodgingWithItinerary(ctx,
		trips.Lodging{
			ID:                    lodgingID,
			TripID:                tripID,
			Name:                  "Aman Tokyo",
			Address:               "Tokyo Updated",
			Latitude:              35.0,
			Longitude:             139.0,
			CheckInAt:             "2026-06-02T16:00",
			CheckOutAt:            "2026-06-04T10:00",
			BookingConfirmation:   "XYZ999",
			CostCents:             47000,
			Notes:                 "Updated notes",
			CheckInItineraryID:    checkInID,
			CheckOutItineraryID:   checkOutID,
			ExpectedUpdatedAt:     beforeLodging.UpdatedAt,
			EnforceOptimisticLock: true,
		},
		"Aman Tokyo",
		2, "16:00",
		4, "10:00",
		"new check-in notes",
		"new check-out notes",
	)
	if err == nil {
		t.Fatal("expected injected itinerary delete failure")
	}
	if !strings.Contains(err.Error(), "injected itinerary delete failure") {
		t.Fatalf("unexpected error: %v", err)
	}

	afterLodging, err := repo.GetLodging(ctx, tripID, lodgingID)
	if err != nil {
		t.Fatal(err)
	}
	if afterLodging.Address != beforeLodging.Address || afterLodging.BookingConfirmation != beforeLodging.BookingConfirmation || afterLodging.CostCents != beforeLodging.CostCents || afterLodging.Notes != beforeLodging.Notes {
		t.Fatalf("lodging changed despite rollback: before=%+v after=%+v", beforeLodging, afterLodging)
	}

	afterItems, err := repo.ListItineraryItems(ctx, tripID)
	if err != nil {
		t.Fatal(err)
	}
	if len(afterItems) != len(beforeItems) {
		t.Fatalf("itinerary count changed: got %d want %d", len(afterItems), len(beforeItems))
	}
	got := map[string]trips.ItineraryItem{}
	for _, item := range afterItems {
		got[item.ID] = item
	}
	for _, item := range beforeItems {
		gotItem, ok := got[item.ID]
		if !ok {
			t.Fatalf("missing itinerary item after rollback: %s", item.ID)
		}
		if gotItem.Title != item.Title || gotItem.Location != item.Location || gotItem.DayNumber != item.DayNumber || gotItem.EstCostCents != item.EstCostCents || gotItem.StartTime != item.StartTime || gotItem.EndTime != item.EndTime {
			t.Fatalf("itinerary item changed despite rollback: before=%+v after=%+v", item, gotItem)
		}
	}
}
