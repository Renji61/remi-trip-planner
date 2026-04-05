package httpapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTripSettingsCalendarSyncCopy(t *testing.T) {
	root := findModuleRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip_settings.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)

	mustContain := []string{
		`Subscribe via a private calendar feed link in <strong>Google Calendar</strong> (<strong>From URL</strong>) or <strong>Apple Calendar</strong> (<strong>New Subscription</strong>). Updates may take time to sync (not instant).`,
		`No subscription link generated yet. Create one to sync this trip with your calendar.`,
		`>Generate Link</button>`,
	}
	for _, s := range mustContain {
		if !strings.Contains(html, s) {
			t.Errorf("trip_settings.html missing expected Calendar Sync copy: %q", s)
		}
	}

	mustNotContain := []string{
		`using a secret feed link. External apps refresh subscribed feeds`,
		`No subscription link yet — generate one to copy into your calendar app.`,
		`>Generate subscription link</button>`,
	}
	for _, s := range mustNotContain {
		if strings.Contains(html, s) {
			t.Errorf("trip_settings.html still contains retired Calendar Sync copy: %q", s)
		}
	}
}
