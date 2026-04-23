// Package airlabs calls the AirLabs Data API (v9) for airport and airline autocomplete and lookups.
package airlabs

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"unicode"
)

const defaultBaseURL = "https://airlabs.co/api/v9"

// AirportRow is one airport from AirLabs suggest or airports endpoints.
type AirportRow struct {
	IATACode    string
	ICAOCode    string
	Name        string
	City        string
	CountryCode string
	Timezone    string
	Lat, Lng    float64
}

// SuggestAirports calls GET /suggest?q= (AirLabs v9 airport autocomplete).
// The /airports endpoint only supports iata_code, icao_code, city_code, and country_code — not free-text search.
// On HTTP 401, 429, or API error payload, it returns (nil, statusCode, err) with err non-nil for 401/429.
func SuggestAirports(ctx context.Context, hc *http.Client, apiKey, query, lang string) ([]AirportRow, int, error) {
	apiKey = strings.TrimSpace(apiKey)
	q := normalizeSuggestQuery(query)
	if apiKey == "" || q == "" {
		return nil, 0, nil
	}
	if hc == nil {
		hc = &http.Client{Timeout: 20 * time.Second}
	}

	// Exact 3-letter IATA: use /airports (documented filter) first.
	if len(q) == 3 && isAllLetters(q) {
		rows, status, err := fetchAirportsByIATA(ctx, hc, apiKey, q)
		if err != nil {
			if status == http.StatusUnauthorized || status == http.StatusTooManyRequests {
				return nil, status, err
			}
			// Other errors (network, 5xx): try /suggest below.
		} else if len(rows) > 0 {
			return rows, status, nil
		}
	}

	u, err := url.Parse(defaultBaseURL + "/suggest")
	if err != nil {
		return nil, 0, err
	}
	uv := u.Query()
	uv.Set("api_key", apiKey)
	uv.Set("q", q)
	if t := suggestLangParam(lang); t != "" {
		uv.Set("lang", t)
	}
	u.RawQuery = uv.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	res, err := hc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	status := res.StatusCode
	if status == http.StatusUnauthorized || status == http.StatusTooManyRequests {
		return nil, status, fmt.Errorf("airlabs: status %d", status)
	}
	if status < 200 || status >= 300 {
		return nil, status, fmt.Errorf("airlabs: status %d: %s", status, strings.TrimSpace(string(body)))
	}
	if st, err := apiErrorFromBody(body); err != nil {
		if st == 0 {
			st = status
		}
		return nil, st, err
	}
	rows, err := parseSuggestResponse(body)
	if err != nil {
		return nil, status, err
	}
	return rows, status, nil
}

func normalizeSuggestQuery(s string) string {
	s = strings.TrimSpace(s)
	if len(s) < 3 {
		return ""
	}
	r := []rune(s)
	if len(r) > 30 {
		s = string(r[:30])
	}
	return s
}

func isAllLetters(s string) bool {
	for _, r := range s {
		if !unicode.IsLetter(r) {
			return false
		}
	}
	return true
}

func suggestLangParam(lang string) string {
	lang = strings.TrimSpace(strings.ToLower(lang))
	if lang == "" {
		return ""
	}
	base := lang
	if i := strings.IndexByte(lang, '-'); i > 0 {
		base = lang[:i]
	}
	if len(base) < 2 {
		return ""
	}
	two := base[:2]
	for _, r := range two {
		if r < 'a' || r > 'z' {
			return ""
		}
	}
	return two
}

func fetchAirportsByIATA(ctx context.Context, hc *http.Client, apiKey, iata string) ([]AirportRow, int, error) {
	iata = strings.ToUpper(strings.TrimSpace(iata))
	u, err := url.Parse(defaultBaseURL + "/airports")
	if err != nil {
		return nil, 0, err
	}
	q := u.Query()
	q.Set("api_key", strings.TrimSpace(apiKey))
	q.Set("iata_code", iata)
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Accept", "application/json")
	res, err := hc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	status := res.StatusCode
	if status == http.StatusUnauthorized || status == http.StatusTooManyRequests {
		return nil, status, fmt.Errorf("airlabs: status %d", status)
	}
	if status < 200 || status >= 300 {
		return nil, status, fmt.Errorf("airlabs: status %d: %s", status, strings.TrimSpace(string(body)))
	}
	if st, err := apiErrorFromBody(body); err != nil {
		if st == 0 {
			st = status
		}
		return nil, st, err
	}
	var wrapped struct {
		Response json.RawMessage `json:"response"`
	}
	if err := json.Unmarshal(body, &wrapped); err != nil {
		return nil, status, fmt.Errorf("airlabs: json: %w", err)
	}
	raw := wrapped.Response
	if len(strings.TrimSpace(string(raw))) == 0 || string(raw) == "null" {
		raw = body
	}
	rows, err := parseAirportListJSON(raw)
	return rows, status, err
}

// apiErrorFromBody returns (HTTP-style status, err) when the JSON payload is an API error; (0, nil) otherwise.
func apiErrorFromBody(body []byte) (int, error) {
	var errWrap struct {
		Error *struct {
			Message string `json:"message"`
			Code    string `json:"code"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &errWrap) != nil || errWrap.Error == nil {
		return 0, nil
	}
	code := strings.ToLower(strings.TrimSpace(errWrap.Error.Code))
	msg := strings.TrimSpace(errWrap.Error.Message)
	switch code {
	case "unknown_api_key", "expired_api_key":
		return http.StatusUnauthorized, fmt.Errorf("airlabs: %s", msg)
	default:
		if strings.Contains(strings.ToLower(msg), "rate") || strings.Contains(msg, "429") {
			return http.StatusTooManyRequests, fmt.Errorf("airlabs: %s", msg)
		}
		return 0, fmt.Errorf("airlabs: %s (%s)", msg, code)
	}
}

type suggestAirportJSON struct {
	Name        string  `json:"name"`
	IataCode    string  `json:"iata_code"`
	IcaoCode    string  `json:"icao_code"`
	City        string  `json:"city"`
	CountryCode string  `json:"country_code"`
	Timezone    string  `json:"timezone"`
	Lat         float64 `json:"lat"`
	Lng         float64 `json:"lng"`
}

type suggestPayload struct {
	Airports            []suggestAirportJSON `json:"airports"`
	AirportsByCities    []suggestAirportJSON `json:"airports_by_cities"`
	AirportsByCountries []suggestAirportJSON `json:"airports_by_countries"`
}

func parseSuggestResponse(body []byte) ([]AirportRow, error) {
	var wrapped struct {
		Response json.RawMessage `json:"response"`
	}
	inner := body
	if err := json.Unmarshal(body, &wrapped); err == nil && len(strings.TrimSpace(string(wrapped.Response))) > 0 && string(wrapped.Response) != "null" {
		inner = wrapped.Response
	}
	var obj suggestPayload
	if err := json.Unmarshal(inner, &obj); err != nil {
		return nil, fmt.Errorf("airlabs: parse suggest: %w", err)
	}
	seen := map[string]struct{}{}
	var out []AirportRow
	appendUnique := func(slice []suggestAirportJSON) {
		for _, d := range slice {
			row, ok := rowFromSuggestJSON(d)
			if !ok {
				continue
			}
			if _, dup := seen[row.IATACode]; dup {
				continue
			}
			seen[row.IATACode] = struct{}{}
			out = append(out, row)
		}
	}
	appendUnique(obj.Airports)
	appendUnique(obj.AirportsByCities)
	appendUnique(obj.AirportsByCountries)
	return out, nil
}

func rowFromSuggestJSON(d suggestAirportJSON) (AirportRow, bool) {
	iata := strings.ToUpper(strings.TrimSpace(d.IataCode))
	if len(iata) != 3 {
		return AirportRow{}, false
	}
	return AirportRow{
		IATACode:    iata,
		ICAOCode:    strings.TrimSpace(d.IcaoCode),
		Name:        strings.TrimSpace(d.Name),
		City:        strings.TrimSpace(d.City),
		CountryCode: strings.TrimSpace(d.CountryCode),
		Timezone:    strings.TrimSpace(d.Timezone),
		Lat:         d.Lat,
		Lng:         d.Lng,
	}, true
}

func parseAirportListJSON(raw []byte) ([]AirportRow, error) {
	return parseAirportJSONArray(raw)
}

func parseAirportJSONArray(raw []byte) ([]AirportRow, error) {
	raw = []byte(strings.TrimSpace(string(raw)))
	if len(raw) == 0 {
		return nil, nil
	}
	var arr []suggestAirportJSON
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("airlabs: parse airports: %w", err)
	}
	var out []AirportRow
	for _, d := range arr {
		if row, ok := rowFromSuggestJSON(d); ok {
			out = append(out, row)
		}
	}
	return out, nil
}
