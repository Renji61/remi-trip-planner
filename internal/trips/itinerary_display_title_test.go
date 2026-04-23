package trips

import "testing"

func TestItineraryLodgingPropertyDisplay(t *testing.T) {
	l := Lodging{Name: "Aman"}
	if got := ItineraryLodgingPropertyDisplay(l, "Accommodation check-in: Old"); got != "Aman" {
		t.Fatalf("name from lodging: %q", got)
	}
	l2 := Lodging{}
	if got := ItineraryLodgingPropertyDisplay(l2, "Accommodation check-in: Old Inn"); got != "Old Inn" {
		t.Fatalf("parsed: %q", got)
	}
}

func TestItineraryVehicleRentalNameDisplay(t *testing.T) {
	v := VehicleRental{VehicleDetail: "Velar"}
	if got := ItineraryVehicleRentalNameDisplay(v, "Vehicle rental: ignored"); got != "Velar" {
		t.Fatalf("detail: %q", got)
	}
	v2 := VehicleRental{PickUpLocation: "SIN"}
	if got := ItineraryVehicleRentalNameDisplay(v2, "Vehicle rental: SIN"); got != "SIN" {
		t.Fatalf("from title: %q", got)
	}
}

func TestItineraryFlightAirlineDisplay(t *testing.T) {
	f := Flight{FlightName: "SQ"}
	if got := ItineraryFlightAirlineDisplay(f, "Flight: X"); got != "SQ" {
		t.Fatalf("flight name: %q", got)
	}
	f2 := Flight{FlightNumber: "123"}
	if got := ItineraryFlightAirlineDisplay(f2, "Flight: Legacy Air"); got != "Legacy Air" {
		t.Fatalf("from stored title: %q", got)
	}
}
