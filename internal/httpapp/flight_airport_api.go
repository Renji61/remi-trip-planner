package httpapp

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"remi-trip-planner/internal/airlabs"
	"remi-trip-planner/internal/trips"
)

// flightAirportSuggestion is JSON for the flight airport autocomplete (extends location fields).
type flightAirportSuggestion struct {
	Lat              float64 `json:"lat"`
	Lng              float64 `json:"lng"`
	DisplayName      string  `json:"displayName"`
	ShortName        string  `json:"shortName"`
	PlaceID          string  `json:"placeId"`
	PlaceName        string  `json:"placeName"`
	FormattedAddress string  `json:"formattedAddress"`
	IATACode         string  `json:"iataCode"`
	ICAOCode         string  `json:"icaoCode"`
	PrimaryLine      string  `json:"primaryLine"`
	SecondaryLine    string  `json:"secondaryLine"`
	Source           string  `json:"source"`
	Timezone         string  `json:"timezone"`
	Country          string  `json:"country"`
	CityName         string  `json:"cityName"`
}

func (a *app) apiFlightAirportSuggest(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < 2 {
		_ = json.NewEncoder(w).Encode([]flightAirportSuggestion{})
		return
	}
	lang := locationLangFromRequest(r)
	var out []flightAirportSuggestion
	seen := map[string]struct{}{}

	add := func(s flightAirportSuggestion) {
		key := strings.ToUpper(s.IATACode) + "\x00" + strings.ToLower(s.PlaceID) + "\x00" + strings.ToLower(s.DisplayName)
		if _, ok := seen[key]; ok {
			return
		}
		seen[key] = struct{}{}
		out = append(out, s)
	}

	local, _ := a.tripService.SearchAirportsByKeyword(r.Context(), q, 8)
	for _, ap := range local {
		city := strings.TrimSpace(ap.City)
		iata := strings.ToUpper(strings.TrimSpace(ap.IATACode))
		name := strings.TrimSpace(ap.Name)
		primary := primaryLineForAirport(city, iata, name)
		cc := strings.TrimSpace(ap.CountryCode)
		fmtAddr := cc
		if city != "" && cc != "" {
			fmtAddr = city + ", " + cc
		} else if city != "" {
			fmtAddr = city
		}
		add(flightAirportSuggestion{
			Lat:              0,
			Lng:              0,
			DisplayName:      name,
			ShortName:        name,
			PlaceID:          "",
			PlaceName:        name,
			FormattedAddress: fmtAddr,
			IATACode:         iata,
			ICAOCode:         strings.TrimSpace(ap.ICAOCode),
			PrimaryLine:      primary,
			SecondaryLine:    name,
			Source:           "cache",
			Timezone:         ap.Timezone,
			Country:          cc,
			CityName:         ap.City,
		})
	}

	appSettings, err := a.tripService.GetAppSettings(r.Context())
	if err != nil {
		_ = json.NewEncoder(w).Encode(out)
		return
	}

	if len(local) == 0 && trips.IsFlightAPIActive(appSettings) && len(out) < 10 {
		rows, status, aerr := airlabs.SuggestAirports(r.Context(), nil, appSettings.AirLabsAPIKey, q, lang)
		if aerr != nil {
			if status == http.StatusUnauthorized || status == http.StatusTooManyRequests {
				slog.WarnContext(r.Context(), "airlabs_airport_search",
					slog.Int("http_status", status),
					slog.String("err", aerr.Error()),
					slog.String("query", q),
				)
				w.Header().Set("X-Flight-Search-Manual", "1")
			} else {
				slog.WarnContext(r.Context(), "airlabs_airport_search",
					slog.Int("http_status", status),
					slog.String("err", aerr.Error()),
					slog.String("query", q),
				)
			}
		} else {
			for _, loc := range rows {
				if len(out) >= 10 {
					break
				}
				city := strings.TrimSpace(loc.City)
				iata := strings.ToUpper(strings.TrimSpace(loc.IATACode))
				name := strings.TrimSpace(loc.Name)
				if name == "" {
					name = iata
				}
				primary := primaryLineForAirport(city, iata, name)
				cc := strings.TrimSpace(loc.CountryCode)
				fmtAddr := cc
				if city != "" && cc != "" {
					fmtAddr = city + ", " + cc
				} else if city != "" {
					fmtAddr = city
				}
				_ = a.tripService.UpsertAirport(r.Context(), trips.Airport{
					IATACode:    iata,
					ICAOCode:    strings.TrimSpace(loc.ICAOCode),
					Name:        name,
					City:        city,
					CountryCode: cc,
					Timezone:    strings.TrimSpace(loc.Timezone),
				})
				add(flightAirportSuggestion{
					Lat:              loc.Lat,
					Lng:              loc.Lng,
					DisplayName:      name,
					ShortName:        name,
					PlaceID:          "",
					PlaceName:        name,
					FormattedAddress: fmtAddr,
					IATACode:         iata,
					ICAOCode:         strings.TrimSpace(loc.ICAOCode),
					PrimaryLine:      primary,
					SecondaryLine:    name,
					Source:           "airlabs",
					Timezone:         loc.Timezone,
					Country:          cc,
					CityName:         city,
				})
			}
		}
	}

	if !appSettings.EnableLocationLookup {
		if len(out) > 0 {
			w.Header().Set("Cache-Control", "private, max-age=60")
		}
		_ = json.NewEncoder(w).Encode(out)
		return
	}
	if len(out) < 10 {
		mapsN := 10 - len(out)
		if mapsN > 0 {
			var locs []locationSuggestion
			if strings.TrimSpace(appSettings.GoogleMapsAPIKey) != "" {
				locs = googlePlaceSuggestions(r.Context(), q, strings.TrimSpace(appSettings.GoogleMapsAPIKey), mapsN, lang)
			}
			if len(locs) == 0 {
				locs = nominatimSuggestions(r.Context(), q, mapsN, lang)
			}
			for _, s := range locs {
				if len(out) >= 10 {
					break
				}
				placeName := s.PlaceName
				if placeName == "" {
					placeName = s.ShortName
				}
				disp := s.DisplayName
				if disp == "" {
					disp = s.FormattedAddress
				}
				add(flightAirportSuggestion{
					Lat:              s.Lat,
					Lng:              s.Lng,
					DisplayName:      disp,
					ShortName:        s.ShortName,
					PlaceID:          s.PlaceID,
					PlaceName:        s.PlaceName,
					FormattedAddress: s.FormattedAddress,
					IATACode:         "",
					PrimaryLine:      placeName,
					SecondaryLine:    s.FormattedAddress,
					Source:           "maps",
					Timezone:         "",
					Country:          "",
					CityName:         "",
				})
			}
		}
	}

	if len(out) > 0 {
		w.Header().Set("Cache-Control", "private, max-age=60")
	}
	_ = json.NewEncoder(w).Encode(out)
}

func primaryLineForAirport(city, iata, name string) string {
	city = strings.TrimSpace(city)
	iata = strings.TrimSpace(iata)
	name = strings.TrimSpace(name)
	if city != "" && iata != "" {
		return city + " - " + iata
	}
	if name != "" && iata != "" {
		return name + " - " + iata
	}
	if name != "" {
		return name
	}
	return iata
}

type flightAirportRememberRequest struct {
	IATACode string `json:"iataCode"`
	ICAOCode string `json:"icaoCode"`
	Name     string `json:"name"`
	City     string `json:"city"`
	Country  string `json:"country"`
	Timezone string `json:"timezone"`
}

func (a *app) apiFlightAirportRemember(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	var body flightAirportRememberRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
		return
	}
	body.IATACode = strings.ToUpper(strings.TrimSpace(body.IATACode))
	if len(body.IATACode) != 3 {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
		return
	}
	err := a.tripService.UpsertAirport(r.Context(), trips.Airport{
		IATACode:    body.IATACode,
		ICAOCode:    strings.TrimSpace(body.ICAOCode),
		Name:        body.Name,
		City:        body.City,
		CountryCode: strings.TrimSpace(body.Country),
		Timezone:    body.Timezone,
	})
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}
