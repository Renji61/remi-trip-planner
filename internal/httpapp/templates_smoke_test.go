package httpapp

import (
	"bytes"
	"fmt"
	ht "html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"remi-trip-planner/internal/trips"
)

func TestDashboardTripCardTemplateRenders(t *testing.T) {
	root := findModuleRoot(t)
	tmpl := ht.Must(
		ht.New("").
			Funcs(ht.FuncMap{
				"formatDateTime":                  func(s string) string { return s },
				"formatUIDate":                    func(s string) string { return s },
				"formatTripDateTime":              func(_ trips.Trip, s string) string { return s },
				"formatTripClock":                 func(_ trips.Trip, s string) string { return s },
				"formatTripDateRange":             func(a, b string) string { return a + "–" + b },
				"formatTripDateShort":             func(a, b string) string { return "Jan 1 – 7" },
				"formatTripMoney":                 func(f float64) string { return fmt.Sprintf("%.0f", f) },
				"abbrevMoney":                     func(sym string, f float64) string { return sym + fmt.Sprintf("%.2f", f) },
				"expenseCategoryStyle":            func(s string) string { return "" },
				"expenseCategoryIcon":             func(s string) string { return "" },
				"listContains":                    func(a string, b []string) bool { return false },
				"hasPrefix":                       strings.HasPrefix,
				"mainSectionVisible":              func(string, trips.Trip) bool { return true },
				"tripSectionEnabled":              func(string, trips.Trip) bool { return true },
				"sidebarWidgetVisible":            func(string, trips.Trip) bool { return true },
				"tripMainSectionLabel":            func(s string) string { return s },
				"tripSidebarWidgetLabel":          func(s string) string { return s },
				"tripMainSectionVisibilityIcon":   trips.MainSectionVisibilityIcon,
				"tripSidebarWidgetVisibilityIcon": trips.SidebarWidgetVisibilityIcon,
				"googleMapsSearchURL": func(lat, lng float64, hint string) string {
					return ""
				},
				"locationLineBeforeComma": func(s string) string { return s },
				"itineraryGeocodeQuery":   func(any) string { return "" },
				"profileInitial": func(u trips.User) string {
					p := trips.UserProfile{DisplayName: u.DisplayName, Username: u.Username, Email: u.Email}
					return p.InitialForAvatar()
				},
				"profileAvatarURL": func(u trips.User) string { return "" },
				"sub":              func(a, b int) int { return a - b },
			}).
			ParseGlob(filepath.Join(root, "web", "templates", "*.html")),
	)

	card := dashboardTripCard{
		Trip: trips.Trip{
			ID:             "t1",
			Name:           "Test Trip",
			Description:    "Fun",
			StartDate:      "2026-04-01",
			EndDate:        "2026-04-10",
			CurrencySymbol: "$",
		},
		BudgetTotal:           100,
		BudgetPercent:         50,
		StatusLabel:           "Upcoming",
		StatusSlug:            "upcoming",
		TripSubtitle:          "General",
		HasValidSchedule:      true,
		ScheduleDurationLabel: "10 Days",
		DashboardListLayout:   false,
	}

	data := map[string]any{
		"ActiveTripCards":   []dashboardTripCard{card},
		"SharedTripCards":   []dashboardTripCard(nil),
		"DraftTripCards":    []dashboardTripCard(nil),
		"ArchivedTripCards": []dashboardTripCard(nil),
		"Settings": trips.AppSettings{
			AppTitle:              "App",
			TripDashboardHeading:  "TD",
			DefaultCurrencyName:   "USD",
			DefaultCurrencySymbol: "$",
			ThemePreference:       "light",
			DashboardTripLayout:   "grid",
		},
		"TravelStats":            trips.TravelStats{MilesDisplay: "0"},
		"DashboardListLayout":    false,
		"HeroPatternClass":       "",
		"HeroImageURL":           "",
		"Saved":                  false,
		"HasError":               false,
		"ErrorText":              "",
		"CSRFToken":              "test-csrf",
		"CurrentUser":            trips.User{DisplayName: "Test", Username: "test"},
		"SidebarNavActive":       "home",
		"SidebarInProgressTrips": []sidebarInProgressTrip{{ID: "live", Name: "Live Trip", DateRange: "Jan 1 – 7"}},
		"SidebarTripID":          "",
		"TripID":                 "",
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "home.html", data); err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `class="trip-card"`) {
		t.Fatalf("expected trip-card in output, got len=%d snippet=%q", len(out), truncate(out, 400))
	}

	card.DashboardListLayout = true
	data["ActiveTripCards"] = []dashboardTripCard{card}
	data["DashboardListLayout"] = true
	buf.Reset()
	if err := tmpl.ExecuteTemplate(&buf, "home.html", data); err != nil {
		t.Fatalf("execute list layout: %v", err)
	}
	out = buf.String()
	if !strings.Contains(out, `trip-card--list-row`) {
		t.Fatalf("expected list row card, got snippet=%q", truncate(out, 500))
	}
	if !strings.Contains(out, `trip-list-row-mobile`) {
		t.Fatalf("expected mobile list row markup in output")
	}
}

func TestAboutPageTemplateRenders(t *testing.T) {
	root := findModuleRoot(t)
	tmpl := ht.Must(
		ht.New("").
			Funcs(ht.FuncMap{
				"hasPrefix":                       strings.HasPrefix,
				"formatDateTime":                  func(s string) string { return s },
				"formatUIDate":                    func(s string) string { return s },
				"formatTripDateTime":              func(_ trips.Trip, s string) string { return s },
				"formatTripClock":                 func(_ trips.Trip, s string) string { return s },
				"formatTripDateRange":             func(a, b string) string { return a + "–" + b },
				"formatTripDateShort":             func(a, b string) string { return "Jan 1 – 7" },
				"formatTripMoney":                 func(f float64) string { return fmt.Sprintf("%.0f", f) },
				"abbrevMoney":                     func(sym string, f float64) string { return sym + fmt.Sprintf("%.2f", f) },
				"expenseCategoryStyle":            func(s string) string { return "" },
				"expenseCategoryIcon":             func(s string) string { return "" },
				"listContains":                    func(a string, b []string) bool { return false },
				"mainSectionVisible":              func(string, trips.Trip) bool { return true },
				"tripSectionEnabled":              func(string, trips.Trip) bool { return true },
				"sidebarWidgetVisible":            func(string, trips.Trip) bool { return true },
				"tripMainSectionLabel":            func(s string) string { return s },
				"tripSidebarWidgetLabel":          func(s string) string { return s },
				"tripMainSectionVisibilityIcon":   trips.MainSectionVisibilityIcon,
				"tripSidebarWidgetVisibilityIcon": trips.SidebarWidgetVisibilityIcon,
				"googleMapsSearchURL":             func(lat, lng float64, hint string) string { return "" },
				"locationLineBeforeComma":         func(s string) string { return s },
				"itineraryGeocodeQuery":           func(any) string { return "" },
				"profileInitial": func(u trips.User) string {
					p := trips.UserProfile{DisplayName: u.DisplayName, Username: u.Username, Email: u.Email}
					return p.InitialForAvatar()
				},
				"profileAvatarURL": func(u trips.User) string { return "" },
				"sub":              func(a, b int) int { return a - b },
			}).
			ParseGlob(filepath.Join(root, "web", "templates", "*.html")),
	)
	data := map[string]any{
		"Settings": trips.AppSettings{
			AppTitle:             "App",
			ThemePreference:      "light",
			TripDashboardHeading: "Trips",
		},
		"ClearThemeOverride":     false,
		"CSRFToken":              "test-csrf",
		"CurrentUser":            trips.User{DisplayName: "Test"},
		"AppVersion":             "9.9.9",
		"ChangelogHTML":          ht.HTML("<p class=\"t\">Note</p>"),
		"SidebarNavActive":       "about",
		"SidebarInProgressTrips": []sidebarInProgressTrip(nil),
		"SidebarTripID":          "",
		"TripID":                 "",
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "about.html", data); err != nil {
		t.Fatalf("execute about: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "9.9.9") || !strings.Contains(out, "about-check-update-btn") {
		t.Fatalf("unexpected about output: %s", truncate(out, 500))
	}
}

func findModuleRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	dir := wd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatal("go.mod not found")
		}
		dir = parent
	}
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "…"
}
