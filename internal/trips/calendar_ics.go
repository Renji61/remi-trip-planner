package trips

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	ics "github.com/arran4/golang-ical"
)

// CalendarFeedTokenHash returns SHA-256 hex of the plaintext feed token (matches sqlite hashToken).
func CalendarFeedTokenHash(plain string) string {
	h := sha256.Sum256([]byte(strings.TrimSpace(plain)))
	return hex.EncodeToString(h[:])
}

// RegenerateCalendarFeedToken replaces the secret feed token; returns plaintext once for the user to copy.
func (s *Service) RegenerateCalendarFeedToken(ctx context.Context, tripID, actorUserID string) (plain string, err error) {
	acc, err := s.TripAccess(ctx, tripID, actorUserID)
	if err != nil {
		return "", err
	}
	if !acc.IsOwner {
		return "", ErrTripAccessDenied
	}
	b := make([]byte, 20)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	plain = hex.EncodeToString(b)
	return plain, s.repo.UpsertCalendarFeedToken(ctx, tripID, CalendarFeedTokenHash(plain), actorUserID)
}

// HasCalendarFeedToken reports whether a subscription URL has been generated for the trip.
func (s *Service) HasCalendarFeedToken(ctx context.Context, tripID string) (bool, error) {
	_, ok, err := s.repo.GetCalendarFeedTokenHash(ctx, tripID)
	return ok, err
}

// CalendarFeedMatches returns true if plainToken matches the stored hash for the trip.
func (s *Service) CalendarFeedMatches(ctx context.Context, tripID, plainToken string) bool {
	stored, ok, err := s.repo.GetCalendarFeedTokenHash(ctx, tripID)
	if err != nil || !ok {
		return false
	}
	return strings.EqualFold(stored, CalendarFeedTokenHash(plainToken))
}

// BuildTripICSBytes builds a PUBLISH calendar for the trip (for subscribed Google/Apple calendars).
func (s *Service) BuildTripICSBytes(ctx context.Context, tripID string) ([]byte, error) {
	trip, err := s.repo.GetTrip(ctx, tripID)
	if err != nil {
		return nil, err
	}
	items, err := s.repo.ListItineraryItems(ctx, tripID)
	if err != nil {
		return nil, err
	}
	lodgings, err := s.repo.ListLodgings(ctx, tripID)
	if err != nil {
		return nil, err
	}
	flights, err := s.repo.ListFlights(ctx, tripID)
	if err != nil {
		return nil, err
	}
	vehicles, err := s.repo.ListVehicleRentals(ctx, tripID)
	if err != nil {
		return nil, err
	}

	cal := ics.NewCalendar()
	cal.SetMethod(ics.MethodPublish)
	cal.SetProductId("-//REMI Trip Planner//EN")
	cal.SetVersion("2.0")
	cal.SetName(trip.Name + " — REMI")

	addEvent := func(uid, summary, loc string, start, end time.Time) {
		ev := ics.NewEvent(uid)
		ev.SetDtStampTime(time.Now().UTC())
		ev.SetStartAt(start)
		ev.SetEndAt(end)
		ev.SetSummary(summary)
		if strings.TrimSpace(loc) != "" {
			ev.SetLocation(loc)
		}
		cal.AddVEvent(ev)
	}

	for _, it := range items {
		st, ok := ItineraryItemStartLocal(trip, it)
		if !ok {
			continue
		}
		en := st.Add(time.Hour)
		if et, ok2 := parseLocalTimeOnSameDay(st, it.EndTime); ok2 && !et.Before(st) {
			en = et
		}
		addEvent("remi-itin-"+it.ID, it.Title, it.Location, st, en)
	}
	for _, l := range lodgings {
		if ci, ok := parseLocalDateTime("2006-01-02T15:04", l.CheckInAt); ok {
			addEvent("remi-stay-in-"+l.ID, "Check-in: "+l.Name, l.Address, ci, ci.Add(30*time.Minute))
		}
		if co, ok := parseLocalDateTime("2006-01-02T15:04", l.CheckOutAt); ok {
			addEvent("remi-stay-out-"+l.ID, "Check-out: "+l.Name, l.Address, co, co.Add(30*time.Minute))
		}
	}
	for _, f := range flights {
		if dep, ok := parseLocalDateTime("2006-01-02T15:04", f.DepartAt); ok {
			end := dep.Add(90 * time.Minute)
			if arr, ok2 := parseLocalDateTime("2006-01-02T15:04", f.ArriveAt); ok2 && arr.After(dep) {
				end = arr
			}
			addEvent("remi-fl-dep-"+f.ID, "Flight: "+strings.TrimSpace(f.FlightName)+" "+strings.TrimSpace(f.FlightNumber), f.DepartAirport, dep, end)
		}
		if arr, ok := parseLocalDateTime("2006-01-02T15:04", f.ArriveAt); ok {
			addEvent("remi-fl-arr-"+f.ID, "Arrive: "+strings.TrimSpace(f.FlightName), f.ArriveAirport, arr.Add(-30*time.Minute), arr)
		}
	}
	for _, v := range vehicles {
		if pu, ok := parseLocalDateTime("2006-01-02T15:04", v.PickUpAt); ok {
			addEvent("remi-car-pu-"+v.ID, "Vehicle pick-up", v.PickUpLocation, pu, pu.Add(time.Hour))
		}
		if dr, ok := parseLocalDateTime("2006-01-02T15:04", v.DropOffAt); ok {
			loc := v.DropOffLocation
			if strings.TrimSpace(loc) == "" {
				loc = v.PickUpLocation
			}
			addEvent("remi-car-do-"+v.ID, "Vehicle drop-off", loc, dr, dr.Add(time.Hour))
		}
	}

	return []byte(cal.Serialize()), nil
}

func parseLocalTimeOnSameDay(day time.Time, hhmm string) (time.Time, bool) {
	hhmm = strings.TrimSpace(hhmm)
	if hhmm == "" {
		return time.Time{}, false
	}
	parts := strings.SplitN(hhmm, ":", 3)
	h, _ := parseTinyInt(parts, 0)
	m, _ := parseTinyInt(parts, 1)
	return time.Date(day.Year(), day.Month(), day.Day(), h, m, 0, 0, day.Location()), true
}

func parseTinyInt(parts []string, idx int) (int, bool) {
	if idx >= len(parts) {
		return 0, false
	}
	var v int
	_, err := fmt.Sscanf(parts[idx], "%d", &v)
	return v, err == nil
}
