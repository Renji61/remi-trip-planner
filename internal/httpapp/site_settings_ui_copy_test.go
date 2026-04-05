package httpapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestSiteSettingsPageCopy validates Site settings template copy (desktop + mobile share one template).
func TestSiteSettingsPageCopy(t *testing.T) {
	root := findModuleRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "web", "templates", "settings.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)

	want := []string{
		`{{else}}Configure map behavior, registration, and dashboard preferences.{{end}}`,
		`App title, map defaults, and location lookup apply to all users. Dashboard title, currency, and registration settings below are saved to your account.`,
		`Shown as the main heading on your dashboard. The sidebar brand remains <strong>REMI Trip Planner</strong>.`,
		`<label>Default Currency`,
		`<label>Currency Symbol`,
		`Used for new trips and dashboard totals (e.g. USD, $).`,
		`Used across the app unless overridden in Trip Settings → Display Preferences.`,
		`placeholder="Search for a place or enter a location"`,
		`Sets the default map center and zoom for trips without a defined location. Uses OpenStreetMap unless a valid Google Maps key is provided. Zoom range: 1–20.`,
		`Enables location suggestions using external services.`,
		`When provided, Google Maps is used for location suggestions and trip maps. Otherwise, OpenStreetMap is used. Changes apply after saving.`,
		`placeholder="Enter a browser API key for Maps, Places, and Geocoding"`,
		`Used when not overridden by user or trip settings.`,
		`<label>Max Upload Size (MB)`,
		`Applies to all document and image uploads across the app.`,
		`Allow Public Registration`,
		`When disabled, <code>/register</code> is unavailable. The first account can still be created via <code>/setup</code>.`,
		`Configure theme and dashboard banner.`,
		`Theme changes are saved automatically.`,
		`>Upload Image</option>`,
		`>Custom Image URL (HTTPS only)</option>`,
		`to set a banner. Paths are stored from the site root to remain valid if the domain changes.`,
		`Configure how trips are displayed on your dashboard.`,
		`<label class="full">Your Distance Unit`,
		`Applies to dashboard stats and where trips don’t override units.`,
		`Grid uses responsive columns; list uses compact rows.`,
		`Applies to Your Trips, Drafts, and Archived sections.`,
		`<strong>Save All Changes</strong> applies all sections at once. Map and registration affect all users; theme, banner, layout, currency, and dashboard title are saved to your account.`,
		`class="trip-settings-save-btn">Save All Changes</button>`,
		`aria-label="Save All Changes"`,
	}
	for _, s := range want {
		if !strings.Contains(html, s) {
			t.Errorf("settings.html missing %q", s)
		}
	}

	avoid := []string{
		`{{else}}Configure map behavior, registration, and your personal dashboard.{{end}}`,
		`apply to <strong>everyone</strong> on this server`,
		`saved to <strong>your account</strong> when you save`,
		`Shown as the main heading on your home dashboard. The sidebar brand stays`,
		`<label>Default currency name`,
		`<label>Default currency symbol`,
		`Used for new trips and dashboard totals (e.g. USD and $).`,
		`Used for date fields and labels everywhere unless a trip sets its own order under Trip settings`,
		`placeholder="Search places or type a name"`,
		`Default center and zoom for the <strong>Trip Map</strong> on trip details`,
		`When enabled, the app can suggest coordinates for locations (may use external lookup).`,
		`When set to a valid browser API key (typically starting with <code>AIza</code>)`,
		`Paste a browser key for Maps, Places, and Geocoding`,
		`Used when a user or trip does not override distance units.`,
		`Max upload size per file (MB)`,
		`Applies to Trip Documents and document/image attachments across trip entry forms.`,
		`Allow new users to register`,
		`When off, <code>/register</code> is unavailable; the first account is still created via`,
		`Theme and dashboard banner image or URL.`,
		`The header control switches light/dark and saves here.`,
		`Select “Upload image” or “Custom image URL”`,
		`How trips are listed on your home dashboard.`,
		`Your distance unit <span class="muted">(personal)</span>`,
		`Applies to dashboard stats and anywhere a trip does not set its own unit.`,
		`list uses a compact horizontal card.`,
		`Applies within Your Trips, Draft, and Archived.`,
		`<strong>Save all changes</strong> applies every section on this page at once. Map and registration affect all users; theme, dashboard banner, layout, currency, and dashboard title are saved to your account.`,
		`class="trip-settings-save-btn">Save all changes</button>`,
	}
	for _, s := range avoid {
		if strings.Contains(html, s) {
			t.Errorf("settings.html should not contain %q", s)
		}
	}
}
