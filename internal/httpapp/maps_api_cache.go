package httpapp

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"sync"
	"time"
)

// In-memory caches for Google Geocoding / Places (and Nominatim fallback) to cut
// repeat HTTP calls for identical queries across users and requests.

const (
	geocodeGoogleSuccessTTL = 7 * 24 * time.Hour
	geocodeGoogleMissTTL    = 20 * time.Minute
	geocodeNominatimTTL     = 24 * time.Hour
	placeSuggestTTL         = 6 * time.Hour
	placeDetailTTL          = 30 * 24 * time.Hour
	nominatimSuggestTTL     = 1 * time.Hour

	mapsCacheMaxGeocode = 2048
	mapsCacheMaxSuggest = 384
	mapsCacheMaxPlace   = 4096
)

type mapsAPIKeyTag string

func mapsAPIKeyTagFrom(key string) mapsAPIKeyTag {
	sum := sha256.Sum256([]byte(strings.TrimSpace(key)))
	return mapsAPIKeyTag(hex.EncodeToString(sum[:8]))
}

type geoCacheEntry struct {
	lat, lng float64
	neg      bool // true = cached "no coordinates" (do not call upstream)
	expires  time.Time
}

type suggestCacheEntry struct {
	items   []locationSuggestion
	expires time.Time
}

type placeDetailCacheEntry struct {
	lat, lng float64
	display  string
	expires  time.Time
}

type mapsLocationCache struct {
	mu sync.Mutex

	geocode map[string]geoCacheEntry
	suggest map[string]suggestCacheEntry
	place   map[string]placeDetailCacheEntry
}

var globalMapsLocationCache = &mapsLocationCache{
	geocode: make(map[string]geoCacheEntry),
	suggest: make(map[string]suggestCacheEntry),
	place:   make(map[string]placeDetailCacheEntry),
}

func normalizeMapsQuery(q string) string {
	return strings.ToLower(strings.TrimSpace(q))
}

func (c *mapsLocationCache) geoKeyGoogle(tag mapsAPIKeyTag, lang, q string) string {
	return "g:" + string(tag) + ":" + normalizeMapsQuery(lang) + ":" + normalizeMapsQuery(q)
}

func (c *mapsLocationCache) geoKeyNominatim(lang, q string) string {
	return "n:" + normalizeMapsQuery(lang) + ":" + normalizeMapsQuery(q)
}

func (c *mapsLocationCache) suggestKeyGoogle(tag mapsAPIKeyTag, lang, q string) string {
	return "gs:" + string(tag) + ":" + normalizeMapsQuery(lang) + ":" + normalizeMapsQuery(q)
}

func (c *mapsLocationCache) suggestKeyNominatim(lang, q string) string {
	return "ns:" + normalizeMapsQuery(lang) + ":" + normalizeMapsQuery(q)
}

func (c *mapsLocationCache) placeKey(tag mapsAPIKeyTag, placeID, lang string) string {
	return "p:" + string(tag) + ":" + strings.TrimSpace(placeID) + ":" + normalizeMapsQuery(lang)
}

func (c *mapsLocationCache) geoGet(key string) (lat, lng float64, hit bool, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, found := c.geocode[key]
	if !found || time.Now().After(e.expires) {
		if found {
			delete(c.geocode, key)
		}
		return 0, 0, false, false
	}
	if e.neg {
		return 0, 0, false, true
	}
	return e.lat, e.lng, true, true
}

func (c *mapsLocationCache) geoSetNegative(key string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.geocode[key] = geoCacheEntry{neg: true, expires: time.Now().Add(ttl)}
	c.trimGeocodeLocked()
}

func (c *mapsLocationCache) geoSetCoords(key string, lat, lng float64, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.geocode[key] = geoCacheEntry{lat: lat, lng: lng, neg: false, expires: time.Now().Add(ttl)}
	c.trimGeocodeLocked()
}

func (c *mapsLocationCache) trimGeocodeLocked() {
	c.purgeExpiredGeocodeLocked()
	for len(c.geocode) > mapsCacheMaxGeocode {
		for k := range c.geocode {
			delete(c.geocode, k)
			break
		}
	}
}

func (c *mapsLocationCache) purgeExpiredGeocodeLocked() {
	now := time.Now()
	for k, v := range c.geocode {
		if now.After(v.expires) {
			delete(c.geocode, k)
		}
	}
}

func (c *mapsLocationCache) suggestGet(key string) ([]locationSuggestion, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, found := c.suggest[key]
	if !found || time.Now().After(e.expires) {
		if found {
			delete(c.suggest, key)
		}
		return nil, false
	}
	out := make([]locationSuggestion, len(e.items))
	copy(out, e.items)
	return out, true
}

func (c *mapsLocationCache) suggestSet(key string, items []locationSuggestion, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	stored := make([]locationSuggestion, len(items))
	copy(stored, items)
	c.suggest[key] = suggestCacheEntry{items: stored, expires: time.Now().Add(ttl)}
	c.purgeExpiredSuggestLocked()
	for len(c.suggest) > mapsCacheMaxSuggest {
		for k := range c.suggest {
			delete(c.suggest, k)
			break
		}
	}
}

func (c *mapsLocationCache) purgeExpiredSuggestLocked() {
	now := time.Now()
	for k, v := range c.suggest {
		if now.After(v.expires) {
			delete(c.suggest, k)
		}
	}
}

func (c *mapsLocationCache) placeGet(key string) (lat, lng float64, display string, ok bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	e, found := c.place[key]
	if !found || time.Now().After(e.expires) {
		if found {
			delete(c.place, key)
		}
		return 0, 0, "", false
	}
	return e.lat, e.lng, e.display, true
}

func (c *mapsLocationCache) placeSet(key string, lat, lng float64, display string, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.place[key] = placeDetailCacheEntry{
		lat: lat, lng: lng, display: strings.TrimSpace(display),
		expires: time.Now().Add(ttl),
	}
	c.purgeExpiredPlaceLocked()
	for len(c.place) > mapsCacheMaxPlace {
		for k := range c.place {
			delete(c.place, k)
			break
		}
	}
}

func (c *mapsLocationCache) purgeExpiredPlaceLocked() {
	now := time.Now()
	for k, v := range c.place {
		if now.After(v.expires) {
			delete(c.place, k)
		}
	}
}
