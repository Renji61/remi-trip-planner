package trips

import (
	"context"
	"encoding/json"
	"math"
	"strings"
	"time"

	"remi-trip-planner/internal/openweather"
)

// StopWeatherPreview is returned for plain itinerary stops (JSON to the trip page).
type StopWeatherPreview struct {
	OK       bool    `json:"ok"`
	Icon     string  `json:"icon"`
	HighC    float64 `json:"highC"`
	LowC     float64 `json:"lowC"`
	HasAlert bool    `json:"hasAlert"`
	Reason   string  `json:"reason,omitempty"`
}

type weatherPayload struct {
	Icon     string  `json:"icon"`
	HighC    float64 `json:"highC"`
	LowC     float64 `json:"lowC"`
	HasAlert bool    `json:"hasAlert"`
}

const weatherCacheTTL = 3 * time.Hour

// ScheduleStopWeatherPrefetchForNewPlainStop warms the weather cache after a user-created plain
// itinerary stop is committed (HTTP add-stop or sync create). Do not call from booking flows
// (lodging/vehicle/flight) where itinerary rows are created before the booking link exists.
func (s *Service) ScheduleStopWeatherPrefetchForNewPlainStop(tripID, itemID string, dayNumber int) {
	tripID = strings.TrimSpace(tripID)
	itemID = strings.TrimSpace(itemID)
	if tripID == "" || itemID == "" {
		return
	}
	if dayNumber < 1 {
		dayNumber = 1
	}
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
		defer cancel()
		trip, err := s.GetTrip(ctx, tripID)
		if err != nil {
			return
		}
		dateISO, ok := ItineraryDayDateISO(trip.StartDate, dayNumber)
		if !ok || strings.TrimSpace(dateISO) == "" {
			return
		}
		_, _ = s.GetStopWeatherPreview(ctx, tripID, itemID, dateISO)
	}()
}

// GetStopWeatherPreview returns cached or fresh OpenWeather data for a plain stop on a given calendar day.
// dateISO is YYYY-MM-DD (the itinerary day for that row). No API key, out-of-range dates, or non-stops yield ok=false.
func (s *Service) GetStopWeatherPreview(ctx context.Context, tripID, itemID, dateISO string) (StopWeatherPreview, error) {
	var out StopWeatherPreview
	dateISO = strings.TrimSpace(dateISO)
	if dateISO == "" {
		out.Reason = "missing_date"
		return out, nil
	}
	app, err := s.GetAppSettings(ctx)
	if err != nil {
		return out, err
	}
	if strings.TrimSpace(app.OpenWeatherAPIKey) == "" {
		out.Reason = "no_key"
		return out, nil
	}
	tripID = strings.TrimSpace(tripID)
	itemID = strings.TrimSpace(itemID)
	items, err := s.repo.ListItineraryItems(ctx, tripID)
	if err != nil {
		return out, err
	}
	var it *ItineraryItem
	for i := range items {
		if items[i].ID == itemID {
			it = &items[i]
			break
		}
	}
	if it == nil {
		out.Reason = "not_found"
		return out, nil
	}
	if NormalizeItineraryItemKind(it.ItemKind) == ItineraryItemKindCommute {
		out.Reason = "commute"
		return out, nil
	}
	lodgings, _ := s.repo.ListLodgings(ctx, tripID)
	vehicles, _ := s.repo.ListVehicleRentals(ctx, tripID)
	flights, _ := s.repo.ListFlights(ctx, tripID)
	if l, has := LodgingByItineraryItemID(lodgings, items)[it.ID]; has && strings.TrimSpace(l.ID) != "" {
		out.Reason = "booking"
		return out, nil
	}
	if v, has := VehicleRentalByItineraryItemID(vehicles, items)[it.ID]; has && strings.TrimSpace(v.ID) != "" {
		out.Reason = "booking"
		return out, nil
	}
	if f, has := FlightByItineraryItemID(flights, items)[it.ID]; has && strings.TrimSpace(f.ID) != "" {
		out.Reason = "booking"
		return out, nil
	}
	if math.Abs(it.Latitude) < 1e-6 && math.Abs(it.Longitude) < 1e-6 {
		out.Reason = "no_coords"
		return out, nil
	}
	targetDay, err := time.Parse("2006-01-02", dateISO)
	if err != nil {
		out.Reason = "bad_date"
		return out, nil
	}
	startToday := time.Date(time.Now().Year(), time.Now().Month(), time.Now().Day(), 0, 0, 0, 0, time.Local)
	day := time.Date(targetDay.Year(), targetDay.Month(), targetDay.Day(), 0, 0, 0, 0, time.Local)
	endWindow := startToday.AddDate(0, 0, 14)
	if day.Before(startToday) || day.After(endWindow) {
		out.Reason = "out_of_range"
		return out, nil
	}

	payload, at, hit, err := s.repo.GetWeatherCache(ctx, tripID, itemID, dateISO)
	if err != nil {
		return out, err
	}
	if hit && !at.IsZero() && at.Add(weatherCacheTTL).After(time.Now().UTC()) {
		var w weatherPayload
		if err := json.Unmarshal([]byte(payload), &w); err == nil {
			out.OK = true
			out.Icon = w.Icon
			out.HighC = w.HighC
			out.LowC = w.LowC
			out.HasAlert = w.HasAlert
			return out, nil
		}
	}

	summary, err := openweather.FetchDaySummary(ctx, nil, app.OpenWeatherAPIKey, it.Latitude, it.Longitude, dateISO)
	if err != nil {
		out.Reason = "fetch_failed"
		return out, nil
	}
	wp := weatherPayload{Icon: summary.Icon, HighC: summary.HighC, LowC: summary.LowC, HasAlert: summary.HasAlert}
	b, _ := json.Marshal(wp)
	if err := s.repo.UpsertWeatherCache(ctx, tripID, itemID, dateISO, it.Latitude, it.Longitude, string(b)); err != nil {
		return out, err
	}
	out.OK = true
	out.Icon = summary.Icon
	out.HighC = summary.HighC
	out.LowC = summary.LowC
	out.HasAlert = summary.HasAlert
	return out, nil
}
