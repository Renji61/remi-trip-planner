package trips

import "strings"

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

// StaySectionTitle is the sidebar/nav label for the stay section (custom per trip).
func (t Trip) StaySectionTitle() string {
	return sectionTitleOrDefault(t.UILabelStay, "Stay")
}

func (t Trip) VehicleSectionTitle() string {
	return sectionTitleOrDefault(t.UILabelVehicle, "Vehicle")
}

func (t Trip) FlightsSectionTitle() string {
	return sectionTitleOrDefault(t.UILabelFlights, "Flights")
}

func (t Trip) SpendsSectionTitle() string {
	return sectionTitleOrDefault(t.UILabelSpends, "Spends")
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

// SectionEnabled reports whether a trip sub-area is shown and reachable.
func (t Trip) SectionEnabledStay() bool    { return t.UIShowStay }
func (t Trip) SectionEnabledVehicle() bool { return t.UIShowVehicle }
func (t Trip) SectionEnabledFlights() bool { return t.UIShowFlights }
func (t Trip) SectionEnabledSpends() bool  { return t.UIShowSpends }
