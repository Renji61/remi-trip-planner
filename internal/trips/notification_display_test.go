package trips

import (
	"testing"
	"time"
)

func TestShortenFlightNotificationBody(t *testing.T) {
	in := "Singapore Airlines SQ 637 — departs Thu Apr 2, 14:34 (Tokyo International Airport, Tokyo, 144-0041, Japan → Kempegowda International Airport, KIAL T2 Exit Road, Bengaluru, 560300, India)"
	want := "Singapore Airlines SQ 637 — departs Thu Apr 2, 14:34 (Tokyo International Airport → Kempegowda International Airport)"
	if got := ShortenFlightNotificationBody(in); got != want {
		t.Fatalf("got %q want %q", got, want)
	}
	if got := ShortenFlightNotificationBody("No flight here"); got != "No flight here" {
		t.Fatalf("passthrough: got %q", got)
	}
	newFmt := "✈️ Flight SQ 637 | Departs at 14:34 > Tokyo (HND) → Bengaluru (BLR)"
	if got := ShortenFlightNotificationBody(newFmt); got != newFmt {
		t.Fatalf("emoji line should pass through: got %q", got)
	}
}

func TestFormatFlightReminderNotificationBody(t *testing.T) {
	loc := time.Local
	dep := time.Date(2026, 4, 2, 14, 34, 0, 0, loc)
	trip := Trip{UITimeFormat: "24h"}
	f := Flight{
		FlightNumber:  "SQ 637",
		DepartAirport: "Tokyo International Airport (HND), Tokyo",
		ArriveAirport: "Kempegowda International Airport (BLR), Bengaluru",
	}
	got := formatFlightReminderNotificationBody(trip, f, dep)
	want := "✈️ Flight SQ 637 | Departs at 14:34 > Tokyo International Airport (HND) → Kempegowda International Airport (BLR)"
	if got != want {
		t.Fatalf("24h: got %q want %q", got, want)
	}
	trip.UITimeFormat = "12h"
	got2 := formatFlightReminderNotificationBody(trip, f, dep)
	if got2 != "✈️ Flight SQ 637 | Departs at 2:34 PM > Tokyo International Airport (HND) → Kempegowda International Airport (BLR)" {
		t.Fatalf("12h: got %q", got2)
	}
}

func TestAppNotificationTitleWithTrip(t *testing.T) {
	if got := AppNotificationTitleWithTrip("Tokyo Spring", "Accommodation Check-In"); got != "Tokyo Spring: Accommodation Check-In" {
		t.Fatalf("got %q", got)
	}
	if got := AppNotificationTitleWithTrip("", "Only title"); got != "Only title" {
		t.Fatalf("empty trip name: got %q", got)
	}
	if got := AppNotificationTitleWithTrip("Trip", ""); got != "Trip" {
		t.Fatalf("empty title: got %q", got)
	}
}

func TestFormatAirportForNotificationLine(t *testing.T) {
	if got := formatAirportForNotificationLine("Narita (NRT), Chiba"); got != "Narita (NRT)" {
		t.Fatalf("got %q", got)
	}
	if got := formatAirportForNotificationLine("Kempegowda International Airport, Bengaluru"); got != "Kempegowda International Airport" {
		t.Fatalf("no code: got %q", got)
	}
}
