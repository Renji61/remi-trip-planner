package trips

import (
	"math"
	"strings"
)

// UITimeFormatIs24h reports whether trip time display should use 24-hour clock.
func UITimeFormatIs24h(raw string) bool {
	return strings.TrimSpace(strings.ToLower(raw)) == "24h"
}

func sectionTitleOrDefault(custom, def string) string {
	if s := strings.TrimSpace(custom); s != "" {
		return s
	}
	return def
}

// StaySectionTitle is the sidebar/nav label for the accommodation section (custom per trip).
func (t Trip) StaySectionTitle() string {
	return sectionTitleOrDefault(t.UILabelStay, "Accommodation")
}

func (t Trip) VehicleSectionTitle() string {
	return sectionTitleOrDefault(t.UILabelVehicle, "Vehicle Rental")
}

func (t Trip) FlightsSectionTitle() string {
	return sectionTitleOrDefault(t.UILabelFlights, "Flights")
}

func (t Trip) SpendsSectionTitle() string {
	return sectionTitleOrDefault(t.UILabelSpends, "Expenses")
}

func (t Trip) GroupExpensesSectionTitle() string {
	return sectionTitleOrDefault(t.UILabelGroupExpenses, "Group Expenses")
}

// ItineraryDayDefaultOpen controls initial <details open> for itinerary day groups (index 0 = first day).
func (t Trip) ItineraryDayDefaultOpen(index int) bool {
	switch strings.TrimSpace(strings.ToLower(t.UIItineraryExpand)) {
	case "all":
		return true
	case "none":
		return false
	default:
		return index == 0
	}
}

// SpendsDayDefaultOpen controls initial open state for the spends-by-day groups on the trip page.
func (t Trip) SpendsDayDefaultOpen(index int) bool {
	switch strings.TrimSpace(strings.ToLower(t.UISpendsExpand)) {
	case "all":
		return true
	case "none":
		return false
	default:
		return index == 0
	}
}

// HasHomeMapCenter is true when this trip stores its own map center (non-zero lat or lng).
func (t Trip) HasHomeMapCenter() bool {
	return math.Abs(t.HomeMapLatitude) > 1e-9 || math.Abs(t.HomeMapLongitude) > 1e-9
}

// SectionEnabled reports whether a trip sub-area is shown and reachable.
func (t Trip) SectionEnabledStay() bool      { return t.UIShowStay }
func (t Trip) SectionEnabledVehicle() bool   { return t.UIShowVehicle }
func (t Trip) SectionEnabledFlights() bool   { return t.UIShowFlights }
func (t Trip) SectionEnabledSpends() bool    { return t.UIShowSpends }
func (t Trip) SectionEnabledItinerary() bool { return t.UIShowItinerary }
func (t Trip) SectionEnabledChecklist() bool { return t.UIShowChecklist }
func (t Trip) SectionEnabledTheTab() bool    { return t.UIShowTheTab && t.UIShowSpends }
