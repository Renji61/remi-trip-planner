package trips

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ParseBookingStreamTime parses stored local datetimes used for itinerary/booking fields.
func ParseBookingStreamTime(s string) (time.Time, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}, fmt.Errorf("empty time")
	}
	layouts := []string{
		"2006-01-02T15:04",
		"2006-01-02T15:04:05",
		time.RFC3339,
		"2006-01-02T15:04:05Z07:00",
	}
	for _, layout := range layouts {
		if t, err := time.ParseInLocation(layout, s, time.Local); err == nil {
			return t, nil
		}
	}
	return time.Parse("2006-01-02T15:04", s)
}

// FormatFlightBlockDuration returns a short label like "6h 50m" between depart and arrive.
func FormatFlightBlockDuration(f Flight) string {
	d, err1 := ParseBookingStreamTime(f.DepartAt)
	a, err2 := ParseBookingStreamTime(f.ArriveAt)
	if err1 != nil || err2 != nil || !a.After(d) {
		return ""
	}
	mins := int(a.Sub(d).Round(time.Minute).Minutes())
	if mins < 0 {
		return ""
	}
	h := mins / 60
	m := mins % 60
	if h > 0 && m > 0 {
		return fmt.Sprintf("%dh %dm", h, m)
	}
	if h > 0 {
		return fmt.Sprintf("%dh", h)
	}
	return fmt.Sprintf("%dm", m)
}

// FormatVehicleRentalDurationLabel returns a short rental span label for the unified booking strip.
// Under 24 hours: "4 hrs", "1 hr", "45 min", "3 hrs 15 min" as needed.
// At or over 24 hours: "1 day", "2 days & 4 hrs", "1 day & 1 hr", "1 day & 45 min" (singular/plural for day/hr).
func FormatVehicleRentalDurationLabel(v VehicleRental) string {
	pick, e1 := ParseBookingStreamTime(v.PickUpAt)
	drop, e2 := ParseBookingStreamTime(v.DropOffAt)
	if e1 != nil || e2 != nil || !drop.After(pick) {
		return ""
	}
	d := drop.Sub(pick).Round(time.Minute)
	totalMins := int(d / time.Minute)
	if totalMins <= 0 {
		return ""
	}

	const dayMins = 24 * 60
	fullDays := totalMins / dayMins
	rem := totalMins % dayMins
	remH := rem / 60
	remMin := rem % 60

	dayWord := func(n int) string {
		if n == 1 {
			return "1 day"
		}
		return fmt.Sprintf("%d days", n)
	}
	hrWord := func(n int) string {
		if n == 1 {
			return "1 hr"
		}
		return fmt.Sprintf("%d hrs", n)
	}

	if fullDays == 0 {
		h := totalMins / 60
		m := totalMins % 60
		if h == 0 {
			if m == 1 {
				return "1 min"
			}
			return fmt.Sprintf("%d min", m)
		}
		if m == 0 {
			return hrWord(h)
		}
		return fmt.Sprintf("%s %d min", hrWord(h), m)
	}

	// At least one full 24h bucket (total duration ≥ 24h).
	if rem == 0 {
		return dayWord(fullDays)
	}
	var b strings.Builder
	b.WriteString(dayWord(fullDays))
	b.WriteString(" & ")
	if remH > 0 {
		b.WriteString(hrWord(remH))
		if remMin > 0 {
			b.WriteString(" ")
			if remMin == 1 {
				b.WriteString("1 min")
			} else {
				b.WriteString(fmt.Sprintf("%d min", remMin))
			}
		}
		return b.String()
	}
	if remMin == 1 {
		b.WriteString("1 min")
	} else {
		b.WriteString(fmt.Sprintf("%d min", remMin))
	}
	return b.String()
}

// FormatLodgingNightsLabel returns a short label like "2 nights" from check-in / check-out.
func FormatLodgingNightsLabel(l Lodging) string {
	a, e1 := ParseBookingStreamTime(l.CheckInAt)
	b, e2 := ParseBookingStreamTime(l.CheckOutAt)
	if e1 != nil || e2 != nil || !b.After(a) {
		return ""
	}
	// Hotel-style "nights" as whole-day span between local anchors.
	full := int(b.Sub(a).Hours() / 24)
	if full < 1 {
		full = 1
	}
	if full == 1 {
		return "1 night"
	}
	return fmt.Sprintf("%d nights", full)
}

// BookingStreamEntry is one row in the unified trip booking list (chronological).
type BookingStreamEntry struct {
	Kind          string // "lodging" | "vehicle" | "flight"
	Sort          time.Time
	BookingStatus string
	Lodging       *Lodging
	Vehicle       *VehicleRental
	Flight        *Flight
}

// BuildBookingStream merges and sorts lodgings, vehicles, and flights by the primary
// sort time (check-in, pick-up, departure respectively). Respects trip section visibility
// and per-section main-column hide flags.
func BuildBookingStream(trip Trip, lodgings []Lodging, vehicles []VehicleRental, flights []Flight) []BookingStreamEntry {
	hidden := parseCommaKeySet(trip.UIMainSectionHidden) // layout_order.go
	out := make([]BookingStreamEntry, 0, len(lodgings)+len(vehicles)+len(flights))

	if trip.UIShowStay && !hidden[MainSectionStay] {
		for i := range lodgings {
			l := &lodgings[i]
			ts, err := ParseBookingStreamTime(l.CheckInAt)
			if err != nil {
				ts = time.Unix(0, 0)
			}
			out = append(out, BookingStreamEntry{
				Kind:          "lodging",
				Sort:          ts,
				BookingStatus: NormalizeBookingStatus(l.BookingStatus),
				Lodging:       l,
			})
		}
	}
	if trip.UIShowVehicle && !hidden[MainSectionVehicle] {
		for i := range vehicles {
			v := &vehicles[i]
			ts, err := ParseBookingStreamTime(v.PickUpAt)
			if err != nil {
				ts = time.Unix(0, 0)
			}
			out = append(out, BookingStreamEntry{
				Kind:          "vehicle",
				Sort:          ts,
				BookingStatus: NormalizeBookingStatus(v.BookingStatus),
				Vehicle:       v,
			})
		}
	}
	if trip.UIShowFlights && !hidden[MainSectionFlights] {
		for i := range flights {
			f := &flights[i]
			ts, err := ParseBookingStreamTime(f.DepartAt)
			if err != nil {
				ts = time.Unix(0, 0)
			}
			out = append(out, BookingStreamEntry{
				Kind:          "flight",
				Sort:          ts,
				BookingStatus: NormalizeBookingStatus(f.BookingStatus),
				Flight:        f,
			})
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Sort.Equal(out[j].Sort) {
			return bookingStreamTieBreak(&out[i], &out[j])
		}
		return out[i].Sort.Before(out[j].Sort)
	})
	return out
}

func bookingStreamTieBreak(a, b *BookingStreamEntry) bool {
	if a.Kind != b.Kind {
		return a.Kind < b.Kind
	}
	switch a.Kind {
	case "lodging":
		return a.Lodging.ID < b.Lodging.ID
	case "vehicle":
		return a.Vehicle.ID < b.Vehicle.ID
	case "flight":
		return a.Flight.ID < b.Flight.ID
	}
	return false
}
