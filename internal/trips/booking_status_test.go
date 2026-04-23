package trips

import "testing"

func TestParseBookFlightChecklistTitle(t *testing.T) {
	t.Parallel()
	a, ok := ParseBookFlightChecklistTitle("Book Flight: EVA Air")
	if !ok || a != "EVA Air" {
		t.Fatalf("got %q %v", a, ok)
	}
	a, ok = ParseBookFlightChecklistTitle("Book: IndiGo - BLR to COK")
	if !ok || a != "IndiGo" {
		t.Fatalf("new form: got %q %v", a, ok)
	}
	_, ok = ParseBookFlightChecklistTitle("Random text")
	if ok {
		t.Fatal("expected no prefix")
	}
	_, ok = ParseBookFlightChecklistTitle("Book: IndiGo - BLR")
	if ok {
		t.Fatal("incomplete line should not parse as new form")
	}
}

func TestBookFlightChecklistTitle(t *testing.T) {
	t.Parallel()
	got := BookFlightChecklistTitle(Flight{
		FlightName:        "IndiGo - 6E",
		DepartAirportIATA: "BLR",
		ArriveAirportIATA: "COK",
	})
	if want := "Book: IndiGo - BLR to COK"; got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestNormalizeBookingStatus(t *testing.T) {
	t.Parallel()
	cases := []struct {
		in, want string
	}{
		{"", BookingStatusToBeDone},
		{"  ", BookingStatusToBeDone},
		{BookingStatusToBeDone, BookingStatusToBeDone},
		{BookingStatusDone, BookingStatusDone},
		{BookingStatusNotRequired, BookingStatusNotRequired},
		{"DONE", BookingStatusDone},
		{"not_required", BookingStatusNotRequired},
		{"garbage", BookingStatusToBeDone},
	}
	for _, tc := range cases {
		if got := NormalizeBookingStatus(tc.in); got != tc.want {
			t.Errorf("NormalizeBookingStatus(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}
