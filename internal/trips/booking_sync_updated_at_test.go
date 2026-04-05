package trips_test

import (
	"context"
	"testing"

	"remi-trip-planner/internal/trips"
)

func TestSyncExpenseForVehicleRentalDoesNotBumpUpdatedAtWhenExpenseLinksStable(t *testing.T) {
	repo, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	svc := trips.NewService(repo)

	ownerID, err := repo.CreateUser(ctx, trips.User{
		Email:        "owner-vehicle-sync@example.com",
		Username:     "owner_vehicle_sync",
		DisplayName:  "Owner",
		PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{
		Name:        "Vehicle Sync",
		OwnerUserID: ownerID,
		StartDate:   "2026-06-01",
		EndDate:     "2026-06-07",
	})
	if err != nil {
		t.Fatal(err)
	}

	rentalID := "vehicle-sync-1"
	pickUpID := "vehicle-sync-pickup-1"
	dropOffID := "vehicle-sync-dropoff-1"
	if err := svc.AddVehicleRentalWithItinerary(ctx,
		trips.VehicleRental{
			ID:                 rentalID,
			TripID:             tripID,
			PickUpLocation:     "Tokyo Station",
			DropOffLocation:    "Haneda Airport",
			VehicleDetail:      "Compact Car",
			PickUpAt:           "2026-06-02T09:00",
			DropOffAt:          "2026-06-04T18:00",
			CostCents:          18000,
			InsuranceCostCents: 3500,
			PickUpItineraryID:  pickUpID,
			DropOffItineraryID: dropOffID,
		},
		trips.ItineraryItem{
			ID:        pickUpID,
			TripID:    tripID,
			DayNumber: 2,
			Title:     trips.VehicleRentalItineraryPickUpTitle("Compact Car"),
			Location:  "Tokyo Station",
			StartTime: "09:00",
			EndTime:   "09:00",
		},
		trips.ItineraryItem{
			ID:        dropOffID,
			TripID:    tripID,
			DayNumber: 4,
			Title:     trips.VehicleRentalItineraryDropOffTitle("Compact Car"),
			Location:  "Haneda Airport",
			StartTime: "18:00",
			EndTime:   "18:00",
		},
	); err != nil {
		t.Fatal(err)
	}

	before, err := repo.GetVehicleRental(ctx, tripID, rentalID)
	if err != nil {
		t.Fatal(err)
	}
	if before.RentalExpenseID == "" || before.InsuranceExpenseID == "" {
		t.Fatalf("expected linked expense IDs, got rental=%q insurance=%q", before.RentalExpenseID, before.InsuranceExpenseID)
	}

	if err := svc.SyncExpenseForVehicleRental(ctx, before); err != nil {
		t.Fatal(err)
	}
	after, err := repo.GetVehicleRental(ctx, tripID, rentalID)
	if err != nil {
		t.Fatal(err)
	}
	if !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("vehicle updated_at changed unexpectedly: before=%s after=%s", before.UpdatedAt, after.UpdatedAt)
	}
}

func TestSyncExpenseForLodgingDoesNotBumpUpdatedAt(t *testing.T) {
	repo, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	svc := trips.NewService(repo)

	ownerID, err := repo.CreateUser(ctx, trips.User{
		Email:        "owner-lodging-sync@example.com",
		Username:     "owner_lodging_sync",
		DisplayName:  "Owner",
		PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{
		Name:        "Lodging Sync",
		OwnerUserID: ownerID,
		StartDate:   "2026-06-01",
		EndDate:     "2026-06-07",
	})
	if err != nil {
		t.Fatal(err)
	}

	lodgingID := "lodging-sync-1"
	checkInID := "lodging-sync-checkin-1"
	checkOutID := "lodging-sync-checkout-1"
	if err := svc.AddLodgingWithItinerary(ctx,
		trips.Lodging{
			ID:                  lodgingID,
			TripID:              tripID,
			Name:                "Sync Hotel",
			Address:             "Shinjuku",
			CheckInAt:           "2026-06-02T15:00",
			CheckOutAt:          "2026-06-04T10:00",
			CostCents:           28000,
			CheckInItineraryID:  checkInID,
			CheckOutItineraryID: checkOutID,
		},
		trips.ItineraryItem{
			ID:        checkInID,
			TripID:    tripID,
			DayNumber: 2,
			Title:     trips.AccommodationItineraryCheckInTitle("Sync Hotel"),
			Location:  "Shinjuku",
			StartTime: "15:00",
			EndTime:   "15:00",
		},
		trips.ItineraryItem{
			ID:        checkOutID,
			TripID:    tripID,
			DayNumber: 4,
			Title:     trips.AccommodationItineraryCheckOutTitle("Sync Hotel"),
			Location:  "Shinjuku",
			StartTime: "10:00",
			EndTime:   "10:00",
		},
	); err != nil {
		t.Fatal(err)
	}

	before, err := repo.GetLodging(ctx, tripID, lodgingID)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.SyncExpenseForLodging(ctx, before); err != nil {
		t.Fatal(err)
	}
	after, err := repo.GetLodging(ctx, tripID, lodgingID)
	if err != nil {
		t.Fatal(err)
	}
	if !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("lodging updated_at changed unexpectedly: before=%s after=%s", before.UpdatedAt, after.UpdatedAt)
	}
}

func TestSyncExpenseForFlightDoesNotBumpUpdatedAtWhenExpenseLinkStable(t *testing.T) {
	repo, cleanup := testDB(t)
	defer cleanup()

	ctx := context.Background()
	svc := trips.NewService(repo)

	ownerID, err := repo.CreateUser(ctx, trips.User{
		Email:        "owner-flight-sync@example.com",
		Username:     "owner_flight_sync",
		DisplayName:  "Owner",
		PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{
		Name:        "Flight Sync",
		OwnerUserID: ownerID,
		StartDate:   "2026-06-01",
		EndDate:     "2026-06-07",
	})
	if err != nil {
		t.Fatal(err)
	}

	flightID := "flight-sync-1"
	departID := "flight-sync-depart-1"
	arriveID := "flight-sync-arrive-1"
	if err := svc.AddFlightWithItinerary(ctx,
		trips.Flight{
			ID:                flightID,
			TripID:            tripID,
			FlightName:        "Sync Air",
			FlightNumber:      "SA 101",
			DepartAirport:     "NRT",
			ArriveAirport:     "KIX",
			DepartAt:          "2026-06-03T07:00",
			ArriveAt:          "2026-06-03T08:30",
			CostCents:         12345,
			DepartItineraryID: departID,
			ArriveItineraryID: arriveID,
		},
		trips.ItineraryItem{
			ID:        departID,
			TripID:    tripID,
			DayNumber: 3,
			Title:     trips.FlightItineraryDepartTitle("Sync Air (SA 101)"),
			Location:  "NRT",
			StartTime: "07:00",
			EndTime:   "07:00",
		},
		trips.ItineraryItem{
			ID:        arriveID,
			TripID:    tripID,
			DayNumber: 3,
			Title:     trips.FlightItineraryArriveTitle("Sync Air (SA 101)"),
			Location:  "KIX",
			StartTime: "08:30",
			EndTime:   "08:30",
		},
	); err != nil {
		t.Fatal(err)
	}

	before, err := repo.GetFlight(ctx, tripID, flightID)
	if err != nil {
		t.Fatal(err)
	}
	if before.ExpenseID == "" {
		t.Fatal("expected linked expense ID")
	}

	if err := svc.SyncExpenseForFlight(ctx, before); err != nil {
		t.Fatal(err)
	}
	after, err := repo.GetFlight(ctx, tripID, flightID)
	if err != nil {
		t.Fatal(err)
	}
	if !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatalf("flight updated_at changed unexpectedly: before=%s after=%s", before.UpdatedAt, after.UpdatedAt)
	}
}
