package httpapp

import (
	"strings"
	"testing"
)

func TestGoogleMapsMultiPlaceURL(t *testing.T) {
	t.Parallel()
	if got := googleMapsMultiPlaceURL(nil); got != "" {
		t.Fatalf("empty: got %q", got)
	}
	if got := googleMapsMultiPlaceURL([]string{"", "  "}); got != "" {
		t.Fatalf("all blank: got %q", got)
	}
	one := googleMapsMultiPlaceURL([]string{"40.7128,-74.006"})
	if !strings.Contains(one, "google.com/maps/search") || !strings.Contains(one, "40.7128") {
		t.Fatalf("single coords: %q", one)
	}
	oneText := googleMapsMultiPlaceURL([]string{"Seattle, WA"})
	if !strings.Contains(oneText, "google.com/maps/search") || !strings.Contains(oneText, "Seattle") {
		t.Fatalf("single text: %q", oneText)
	}
	multi := googleMapsMultiPlaceURL([]string{"40.7,-74.0", "34.0,-118.2"})
	if !strings.HasPrefix(multi, "https://www.google.com/maps/dir/") {
		t.Fatalf("multi prefix: %q", multi)
	}
	if !strings.Contains(multi, "40.7%2C-74") && !strings.Contains(multi, "40.7,-74") {
		t.Fatalf("multi should contain first waypoint: %q", multi)
	}
}
