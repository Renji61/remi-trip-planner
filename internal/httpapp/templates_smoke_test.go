package httpapp

import (
	"bytes"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"remi-trip-planner/internal/trips"
)

func TestDashboardTripCardTemplateRenders(t *testing.T) {
	root := findModuleRoot(t)
	tmpl := template.Must(
		template.New("").
			Funcs(template.FuncMap{
				"formatDateTime":         func(s string) string { return s },
				"formatUIDate":           func(s string) string { return s },
				"formatTripDateTime":     func(_ trips.Trip, s string) string { return s },
				"formatTripClock":        func(_ trips.Trip, s string) string { return s },
				"formatTripDateRange":    func(a, b string) string { return a + "–" + b },
				"formatTripDateShort":    func(a, b string) string { return "Jan 1 – 7" },
				"formatTripMoney":        func(f float64) string { return fmt.Sprintf("%.0f", f) },
				"expenseCategoryStyle":   func(s string) string { return "" },
				"expenseCategoryIcon":    func(s string) string { return "" },
				"listContains":           func(a string, b []string) bool { return false },
				"hasPrefix":              strings.HasPrefix,
				"mainSectionVisible":     func(string, trips.Trip) bool { return true },
				"sidebarWidgetVisible":   func(string, trips.Trip) bool { return true },
				"tripMainSectionLabel":   func(s string) string { return s },
				"tripSidebarWidgetLabel": func(s string) string { return s },
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
		"TravelStats":         trips.TravelStats{MilesDisplay: "0"},
		"DashboardListLayout": false,
		"HeroPatternClass":    "",
		"HeroImageURL":        "",
		"Saved":               false,
		"HasError":            false,
		"ErrorText":           "",
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
