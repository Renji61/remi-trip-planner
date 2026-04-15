package httpapp

import (
	"fmt"
	"math"
	"net/url"
	"path/filepath"
	"strings"
)

// googleMapsSearchURL builds a Google Maps search URL: coordinates when valid,
// otherwise the trimmed hint string.
func googleMapsDirectionsURL(fromLat, fromLng, toLat, toLng float64, transportMode string) string {
	has := func(lat, lng float64) bool {
		return !math.IsNaN(lat) && !math.IsNaN(lng) && (math.Abs(lat) > 1e-7 || math.Abs(lng) > 1e-7)
	}
	if !has(fromLat, fromLng) || !has(toLat, toLng) {
		return ""
	}
	tm := strings.TrimSpace(strings.ToLower(transportMode))
	mode := "driving"
	switch tm {
	case "walk", "walking":
		mode = "walking"
	case "transit", "train", "bus", "ferry":
		mode = "transit"
	case "bicycle", "cycling":
		mode = "bicycling"
	case "drive", "driving", "taxi", "car", "":
		mode = "driving"
	default:
		mode = "driving"
	}
	o := fmt.Sprintf("%g,%g", fromLat, fromLng)
	d := fmt.Sprintf("%g,%g", toLat, toLng)
	return "https://www.google.com/maps/dir/?api=1&origin=" + url.QueryEscape(o) + "&destination=" + url.QueryEscape(d) + "&travelmode=" + url.QueryEscape(mode)
}

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

// itineraryNotesDisplay removes noisy attachment segments that are internal plumbing.
// buildLodgingCheckInNotes may append " | Attachment: /static/..." or, when nothing else
// is present, only "Attachment: /static/..." (no leading pipe).
func itineraryNotesDisplay(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	parts := strings.Split(s, " | ")
	var kept []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if strings.HasPrefix(strings.ToLower(p), "attachment:") {
			continue
		}
		kept = append(kept, p)
	}
	out := strings.TrimSpace(strings.Join(kept, " | "))
	if strings.Contains(out, "\n") {
		lines := strings.Split(out, "\n")
		var kl []string
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			if strings.HasPrefix(strings.ToLower(line), "attachment:") {
				continue
			}
			kl = append(kl, line)
		}
		out = strings.TrimSpace(strings.Join(kl, "\n"))
	}
	return out
}

// isImageWebPath reports whether a stored web path likely points to an image file.
func isImageWebPath(s string) bool {
	s = strings.TrimSpace(s)
	if s == "" {
		return false
	}
	if i := strings.IndexAny(s, "?#"); i >= 0 {
		s = s[:i]
	}
	switch strings.ToLower(filepath.Ext(s)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp", ".svg", ".avif":
		return true
	default:
		return false
	}
}
