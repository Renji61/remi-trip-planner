package trips_test

import (
	"context"
	"errors"
	"strings"
	"testing"

	"remi-trip-planner/internal/trips"
)

type txCapableRepo interface {
	RunInTx(context.Context, func(trips.Repository) error) error
}

type failingFlightDeleteRepo struct {
	trips.Repository
}

func (r failingFlightDeleteRepo) DeleteFlight(ctx context.Context, tripID, flightID string) error {
	return errors.New("injected flight delete failure")
}

func (r failingFlightDeleteRepo) RunInTx(ctx context.Context, fn func(trips.Repository) error) error {
	txRepo, ok := r.Repository.(txCapableRepo)
	if !ok {
		return errors.New("repository does not support transactions")
	}
	return txRepo.RunInTx(ctx, func(repo trips.Repository) error {
		return fn(failingFlightDeleteRepo{Repository: repo})
	})
}

func TestDeleteFlightRollsBackLinkedDeletesOnFailure(t *testing.T) {
	repo, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	svc := trips.NewService(repo)

	ownerID, err := repo.CreateUser(ctx, trips.User{
		Email:        "owner-delete@example.com",
		Username:     "owner_delete_atomic",
		DisplayName:  "Owner",
		PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{
		Name:        "Atomic Delete",
		OwnerUserID: ownerID,
		StartDate:   "2026-06-01",
		EndDate:     "2026-06-07",
	})
	if err != nil {
		t.Fatal(err)
	}

	flightID := "flight-delete-1"
	departID := "depart-delete-1"
	arriveID := "arrive-delete-1"
	if err := svc.AddFlightWithItinerary(ctx,
		trips.Flight{
			ID:                flightID,
			TripID:            tripID,
			FlightName:        "Rollback Air",
			FlightNumber:      "RB 101",
			DepartAirport:     "MNL",
			ArriveAirport:     "CEB",
			DepartAt:          "2026-06-02T08:00",
			ArriveAt:          "2026-06-02T09:30",
			CostCents:         54321,
			DepartItineraryID: departID,
			ArriveItineraryID: arriveID,
		},
		trips.ItineraryItem{
			ID:           departID,
			TripID:       tripID,
			DayNumber:    2,
			Title:        trips.FlightItineraryDepartTitle("Rollback Air (RB 101)"),
			Location:     "MNL",
			EstCostCents: 54321,
			StartTime:    "08:00",
			EndTime:      "08:00",
		},
		trips.ItineraryItem{
			ID:           arriveID,
			TripID:       tripID,
			DayNumber:    2,
			Title:        trips.FlightItineraryArriveTitle("Rollback Air (RB 101)"),
			Location:     "CEB",
			EstCostCents: 54321,
			StartTime:    "09:30",
			EndTime:      "09:30",
		},
	); err != nil {
		t.Fatal(err)
	}

	flightBefore, err := repo.GetFlight(ctx, tripID, flightID)
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(flightBefore.ExpenseID) == "" {
		t.Fatal("expected linked expense id")
	}

	failingSvc := trips.NewService(failingFlightDeleteRepo{Repository: repo})
	err = failingSvc.DeleteFlight(ctx, tripID, flightID)
	if err == nil {
		t.Fatal("expected injected delete failure")
	}
	if !strings.Contains(err.Error(), "injected flight delete failure") {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, err := repo.GetFlight(ctx, tripID, flightID); err != nil {
		t.Fatalf("flight should still exist after rollback: %v", err)
	}
	if _, err := repo.GetExpense(ctx, tripID, flightBefore.ExpenseID); err != nil {
		t.Fatalf("linked expense should still exist after rollback: %v", err)
	}
	items, err := repo.ListItineraryItems(ctx, tripID)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 {
		t.Fatalf("expected itinerary rows to roll back, got %d", len(items))
	}
	gotIDs := map[string]struct{}{}
	for _, item := range items {
		gotIDs[item.ID] = struct{}{}
	}
	if _, ok := gotIDs[departID]; !ok {
		t.Fatal("departure itinerary row missing after rollback")
	}
	if _, ok := gotIDs[arriveID]; !ok {
		t.Fatal("arrival itinerary row missing after rollback")
	}
}
