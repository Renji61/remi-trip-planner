package httpapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestProfilePageCopy validates /profile template copy (desktop + mobile share one template).
func TestProfilePageCopy(t *testing.T) {
	root := findModuleRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "web", "templates", "profile.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)

	want := []string{
		`Download a JSON snapshot of your profile, settings, and accessible trips (passwords excluded; API keys are redacted).`,
		`href="/profile/export" download>Download JSON</a>`,
		`Use a strong, unique password.`,
		`<p id="profile-password-hint" class="settings-field-hint muted full auth-password-hint">Minimum 8 characters</p>`,
	}
	for _, s := range want {
		if !strings.Contains(html, s) {
			t.Errorf("profile.html missing %q", s)
		}
	}

	avoid := []string{
		"Download a JSON snapshot of your profile, settings, and trips you can access (no password; API keys in app settings are redacted).",
		`download>Download JSON export</a>`,
		"Use a strong password you do not reuse elsewhere.",
		`<p id="profile-password-hint" class="settings-field-hint muted full auth-password-hint">Minimum 8 characters required.</p>`,
	}
	for _, s := range avoid {
		if strings.Contains(html, s) {
			t.Errorf("profile.html should not contain %q", s)
		}
	}
}
