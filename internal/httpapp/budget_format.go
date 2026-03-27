package httpapp

import (
	"fmt"
	"math"
)

const abbrevMoneyThreshold = 100_000

// abbrevMoney formats amounts for compact UI: full precision below abbrevMoneyThreshold,
// otherwise uses k / M / B / T suffixes (always with symbol prefix).
func abbrevMoney(symbol string, v float64) string {
	abs := math.Abs(v)
	if abs < abbrevMoneyThreshold && abs > 0 || abs == 0 {
		return fmt.Sprintf("%s%.2f", symbol, v)
	}
	sign := 1.0
	if v < 0 {
		sign = -1
	}
	x := abs
	var div float64
	var suf string
	switch {
	case x >= 1e12:
		div, suf = 1e12, "T"
	case x >= 1e9:
		div, suf = 1e9, "B"
	case x >= 1e6:
		div, suf = 1e6, "M"
	case x >= 1e3:
		div, suf = 1e3, "k"
	default:
		return fmt.Sprintf("%s%.2f", symbol, v)
	}
	val := sign * x / div
	return fmt.Sprintf("%s%.2f%s", symbol, val, suf)
}
