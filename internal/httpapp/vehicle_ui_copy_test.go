package httpapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestVehicleRentalPageAndFormsCopy validates vehicle rental UI strings on /vehicle-rental,
// trip page vehicle forms (itinerary inline, carousel edit, mobile sheet), and shared
// attachment control copy in app.js.
func TestVehicleRentalPageAndFormsCopy(t *testing.T) {
	root := findModuleRoot(t)

	vB, err := os.ReadFile(filepath.Join(root, "web", "templates", "vehicle_rental.html"))
	if err != nil {
		t.Fatal(err)
	}
	v := string(vB)

	tripB, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip.html"))
	if err != nil {
		t.Fatal(err)
	}
	trip := string(tripB)

	jsB, err := os.ReadFile(filepath.Join(root, "web", "static", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	js := string(jsB)

	// —— vehicle_rental.html ——
	vWant := []string{
		"Manage your vehicle rental bookings.",
		`id="vehicle-form-title">New {{.Details.Trip.VehicleSectionTitle}} Booking</h3>`,
		"Details sync with your itinerary and expenses.",
		`<label>Vehicle<input type="text" name="vehicle_detail" placeholder="e.g. Range Rover Velar"></label>`,
		`<label>Booking Reference<input type="text" name="booking_confirmation" placeholder="#REMI-9921"></label>`,
		`<label>Pickup Location<input type="text" name="pick_up_location"`,
		`>Pickup Date & Time{{template "remiDateTimeField"`,
		`<legend class="vehicle-dropoff-legend">Drop-off Location</legend>`,
		`aria-label="Drop-off relative to pickup"`,
		`> Same as pickup</label>`,
		`> Different drop-off location</label>`,
		`>Drop-off Location<input type="text" name="drop_off_location"`,
		`type="submit">Save Vehicle Rental</button>`,
		`<h3>Edit Vehicle Rental</h3>`,
		`class="vehicle-edit-subtitle muted">Update details and they will sync with your itinerary and expenses.`,
		`<small>Booking Reference</small>`,
		`vehicle-meta-same-pickup">Same as pickup</strong>`,
		`aria-label="Open pickup location in Google Maps"`,
	}
	for _, s := range vWant {
		if !strings.Contains(v, s) {
			t.Errorf("vehicle_rental.html missing %q", s)
		}
	}
	vAvoid := []string{
		"Manage your {{.Details.Trip.VehicleSectionTitle}} bookings.",
		"Details sync to your itinerary and expenses.",
		"Vehicle rental name<input",
		`<label>Booking Confirmation<input type="text" name="booking_confirmation"`,
		"Pick-up Location<input",
		"Pick-up Date & Time{{template",
		`<legend class="vehicle-dropoff-legend">Drop-off location</legend>`,
		`>Drop-off location<input type="text" name="drop_off_location"`,
		"> Same as pick-up</label>",
		"> Different drop-off</label>",
		`Save {{.Details.Trip.VehicleSectionTitle}} booking`,
		`<h3>Edit {{$.Details.Trip.VehicleSectionTitle}}</h3>`,
		"Update details here and it will sync to itinerary and expenses.",
		`vehicle-meta-same-pickup">Same as pick-up</strong>`,
	}
	for _, s := range vAvoid {
		if strings.Contains(v, s) {
			t.Errorf("vehicle_rental.html should not contain %q", s)
		}
	}

	// —— trip.html (vehicle-related) ——
	tripWant := []string{
		`id="vehicle-rental-itinerary-edit-`,
		`label>Vehicle<input type="text" name="vehicle_detail" value="{{.Vehicle.VehicleDetail}}"`,
		`<span class="itinerary-label">Vehicle</span>`,
		`id="vehicle-rental-edit-`,
		`<h3>Edit Vehicle Rental</h3>`,
		`class="vehicle-edit-subtitle muted">Update details and they will sync with your itinerary and expenses.`,
		`id="mobile-sheet-vehicle"`,
		`class="trip-resource-form-subtitle mobile-sheet-subtitle">Details sync with your itinerary and expenses.`,
		`<label>Pickup Location<input type="text" name="pick_up_location" value="{{.Vehicle.PickUpLocation}}"`,
		`<label>Pickup Location<input type="text" name="pick_up_location" value="{{.PickUpLocation}}"`,
		`<label>Booking Reference<input type="text" name="booking_confirmation" value="{{.Vehicle.BookingConfirmation}}"`,
		`<label>Booking Reference<input type="text" name="booking_confirmation" value="{{.BookingConfirmation}}"`,
		`<label>Booking Reference<input type="text" name="booking_confirmation" placeholder="#REMI-9921"></label>`,
		`type="submit">Save Vehicle Rental</button>`,
		`<strong>{{.BookingConfirmation}}</strong><small>Booking Reference</small>`,
	}
	for _, s := range tripWant {
		if !strings.Contains(trip, s) {
			t.Errorf("trip.html (vehicle) missing %q", s)
		}
	}
	tripVehicleAvoid := []string{
		`label>Vehicle rental name<input type="text" name="vehicle_detail"`,
		`<label>Pick-up Location<input type="text" name="pick_up_location"`,
		`<label>Booking Confirmation<input type="text" name="booking_confirmation" value="{{.Vehicle.BookingConfirmation}}"`,
		`<label>Booking Confirmation<input type="text" name="booking_confirmation" value="{{.BookingConfirmation}}"`,
		`<label>Booking Confirmation<input type="text" name="booking_confirmation" placeholder="#REMI-9921"></label>`,
		`<legend class="vehicle-dropoff-legend">Drop-off location</legend>`,
		`aria-label="Drop-off relative to pick-up"`,
	}
	for _, s := range tripVehicleAvoid {
		if strings.Contains(trip, s) {
			t.Errorf("trip.html should not contain (vehicle legacy) %q", s)
		}
	}

	// Vehicle itinerary row must not use "Same as pick-up" in the meta line.
	if strings.Contains(trip, `vehicle-meta-same-pickup">Same as pick-up`) {
		t.Error(`trip.html vehicle card should use "Same as pickup" in .vehicle-meta-same-pickup`)
	}

	// —— app.js attachment control (vehicle forms use same picker) ——
	for _, s := range []string{`"No image uploaded"`, `"No file attached"`, `"Upload File"`} {
		if !strings.Contains(js, s) {
			t.Errorf("app.js missing %q", s)
		}
	}
	for _, s := range []string{`"No image attached"`, `"No attachment"`, `: "Choose File"`} {
		if strings.Contains(js, s) {
			t.Errorf("app.js should not contain %q", s)
		}
	}
}
