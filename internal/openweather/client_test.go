package openweather

import (
	"testing"
	"time"
)

func TestSummarizeHourlySlots(t *testing.T) {
	// UTC+0: noon on 2026-04-22
	ts := time.Date(2026, 4, 22, 12, 0, 0, 0, time.UTC).Unix()
	slots := []forecastSlot{{
		Dt: ts,
		Main: struct {
			Temp    float64 `json:"temp"`
			TempMin float64 `json:"temp_min"`
			TempMax float64 `json:"temp_max"`
		}{Temp: 20, TempMin: 18, TempMax: 22},
		Weather: []struct {
			Main string `json:"main"`
		}{{Main: "Clear"}},
	}}
	got, ok := summarizeHourlySlots(slots, 0, "2026-04-22")
	if !ok {
		t.Fatal("expected match")
	}
	if got.LowC != 18 || got.HighC != 22 {
		t.Fatalf("temps: %+v", got)
	}
	if got.Icon != "☀️" {
		t.Fatalf("icon: %q", got.Icon)
	}
}

func TestSummarizeDailySlots(t *testing.T) {
	const tz = 19800 // +5:30 like Hyderabad
	// Noon local on 2026-07-10
	localNoon := time.Date(2026, 7, 10, 12, 0, 0, 0, time.FixedZone("loc", tz))
	slots := []dailyForecastSlot{{
		Dt: localNoon.Unix(),
		Temp: struct {
			Day   float64 `json:"day"`
			Min   float64 `json:"min"`
			Max   float64 `json:"max"`
			Morn  float64 `json:"morn"`
			Eve   float64 `json:"eve"`
			Night float64 `json:"night"`
		}{Min: 22, Max: 31, Day: 28},
		Weather: []struct {
			Main string `json:"main"`
		}{{Main: "Rain"}},
	}}
	got, ok := summarizeDailySlots(slots, tz, "2026-07-10")
	if !ok {
		t.Fatal("expected match")
	}
	if got.LowC != 22 || got.HighC != 31 {
		t.Fatalf("temps: %+v", got)
	}
	if got.Icon != "🌧️" {
		t.Fatalf("icon: %q", got.Icon)
	}
}
