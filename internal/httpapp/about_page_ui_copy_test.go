package httpapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestAboutPageCopy validates /about desktop and mobile shared template copy (single template).
func TestAboutPageCopy(t *testing.T) {
	root := findModuleRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "web", "templates", "about.html"))
	if err != nil {
		t.Fatal(err)
	}
	html := string(b)

	want := []string{
		`Version details and features of {{.Settings.AppTitle}}.`,
		`<h3 id="about-version-title">Current Version</h3>`,
		`setStatus("You're using the latest version (v" + data.latest + ").");`,
	}
	for _, s := range want {
		if !strings.Contains(html, s) {
			t.Errorf("about.html missing %q", s)
		}
	}

	avoid := []string{
		"Version information and what you can do with",
		`<h3 id="about-version-title">This install</h3>`,
		`You are up to date with the latest published release`,
	}
	for _, s := range avoid {
		if strings.Contains(html, s) {
			t.Errorf("about.html should not contain %q", s)
		}
	}
}
