package trips

import (
	"strings"
	"time"
)

// TripScheduleBounds returns calendar start/end midnight in local time when both dates parse and end is on or after start.
func TripScheduleBounds(t Trip) (startD, endD time.Time, ok bool) {
	start, err1 := time.ParseInLocation("2006-01-02", strings.TrimSpace(t.StartDate), time.Local)
	end, err2 := time.ParseInLocation("2006-01-02", strings.TrimSpace(t.EndDate), time.Local)
	if err1 != nil || err2 != nil {
		return time.Time{}, time.Time{}, false
	}
	startD = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
	endD = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, time.Local)
	if endD.Before(startD) {
		return time.Time{}, time.Time{}, false
	}
	return startD, endD, true
}

// TripEligibleForInAppNotifications is true only for active (non-archived) trips whose calendar window includes today's local date — i.e. "in progress" on the dashboard, not draft, upcoming, or completed.
func TripEligibleForInAppNotifications(t Trip, now time.Time) bool {
	if t.IsArchived {
		return false
	}
	startD, endD, ok := TripScheduleBounds(t)
	if !ok {
		return false
	}
	nowD := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.Local)
	if nowD.Before(startD) || nowD.After(endD) {
		return false
	}
	return true
}
