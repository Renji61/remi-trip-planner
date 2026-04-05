package httpapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTripSettingsDisplayPreferencesCopy(t *testing.T) {
	root := findModuleRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip_settings.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)

	mustContain := []string{
		`>Default Itinerary View`,
		`>Default Expense View`,
		`>Default Group Expense View`,
		`>Time Format`,
		`>Date Format`,
		`>Use site default</option>`,
		`Overrides the default date format from site settings for this trip only.`,
	}
	for _, s := range mustContain {
		if !strings.Contains(html, s) {
			t.Errorf("trip_settings.html missing Display Preferences copy: %q", s)
		}
	}

	// Ensure date-format inherit option text changed but distance unit option unchanged.
	if !strings.Contains(html, `name="distance_unit"`) || !strings.Contains(html, `>Use unit set in Site Settings</option>`) {
		t.Fatal("expected distance unit dropdown to keep 'Use unit set in Site Settings'")
	}
	if strings.Count(html, `>Use site default</option>`) != 1 {
		t.Fatalf("expected exactly one 'Use site default' option (date format only), got count check in file")
	}

	mustNotContain := []string{
		`Default expanded itinerary days`,
		`Default expanded expense days (on trip page)`,
		`Default expanded group expense days (on trip page)`,
		`Time display on this trip`,
		`Date display on this trip`,
		`Overrides the default date order from Site settings for this trip only.`,
	}
	for _, s := range mustNotContain {
		if strings.Contains(html, s) {
			t.Errorf("trip_settings.html still contains retired Display Preferences copy: %q", s)
		}
	}
}
