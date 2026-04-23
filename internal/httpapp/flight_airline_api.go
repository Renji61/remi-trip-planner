package httpapp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"remi-trip-planner/internal/airlabs"
	"remi-trip-planner/internal/trips"
)

// flightAirlineSuggestion is JSON for the flight airline autocomplete (mirrors location/airport shape for shared UI).
type flightAirlineSuggestion struct {
	DisplayName      string `json:"displayName"`
	ShortName        string `json:"shortName"`
	PrimaryLine      string `json:"primaryLine"`
	SecondaryLine    string `json:"secondaryLine"`
	IATACode         string `json:"iataCode"`
	ICAOCode         string `json:"icaoCode"`
	Country          string `json:"country"`
	Source           string `json:"source"`
	FormattedAddress string `json:"formattedAddress"`
}

func (a *app) apiFlightAirlineSuggest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len([]rune(q)) < 2 {
		_ = json.NewEncoder(w).Encode([]flightAirlineSuggestion{})
		return
	}

	appSettings, err := a.tripService.GetAppSettings(r.Context())
	if err != nil {
		_ = json.NewEncoder(w).Encode([]flightAirlineSuggestion{})
		return
	}

	var out []flightAirlineSuggestion
	if !trips.IsFlightAPIActive(appSettings) {
		_ = json.NewEncoder(w).Encode(out)
		return
	}

	rows, status, aerr := airlabs.SuggestAirlines(r.Context(), nil, appSettings.AirLabsAPIKey, q)
	if aerr != nil {
		if status == http.StatusUnauthorized || status == http.StatusTooManyRequests {
			slog.WarnContext(r.Context(), "airlabs_airline_search",
				slog.Int("http_status", status),
				slog.String("err", aerr.Error()),
				slog.String("query", q),
			)
			w.Header().Set("X-Flight-Airline-Search-Manual", "1")
		} else {
			slog.WarnContext(r.Context(), "airlabs_airline_search",
				slog.Int("http_status", status),
				slog.String("err", aerr.Error()),
				slog.String("query", q),
			)
		}
		_ = json.NewEncoder(w).Encode(out)
		return
	}

	for _, row := range rows {
		if len(out) >= 10 {
			break
		}
		iata := strings.ToUpper(strings.TrimSpace(row.IATACode))
		name := strings.TrimSpace(row.Name)
		if iata == "" || name == "" {
			continue
		}
		line := name + " - " + iata
		cc := strings.TrimSpace(row.CountryCode)
		sec := strings.TrimSpace(row.ICAOCode)
		if cc != "" {
			if sec != "" {
				sec = sec + " · " + cc
			} else {
				sec = cc
			}
		}
		out = append(out, flightAirlineSuggestion{
			DisplayName:      name,
			ShortName:        name,
			PrimaryLine:      line,
			SecondaryLine:    sec,
			IATACode:         iata,
			ICAOCode:         strings.TrimSpace(row.ICAOCode),
			Country:          cc,
			Source:           "airlabs",
			FormattedAddress: sec,
		})
	}

	if len(out) > 0 {
		w.Header().Set("Cache-Control", "private, max-age=60")
	}
	_ = json.NewEncoder(w).Encode(out)
}
