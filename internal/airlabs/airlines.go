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
)

// AirlineRow is one airline from GET /airlines (AirLabs v9).
type AirlineRow struct {
	Name        string
	IATACode    string
	ICAOCode    string
	CountryCode string
}

// normalizeAirlineSuggestQuery returns a trimmed query or "" if too short for lookup.
func normalizeAirlineSuggestQuery(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	r := []rune(s)
	if len(r) < 2 {
		return ""
	}
	if len(r) > 40 {
		s = string(r[:40])
	}
	return s
}

func isTwoCharAirlineDesignator(s string) bool {
	s = strings.ToUpper(strings.TrimSpace(s))
	if len(s) != 2 {
		return false
	}
	for _, c := range s {
		if (c < 'A' || c > 'Z') && (c < '0' || c > '9') {
			return false
		}
	}
	return true
}

// SuggestAirlines queries GET /airlines with name= and/or iata_code= (two-character designator).
// For a 2-character query it tries iata_code first, then merges name= results (deduped by IATA).
func SuggestAirlines(ctx context.Context, hc *http.Client, apiKey, query string) ([]AirlineRow, int, error) {
	apiKey = strings.TrimSpace(apiKey)
	q := normalizeAirlineSuggestQuery(query)
	if apiKey == "" || q == "" {
		return nil, 0, nil
	}
	if hc == nil {
		hc = &http.Client{Timeout: 20 * time.Second}
	}

	seen := map[string]struct{}{}
	var out []AirlineRow
	var lastStatus int

	addRows := func(rows []AirlineRow, status int) {
		lastStatus = status
		for _, row := range rows {
			iata := strings.ToUpper(strings.TrimSpace(row.IATACode))
			if iata == "" || len(iata) < 2 {
				continue
			}
			name := strings.TrimSpace(row.Name)
			if name == "" {
				continue
			}
			if _, dup := seen[iata]; dup {
				continue
			}
			seen[iata] = struct{}{}
			out = append(out, AirlineRow{
				Name:        name,
				IATACode:    iata,
				ICAOCode:    strings.TrimSpace(row.ICAOCode),
				CountryCode: strings.TrimSpace(row.CountryCode),
			})
			if len(out) >= 15 {
				break
			}
		}
	}

	if isTwoCharAirlineDesignator(q) {
		uv := url.Values{}
		uv.Set("api_key", apiKey)
		uv.Set("iata_code", strings.ToUpper(strings.TrimSpace(q)))
		uv.Set("_fields", "name,iata_code,icao_code,country_code")
		rows, status, err := fetchAirlines(ctx, hc, uv)
		if err != nil {
			if status == http.StatusUnauthorized || status == http.StatusTooManyRequests {
				return nil, status, err
			}
		} else {
			addRows(rows, status)
		}
	}

	if len(out) < 15 {
		uv := url.Values{}
		uv.Set("api_key", apiKey)
		uv.Set("name", q)
		uv.Set("_fields", "name,iata_code,icao_code,country_code")
		rows, status, err := fetchAirlines(ctx, hc, uv)
		if err != nil {
			if status == http.StatusUnauthorized || status == http.StatusTooManyRequests {
				return nil, status, err
			}
			if len(out) == 0 {
				return nil, status, err
			}
		} else {
			addRows(rows, status)
		}
	}

	if lastStatus == 0 {
		lastStatus = http.StatusOK
	}
	return out, lastStatus, nil
}

func fetchAirlines(ctx context.Context, hc *http.Client, q url.Values) ([]AirlineRow, int, error) {
	u, err := url.Parse(defaultBaseURL + "/airlines")
	if err != nil {
		return nil, 0, err
	}
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
	rows, err := parseAirlinesResponse(body)
	return rows, status, err
}

type airlineJSON struct {
	Name        string `json:"name"`
	IataCode    string `json:"iata_code"`
	IcaoCode    string `json:"icao_code"`
	CountryCode string `json:"country_code"`
}

func parseAirlinesResponse(body []byte) ([]AirlineRow, error) {
	var wrapped struct {
		Response json.RawMessage `json:"response"`
	}
	raw := body
	if err := json.Unmarshal(body, &wrapped); err == nil && len(strings.TrimSpace(string(wrapped.Response))) > 0 && string(wrapped.Response) != "null" {
		raw = wrapped.Response
	}
	raw = []byte(strings.TrimSpace(string(raw)))
	if len(raw) == 0 || string(raw) == "null" {
		return nil, nil
	}
	var arr []airlineJSON
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("airlabs: parse airlines: %w", err)
	}
	var out []AirlineRow
	for _, d := range arr {
		iata := strings.ToUpper(strings.TrimSpace(d.IataCode))
		if iata == "" {
			continue
		}
		name := strings.TrimSpace(d.Name)
		if name == "" {
			continue
		}
		out = append(out, AirlineRow{
			Name:        name,
			IATACode:    iata,
			ICAOCode:    strings.TrimSpace(d.IcaoCode),
			CountryCode: strings.TrimSpace(d.CountryCode),
		})
	}
	return out, nil
}
