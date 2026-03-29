package httpapp

import "testing"

func TestIsSafeSiteSettingsReturn(t *testing.T) {
	tests := []struct {
		raw  string
		want bool
	}{
		{"", false},
		{"/settings", true},
		{"/settings?trip_id=x", true},
		{"/profile", true},
		{"//evil.example/phish", false},
		{"///evil.example/phish", false},
		{"https://evil.example/", false},
		{"http://evil.example/", false},
		{"evil.example", false},
		{"/path@host", false},
		{"/path\\evil", false},
		{"   ", false},
		{" //evil.example/x", false},
		{"/", true},
	}
	for _, tt := range tests {
		if got := isSafeSiteSettingsReturn(tt.raw); got != tt.want {
			t.Errorf("isSafeSiteSettingsReturn(%q) = %v, want %v", tt.raw, got, tt.want)
		}
	}
}
