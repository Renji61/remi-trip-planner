package trips

import "strings"

func trimLabelBeforeComma(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, ","); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}

func isValidIATACode(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != 3 {
		return false
	}
	for _, c := range strings.ToUpper(s) {
		if c < 'A' || c > 'Z' {
			return false
		}
	}
	return true
}

func airportNameFromLookup(iata string, byIATA map[string]Airport) string {
	if byIATA == nil {
		return ""
	}
	a, ok := byIATA[iata]
	if !ok {
		return ""
	}
	name := strings.TrimSpace(a.Name)
	if name != "" {
		return name
	}
	return strings.TrimSpace(a.City)
}

// FormatFlightAirportHeadline returns "{airport name or label} - {IATA}" for itinerary
// and flight cards. When IATA is missing or invalid, falls back to a compact line from display.
func FormatFlightAirportHeadline(display, iata string, byIATA map[string]Airport) string {
	display = strings.TrimSpace(display)
	iata = strings.TrimSpace(iata)
	if !isValidIATACode(iata) {
		if display != "" {
			return trimLabelBeforeComma(display)
		}
		return ""
	}
	ui := strings.ToUpper(iata)
	suf := " - " + ui
	du := strings.ToUpper(display)

	if display != "" && strings.HasSuffix(du, strings.ToUpper(suf)) {
		return display
	}
	if display != "" && strings.EqualFold(display, ui) {
		if name := airportNameFromLookup(ui, byIATA); name != "" {
			return name + suf
		}
		return ui
	}
	if display == "" {
		if name := airportNameFromLookup(ui, byIATA); name != "" {
			return name + suf
		}
		return ui
	}
	return trimLabelBeforeComma(display) + suf
}
