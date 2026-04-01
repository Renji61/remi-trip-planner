package httpapp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// locationSuggestion is the JSON shape consumed by web/static/app.js location autocomplete.
type locationSuggestion struct {
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
	DisplayName string  `json:"displayName"`
	ShortName   string  `json:"shortName"`
}

// geocodeCoords resolves a free-text location to coordinates using Google Geocoding when
// googleAPIKey is non-empty, otherwise Nominatim. lang is a BCP 47 tag (e.g. "en") for result language.
func geocodeCoords(ctx context.Context, query, googleAPIKey, lang string) (lat, lng float64) {
	q := strings.TrimSpace(query)
	if q == "" {
		return 0, 0
	}
	key := strings.TrimSpace(googleAPIKey)
	lang = strings.TrimSpace(lang)
	if lang == "" {
		lang = "en"
	}
	resolveOne := func(candidate string) (float64, float64) {
		c := strings.TrimSpace(candidate)
		if c == "" {
			return 0, 0
		}
		if key != "" {
			if lat, lng := geocodeGoogle(ctx, c, key, lang); lat != 0 || lng != 0 {
				return lat, lng
			}
		}
		return geocodeNominatim(ctx, c, lang)
	}
	if lat, lng := resolveOne(q); lat != 0 || lng != 0 {
		return lat, lng
	}
	retry := normalizeGeocodeRetryQuery(q)
	if retry != "" && retry != q {
		return resolveOne(retry)
	}
	return 0, 0
}

func normalizeGeocodeRetryQuery(query string) string {
	q := strings.TrimSpace(query)
	if q == "" {
		return ""
	}
	// Normalize commonly pasted address strings before one retry.
	q = strings.Join(strings.Fields(q), " ")
	q = strings.ReplaceAll(q, ";", ",")
	q = strings.ReplaceAll(q, "|", ",")
	q = strings.TrimSpace(strings.TrimRight(q, ","))
	return q
}

func geocodeNominatim(ctx context.Context, query, lang string) (lat, lng float64) {
	if strings.TrimSpace(lang) == "" {
		lang = "en"
	}
	cacheKey := globalMapsLocationCache.geoKeyNominatim(lang, query)
	if la, ln, hit, ok := globalMapsLocationCache.geoGet(cacheKey); ok {
		if hit {
			return la, ln
		}
		return 0, 0
	}
	reqURL := fmt.Sprintf(
		"https://nominatim.openstreetmap.org/search?q=%s&format=jsonv2&limit=1",
		url.QueryEscape(query),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, 0
	}
	req.Header.Set("User-Agent", "REMI-Trip-Planner/1.0")
	req.Header.Set("Accept-Language", lang)
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0
	}
	defer resp.Body.Close()
	var results []struct {
		Lat string `json:"lat"`
		Lon string `json:"lon"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil {
		return 0, 0
	}
	if len(results) == 0 {
		globalMapsLocationCache.geoSetNegative(cacheKey, geocodeGoogleMissTTL)
		return 0, 0
	}
	lat, _ = strconv.ParseFloat(results[0].Lat, 64)
	lng, _ = strconv.ParseFloat(results[0].Lon, 64)
	if lat == 0 && lng == 0 {
		globalMapsLocationCache.geoSetNegative(cacheKey, geocodeGoogleMissTTL)
	} else {
		globalMapsLocationCache.geoSetCoords(cacheKey, lat, lng, geocodeNominatimTTL)
	}
	return lat, lng
}

func geocodeGoogle(ctx context.Context, address, apiKey, lang string) (lat, lng float64) {
	if strings.TrimSpace(lang) == "" {
		lang = "en"
	}
	tag := mapsAPIKeyTagFrom(apiKey)
	cacheKey := globalMapsLocationCache.geoKeyGoogle(tag, lang, address)
	if la, ln, hit, ok := globalMapsLocationCache.geoGet(cacheKey); ok {
		if hit {
			return la, ln
		}
		return 0, 0
	}
	u := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/geocode/json?address=%s&key=%s",
		url.QueryEscape(address),
		url.QueryEscape(apiKey),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, 0
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0
	}
	defer resp.Body.Close()
	var payload struct {
		Status  string `json:"status"`
		Results []struct {
			Geometry struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		} `json:"results"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return 0, 0
	}
	if payload.Status != "OK" && payload.Status != "ZERO_RESULTS" {
		globalMapsLocationCache.geoSetNegative(cacheKey, geocodeGoogleMissTTL)
		return 0, 0
	}
	if len(payload.Results) == 0 {
		globalMapsLocationCache.geoSetNegative(cacheKey, geocodeGoogleMissTTL)
		return 0, 0
	}
	loc := payload.Results[0].Geometry.Location
	globalMapsLocationCache.geoSetCoords(cacheKey, loc.Lat, loc.Lng, geocodeGoogleSuccessTTL)
	return loc.Lat, loc.Lng
}

func nominatimSuggestions(ctx context.Context, query string, limit int, lang string) []locationSuggestion {
	if limit <= 0 {
		limit = 5
	}
	if strings.TrimSpace(lang) == "" {
		lang = "en"
	}
	sKey := globalMapsLocationCache.suggestKeyNominatim(lang, query)
	if cached, ok := globalMapsLocationCache.suggestGet(sKey); ok {
		if len(cached) <= limit {
			return cached
		}
		return cached[:limit]
	}
	reqURL := fmt.Sprintf(
		"https://nominatim.openstreetmap.org/search?q=%s&format=jsonv2&limit=%d",
		url.QueryEscape(strings.TrimSpace(query)),
		limit,
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil
	}
	req.Header.Set("User-Agent", "REMI-Trip-Planner/1.0")
	req.Header.Set("Accept-Language", lang)
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var data []struct {
		Lat         string `json:"lat"`
		Lon         string `json:"lon"`
		DisplayName string `json:"display_name"`
		Name        string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil
	}
	var out []locationSuggestion
	for _, item := range data {
		lat, _ := strconv.ParseFloat(item.Lat, 64)
		lng, _ := strconv.ParseFloat(item.Lon, 64)
		displayName := strings.TrimSpace(item.DisplayName)
		name := strings.TrimSpace(item.Name)
		// Prefer display_name (follows Accept-Language) over OSM's primary name (often endonym / local script).
		shortName := ""
		if displayName != "" {
			shortName = strings.TrimSpace(strings.Split(displayName, ",")[0])
		}
		if shortName == "" {
			shortName = name
		}
		if shortName == "" {
			shortName = displayName
		}
		if displayName == "" || (lat == 0 && lng == 0) {
			continue
		}
		out = append(out, locationSuggestion{
			Lat: lat, Lng: lng,
			DisplayName: displayName,
			ShortName:   shortName,
		})
	}
	globalMapsLocationCache.suggestSet(sKey, out, nominatimSuggestTTL)
	return out
}

func googlePlaceSuggestions(ctx context.Context, input, apiKey string, limit int, lang string) []locationSuggestion {
	if limit <= 0 {
		limit = 5
	}
	if strings.TrimSpace(lang) == "" {
		lang = "en"
	}
	tag := mapsAPIKeyTagFrom(apiKey)
	sKey := globalMapsLocationCache.suggestKeyGoogle(tag, lang, input)
	if cached, ok := globalMapsLocationCache.suggestGet(sKey); ok {
		if len(cached) <= limit {
			return cached
		}
		return cached[:limit]
	}
	autoURL := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/place/autocomplete/json?input=%s&key=%s",
		url.QueryEscape(strings.TrimSpace(input)),
		url.QueryEscape(apiKey),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, autoURL, nil)
	if err != nil {
		return nil
	}
	client := &http.Client{Timeout: 8 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil
	}
	defer resp.Body.Close()
	var auto struct {
		Status      string `json:"status"`
		Predictions []struct {
			PlaceID              string `json:"place_id"`
			StructuredFormatting struct {
				MainText string `json:"main_text"`
			} `json:"structured_formatting"`
			Description string `json:"description"`
		} `json:"predictions"`
		ErrorMessage string `json:"error_message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&auto); err != nil {
		return nil
	}
	if auto.Status != "OK" && auto.Status != "ZERO_RESULTS" {
		return nil
	}
	var out []locationSuggestion
	for _, p := range auto.Predictions {
		if len(out) >= limit {
			break
		}
		if strings.TrimSpace(p.PlaceID) == "" {
			continue
		}
		lat, lng, display := googlePlaceDetail(ctx, p.PlaceID, apiKey, client, lang)
		if display == "" {
			display = p.Description
		}
		shortName := strings.TrimSpace(p.StructuredFormatting.MainText)
		if shortName == "" && display != "" {
			shortName = strings.TrimSpace(strings.Split(display, ",")[0])
		}
		if shortName == "" {
			shortName = display
		}
		if display == "" {
			continue
		}
		out = append(out, locationSuggestion{
			Lat: lat, Lng: lng,
			DisplayName: display,
			ShortName:   shortName,
		})
	}
	globalMapsLocationCache.suggestSet(sKey, out, placeSuggestTTL)
	return out
}

func googlePlaceDetail(ctx context.Context, placeID, apiKey string, client *http.Client, lang string) (lat, lng float64, formatted string) {
	if strings.TrimSpace(lang) == "" {
		lang = "en"
	}
	tag := mapsAPIKeyTagFrom(apiKey)
	pKey := globalMapsLocationCache.placeKey(tag, placeID, lang)
	if la, ln, disp, ok := globalMapsLocationCache.placeGet(pKey); ok {
		return la, ln, disp
	}
	u := fmt.Sprintf(
		"https://maps.googleapis.com/maps/api/place/details/json?place_id=%s&fields=geometry%%2Flocation%%2Cformatted_address%%2Cname&key=%s&language=%s",
		url.QueryEscape(placeID),
		url.QueryEscape(apiKey),
		url.QueryEscape(lang),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return 0, 0, ""
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, ""
	}
	defer resp.Body.Close()
	var payload struct {
		Status string `json:"status"`
		Result struct {
			FormattedAddress string `json:"formatted_address"`
			Name             string `json:"name"`
			Geometry         struct {
				Location struct {
					Lat float64 `json:"lat"`
					Lng float64 `json:"lng"`
				} `json:"location"`
			} `json:"geometry"`
		} `json:"result"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil || payload.Status != "OK" {
		return 0, 0, ""
	}
	formatted = strings.TrimSpace(payload.Result.FormattedAddress)
	if formatted == "" {
		formatted = strings.TrimSpace(payload.Result.Name)
	}
	loc := payload.Result.Geometry.Location
	globalMapsLocationCache.placeSet(pKey, loc.Lat, loc.Lng, formatted, placeDetailTTL)
	return loc.Lat, loc.Lng, formatted
}
