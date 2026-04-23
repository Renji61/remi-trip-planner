package trips

import "time"

// Airport is a cached row from AirLabs / maps for flight airport fields.
type Airport struct {
	IATACode    string
	ICAOCode    string
	Name        string
	City        string
	CountryCode string
	Timezone    string
	LastUpdated time.Time
}
