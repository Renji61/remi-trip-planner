package httpapp

import "testing"

func TestAbbrevMoney(t *testing.T) {
	t.Parallel()
	cases := []struct {
		symbol string
		v      float64
		want   string
	}{
		{"$", 0, "$0.00"},
		{"$", 123.45, "$123.45"},
		{"$", 99999.99, "$99999.99"},
		{"$", 100000, "$100.00k"},
		{"₱", 1500000, "₱1.50M"},
		{"$", -250000, "$-250.00k"},
	}
	for _, tc := range cases {
		got := abbrevMoney(tc.symbol, tc.v)
		if got != tc.want {
			t.Errorf("abbrevMoney(%q, %v) = %q, want %q", tc.symbol, tc.v, got, tc.want)
		}
	}
}
