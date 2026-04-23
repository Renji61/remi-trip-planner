package trips_test

import (
	"context"
	"strings"
	"testing"

	"remi-trip-planner/internal/storage/sqlite"
	"remi-trip-planner/internal/trips"
)

func TestBookFlightChecklistTitle_Integration(t *testing.T) {
	t.Parallel()
	f := trips.Flight{FlightName: "Cebu Pacific", DepartAirportIATA: "MNL", ArriveAirportIATA: "CEB"}
	if g := trips.BookFlightChecklistTitle(f); g != "Book: Cebu Pacific - MNL to CEB" {
		t.Fatalf("got %q", g)
	}
	f2 := trips.Flight{FlightName: "  ", DepartAirportIATA: "MNL", ArriveAirportIATA: "CEB"}
	if g := trips.BookFlightChecklistTitle(f2); g != "Book: Flight - MNL to CEB" {
		t.Fatalf("empty name: got %q", g)
	}
}

func TestTripBookings_FlightToBeDoneCreatesChecklist(t *testing.T) {
	repo, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	svc := trips.NewService(repo)

	tripID := seedTripForFlights(t, ctx, repo, "tb1")

	dep, arr := "dep-1", "arr-1"
	flightID := "flight-cl-1"
	if err := svc.AddFlightWithItinerary(ctx,
		trips.Flight{
			ID:                flightID,
			TripID:            tripID,
			FlightName:        "Try Air",
			FlightNumber:      "TR 1",
			BookingStatus:     trips.BookingStatusToBeDone,
			DepartAirport:     "MNL",
			ArriveAirport:     "CEB",
			DepartAirportIATA: "MNL",
			ArriveAirportIATA: "CEB",
			DepartAt:          "2026-06-02T08:00",
			ArriveAt:          "2026-06-02T09:00",
			DepartItineraryID: dep,
			ArriveItineraryID: arr,
		},
		trips.ItineraryItem{ID: dep, TripID: tripID, DayNumber: 2, Title: trips.FlightItineraryDepartTitle("x"), Location: "MNL", StartTime: "08:00", EndTime: "08:00"},
		trips.ItineraryItem{ID: arr, TripID: tripID, DayNumber: 2, Title: trips.FlightItineraryArriveTitle("x"), Location: "CEB", StartTime: "09:00", EndTime: "09:00"},
	); err != nil {
		t.Fatal(err)
	}

	f, err := repo.GetFlight(ctx, tripID, flightID)
	if err != nil {
		t.Fatal(err)
	}
	if f.TripBookingsChecklistItemID == "" {
		t.Fatal("expected linked checklist id")
	}
	cl, err := repo.GetChecklistItem(ctx, f.TripBookingsChecklistItemID)
	if err != nil {
		t.Fatal(err)
	}
	if cl.Category != trips.TripBookingsChecklistCategory {
		t.Fatalf("category: %q", cl.Category)
	}
	if want := "Book: Try Air - MNL to CEB"; cl.Text != want {
		t.Fatalf("text: %q want %q", cl.Text, want)
	}
	if cl.Done {
		t.Fatal("expected not done")
	}
}

func TestTripBookings_ChecklistDoneSyncsFlight(t *testing.T) {
	repo, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	svc := trips.NewService(repo)
	tripID := seedTripForFlights(t, ctx, repo, "tb2")
	dep, arr := "dep-2", "arr-2"
	flightID := "flight-cl-2"
	if err := svc.AddFlightWithItinerary(ctx,
		trips.Flight{
			ID:                flightID,
			TripID:            tripID,
			FlightName:        "Sync Co",
			FlightNumber:      "SC 2",
			DepartAirport:     "MNL",
			ArriveAirport:     "CEB",
			DepartAirportIATA: "MNL",
			ArriveAirportIATA: "CEB",
			DepartAt:          "2026-06-02T10:00",
			ArriveAt:          "2026-06-02T11:00",
			DepartItineraryID: dep,
			ArriveItineraryID: arr,
		},
		trips.ItineraryItem{ID: dep, TripID: tripID, DayNumber: 2, Title: trips.FlightItineraryDepartTitle("s"), Location: "MNL", StartTime: "10:00", EndTime: "10:00"},
		trips.ItineraryItem{ID: arr, TripID: tripID, DayNumber: 2, Title: trips.FlightItineraryArriveTitle("s"), Location: "CEB", StartTime: "11:00", EndTime: "11:00"},
	); err != nil {
		t.Fatal(err)
	}
	f, err := repo.GetFlight(ctx, tripID, flightID)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.ToggleChecklistItem(ctx, f.TripBookingsChecklistItemID, true); err != nil {
		t.Fatal(err)
	}
	after, err := repo.GetFlight(ctx, tripID, flightID)
	if err != nil {
		t.Fatal(err)
	}
	if trips.NormalizeBookingStatus(after.BookingStatus) != trips.BookingStatusDone {
		t.Fatalf("flight status: %q", after.BookingStatus)
	}
}

func TestTripBookings_FlightDoneTicksChecklist(t *testing.T) {
	repo, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	svc := trips.NewService(repo)
	tripID := seedTripForFlights(t, ctx, repo, "tb3")
	dep, arr := "dep-3", "arr-3"
	flightID := "flight-cl-3"
	if err := svc.AddFlightWithItinerary(ctx,
		trips.Flight{
			ID:                flightID,
			TripID:            tripID,
			FlightName:        "Mark Air",
			FlightNumber:      "MA 3",
			DepartAirport:     "MNL",
			ArriveAirport:     "DVO",
			DepartAirportIATA: "MNL",
			ArriveAirportIATA: "DVO",
			DepartAt:          "2026-06-02T12:00",
			ArriveAt:          "2026-06-02T13:00",
			DepartItineraryID: dep,
			ArriveItineraryID: arr,
		},
		trips.ItineraryItem{ID: dep, TripID: tripID, DayNumber: 2, Title: trips.FlightItineraryDepartTitle("m"), Location: "MNL", StartTime: "12:00", EndTime: "12:00"},
		trips.ItineraryItem{ID: arr, TripID: tripID, DayNumber: 2, Title: trips.FlightItineraryArriveTitle("m"), Location: "DVO", StartTime: "13:00", EndTime: "13:00"},
	); err != nil {
		t.Fatal(err)
	}
	f, err := repo.GetFlight(ctx, tripID, flightID)
	if err != nil {
		t.Fatal(err)
	}
	cid := f.TripBookingsChecklistItemID
	f.BookingStatus = trips.BookingStatusDone
	if err := svc.UpdateFlight(ctx, f); err != nil {
		t.Fatal(err)
	}
	cl, err := repo.GetChecklistItem(ctx, cid)
	if err != nil {
		t.Fatal(err)
	}
	if !cl.Done {
		t.Fatal("checklist should be done")
	}
}

func TestTripBookings_DeletedChecklistNotResynced(t *testing.T) {
	repo, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	svc := trips.NewService(repo)
	tripID := seedTripForFlights(t, ctx, repo, "tb4")
	dep, arr := "dep-4", "arr-4"
	flightID := "flight-cl-4"
	if err := svc.AddFlightWithItinerary(ctx,
		trips.Flight{
			ID:                flightID,
			TripID:            tripID,
			FlightName:        "Gone Air",
			FlightNumber:      "GA 4",
			DepartAirport:     "MNL",
			ArriveAirport:     "CEB",
			DepartAirportIATA: "MNL",
			ArriveAirportIATA: "CEB",
			DepartAt:          "2026-06-02T14:00",
			ArriveAt:          "2026-06-02T15:00",
			DepartItineraryID: dep,
			ArriveItineraryID: arr,
		},
		trips.ItineraryItem{ID: dep, TripID: tripID, DayNumber: 2, Title: trips.FlightItineraryDepartTitle("g"), Location: "MNL", StartTime: "14:00", EndTime: "14:00"},
		trips.ItineraryItem{ID: arr, TripID: tripID, DayNumber: 2, Title: trips.FlightItineraryArriveTitle("g"), Location: "CEB", StartTime: "15:00", EndTime: "15:00"},
	); err != nil {
		t.Fatal(err)
	}
	f, err := repo.GetFlight(ctx, tripID, flightID)
	if err != nil {
		t.Fatal(err)
	}
	if err := svc.DeleteChecklistItem(ctx, f.TripBookingsChecklistItemID); err != nil {
		t.Fatal(err)
	}
	after, err := repo.GetFlight(ctx, tripID, flightID)
	if err != nil {
		t.Fatal(err)
	}
	if !after.TripBookingsChecklistDismissed {
		t.Fatal("expected dismissed")
	}
	after.BookingStatus = trips.BookingStatusToBeDone
	if err := svc.UpdateFlight(ctx, after); err != nil {
		t.Fatal(err)
	}
	n2, err := repo.GetFlight(ctx, tripID, flightID)
	if err != nil {
		t.Fatal(err)
	}
	if n2.TripBookingsChecklistItemID != "" {
		t.Fatal("should not recreate checklist when dismissed")
	}
	items, err := repo.ListChecklistItems(ctx, tripID)
	if err != nil {
		t.Fatal(err)
	}
	for _, it := range items {
		if strings.Contains(it.Text, "Gone Air") {
			t.Fatalf("unexpected new checklist: %+v", it)
		}
	}
}

func TestTripBookings_NotRequiredStrikesChecklist(t *testing.T) {
	repo, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	svc := trips.NewService(repo)
	tripID := seedTripForFlights(t, ctx, repo, "nr1")
	dep, arr := "dep-nr", "arr-nr"
	flightID := "flight-nr"
	if err := svc.AddFlightWithItinerary(ctx,
		trips.Flight{
			ID:                flightID,
			TripID:            tripID,
			FlightName:        "NR Air",
			DepartAirport:     "MNL",
			ArriveAirport:     "CEB",
			DepartAirportIATA: "MNL",
			ArriveAirportIATA: "CEB",
			DepartAt:          "2026-06-02T16:00",
			ArriveAt:          "2026-06-02T17:00",
			DepartItineraryID: dep,
			ArriveItineraryID: arr,
		},
		trips.ItineraryItem{ID: dep, TripID: tripID, DayNumber: 2, Title: trips.FlightItineraryDepartTitle("n"), Location: "MNL", StartTime: "16:00", EndTime: "16:00"},
		trips.ItineraryItem{ID: arr, TripID: tripID, DayNumber: 2, Title: trips.FlightItineraryArriveTitle("n"), Location: "CEB", StartTime: "17:00", EndTime: "17:00"},
	); err != nil {
		t.Fatal(err)
	}
	f, err := repo.GetFlight(ctx, tripID, flightID)
	if err != nil {
		t.Fatal(err)
	}
	f.BookingStatus = trips.BookingStatusNotRequired
	if err := svc.UpdateFlight(ctx, f); err != nil {
		t.Fatal(err)
	}
	cl, err := repo.GetChecklistItem(ctx, f.TripBookingsChecklistItemID)
	if err != nil {
		t.Fatal(err)
	}
	if !cl.Done {
		t.Fatal("not required should mark checklist done (strikethrough in UI)")
	}
}

func TestTripBookings_AirlineRenamedWhenBookingNotRequiredUpdatesChecklist(t *testing.T) {
	repo, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	svc := trips.NewService(repo)
	tripID := seedTripForFlights(t, ctx, repo, "rx1")
	dep, arr := "dep-rx", "arr-rx"
	flightID := "flight-rx"
	if err := svc.AddFlightWithItinerary(ctx,
		trips.Flight{
			ID:                flightID,
			TripID:            tripID,
			FlightName:        "Old Name",
			DepartAirport:     "MNL",
			ArriveAirport:     "CEB",
			DepartAirportIATA: "MNL",
			ArriveAirportIATA: "CEB",
			DepartAt:          "2026-06-02T18:00",
			ArriveAt:          "2026-06-02T19:00",
			DepartItineraryID: dep,
			ArriveItineraryID: arr,
		},
		trips.ItineraryItem{ID: dep, TripID: tripID, DayNumber: 2, Title: trips.FlightItineraryDepartTitle("o"), Location: "MNL", StartTime: "18:00", EndTime: "18:00"},
		trips.ItineraryItem{ID: arr, TripID: tripID, DayNumber: 2, Title: trips.FlightItineraryArriveTitle("o"), Location: "CEB", StartTime: "19:00", EndTime: "19:00"},
	); err != nil {
		t.Fatal(err)
	}
	f, err := repo.GetFlight(ctx, tripID, flightID)
	if err != nil {
		t.Fatal(err)
	}
	f.BookingStatus = trips.BookingStatusNotRequired
	if err := svc.UpdateFlight(ctx, f); err != nil {
		t.Fatal(err)
	}
	f, err = repo.GetFlight(ctx, tripID, flightID)
	if err != nil {
		t.Fatal(err)
	}
	f.FlightName = "New Name Co"
	if err := svc.UpdateFlight(ctx, f); err != nil {
		t.Fatal(err)
	}
	f, err = repo.GetFlight(ctx, tripID, flightID)
	if err != nil {
		t.Fatal(err)
	}
	cl, err := repo.GetChecklistItem(ctx, f.TripBookingsChecklistItemID)
	if err != nil {
		t.Fatal(err)
	}
	want := trips.BookFlightChecklistTitle(f)
	if cl.Text != want {
		t.Fatalf("checklist text: %q want %q", cl.Text, want)
	}
	if !cl.Done {
		t.Fatal("checklist should stay struck when renaming under not required")
	}
}

func TestTripBookings_ChecklistTitleEditUpdatesAirline(t *testing.T) {
	repo, cleanup := testDB(t)
	defer cleanup()
	ctx := context.Background()
	svc := trips.NewService(repo)
	tripID := seedTripForFlights(t, ctx, repo, "ed1")
	dep, arr := "dep-ed", "arr-ed"
	flightID := "flight-ed"
	if err := svc.AddFlightWithItinerary(ctx,
		trips.Flight{
			ID:                flightID,
			TripID:            tripID,
			FlightName:        "Start Air",
			DepartAirport:     "MNL",
			ArriveAirport:     "CEB",
			DepartAirportIATA: "MNL",
			ArriveAirportIATA: "CEB",
			DepartAt:          "2026-06-02T20:00",
			ArriveAt:          "2026-06-02T21:00",
			DepartItineraryID: dep,
			ArriveItineraryID: arr,
		},
		trips.ItineraryItem{ID: dep, TripID: tripID, DayNumber: 2, Title: trips.FlightItineraryDepartTitle("s"), Location: "MNL", StartTime: "20:00", EndTime: "20:00"},
		trips.ItineraryItem{ID: arr, TripID: tripID, DayNumber: 2, Title: trips.FlightItineraryArriveTitle("s"), Location: "CEB", StartTime: "21:00", EndTime: "21:00"},
	); err != nil {
		t.Fatal(err)
	}
	f, err := repo.GetFlight(ctx, tripID, flightID)
	if err != nil {
		t.Fatal(err)
	}
	cid := f.TripBookingsChecklistItemID
	if err := svc.UpdateChecklistItem(ctx, trips.ChecklistItem{
		ID:       cid,
		Text:     trips.BookFlightChecklistTitle(trips.Flight{FlightName: "Renamed From Checklist", DepartAirportIATA: "MNL", ArriveAirportIATA: "CEB"}),
		Category: trips.TripBookingsChecklistCategory,
	}); err != nil {
		t.Fatal(err)
	}
	after, err := repo.GetFlight(ctx, tripID, flightID)
	if err != nil {
		t.Fatal(err)
	}
	if after.FlightName != "Renamed From Checklist" {
		t.Fatalf("FlightName: %q", after.FlightName)
	}
}

func seedTripForFlights(t *testing.T, ctx context.Context, repo *sqlite.Repository, userSuffix string) string {
	t.Helper()
	ownerID, err := repo.CreateUser(ctx, trips.User{
		Email:        "tb+" + userSuffix + "@test.local",
		Username:     "tbook_" + userSuffix,
		DisplayName:  "T",
		PasswordHash: "x",
	})
	if err != nil {
		t.Fatal(err)
	}
	tripID, err := repo.CreateTrip(ctx, trips.Trip{
		Name:        "Trip for flights",
		Description: "t",
		OwnerUserID: ownerID,
		StartDate:   "2026-06-01",
		EndDate:     "2026-06-07",
	})
	if err != nil {
		t.Fatal(err)
	}
	return tripID
}
