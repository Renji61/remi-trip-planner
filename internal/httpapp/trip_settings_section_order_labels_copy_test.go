package httpapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTripSettingsSectionOrderSectionLabelsSidebarCopy(t *testing.T) {
	root := findModuleRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip_settings.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)

	mustContain := []string{
		`Drag to reorder sections. The trip hero always stays at the top. Primary content and Quick Access Tools follow this order on larger screens.`,
		`>Primary Content Sections</p>`,
		`id="trip-settings-section-labels-title">Section Labels`,
		`Customize navigation labels (leave blank to use default names).`,
		`>Accommodation Label<input`,
		`>Vehicle Rental Label<input`,
		`>Flights Label<input`,
		`>Expenses Label<input`,
		`>Group Expenses Label<input`,
		`Add up to 3 custom links to the trip sidebar (desktop only). <strong>HTTPS</strong> URLs only. Drag to reorder.`,
		`>Link Name<input`,
	}
	for _, s := range mustContain {
		if !strings.Contains(html, s) {
			t.Errorf("trip_settings.html missing expected copy: %q", s)
		}
	}

	mustNotContain := []string{
		`Drag to reorder. The trip hero stays at the top; primary content sections`,
		`>Primary Content</p>`,
		`id="trip-settings-rename-title">Rename Sections`,
		`Optional labels for navigation (defaults apply if left blank).`,
		`Accommodation nav label<input`,
		`Vehicle rental nav label<input`,
		`Flights nav label<input`,
		`Expenses nav label<input`,
		`Group Expenses nav label<input`,
		`Up to three links in the trip page sidebar (desktop). <strong>https</strong> URLs only. Drag rows or use arrows to set order.`,
		`>Link Text<input`,
	}
	for _, s := range mustNotContain {
		if strings.Contains(html, s) {
			t.Errorf("trip_settings.html still contains retired copy: %q", s)
		}
	}
}
