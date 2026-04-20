package trips

import (
	"encoding/json"
	"strings"
)

// VenueHoursSummary returns a one-line saved venue hours snapshot for itinerary UI, or empty.
func (i ItineraryItem) VenueHoursSummary() string {
	s := strings.TrimSpace(i.VenueHoursJSON)
	if s == "" {
		return ""
	}
	var v struct {
		Summary string `json:"summary"`
	}
	if err := json.Unmarshal([]byte(s), &v); err != nil {
		return ""
	}
	return strings.TrimSpace(v.Summary)
}
