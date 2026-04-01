package httpapp

import (
	"strings"
	"testing"
)

func TestResolveCreateCoordsOrError(t *testing.T) {
	t.Run("uses provided coordinates when present", func(t *testing.T) {
		called := false
		lat, lng, err := resolveCreateCoordsOrError("Manila", 14.5995, 120.9842, func(string) (float64, float64) {
			called = true
			return 0, 0
		}, "Location")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if called {
			t.Fatalf("geocoder should not be called when provided coordinates are present")
		}
		if lat != 14.5995 || lng != 120.9842 {
			t.Fatalf("got (%v,%v), want provided coords", lat, lng)
		}
	})

	t.Run("falls back to geocoder when provided coordinates are missing", func(t *testing.T) {
		lat, lng, err := resolveCreateCoordsOrError("Tokyo", 0, 0, func(string) (float64, float64) {
			return 35.6762, 139.6503
		}, "Location")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if lat != 35.6762 || lng != 139.6503 {
			t.Fatalf("got (%v,%v), want geocoded coords", lat, lng)
		}
	})

	t.Run("returns validation error when location is blank", func(t *testing.T) {
		_, _, err := resolveCreateCoordsOrError("   ", 0, 0, func(string) (float64, float64) {
			return 35.6762, 139.6503
		}, "Location")
		if err == nil {
			t.Fatalf("expected error for blank location")
		}
	})

	t.Run("returns validation error when geocoder cannot resolve location", func(t *testing.T) {
		_, _, err := resolveCreateCoordsOrError("Unknown Place", 0, 0, func(string) (float64, float64) {
			return 0, 0
		}, "Location")
		if err == nil {
			t.Fatalf("expected error when geocoder returns zero coordinates")
		}
		if !strings.Contains(err.Error(), "city/country") {
			t.Fatalf("expected actionable guidance in error, got: %v", err)
		}
	})
}
