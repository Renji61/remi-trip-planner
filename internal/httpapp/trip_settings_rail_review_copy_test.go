package httpapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTripSettingsRailAndReviewCopy(t *testing.T) {
	root := findModuleRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip_settings.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)

	mustContain := []string{
		`Enable or disable sections for this trip (navigation, pages, and layout). Custom sidebar links appear when enabled. Disabling a section will also hide its related content block and navigation link.`,
		`Show or hide content blocks on the trip page. If a section is disabled above, related options here will also be disabled. Quick Access Tools below control mobile and sidebar quick actions.`,
		`Control which quick actions appear in the sidebar and mobile menu.`,
		`Changes apply after saving. Hidden sections will be removed from navigation.`,
		`>Back to Dashboard</a>`,
	}
	for _, s := range mustContain {
		if !strings.Contains(html, s) {
			t.Errorf("trip_settings.html missing expected rail/review copy: %q", s)
		}
	}

	mustNotContain := []string{
		`Turn these areas on or off for this trip (navigation, pages, and layout). Custom Sidebar Links appear`,
		`Show or hide primary content blocks on the trip page. <strong>Trip sections</strong> above control master areas`,
		`These toggles control Quick Access Tools in the sidebar and the same items in the mobile <strong>+</strong> menu.`,
		`Changes will take effect once saved. Hidden sections will be removed from navigation.`,
		`>Back to dashboard</a>`,
	}
	for _, s := range mustNotContain {
		if strings.Contains(html, s) {
			t.Errorf("trip_settings.html still contains retired rail/review copy: %q", s)
		}
	}
}
