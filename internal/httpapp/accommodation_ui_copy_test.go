package httpapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAccommodationPageAndFormsCopy validates accommodation UI strings on /accommodation,
// trip page stay forms (itinerary inline, carousel edit, mobile sheet), and shared
// attachment control copy in app.js.
func TestAccommodationPageAndFormsCopy(t *testing.T) {
	root := findModuleRoot(t)

	accB, err := os.ReadFile(filepath.Join(root, "web", "templates", "accommodation.html"))
	if err != nil {
		t.Fatal(err)
	}
	acc := string(accB)

	tripB, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip.html"))
	if err != nil {
		t.Fatal(err)
	}
	trip := string(tripB)
	fabB, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip_fab_flyouts.html"))
	if err != nil {
		t.Fatal(err)
	}
	ub, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip_unified_bookings.html"))
	if err != nil {
		t.Fatal(err)
	}
	tripAndFab := trip + "\n" + string(fabB) + "\n" + string(ub)

	jsB, err := os.ReadFile(filepath.Join(root, "web", "static", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	js := string(jsB)

	// —— accommodation.html ——
	accWant := []string{
		"Manage your accommodation bookings.",
		"Details sync with your itinerary and expenses.",
		`label class="full">Property Name<input`,
		`data-accommodation-status>Address may auto-fill based on the property name.</p>`,
		`<label class="full">Total Cost<input type="number" step="0.01" min="0" name="cost"`,
		`<label class="full">Booking Reference<input type="text" name="booking_confirmation"`,
		`>Save Accommodation</button>`,
		`<h4>No accommodations yet.</h4>`,
		`<p>Add your first accommodation to keep your itinerary and budget in sync.</p>`,
		`{{template "remiBookingStatusField" (dict "name" "booking_status" "value" .BookingStatus)}}`,
		`<small>Booking Reference</small>`,
	}
	for _, s := range accWant {
		if !strings.Contains(acc, s) {
			t.Errorf("accommodation.html missing %q", s)
		}
	}
	accAvoid := []string{
		"Manage your Accommodation bookings.",
		"Details sync to your itinerary and expenses.",
		"Hotel or Residence Name",
		"We will try to auto-fill address from the name you enter.",
		`<label>Cost<input type="number" step="0.01" min="0" name="cost"`,
		`<label>Booking Confirmation<input type="text" name="booking_confirmation"`,
		"Add Next Destination",
		"Keep your journey organized by adding another accommodation.",
		"Update details here and it will sync to itinerary and expenses.",
		`<small>Booking Confirmation</small>`,
	}
	for _, s := range accAvoid {
		if strings.Contains(acc, s) {
			t.Errorf("accommodation.html should not contain %q", s)
		}
	}

	// —— trip.html (accommodation-related) ——
	tripWant := []string{
		`id="accommodation-itinerary-edit-`,
		`label class="full">Property Name<input type="text" name="name" value="{{.Lodging.Name}}"`,
		`{{template "remiBookingStatusField" (dict "name" "booking_status" "value" .Lodging.BookingStatus)}}`,
		`id="trip-accommodation-edit-`,
		`{{template "remiBookingStatusField" (dict "name" "booking_status" "value" $l.BookingStatus)}}`,
		`id="mobile-sheet-accommodation"`,
		`class="trip-resource-form-subtitle mobile-sheet-subtitle">Details sync with your itinerary and expenses.`,
		`<label class="full">Total Cost<input type="number" step="0.01" min="0" name="cost" value="{{printf "%.2f" .Lodging.Cost}}"`,
		`<label class="full">Total Cost<input type="number" step="0.01" min="0" name="cost" value="{{printf "%.2f" $l.Cost}}"`,
		`<label class="full">Total Cost<input type="number" step="0.01" min="0" name="cost" placeholder="0.00"></label>`,
		`<label class="full">Booking Reference<input type="text" name="booking_confirmation" value="{{.Lodging.BookingConfirmation}}"`,
		`<label class="full">Booking Reference<input type="text" name="booking_confirmation" value="{{$l.BookingConfirmation}}"`,
		`<label class="full">Booking Reference<input type="text" name="booking_confirmation" placeholder="e.g. ABCD1234"></label>`,
		`<strong>{{$.CurrencySymbol}}{{printf "%.2f" .Lodging.Cost}}</strong><small>Accommodation cost</small>`,
		`<strong>{{$l.BookingConfirmation}}</strong><small>Booking Reference</small>`,
	}
	for _, s := range tripWant {
		if !strings.Contains(tripAndFab, s) {
			t.Errorf("trip templates (accommodation) missing %q", s)
		}
	}
	tripAccAvoid := []string{
		"Hotel or Residence Name",
		"We will try to auto-fill address from the name you enter.",
	}
	for _, s := range tripAccAvoid {
		if strings.Contains(tripAndFab, s) {
			t.Errorf("trip templates should not contain (accommodation legacy) %q", s)
		}
	}

	// —— app.js attachment control (accommodation forms use same picker as flights) ——
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
