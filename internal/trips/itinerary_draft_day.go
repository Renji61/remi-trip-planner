package trips

import (
	"strings"
	"time"
)

// minDraftItineraryDayNumber is the smallest day_number value used to encode a calendar
// date (YYYYMMDD) for trips without start/end bounds. It stays well above plausible
// relative "day 1..N" indices so sorting and queries stay coherent.
const minDraftItineraryDayNumber = 19000101

// IsDraftTripForDateBounds is true when the trip has no usable start/end window for
// constraining itinerary dates (missing or unparseable).
func IsDraftTripForDateBounds(startDate, endDate string) bool {
	s := strings.TrimSpace(startDate)
	e := strings.TrimSpace(endDate)
	if s == "" || e == "" {
		return true
	}
	_, err1 := time.Parse("2006-01-02", s)
	_, err2 := time.Parse("2006-01-02", e)
	return err1 != nil || err2 != nil
}

// DraftItineraryDayNumberFromDate encodes a calendar date as day_number when the trip
// has no fixed start/end dates (draft planning).
func DraftItineraryDayNumberFromDate(t time.Time) int {
	y, m, d := t.Date()
	return y*10000 + int(m)*100 + d
}

// ItineraryDayDateISO returns the calendar YYYY-MM-DD for an itinerary day_number,
// matching the trip page DateLabel (relative to trip start, or draft YYYYMMDD encoding).
func ItineraryDayDateISO(tripStartDate string, dayNumber int) (iso string, ok bool) {
	start := strings.TrimSpace(tripStartDate)
	if t, err := time.Parse("2006-01-02", start); err == nil {
		return t.AddDate(0, 0, dayNumber-1).Format("2006-01-02"), true
	}
	return CalendarDateFromDraftItineraryDayNumber(dayNumber)
}

// CalendarDateFromDraftItineraryDayNumber returns the YYYY-MM-DD for a draft-encoded
// day_number, or ok == false if the value is not in that encoding.
func CalendarDateFromDraftItineraryDayNumber(day int) (iso string, ok bool) {
	if day < minDraftItineraryDayNumber {
		return "", false
	}
	y := day / 10000
	mo := (day % 10000) / 100
	dd := day % 100
	if mo < 1 || mo > 12 || dd < 1 || dd > 31 {
		return "", false
	}
	t := time.Date(y, time.Month(mo), dd, 0, 0, 0, 0, time.Local)
	if t.Year() != y || int(t.Month()) != mo || t.Day() != dd {
		return "", false
	}
	return t.Format("2006-01-02"), true
}

// RelativeDayNumberInTripWindow maps a calendar day to a 1-based trip day index using
// local midnight boundaries. The calendar day is clamped to [startDate, endDate] inclusive
// so planning outside the window still lands on the nearest in-range day.
func RelativeDayNumberInTripWindow(startDate, endDate string, calendarDay time.Time) (int, error) {
	start, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(startDate), time.Local)
	if err != nil {
		return 0, err
	}
	end, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(endDate), time.Local)
	if err != nil {
		return 0, err
	}
	truncate := func(t time.Time) time.Time {
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
	}
	ds := truncate(start)
	de := truncate(end)
	d := truncate(calendarDay)
	if d.Before(ds) {
		d = ds
	}
	if d.After(de) {
		d = de
	}
	return int(d.Sub(ds).Hours()/24) + 1, nil
}
