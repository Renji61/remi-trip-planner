package version

import (
	"strconv"
	"strings"
)

// Version is the semver of this build (no "v" prefix). Bump when cutting a release.
const Version = "1.40.0"

// GitHubRepo is "owner/repo" for public release checks (same as upstream).
const GitHubRepo = "Renji61/remi-trip-planner"

// Normalize strips a leading "v" from a tag or version string.
func Normalize(s string) string {
	return strings.TrimPrefix(strings.TrimSpace(s), "v")
}

// Compare returns +1 if a > b, -1 if a < b, 0 if equal (by major.minor.patch only).
func Compare(a, b string) int {
	pa := parseParts(a)
	pb := parseParts(b)
	for i := 0; i < 3; i++ {
		if pa[i] > pb[i] {
			return 1
		}
		if pa[i] < pb[i] {
			return -1
		}
	}
	return 0
}

func parseParts(s string) [3]int {
	s = Normalize(s)
	parts := strings.Split(s, ".")
	var out [3]int
	for i := 0; i < 3; i++ {
		if i < len(parts) {
			// ignore non-numeric suffixes like "1.0.0-beta"
			num := parts[i]
			for j := 0; j < len(num); j++ {
				if num[j] < '0' || num[j] > '9' {
					num = num[:j]
					break
				}
			}
			out[i], _ = strconv.Atoi(num)
		}
	}
	return out
}
