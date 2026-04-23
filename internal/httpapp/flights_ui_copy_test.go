package httpapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestFlightsPageAndFormsCopy validates flight UI strings on /flights, trip page flight forms,
// itinerary flight rows, mobile sheet, and shared attachment control copy in app.js.
func TestFlightsPageAndFormsCopy(t *testing.T) {
	root := findModuleRoot(t)

	flightsB, err := os.ReadFile(filepath.Join(root, "web", "templates", "flights.html"))
	if err != nil {
		t.Fatal(err)
	}
	flights := string(flightsB)

	tripB, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip.html"))
	if err != nil {
		t.Fatal(err)
	}
	trip := string(tripB)
	fabB, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip_fab_flyouts.html"))
	if err != nil {
		t.Fatal(err)
	}
	bookingFieldB, err := os.ReadFile(filepath.Join(root, "web", "templates", "booking_status_field.html"))
	if err != nil {
		t.Fatal(err)
	}
	bookingField := string(bookingFieldB)
	ub, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip_unified_bookings.html"))
	if err != nil {
		t.Fatal(err)
	}
	tripAndFab := trip + "\n" + string(fabB) + "\n" + bookingField + "\n" + string(ub)
	flightsWithBookingField := flights + "\n" + bookingField

	jsB, err := os.ReadFile(filepath.Join(root, "web", "static", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	js := string(jsB)

	// —— flights.html ——
	flightsWant := []string{
		"Manage your flight bookings.",
		`>New Flight Booking</h3>`,
		"Details sync with your itinerary and expenses.",
		`class="full flight-section-head flight-edit-form-head"`,
		"Update details and they will sync with your itinerary and expenses.",
		`name="{{.name}}"`,
		`>Booking Status`,
		`value="to_be_done"`,
		`label class="full">Airline<input`,
		`label class="full">Departure Airport<input`,
		`label class="full">Departure Date & Time`,
		`label class="full">Arrival Airport<input`,
		`label class="full">Arrival Date & Time`,
		`label class="full">Booking Reference<input`,
		`<label class="full">Total Cost<input type="number" step="0.01" min="0" name="cost"`,
		`placeholder="Add any additional details"`,
	}
	for _, s := range flightsWant {
		if !strings.Contains(flightsWithBookingField, s) {
			t.Errorf("flights templates (incl. booking status field) missing %q", s)
		}
	}
	flightsAvoid := []string{
		"Manage your air travel with ease.",
		"Details sync to your itinerary and expenses.",
		"label>Flight Name<input",
		"label>Depart Airport<input",
		"label>Depart Date and Time",
		"label>Arrive Airport<input",
		"label>Arrive Date and Time",
		"label>Booking Confirmation #<input",
		`<label>Cost<input type="number" step="0.01" min="0" name="cost"`,
		`Update details here and it will sync to itinerary and expenses.`,
	}
	for _, s := range flightsAvoid {
		if strings.Contains(flights, s) {
			t.Errorf("flights.html should not contain %q", s)
		}
	}
	if strings.Contains(flights, "New {{.Details.Trip.FlightsSectionTitle}} booking") {
		t.Error("flights.html should use static New Flight Booking title")
	}

	// —— trip.html (flight-related) ——
	tripWant := []string{
		`{{$route := flightRouteIATALine .Flight}}`,
		`flight-meta-grid`,
		`<strong>{{.Flight.BookingConfirmation}}</strong><small>Booking Confirmation</small>`,
		`<strong>{{$.CurrencySymbol}}{{printf "%.2f" .Flight.Cost}}</strong><small>Total Cost</small>`,
		`value="{{.Flight.Notes}}" placeholder="Add any additional details"`,
		`class="full flight-section-head flight-edit-form-head"`,
		`class="flight-edit-subtitle muted">Update details and they will sync with your itinerary and expenses.`,
		`name="{{.name}}"`,
		`value="to_be_done"`,
		`label class="full">Airline<input`,
		`label class="full">Departure Airport<input`,
		`label class="full">Departure Date & Time`,
		`label class="full">Arrival Airport<input`,
		`label class="full">Arrival Date & Time`,
		`label class="full">Booking Reference<input`,
		`<label class="full">Total Cost<input type="number" step="0.01" min="0" name="cost" value="{{printf "%.2f" .Flight.Cost}}"`,
		`<label class="full">Total Cost<input type="number" step="0.01" min="0" name="cost" value="{{printf "%.2f" $f.Cost}}"`,
		`action="/trips/{{.Details.Trip.ID}}/flights"`,
	}
	for _, s := range tripWant {
		if !strings.Contains(tripAndFab, s) {
			t.Errorf("trip templates (flights) missing %q", s)
		}
	}
	tripFlightAvoid := []string{
		`Depart Airport{{else}}Arrive Airport`,
		`<span class="itinerary-label">Flight</span>`,
		`value="{{.Flight.Notes}}" placeholder="Optional details"`,
		`<h3>Edit {{$.Details.Trip.FlightsSectionTitle}}</h3>`,
		`class="flight-edit-subtitle muted">Update details here and it will sync to itinerary and expenses.`,
		`label>Flight Name<input`,
		`label>Depart Airport<input`,
		`label>Booking Confirmation #<input`,
	}
	for _, s := range tripFlightAvoid {
		if strings.Contains(tripAndFab, s) {
			t.Errorf("trip templates should not contain (flight legacy) %q", s)
		}
	}

	// —— app.js attachment control ——
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
