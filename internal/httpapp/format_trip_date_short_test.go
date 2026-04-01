package httpapp

import (
	"testing"

	"remi-trip-planner/internal/trips"
)

func TestFormatTripDateShortForTrip(t *testing.T) {
	trip := trips.Trip{UIDateFormat: "dmy"}
	cases := []struct {
		start, end, want string
	}{
		{"2026-04-01", "2026-04-05", "1 – 5 Apr"},
		{"2026-03-24", "2026-03-29", "24 – 29 Mar"},
		{"2026-03-30", "2026-04-02", "30 Mar – 2 Apr"},
		{"2026-12-30", "2027-01-02", "30 Dec '26 – 2 Jan '27"},
		{"2026-03-24", "2026-03-24", "24 Mar"},
		{"", "2026-01-02", ""},
	}
	for _, tc := range cases {
		got := formatTripDateShortForTrip(trip, tc.start, tc.end)
		if got != tc.want {
			t.Errorf("formatTripDateShortForTrip(..., %q, %q) = %q; want %q", tc.start, tc.end, got, tc.want)
		}
	}
	// Mobile short format is fixed; ignore per-trip US date order preference.
	usTrip := trips.Trip{UIDateFormat: "mdy"}
	got := formatTripDateShortForTrip(usTrip, "2026-03-24", "2026-03-29")
	if got != "24 – 29 Mar" {
		t.Errorf("MDY trip got %q; want same as DMY for mobile short range", got)
	}
}
