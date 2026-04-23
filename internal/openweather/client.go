// Package openweather fetches day-level forecast data from OpenWeatherMap (forecast + optional One Call alerts).
package openweather

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

// DaySummary is a single-day temperature range, display icon, and optional government alert (One Call 3.0 only).
type DaySummary struct {
	HighC    float64
	LowC     float64
	Icon     string // emoji
	HasAlert bool
}

const forecastURL = "https://api.openweathermap.org/data/2.5/forecast"
const forecastDailyURL = "https://api.openweathermap.org/data/2.5/forecast/daily"
const oneCallURL = "https://api.openweathermap.org/data/3.0/onecall"

type forecastSlot struct {
	Dt   int64 `json:"dt"`
	Main struct {
		Temp    float64 `json:"temp"`
		TempMin float64 `json:"temp_min"`
		TempMax float64 `json:"temp_max"`
	} `json:"main"`
	Weather []struct {
		Main string `json:"main"`
	} `json:"weather"`
}

type forecastResponse struct {
	List []forecastSlot `json:"list"`
	City struct {
		Timezone int `json:"timezone"`
	} `json:"city"`
}

type dailyForecastSlot struct {
	Dt   int64 `json:"dt"`
	Temp struct {
		Day   float64 `json:"day"`
		Min   float64 `json:"min"`
		Max   float64 `json:"max"`
		Morn  float64 `json:"morn"`
		Eve   float64 `json:"eve"`
		Night float64 `json:"night"`
	} `json:"temp"`
	Weather []struct {
		Main string `json:"main"`
	} `json:"weather"`
}

type dailyForecastResponse struct {
	List []dailyForecastSlot `json:"list"`
	City struct {
		Timezone int `json:"timezone"`
	} `json:"city"`
}

// FetchDaySummary loads OpenWeather 2.5 data for the calendar day targetDate (YYYY-MM-DD) in the location's timezone:
// first the 5-day / 3-hour forecast (covers roughly five calendar days), then the 16-day daily forecast when the day
// is farther out. Optionally fetches One Call 3.0 for alerts (ignored if the key has no access).
func FetchDaySummary(ctx context.Context, hc *http.Client, apiKey string, lat, lon float64, targetDate string) (DaySummary, error) {
	var out DaySummary
	if hc == nil {
		hc = &http.Client{Timeout: 25 * time.Second}
	}
	key := strings.TrimSpace(apiKey)
	if key == "" {
		return out, fmt.Errorf("openweather: missing api key")
	}
	slots, offsetSec, err := fetchForecast5(ctx, hc, key, lat, lon)
	if err != nil {
		return out, err
	}
	if out, ok := summarizeHourlySlots(slots, offsetSec, targetDate); ok {
		if alerts, err := tryFetchOneCallAlerts(ctx, hc, key, lat, lon, targetDate); err == nil {
			out.HasAlert = alerts
		}
		return out, nil
	}
	dailySlots, dailyTZ, errDaily := fetchForecastDaily16(ctx, hc, key, lat, lon)
	if errDaily != nil {
		return out, fmt.Errorf("openweather: no data for %s in 5-day forecast; daily: %w", targetDate, errDaily)
	}
	tz := dailyTZ
	if tz == 0 {
		tz = offsetSec
	}
	if out, ok := summarizeDailySlots(dailySlots, tz, targetDate); ok {
		if alerts, err := tryFetchOneCallAlerts(ctx, hc, key, lat, lon, targetDate); err == nil {
			out.HasAlert = alerts
		}
		return out, nil
	}
	return out, fmt.Errorf("openweather: no data for %s in hourly or daily forecast", targetDate)
}

func summarizeHourlySlots(slots []forecastSlot, offsetSec int, targetDate string) (DaySummary, bool) {
	var out DaySummary
	loc := time.FixedZone("owm", offsetSec)
	var (
		tmin, tmax  float64
		haveT       bool
		weatherMain string
		seenIcon    bool
	)
	for _, s := range slots {
		lt := time.Unix(s.Dt, 0).In(loc)
		if lt.Format("2006-01-02") != targetDate {
			continue
		}
		tlo := s.Main.TempMin
		thi := s.Main.TempMax
		if tlo == 0 && thi == 0 {
			tlo, thi = s.Main.Temp, s.Main.Temp
		}
		if !haveT {
			tmin, tmax = tlo, thi
		} else {
			if tlo < tmin {
				tmin = tlo
			}
			if thi > tmax {
				tmax = thi
			}
		}
		haveT = true
		if !seenIcon && len(s.Weather) > 0 {
			weatherMain = s.Weather[0].Main
			seenIcon = true
		}
	}
	if !haveT {
		return out, false
	}
	out.HighC = tmax
	out.LowC = tmin
	out.Icon = iconEmoji(weatherMain)
	return out, true
}

func summarizeDailySlots(slots []dailyForecastSlot, offsetSec int, targetDate string) (DaySummary, bool) {
	var out DaySummary
	loc := time.FixedZone("owm-daily", offsetSec)
	for _, s := range slots {
		lt := time.Unix(s.Dt, 0).In(loc)
		if lt.Format("2006-01-02") != targetDate {
			continue
		}
		hi := s.Temp.Max
		lo := s.Temp.Min
		if hi == 0 && lo == 0 {
			hi, lo = s.Temp.Day, s.Temp.Day
		}
		if hi == 0 && lo == 0 {
			// Rare malformed row
			continue
		}
		if lo > hi {
			lo, hi = hi, lo
		}
		main := ""
		if len(s.Weather) > 0 {
			main = s.Weather[0].Main
		}
		out.HighC = hi
		out.LowC = lo
		out.Icon = iconEmoji(main)
		return out, true
	}
	return out, false
}

func fetchForecast5(ctx context.Context, hc *http.Client, apiKey string, lat, lon float64) ([]forecastSlot, int, error) {
	u, err := url.Parse(forecastURL)
	if err != nil {
		return nil, 0, err
	}
	q := u.Query()
	q.Set("lat", fmt.Sprintf("%.5f", lat))
	q.Set("lon", fmt.Sprintf("%.5f", lon))
	q.Set("units", "metric")
	q.Set("appid", apiKey)
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	res, err := hc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("openweather: forecast: status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	var w forecastResponse
	if err := json.Unmarshal(body, &w); err != nil {
		return nil, 0, fmt.Errorf("openweather: json: %w", err)
	}
	if len(w.List) == 0 {
		return nil, w.City.Timezone, fmt.Errorf("openweather: empty list")
	}
	return w.List, w.City.Timezone, nil
}

func fetchForecastDaily16(ctx context.Context, hc *http.Client, apiKey string, lat, lon float64) ([]dailyForecastSlot, int, error) {
	u, err := url.Parse(forecastDailyURL)
	if err != nil {
		return nil, 0, err
	}
	q := u.Query()
	q.Set("lat", fmt.Sprintf("%.5f", lat))
	q.Set("lon", fmt.Sprintf("%.5f", lon))
	q.Set("cnt", "16")
	q.Set("units", "metric")
	q.Set("appid", apiKey)
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, 0, err
	}
	res, err := hc.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("openweather: forecast/daily: status %d: %s", res.StatusCode, strings.TrimSpace(string(body)))
	}
	var w dailyForecastResponse
	if err := json.Unmarshal(body, &w); err != nil {
		return nil, 0, fmt.Errorf("openweather: daily json: %w", err)
	}
	if len(w.List) == 0 {
		return nil, w.City.Timezone, fmt.Errorf("openweather: daily empty list")
	}
	return w.List, w.City.Timezone, nil
}

// tryFetchOneCallAlerts returns true if the One Call 3.0 response includes a national alert that overlaps targetDate.
func tryFetchOneCallAlerts(ctx context.Context, hc *http.Client, apiKey string, lat, lon float64, targetDate string) (bool, error) {
	u, _ := url.Parse(oneCallURL)
	q := u.Query()
	q.Set("lat", fmt.Sprintf("%.5f", lat))
	q.Set("lon", fmt.Sprintf("%.5f", lon))
	q.Set("units", "metric")
	q.Set("exclude", "current,minutely,hourly,daily")
	q.Set("appid", apiKey)
	u.RawQuery = q.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return false, err
	}
	res, err := hc.Do(req)
	if err != nil {
		return false, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	if res.StatusCode != http.StatusOK {
		return false, fmt.Errorf("onecall: %d", res.StatusCode)
	}
	var oc struct {
		Alerts []struct {
			Event string   `json:"event"`
			Start int64    `json:"start"`
			End   int64    `json:"end"`
			Tags  []string `json:"tags"`
		} `json:"alerts"`
	}
	if err := json.Unmarshal(body, &oc); err != nil {
		return false, err
	}
	day, err := time.Parse("2006-01-02", targetDate)
	if err != nil {
		return false, err
	}
	dayStart := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, time.UTC)
	dayEnd := dayStart.Add(24 * time.Hour)
	for _, a := range oc.Alerts {
		if a.End <= 0 {
			if strings.TrimSpace(a.Event) != "" || len(a.Tags) > 0 {
				return true, nil
			}
			continue
		}
		ast := time.Unix(a.Start, 0).UTC()
		aen := time.Unix(a.End, 0).UTC()
		if aen.After(dayStart) && ast.Before(dayEnd) {
			return true, nil
		}
	}
	return false, nil
}

func iconEmoji(main string) string {
	m := strings.ToLower(strings.TrimSpace(main))
	switch m {
	case "clear":
		return "☀️"
	case "rain", "drizzle":
		return "🌧️"
	case "thunderstorm":
		return "⛈️"
	case "snow":
		return "🌨️"
	case "mist", "fog", "haze", "smoke", "dust", "sand", "tornado", "ash", "squall":
		return "🌫️"
	case "clouds":
		return "☁️"
	default:
		if m != "" {
			return "☁️"
		}
		return "—"
	}
}
