package httpapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTripSettingsTopbarSubtitleCopy(t *testing.T) {
	root := findModuleRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip_settings.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)
	want := `Manage dates, currency, labels, and layout for {{.Details.Trip.Name}}.`
	if !strings.Contains(html, want) {
		t.Fatalf("expected topbar subtitle: %q", want)
	}
	if strings.Contains(html, "Dates, currency, navigation labels, and layout for {{.Details.Trip.Name}}.") {
		t.Fatal("old topbar subtitle should be removed")
	}
}
