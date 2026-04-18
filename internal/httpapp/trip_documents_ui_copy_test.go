package httpapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestTripDocumentsPageCopy validates trip documents page strings and the upload button
// label logic in app.js for /documents (desktop and mobile upload card on that page).
func TestTripDocumentsPageCopy(t *testing.T) {
	root := findModuleRoot(t)

	docB, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip_documents.html"))
	if err != nil {
		t.Fatal(err)
	}
	doc := string(docB)

	tripB, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip.html"))
	if err != nil {
		t.Fatal(err)
	}
	trip := string(tripB)

	fabB, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip_mobile_fab_links.html"))
	if err != nil {
		t.Fatal(err)
	}
	fab := string(fabB)

	jsB, err := os.ReadFile(filepath.Join(root, "web", "static", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	js := string(jsB)

	docWant := []string{
		"Keep all your travel documents organized and accessible.",
		`Drag &amp; drop files or click to upload`,
		`data-trip-doc-selection-status>No file selected</p>`,
		`type="submit">Upload</button>`,
		`>Search Documents`,
		`placeholder="Search files, sections, or categories..."`,
		`<option value="">All Categories</option>`,
	}
	for _, s := range docWant {
		if !strings.Contains(doc, s) {
			t.Errorf("trip_documents.html missing %q", s)
		}
	}
	docAvoid := []string{
		"All your travel essentials, organized and ready for the road.",
		"Drop files or click to browse",
		"Quick search",
		"Search file, section, item, category...",
		`<option value="">All categories</option>`,
		"type=\"submit\">Upload Document</button>",
	}
	for _, s := range docAvoid {
		if strings.Contains(doc, s) {
			t.Errorf("trip_documents.html should not contain %q", s)
		}
	}

	// Trip details FAB uses tripMobileFabLinks; documents shortcut lives there.
	if !strings.Contains(trip, `template "tripMobileFabLinks"`) {
		t.Error(`trip.html should embed tripMobileFabLinks for the FAB menu`)
	}
	if !strings.Contains(fab, `href="/trips/{{$t.ID}}/documents"`) {
		t.Error(`trip_mobile_fab_links.html should link to /documents when UIShowDocuments`)
	}
	if !strings.Contains(fab, "Trip Documents") {
		t.Error("trip_mobile_fab_links.html should contain Trip Documents label for documents link")
	}

	// Trip documents upload: single-word submit label in JS (replaces Upload Document / Upload Documents).
	if !strings.Contains(js, `const label = "Upload";`) {
		t.Error(`app.js trip documents block should set upload button label to "Upload"`)
	}
	if strings.Contains(js, `? "Upload Documents" : "Upload Document"`) {
		t.Error(`app.js should not use legacy Upload Document(s) labels for trip documents`)
	}
}
