package httpapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTripSettingsTripMembersCopy(t *testing.T) {
	root := findModuleRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip_settings.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)

	mustContain := []string{
		`Manage travelers and control who can view or edit this trip.`,
		`>Collaborators (Editors)</p>`,
		`Collaborators can view and edit all trip details.`,
		`>Guests (For Expense Splitting)</p>`,
		`No guests added yet.`,
		`>Guest Name<input`,
		`placeholder="e.g. Alex (Friend)"`,
		`Guests are used for expense tracking and splitting only.`,
	}
	for _, s := range mustContain {
		if !strings.Contains(html, s) {
			t.Errorf("trip_settings.html missing expected Trip Members copy: %q", s)
		}
	}

	mustNotContain := []string{
		`Manage who is traveling and who can edit this plan.`,
		`<p class="trip-settings-people-field-label">Collaborators</p>`,
		`<strong>Collaborators:</strong> Can view and edit all trip details.`,
		`<p class="trip-settings-people-field-label">Guests</p>`,
		`No guests yet.`,
		`Guest display name<input`,
		`placeholder="e.g. Alex (friend)"`,
		`<strong>Guests:</strong> Passive members for cost tracking and splitting.`,
	}
	for _, s := range mustNotContain {
		if strings.Contains(html, s) {
			t.Errorf("trip_settings.html still contains retired Trip Members copy: %q", s)
		}
	}
}
