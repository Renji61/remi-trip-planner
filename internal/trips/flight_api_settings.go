package trips

import "strings"

// IsFlightAPIActive reports whether the AirLabs API key is configured (decrypted, non-empty).
// This is the Go equivalent of is_flight_api_active().
func IsFlightAPIActive(s AppSettings) bool {
	return strings.TrimSpace(s.AirLabsAPIKey) != ""
}
