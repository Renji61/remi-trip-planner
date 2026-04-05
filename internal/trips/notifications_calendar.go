package trips

import "time"

// AppNotification is an in-app reminder delivered to a user (bell / notifications page).
type AppNotification struct {
	ID        string
	UserID    string
	TripID    string
	Title     string
	Body      string
	Href      string
	Kind      string
	DedupeKey string
	ReadAt    time.Time
	CreatedAt time.Time
}

// ItineraryCustomReminder fires relative to the itinerary stop’s start date/time.
type ItineraryCustomReminder struct {
	ID                 string
	TripID             string
	ItineraryItemID    string
	MinutesBeforeStart int
	Label              string
	CreatedAt          time.Time
}
