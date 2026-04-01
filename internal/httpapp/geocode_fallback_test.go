package httpapp

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func testJSONResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Header:     make(http.Header),
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func resetMapsLocationCacheForTest() {
	globalMapsLocationCache.mu.Lock()
	defer globalMapsLocationCache.mu.Unlock()
	globalMapsLocationCache.geocode = make(map[string]geoCacheEntry)
	globalMapsLocationCache.suggest = make(map[string]suggestCacheEntry)
	globalMapsLocationCache.place = make(map[string]placeDetailCacheEntry)
}

func TestGeocodeCoordsFallsBackWhenGoogleFails(t *testing.T) {
	resetMapsLocationCacheForTest()
	originalTransport := http.DefaultTransport
	defer func() {
		http.DefaultTransport = originalTransport
	}()

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		switch req.URL.Host {
		case "maps.googleapis.com":
			return testJSONResponse(http.StatusOK, `{"status":"REQUEST_DENIED","results":[]}`), nil
		case "nominatim.openstreetmap.org":
			return testJSONResponse(http.StatusOK, `[{"lat":"14.6000","lon":"120.9800"}]`), nil
		default:
			t.Fatalf("unexpected host called: %s", req.URL.Host)
			return nil, nil
		}
	})

	lat, lng := geocodeCoords(context.Background(), "Some complete pasted address", "fake-google-key", "en")
	if lat != 14.6 || lng != 120.98 {
		t.Fatalf("expected nominatim fallback coords, got (%v,%v)", lat, lng)
	}
}

func TestGeocodeCoordsRetriesWithNormalizedQuery(t *testing.T) {
	resetMapsLocationCacheForTest()
	originalTransport := http.DefaultTransport
	defer func() {
		http.DefaultTransport = originalTransport
	}()

	http.DefaultTransport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		if req.URL.Host != "nominatim.openstreetmap.org" {
			t.Fatalf("unexpected host called: %s", req.URL.Host)
			return nil, nil
		}
		q := req.URL.Query().Get("q")
		if q == "1600 Amphitheatre Pkwy; Mountain View, CA" {
			return testJSONResponse(http.StatusOK, `[]`), nil
		}
		if q == "1600 Amphitheatre Pkwy, Mountain View, CA" {
			return testJSONResponse(http.StatusOK, `[{"lat":"37.4220","lon":"-122.0841"}]`), nil
		}
		t.Fatalf("unexpected query value: %q", q)
		return nil, nil
	})

	lat, lng := geocodeCoords(context.Background(), "1600 Amphitheatre Pkwy; Mountain View, CA", "", "en")
	if lat != 37.422 || lng != -122.0841 {
		t.Fatalf("expected normalized retry coords, got (%v,%v)", lat, lng)
	}
}
