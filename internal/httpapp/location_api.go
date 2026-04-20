package httpapp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

func (a *app) geocodeForApp(ctx context.Context, query string) (lat, lng float64) {
	appSettings, err := a.tripService.GetAppSettings(ctx)
	key := ""
	if err == nil {
		key = strings.TrimSpace(appSettings.GoogleMapsAPIKey)
	}
	return geocodeCoords(ctx, query, key, "en")
}

func looksLikeLanguageTag(s string) bool {
	if len(s) < 2 || len(s) > 12 {
		return false
	}
	for _, c := range s {
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' {
			continue
		}
		return false
	}
	return true
}

func preferredLocationLanguage(acceptLanguage string) string {
	s := strings.TrimSpace(acceptLanguage)
	if s == "" {
		return "en"
	}
	part := strings.TrimSpace(strings.Split(s, ",")[0])
	part = strings.TrimSpace(strings.Split(part, ";")[0])
	part = strings.ToLower(part)
	if part == "" || !looksLikeLanguageTag(part) {
		return "en"
	}
	if len(part) > 12 {
		part = part[:12]
	}
	return part
}

// locationLangFromRequest picks a BCP 47 language for geocoder responses: ?lang= when valid, else Accept-Language, else en.
func locationLangFromRequest(r *http.Request) string {
	override := strings.TrimSpace(r.URL.Query().Get("lang"))
	if looksLikeLanguageTag(override) {
		return strings.ToLower(override)
	}
	return preferredLocationLanguage(r.Header.Get("Accept-Language"))
}

// apiLocationSuggest returns JSON suggestions for location autocomplete (same shape as legacy client Nominatim parsing).
func (a *app) apiLocationSuggest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < 3 {
		_ = json.NewEncoder(w).Encode([]locationSuggestion{})
		return
	}
	app, err := a.tripService.GetAppSettings(r.Context())
	if err != nil || !app.EnableLocationLookup {
		_ = json.NewEncoder(w).Encode([]locationSuggestion{})
		return
	}
	lang := locationLangFromRequest(r)
	var suggestions []locationSuggestion
	if strings.TrimSpace(app.GoogleMapsAPIKey) != "" {
		suggestions = googlePlaceSuggestions(r.Context(), q, strings.TrimSpace(app.GoogleMapsAPIKey), 5, lang)
		if len(suggestions) == 0 {
			suggestions = nominatimSuggestions(r.Context(), q, 5, lang)
		}
	} else {
		suggestions = nominatimSuggestions(r.Context(), q, 5, lang)
	}
	if len(suggestions) > 0 {
		w.Header().Set("Cache-Control", "private, max-age=180")
	}
	_ = json.NewEncoder(w).Encode(suggestions)
}

// apiLocationGeocode returns a single best coordinate match as JSON {lat, lng, displayName}.
func (a *app) apiLocationGeocode(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		_ = json.NewEncoder(w).Encode(map[string]any{"lat": 0, "lng": 0, "displayName": ""})
		return
	}
	app, err := a.tripService.GetAppSettings(r.Context())
	if err != nil || !app.EnableLocationLookup {
		_ = json.NewEncoder(w).Encode(map[string]any{"lat": 0, "lng": 0, "displayName": ""})
		return
	}
	key := strings.TrimSpace(app.GoogleMapsAPIKey)
	lang := locationLangFromRequest(r)
	lat, lng := geocodeCoords(r.Context(), q, key, lang)
	if lat != 0 || lng != 0 {
		w.Header().Set("Cache-Control", "private, max-age=600")
	}
	_ = json.NewEncoder(w).Encode(map[string]any{
		"lat": lat, "lng": lng, "displayName": q,
	})
}

// apiLocationPlaceDetails returns coordinates, formatted address, name, and opening_hours for a Google place_id.
func (a *app) apiLocationPlaceDetails(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	placeID := strings.TrimSpace(r.URL.Query().Get("place_id"))
	if placeID == "" {
		http.Error(w, "place_id required", http.StatusBadRequest)
		return
	}
	app, err := a.tripService.GetAppSettings(r.Context())
	if err != nil || !app.EnableLocationLookup || strings.TrimSpace(app.GoogleMapsAPIKey) == "" {
		http.Error(w, "location lookup unavailable", http.StatusBadRequest)
		return
	}
	lang := locationLangFromRequest(r)
	client := &http.Client{Timeout: 12 * time.Second}
	d := fetchGooglePlaceDetails(r.Context(), placeID, strings.TrimSpace(app.GoogleMapsAPIKey), client, lang)
	if d.FormattedAddress == "" && d.Lat == 0 && d.Lng == 0 {
		http.Error(w, "place not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Cache-Control", "private, max-age=600")
	_ = json.NewEncoder(w).Encode(d)
}
