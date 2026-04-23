package trips

import (
	"fmt"
	"strconv"
	"strings"
	"time"
)

// AirportTimezoneLocation maps IANA timezone names (e.g. "Europe/London") or offset strings (e.g. "+01:00") to a *time.Location.
func AirportTimezoneLocation(s string) *time.Location {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	if loc, err := time.LoadLocation(s); err == nil {
		return loc
	}
	if len(s) >= 3 && (s[0] == '+' || s[0] == '-') {
		sign := 1
		if s[0] == '-' {
			sign = -1
		}
		rest := s[1:]
		parts := strings.Split(rest, ":")
		h, errH := strconv.Atoi(parts[0])
		if errH != nil {
			return nil
		}
		mi := 0
		if len(parts) > 1 {
			mi, _ = strconv.Atoi(parts[1])
		}
		sec := sign * (h*3600 + mi*60)
		return time.FixedZone(s, sec)
	}
	return nil
}

func formatDurationWords(d time.Duration) string {
	if d < 0 {
		return ""
	}
	total := int(d.Round(time.Minute).Minutes())
	if total < 0 {
		return ""
	}
	h := total / 60
	m := total % 60
	if h > 0 {
		return fmt.Sprintf("%dh %02dm", h, m)
	}
	return fmt.Sprintf("%dm", m)
}

// FormatFlightDurationDisplay returns a label like "10h 30m" when both datetimes parse and timezones are known;
// otherwise it uses naive local difference between the stored clock values.
func FormatFlightDurationDisplay(depAt, arrAt, depTZ, arrTZ string) string {
	depAt = strings.TrimSpace(depAt)
	arrAt = strings.TrimSpace(arrAt)
	if depAt == "" || arrAt == "" {
		return ""
	}
	tDep, err1 := time.Parse("2006-01-02T15:04", depAt)
	tArr, err2 := time.Parse("2006-01-02T15:04", arrAt)
	if err1 != nil || err2 != nil {
		return ""
	}
	depLoc := AirportTimezoneLocation(depTZ)
	arrLoc := AirportTimezoneLocation(arrTZ)
	if depLoc != nil && arrLoc != nil {
		u1 := time.Date(tDep.Year(), tDep.Month(), tDep.Day(), tDep.Hour(), tDep.Minute(), 0, 0, depLoc)
		u2 := time.Date(tArr.Year(), tArr.Month(), tArr.Day(), tArr.Hour(), tArr.Minute(), 0, 0, arrLoc)
		d := u2.Sub(u1)
		return formatDurationWords(d)
	}
	d := tArr.Sub(tDep)
	return formatDurationWords(d)
}
