package httpapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTripSettingsTripInformationCopy locks in Trip Information field hints and placeholders on the trip settings page.
func TestTripSettingsTripInformationCopy(t *testing.T) {
	root := findModuleRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip_settings.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)

	mustContain := []string{
		`placeholder="e.g. Summer in Japan"`,
		`placeholder="Add a short summary of your trip"`,
		`placeholder="Enter total budget (leave 0 to auto-calculate)"`,
		`If set above 0, this acts as a spending cap. All expenses (including group expenses, personal expenses, accommodation, vehicle rental, and flights) contribute to <strong>Actual Spending</strong>.`,
		`Controls how distances (km/mi) are displayed for this trip.`,
		`Displayed in the trip header. Upload an image or provide a URL. If not set, the default site style will be used.`,
		`<label class="full">Map Center`,
		`placeholder="Search for a location or leave empty"`,
		`Sets the default center for your trip map. Leave empty to use the site default. Uses <strong>OpenStreetMap</strong> unless a Google Maps key is configured.`,
		`placeholder="e.g. USD"`,
		`placeholder="e.g. $"`,
	}
	for _, s := range mustContain {
		if !strings.Contains(html, s) {
			t.Errorf("trip_settings.html missing expected copy: %q", s)
		}
	}

	mustNotContain := []string{
		`placeholder="Short trip summary"`,
		`placeholder="0 = use computed budget"`,
		`When set above zero, <strong>Trip Budget</strong>`,
		`Itinerary connectors and other distances on this trip follow this choice when set.`,
		`Applies only to this trip's top hero on the trip page.`,
		`Map center (search places)`,
		`placeholder="Search places or leave empty for site default"`,
		`unless Site settings has a valid Google Maps browser key.`,
		`placeholder="e.g. PHP"`,
		`placeholder="e.g. £"`,
	}
	for _, s := range mustNotContain {
		if strings.Contains(html, s) {
			t.Errorf("trip_settings.html still contains retired copy: %q", s)
		}
	}
}
