package trips

// DefaultAppSettings returns factory defaults for the single app_settings row.
func DefaultAppSettings() AppSettings {
	return AppSettings{
		AppTitle:                "REMI Trip Planner",
		TripDashboardHeading:    "Trip Dashboard",
		DefaultCurrencyName:     "USD",
		DefaultCurrencySymbol:   "$",
		MapDefaultLatitude:      14.5995,
		MapDefaultLongitude:     120.9842,
		MapDefaultZoom:          6,
		EnableLocationLookup:    true,
		RegistrationEnabled:     true,
		ThemePreference:         "system",
		DashboardTripLayout:     "grid",
		DashboardTripSort:       "name",
		DashboardHeroBackground: "default",
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
	t.UIItineraryExpand = "first"
	t.UISpendsExpand = "first"
	t.UITimeFormat = "12h"
	t.UILabelStay = ""
	t.UILabelVehicle = ""
	t.UILabelFlights = ""
	t.UILabelSpends = ""
	t.UIMainSectionOrder = ""
	t.UISidebarWidgetOrder = ""
	t.UIMainSectionHidden = ""
	t.UISidebarWidgetHidden = ""
	t.UIShowCustomLinks = true
	t.UICustomSidebarLinks = ""
}
