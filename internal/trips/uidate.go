package trips

import (
	"strings"
	"time"
)

// NormalizeUIDateFormat returns "mdy" (MM-DD-YYYY) or "dmy" (DD-MM-YYYY); unknown values default to dmy.
func NormalizeUIDateFormat(raw string) string {
	s := strings.ToLower(strings.TrimSpace(raw))
	if s == "mdy" {
		return "mdy"
	}
	return "dmy"
}

// UIDateIsMDY reports whether numeric dates should use MM-DD-YYYY order.
func UIDateIsMDY(raw string) bool {
	return NormalizeUIDateFormat(raw) == "mdy"
}

// UIDateNumericLayout returns the Go time layout for formatting a calendar date (from YYYY-MM-DD) for UI.
func UIDateNumericLayout(raw string) string {
	if UIDateIsMDY(raw) {
		return "01-02-2006"
	}
	return "02-01-2006"
}

// FormatISODate formats a stored YYYY-MM-DD value with the given layout. Empty input yields ""; unparseable input is returned unchanged.
func FormatISODate(iso, layout string) string {
	s := strings.TrimSpace(iso)
	if s == "" {
		return ""
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return iso
	}
	return t.Format(layout)
}
