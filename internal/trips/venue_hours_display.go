package trips

import (
	"encoding/json"
	"regexp"
	"strings"
	"time"
)

const openingHoursUnavailable = "Opening hours not available."

type venueHoursPayload struct {
	Date             string `json:"date"`
	GooglePlaceID    string `json:"google_place_id"`
	Summary          string `json:"summary"`
	OpenMins         *int   `json:"open_mins,omitempty"`
	CloseMins        *int   `json:"close_mins,omitempty"`
	Status           string `json:"status,omitempty"`
	UserOpeningHours string `json:"user_opening_hours,omitempty"`
}

// VenueHoursSummary returns the legacy one-line snapshot from JSON (for diagnostics / old callers).
func (i ItineraryItem) VenueHoursSummary() string {
	s := strings.TrimSpace(i.VenueHoursJSON)
	if s == "" {
		return ""
	}
	var v struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return ""
	}
	return strings.TrimSpace(v.Summary)
}

// MergeOpeningHoursUserInput merges the plain-text "opening_hours" form field into venue_hours_json.
func MergeOpeningHoursUserInput(venueJSON, openingHours string) string {
	openingHours = strings.TrimSpace(openingHours)
	raw := strings.TrimSpace(venueJSON)
	var m map[string]any
	if raw != "" {
		_ = json.Unmarshal([]byte(raw), &m)
	}
	if m == nil {
		m = map[string]any{}
	}
	if openingHours == "" {
		delete(m, "user_opening_hours")
	} else {
		m["user_opening_hours"] = openingHours
	}
	out, err := json.Marshal(m)
	if err != nil {
		return raw
	}
	return string(out)
}

// OpeningHoursFormValue is the value for the Opening Hours text field (edit / add): user text, else computed line.
func (i ItineraryItem) OpeningHoursFormValue(trip Trip) string {
	if s := openingHoursUserOverride(i.VenueHoursJSON); s != "" {
		return s
	}
	return openingHoursComputedLine(trip, i.VenueHoursJSON)
}

// OpeningHoursCardPrimary is the bold primary line shown on the itinerary stop card.
func (i ItineraryItem) OpeningHoursCardPrimary(trip Trip) string {
	if s := openingHoursUserOverride(i.VenueHoursJSON); s != "" {
		return s
	}
	line := openingHoursComputedLine(trip, i.VenueHoursJSON)
	if strings.TrimSpace(line) == "" {
		return openingHoursUnavailable
	}
	return line
}

func openingHoursUserOverride(venueJSON string) string {
	s := strings.TrimSpace(venueJSON)
	if s == "" {
		return ""
	}
	var v venueHoursPayload
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return ""
	}
	return strings.TrimSpace(v.UserOpeningHours)
}

func openingHoursComputedLine(trip Trip, venueJSON string) string {
	s := strings.TrimSpace(venueJSON)
	if s == "" {
		return ""
	}
	var v venueHoursPayload
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return ""
	}
	if v.OpenMins != nil && v.CloseMins != nil {
		return formatOpeningHoursRange(trip, *v.OpenMins, *v.CloseMins)
	}
	if strings.TrimSpace(v.Summary) != "" {
		if line := legacyOpeningHoursTimesFromSummary(v.Summary); line != "" {
			return line
		}
		sum := strings.TrimSpace(v.Summary)
		if !strings.Contains(sum, "🔴") && !strings.Contains(strings.ToLower(sum), "no hours") {
			return sum
		}
	}
	switch strings.TrimSpace(strings.ToLower(v.Status)) {
	case "closed", "unavailable":
		return ""
	}
	return ""
}

func formatOpeningHoursRange(trip Trip, openMins, closeMins int) string {
	if openMins < 0 || closeMins < 0 || closeMins <= openMins {
		return ""
	}
	o := time.Date(2000, 1, 1, (openMins/60)%24, openMins%60, 0, 0, time.Local)
	c := time.Date(2000, 1, 1, (closeMins/60)%24, closeMins%60, 0, 0, time.Local)
	a := FormatTripDepartureClock(trip, o)
	b := FormatTripDepartureClock(trip, c)
	return a + " – " + b
}

var legacyOpenSummaryRE = regexp.MustCompile(`(?i)\bopen:\s*(.+)$`)

// legacyOpeningHoursTimesFromSummary strips emoji/marketing prefixes from saved JS summaries.
func legacyOpeningHoursTimesFromSummary(summary string) string {
	s := strings.TrimSpace(summary)
	if s == "" {
		return ""
	}
	lo := strings.ToLower(s)
	if strings.Contains(lo, "no hours") || strings.Contains(lo, "unavailable") {
		return ""
	}
	if strings.Contains(lo, "closed on this day") || strings.HasPrefix(lo, "🔴 closed") {
		return ""
	}
	if m := legacyOpenSummaryRE.FindStringSubmatch(s); len(m) > 1 {
		return strings.TrimSpace(m[1])
	}
	// Strip leading emoji / bullet noise
	s = strings.TrimLeft(s, "🟢🔴 \t")
	if strings.HasPrefix(strings.ToLower(s), "open:") {
		return strings.TrimSpace(s[5:])
	}
	return ""
}
