package httpapp

import (
	"fmt"
	"math"
	"net/url"
	"strings"
)

// googleMapsSearchURL builds a Google Maps search URL: coordinates when valid,
// otherwise the trimmed hint string.
func googleMapsSearchURL(lat, lng float64, hint string) string {
	hint = strings.TrimSpace(hint)
	hasCoords := !math.IsNaN(lat) && !math.IsNaN(lng) && (math.Abs(lat) > 1e-7 || math.Abs(lng) > 1e-7)
	if hasCoords {
		q := fmt.Sprintf("%g,%g", lat, lng)
		return "https://www.google.com/maps/search/?api=1&query=" + url.QueryEscape(q)
	}
	if hint == "" {
		return ""
	}
	return "https://www.google.com/maps/search/?api=1&query=" + url.QueryEscape(hint)
}

// locationLineBeforeComma returns the substring before the first comma, trimmed,
// or the full string if there is no comma (for compact airport/address labels).
func locationLineBeforeComma(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(s, ","); i >= 0 {
		return strings.TrimSpace(s[:i])
	}
	return s
}
