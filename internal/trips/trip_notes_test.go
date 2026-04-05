package trips

import "testing"

func TestNormalizeKeepView(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", KeepViewNotes},
		{"notes", KeepViewNotes},
		{"  REMINDERS ", KeepViewReminders},
		{"Archive", KeepViewArchive},
		{"trash", KeepViewTrash},
		{"unknown", KeepViewNotes},
	}
	for _, tc := range tests {
		if got := NormalizeKeepView(tc.in); got != tc.want {
			t.Errorf("NormalizeKeepView(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}
