package trips

import "testing"

func TestFormatFlightAirportHeadline(t *testing.T) {
	cache := map[string]Airport{
		"BLR": {IATACode: "BLR", Name: "Kempegowda International Airport"},
		"COK": {IATACode: "COK", Name: "Cochin International Airport", City: "Kochi"},
	}
	tests := []struct {
		name    string
		display string
		iata    string
		by      map[string]Airport
		want    string
	}{
		{"empty display lookup name", "", "BLR", cache, "Kempegowda International Airport - BLR"},
		{"display only IATA lookup", "BLR", "blr", cache, "Kempegowda International Airport - BLR"},
		{"already formatted", "Kempegowda International Airport - BLR", "BLR", cache, "Kempegowda International Airport - BLR"},
		{"name without suffix", "Kempegowda International Airport", "BLR", cache, "Kempegowda International Airport - BLR"},
		{"comma trimmed", "Kochi, India", "COK", cache, "Kochi - COK"},
		{"no cache falls back IATA", "", "BLR", nil, "BLR"},
		{"invalid IATA uses display", "Some Address, TX", "", cache, "Some Address"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := FormatFlightAirportHeadline(tt.display, tt.iata, tt.by)
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}
