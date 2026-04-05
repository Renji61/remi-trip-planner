package trips

import (
	"testing"
	"time"
)

func TestTripEligibleForInAppNotifications(t *testing.T) {
	loc := time.Local
	mid := time.Date(2026, 4, 10, 12, 0, 0, 0, loc)

	trip := Trip{
		StartDate: "2026-04-01",
		EndDate:   "2026-04-20",
	}
	if !TripEligibleForInAppNotifications(trip, mid) {
		t.Fatal("expected in-window trip to be eligible")
	}
	if TripEligibleForInAppNotifications(Trip{StartDate: "2026-04-01", EndDate: "2026-04-20", IsArchived: true}, mid) {
		t.Fatal("archived trip must not be eligible")
	}
	if TripEligibleForInAppNotifications(Trip{StartDate: "", EndDate: ""}, mid) {
		t.Fatal("draft (no dates) must not be eligible")
	}
	before := time.Date(2026, 3, 31, 12, 0, 0, 0, loc)
	if TripEligibleForInAppNotifications(trip, before) {
		t.Fatal("upcoming trip must not be eligible")
	}
	after := time.Date(2026, 4, 21, 12, 0, 0, 0, loc)
	if TripEligibleForInAppNotifications(trip, after) {
		t.Fatal("completed trip must not be eligible")
	}
	onEnd := time.Date(2026, 4, 20, 23, 0, 0, 0, loc)
	if !TripEligibleForInAppNotifications(trip, onEnd) {
		t.Fatal("last day of trip should still be in progress")
	}
}
