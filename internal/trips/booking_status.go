package trips

import "strings"

// TripBookingsChecklistCategory is the checklist category for auto-generated flight booking tasks.
const TripBookingsChecklistCategory = "Trip Bookings"

// BookFlightChecklistPrefix is the start of new Trip Bookings checklist lines (before airline name).
const BookFlightChecklistPrefix = "Book: "

const bookFlightChecklistPrefixLegacy = "Book Flight: "

// BookFlightChecklistTitle is the checklist line for a flight still to be booked.
// Format: Book: {Airline} - {departure IATA} to {arrival IATA} (e.g. "Book: IndiGo - BLR to COK").
func BookFlightChecklistTitle(f Flight) string {
	airline := airlineLabelForBookChecklist(f.FlightName)
	dep := airportIATAOrPlaceholder(f.DepartAirportIATA)
	arr := airportIATAOrPlaceholder(f.ArriveAirportIATA)
	return BookFlightChecklistPrefix + airline + " - " + dep + " to " + arr
}

// airlineLabelForBookChecklist returns a display label for the airline, trimming a trailing
// " - {2 alnum}" airline designator (e.g. "IndiGo - 6E" → "IndiGo", "American Airlines - AA" → "American Airlines").
func airlineLabelForBookChecklist(flightName string) string {
	n := strings.TrimSpace(flightName)
	if n == "" {
		return "Flight"
	}
	for {
		i := strings.LastIndex(n, " - ")
		if i < 0 {
			return n
		}
		tail := strings.TrimSpace(n[i+3:])
		if len(tail) == 2 && isAirlineDesignatorToken(tail) {
			n = strings.TrimSpace(n[:i])
			continue
		}
		return n
	}
}

func isAirlineDesignatorToken(s string) bool {
	s = strings.ToUpper(strings.TrimSpace(s))
	if len(s) != 2 {
		return false
	}
	for _, c := range s {
		if (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

func airportIATAOrPlaceholder(s string) string {
	t := strings.ToUpper(strings.TrimSpace(s))
	if len(t) == 3 {
		ok := true
		for _, c := range t {
			if c < 'A' || c > 'Z' {
				ok = false
				break
			}
		}
		if ok {
			return t
		}
	}
	return "?"
}

// ParseBookFlightChecklistTitle returns the airline label from a Trip Bookings line, for sync when the checklist is edited.
// It accepts the legacy "Book Flight: {name}" form and the current "Book: {airline} - {DEP} to {ARR}" form.
func ParseBookFlightChecklistTitle(text string) (airline string, ok bool) {
	t := strings.TrimSpace(text)
	if strings.HasPrefix(t, bookFlightChecklistPrefixLegacy) {
		return strings.TrimSpace(strings.TrimPrefix(t, bookFlightChecklistPrefixLegacy)), true
	}
	if !strings.HasPrefix(t, BookFlightChecklistPrefix) {
		return "", false
	}
	rest := strings.TrimSpace(strings.TrimPrefix(t, BookFlightChecklistPrefix))
	if rest == "" {
		return "", false
	}
	toIdx := strings.LastIndex(rest, " to ")
	if toIdx < 0 {
		return "", false
	}
	arr := strings.TrimSpace(rest[toIdx+len(" to "):])
	if !isThreeLetterAirportIATA(arr) {
		return "", false
	}
	before := strings.TrimSpace(rest[:toIdx])
	depSep := strings.LastIndex(before, " - ")
	if depSep < 0 {
		return "", false
	}
	air := strings.TrimSpace(before[:depSep])
	dep := strings.TrimSpace(before[depSep+len(" - "):])
	if air == "" || !isThreeLetterAirportIATA(dep) {
		return "", false
	}
	return air, true
}

func isThreeLetterAirportIATA(s string) bool {
	s = strings.ToUpper(strings.TrimSpace(s))
	if len(s) != 3 {
		return false
	}
	for _, c := range s {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}

// Canonical booking-status values for Flights, Stay, and Vehicle (shared field name: booking_status).
const (
	BookingStatusToBeDone    = "to_be_done"
	BookingStatusDone        = "done"
	BookingStatusNotRequired = "not_required"
)

// NormalizeBookingStatus coerces form/API input to a stored token. Empty or unknown → to_be_done.
func NormalizeBookingStatus(s string) string {
	switch strings.TrimSpace(strings.ToLower(s)) {
	case BookingStatusDone:
		return BookingStatusDone
	case BookingStatusNotRequired:
		return BookingStatusNotRequired
	case BookingStatusToBeDone, "":
		return BookingStatusToBeDone
	default:
		return BookingStatusToBeDone
	}
}
