package httpapp

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
)

func (a *app) geocodeForApp(ctx context.Context, query string) (lat, lng float64) {
	appSettings, err := a.tripService.GetAppSettings(ctx)
	key := ""
	if err == nil {
		key = strings.TrimSpace(appSettings.GoogleMapsAPIKey)
	}
	return geocodeCoords(ctx, query, key)
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
	var suggestions []locationSuggestion
	if strings.TrimSpace(app.GoogleMapsAPIKey) != "" {
		suggestions = googlePlaceSuggestions(r.Context(), q, strings.TrimSpace(app.GoogleMapsAPIKey), 5)
	} else {
		suggestions = nominatimSuggestions(r.Context(), q, 5)
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
	lat, lng := geocodeCoords(r.Context(), q, key)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"lat": lat, "lng": lng, "displayName": q,
	})
}
