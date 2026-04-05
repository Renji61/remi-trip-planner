package trips_test

import (
	"context"
	"strings"
	"testing"

	"remi-trip-planner/internal/trips"
)

func TestUpdateFlightWithItineraryRollsBackOnFlightValidationFailure(t *testing.T) {
	repo, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	svc := trips.NewService(repo)

	ownerID, err := repo.CreateUser(ctx, trips.User{
		Email:        "owner@example.com",
		Username:     "owner_atomic",
		DisplayName:  "Owner",
		PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{
		Name:        "Atomic Flight",
		OwnerUserID: ownerID,
		StartDate:   "2026-06-01",
		EndDate:     "2026-06-07",
	})
	if err != nil {
		t.Fatal(err)
	}

	flightID := "flight-1"
	departID := "depart-1"
	arriveID := "arrive-1"
	if err := svc.AddFlightWithItinerary(ctx,
		trips.Flight{
			ID:                flightID,
			TripID:            tripID,
			FlightName:        "Initial Air",
			FlightNumber:      "IA 100",
			DepartAirport:     "MNL",
			ArriveAirport:     "CEB",
			DepartAt:          "2026-06-02T08:00",
			ArriveAt:          "2026-06-02T09:30",
			CostCents:         12345,
			DepartItineraryID: departID,
			ArriveItineraryID: arriveID,
		},
		trips.ItineraryItem{
			ID:           departID,
			TripID:       tripID,
			DayNumber:    2,
			Title:        trips.FlightItineraryDepartTitle("Initial Air (IA 100)"),
			Location:     "MNL",
			EstCostCents: 12345,
			StartTime:    "08:00",
			EndTime:      "08:00",
		},
		trips.ItineraryItem{
			ID:           arriveID,
			TripID:       tripID,
			DayNumber:    2,
			Title:        trips.FlightItineraryArriveTitle("Initial Air (IA 100)"),
			Location:     "CEB",
			EstCostCents: 12345,
			StartTime:    "09:30",
			EndTime:      "09:30",
		},
	); err != nil {
		t.Fatal(err)
	}

	originalFlight, err := repo.GetFlight(ctx, tripID, flightID)
	if err != nil {
		t.Fatal(err)
	}
	originalItems, err := repo.ListItineraryItems(ctx, tripID)
	if err != nil {
		t.Fatal(err)
	}

	err = svc.UpdateFlightWithItinerary(ctx,
		trips.Flight{
			ID:                    flightID,
			TripID:                tripID,
			FlightName:            "Changed Air",
			FlightNumber:          "CA 200",
			DepartAirport:         "SIN",
			ArriveAirport:         "HND",
			DepartAt:              "2026-06-02T11:00",
			ArriveAt:              "2026-06-02T16:00",
			CostCents:             99999,
			DepartItineraryID:     departID,
			ArriveItineraryID:     arriveID,
			ExpenseID:             originalFlight.ExpenseID,
			EnforceOptimisticLock: true,
		},
		trips.ItineraryItem{
			ID:           departID,
			TripID:       tripID,
			DayNumber:    3,
			Title:        trips.FlightItineraryDepartTitle("Changed Air (CA 200)"),
			Location:     "SIN",
			EstCostCents: 99999,
			StartTime:    "11:00",
			EndTime:      "11:00",
		},
		trips.ItineraryItem{
			ID:           arriveID,
			TripID:       tripID,
			DayNumber:    3,
			Title:        trips.FlightItineraryArriveTitle("Changed Air (CA 200)"),
			Location:     "HND",
			EstCostCents: 99999,
			StartTime:    "16:00",
			EndTime:      "16:00",
		},
	)
	if err == nil {
		t.Fatal("expected optimistic lock validation error")
	}
	if !strings.Contains(err.Error(), "older view") {
		t.Fatalf("unexpected error: %v", err)
	}

	gotFlight, err := repo.GetFlight(ctx, tripID, flightID)
	if err != nil {
		t.Fatal(err)
	}
	if gotFlight.FlightName != originalFlight.FlightName || gotFlight.DepartAirport != originalFlight.DepartAirport || gotFlight.ArriveAirport != originalFlight.ArriveAirport {
		t.Fatalf("flight changed despite rollback: %+v", gotFlight)
	}

	gotItems, err := repo.ListItineraryItems(ctx, tripID)
	if err != nil {
		t.Fatal(err)
	}
	if len(gotItems) != len(originalItems) {
		t.Fatalf("itinerary count changed: got %d want %d", len(gotItems), len(originalItems))
	}
	for i := range gotItems {
		if gotItems[i].ID != originalItems[i].ID ||
			gotItems[i].DayNumber != originalItems[i].DayNumber ||
			gotItems[i].Title != originalItems[i].Title ||
			gotItems[i].Location != originalItems[i].Location ||
			gotItems[i].EstCostCents != originalItems[i].EstCostCents ||
			gotItems[i].StartTime != originalItems[i].StartTime ||
			gotItems[i].EndTime != originalItems[i].EndTime {
			t.Fatalf("itinerary item changed despite rollback: got %+v want %+v", gotItems[i], originalItems[i])
		}
	}
}
