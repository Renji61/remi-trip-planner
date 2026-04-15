package httpapp

import (
	"math"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// itineraryAddStopNeedsDeferredGeocode mirrors web/static/app.js logic in initItineraryForm submit
// (search: itineraryForm.addEventListener("submit")). When this returns false, the browser should
// perform a normal POST on the first click; when true, the client runs resolveLocation then requestSubmit.
func itineraryAddStopNeedsDeferredGeocode(locationLookupEnabled bool, query, latStr, lngStr string) bool {
	q := strings.TrimSpace(query)
	if q == "" {
		return false
	}
	if !locationLookupEnabled {
		return false
	}
	latStr = strings.TrimSpace(latStr)
	lngStr = strings.TrimSpace(lngStr)
	if latStr == "" || lngStr == "" {
		return true
	}
	lat, err1 := strconv.ParseFloat(latStr, 64)
	lng, err2 := strconv.ParseFloat(lngStr, 64)
	if err1 != nil || err2 != nil {
		return true
	}
	if math.IsNaN(lat) || math.IsNaN(lng) || math.IsInf(lat, 0) || math.IsInf(lng, 0) {
		return true
	}
	if math.Abs(lat) > 90 || math.Abs(lng) > 180 {
		return true
	}
	return false
}

func TestItineraryAddStopDeferredGeocodePolicy(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name    string
		lookup  bool
		query   string
		lat     string
		lng     string
		wantDef bool
	}{
		{"empty query", true, "  ", "", "", false},
		{"lookup off", false, "Paris", "", "", false},
		{"lookup off with coords", false, "Paris", "48.8", "2.3", false},
		{"lookup on missing coords", true, "Paris", "", "", true},
		{"lookup on valid coords", true, "Paris", "13.7563", "100.5018", false},
		{"lookup on bogus coords", true, "Paris", "not-a-number", "2", true},
		{"lookup on out of range lat", true, "X", "99", "10", true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			got := itineraryAddStopNeedsDeferredGeocode(tc.lookup, tc.query, tc.lat, tc.lng)
			if got != tc.wantDef {
				t.Fatalf("needsDeferredGeocode=%v want %v (lookup=%v q=%q lat=%q lng=%q)",
					got, tc.wantDef, tc.lookup, tc.query, tc.lat, tc.lng)
			}
		})
	}
}

func TestAppJsItineraryAndAjaxSubmitListenersStaySync(t *testing.T) {
	t.Parallel()
	root := findModuleRoot(t)
	b, err := os.ReadFile(filepath.Join(root, "web", "static", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	if strings.Contains(s, `itineraryForm.addEventListener("submit", async`) {
		t.Fatal(`itinerary form submit must not use an async listener (regression: first click no-op)`)
	}
	if !strings.Contains(s, `itineraryForm.addEventListener("submit", (event) =>`) {
		t.Fatal(`expected sync itineraryForm submit listener (event) =>`)
	}
	if strings.Contains(s, "const handleAjaxFormSubmit = async (event)") {
		t.Fatal(`handleAjaxFormSubmit must not be an async function; use sync listener + void (async () => { ... })()`)
	}
	if !strings.Contains(s, "const handleAjaxFormSubmit = (event) =>") {
		t.Fatal(`expected sync handleAjaxFormSubmit = (event) =>`)
	}
}
