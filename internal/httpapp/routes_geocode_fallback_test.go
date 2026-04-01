package httpapp

import (
	"testing"

	"remi-trip-planner/internal/trips"
)

func TestFallbackItineraryCoordsOnGeocodeMiss(t *testing.T) {
	t.Run("keeps geocoded coords when available", func(t *testing.T) {
		gotLat, gotLng := fallbackItineraryCoordsOnGeocodeMiss(14.5995, 120.9842, trips.ItineraryItem{
			Latitude:  35.6762,
			Longitude: 139.6503,
		})
		if gotLat != 14.5995 || gotLng != 120.9842 {
			t.Fatalf("got (%v,%v), want geocoded coords", gotLat, gotLng)
		}
	})

	t.Run("reuses existing coords when geocode misses", func(t *testing.T) {
		gotLat, gotLng := fallbackItineraryCoordsOnGeocodeMiss(0, 0, trips.ItineraryItem{
			Latitude:  35.6762,
			Longitude: 139.6503,
		})
		if gotLat != 35.6762 || gotLng != 139.6503 {
			t.Fatalf("got (%v,%v), want existing coords", gotLat, gotLng)
		}
	})

	t.Run("keeps zero coords when neither source has data", func(t *testing.T) {
		gotLat, gotLng := fallbackItineraryCoordsOnGeocodeMiss(0, 0, trips.ItineraryItem{})
		if gotLat != 0 || gotLng != 0 {
			t.Fatalf("got (%v,%v), want (0,0)", gotLat, gotLng)
		}
	})
}
