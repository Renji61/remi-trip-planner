package trips

// DefaultAppSettings returns factory defaults for the single app_settings row.
func DefaultAppSettings() AppSettings {
	return AppSettings{
		AppTitle:                "REMI Trip Planner",
		TripDashboardHeading:    "Trip Dashboard",
		DefaultCurrencyName:     "USD",
		DefaultCurrencySymbol:   "$",
		MapDefaultPlaceLabel:    DefaultMapPlaceLabel,
		MapDefaultLatitude:      DefaultMapLatitude,
		MapDefaultLongitude:     DefaultMapLongitude,
		MapDefaultZoom:          6,
		EnableLocationLookup:    true,
		RegistrationEnabled:     true,
		ThemePreference:         "system",
		DashboardTripLayout:     "grid",
		DashboardTripSort:       "name",
		DashboardHeroBackground: "default",
		DefaultDistanceUnit:     "km",
		GoogleMapsAPIKey:        "",
		GoogleMapsMapID:         "",
		AirLabsAPIKey:           "",
		OpenWeatherAPIKey:       "",
		MaxUploadFileSizeMB:     5,
		DefaultUIDateFormat:     "dmy",
	}
}

// DefaultUserUISettings returns per-account UI defaults aligned with DefaultAppSettings.
func DefaultUserUISettings(userID string) UserSettings {
	app := DefaultAppSettings()
	return UserSettings{
		UserID:                  userID,
		ThemePreference:         app.ThemePreference,
		DashboardTripLayout:     app.DashboardTripLayout,
		DashboardTripSort:       app.DashboardTripSort,
		DashboardHeroBackground: app.DashboardHeroBackground,
		TripDashboardHeading:    app.TripDashboardHeading,
		DefaultCurrencyName:     app.DefaultCurrencyName,
		DefaultCurrencySymbol:   app.DefaultCurrencySymbol,
	}
}

// ApplyDefaultTripUIPresets resets trip layout / visibility options to defaults (not name, dates, or currency).
func ApplyDefaultTripUIPresets(t *Trip) {
	t.UIShowStay = true
	t.UIShowVehicle = true
	t.UIShowFlights = true
	t.UIShowSpends = true
	t.UIShowItinerary = true
	t.UIShowChecklist = true
	t.UIShowTheTab = true
	t.UIShowDocuments = true
	t.UICollaborationEnabled = true
	t.UIItineraryExpand = "first"
	t.UISpendsExpand = "first"
	t.UITabExpand = "first"
	t.UITimeFormat = "12h"
	t.UIDateFormat = "inherit"
	t.UILabelStay = ""
	t.UILabelVehicle = ""
	t.UILabelFlights = ""
	t.UILabelSpends = ""
	t.UILabelGroupExpenses = ""
	t.UIMainSectionOrder = ""
	t.UISidebarWidgetOrder = ""
	t.UIMainSectionHidden = ""
	t.UISidebarWidgetHidden = ""
	t.UIShowCustomLinks = true
	t.UICustomSidebarLinks = ""
}
