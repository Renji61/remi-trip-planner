package trips

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// AppNotificationTitleWithTrip returns "{trip name}: {title}" for the notifications page and related UI.
func AppNotificationTitleWithTrip(tripName, title string) string {
	tn := strings.TrimSpace(tripName)
	tt := strings.TrimSpace(title)
	switch {
	case tn == "":
		return tt
	case tt == "":
		return tn
	default:
		return tn + ": " + tt
	}
}

// airportLineLabel returns the substring before the first comma, trimmed, for shorter in-app copy.
func airportLineLabel(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return s
	}
	if i := strings.IndexByte(s, ','); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

var iataInParensRE = regexp.MustCompile(`(?i)\(([A-Z]{3})\)`)

// formatAirportForNotificationLine builds "City (XXX)" when a 3-letter IATA-style code appears
// in parentheses anywhere in the stored airport string; otherwise the comma-first line / city label.
func formatAirportForNotificationLine(full string) string {
	full = strings.TrimSpace(full)
	if full == "" {
		return ""
	}
	head := airportLineLabel(full)
	city := head
	if idx := strings.IndexByte(head, '('); idx >= 0 {
		city = strings.TrimSpace(head[:idx])
	}
	var code string
	if m := iataInParensRE.FindStringSubmatch(full); len(m) == 2 {
		code = strings.ToUpper(m[1])
	}
	if code != "" {
		if city == "" {
			city = head
		}
		return city + " (" + code + ")"
	}
	if city != "" {
		return city
	}
	return head
}

// formatFlightReminderNotificationBody is the in-app flight reminder line (bell / notifications page).
func formatFlightReminderNotificationBody(trip Trip, f Flight, dep time.Time) string {
	fn := strings.TrimSpace(f.FlightNumber)
	if fn == "" {
		fn = strings.TrimSpace(f.FlightName)
	}
	if fn == "" {
		fn = "—"
	}
	clock := FormatTripDepartureClock(trip, dep)
	d := formatAirportForNotificationLine(f.DepartAirport)
	a := formatAirportForNotificationLine(f.ArriveAirport)
	return fmt.Sprintf("✈️ Flight %s | Departs at %s > %s → %s", fn, clock, d, a)
}

// ShortenFlightNotificationBody compacts departure/arrival airport text in legacy flight reminder bodies
// (pattern: " — departs … (dep → arr)"). Bodies already using the ✈️ line are left unchanged.
func ShortenFlightNotificationBody(body string) string {
	if strings.HasPrefix(strings.TrimSpace(body), "✈️") {
		return body
	}
	if !strings.Contains(body, " — departs ") {
		return body
	}
	arrow := strings.LastIndex(body, " → ")
	if arrow < 0 {
		return body
	}
	open := strings.LastIndex(body[:arrow], "(")
	if open < 0 {
		return body
	}
	closeRel := strings.IndexByte(body[arrow:], ')')
	if closeRel < 0 {
		return body
	}
	closeIdx := arrow + closeRel
	dep := strings.TrimSpace(body[open+1 : arrow])
	arr := strings.TrimSpace(body[arrow+len(" → ") : closeIdx])
	if dep == "" || arr == "" {
		return body
	}
	return body[:open+1] + airportLineLabel(dep) + " → " + airportLineLabel(arr) + body[closeIdx:]
}
