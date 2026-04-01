package httpapp

import "testing"

func TestItineraryNotesDisplay(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"", ""},
		{"Hello", "Hello"},
		{"Attachment: /static/uploads/bookings/x.png", ""},
		{"  Attachment: /static/x.png  ", ""},
		{"Notes here | Attachment: /static/uploads/a.pdf", "Notes here"},
		{"Booking: ABC | Attachment: /static/x.png", "Booking: ABC"},
		{"A | B | Attachment: /y.png", "A | B"},
		{"Line1\nAttachment: /z.png", "Line1"},
	}
	for _, tc := range tests {
		if got := itineraryNotesDisplay(tc.in); got != tc.want {
			t.Errorf("itineraryNotesDisplay(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}
