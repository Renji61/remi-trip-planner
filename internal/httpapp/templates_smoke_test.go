package httpapp

import (
	"bytes"
	"fmt"
	ht "html/template"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"remi-trip-planner/internal/trips"
)

// stubJoinItineraryLocalDateTime mirrors routes.joinItineraryLocalDateTime for template parse smoke tests.
func stubJoinItineraryLocalDateTime(dateISO, timeHM string) string {
	dateISO = strings.TrimSpace(dateISO)
	timeHM = strings.TrimSpace(timeHM)
	if dateISO == "" {
		return ""
	}
	if timeHM == "" {
		timeHM = "09:00"
	}
	if len(timeHM) > 5 {
		timeHM = timeHM[:5]
	}
	return dateISO + "T" + timeHM
}

func TestDashboardTripCardTemplateRenders(t *testing.T) {
	root := findModuleRoot(t)
	tmpl := ht.Must(
		ht.New("").
			Funcs(ht.FuncMap{
				"formatDateTime":       func(s string) string { return s },
				"formatTripUIDate":     func(any, trips.AppSettings, string) string { return "d" },
				"formatTripDateTime":   func(_ trips.Trip, _ trips.AppSettings, s string) string { return s },
				"formatTripClock":      func(_ trips.Trip, s string) string { return s },
				"formatTripDateRange":  func(any, trips.AppSettings, string, string) string { return "a–b" },
				"formatTripDateShort":  func(any, trips.AppSettings, string, string) string { return "Jan 1 – 7" },
				"siteUIDateIsMDY":      func(trips.AppSettings) bool { return false },
				"effectiveUIDateIsMDY": func(trips.Trip, trips.AppSettings) bool { return false },
				"effectiveUITimeIs24h": func(trips.Trip) bool { return false },
				"formatTripMoney":      func(f float64) string { return fmt.Sprintf("%.0f", f) },
				"humanFileSize":        func(_ int64) string { return "1 MB" },
				"abbrevMoney":          func(sym string, f float64) string { return sym + fmt.Sprintf("%.2f", f) },
				"expenseCategoryStyle": func(s string) string { return "" },
				"expenseCategoryIcon":  func(s string) string { return "" },
				"listContains": func(list []string, s string) bool {
					for _, x := range list {
						if x == s {
							return true
						}
					}
					return false
				},
				"hasPrefix":                           strings.HasPrefix,
				"trimSpace":                           strings.TrimSpace,
				"keepNoteBodyPreview":                 keepNoteBodyPreview,
				"keepNoteColorInPicker":               keepNoteColorInPicker,
				"mainSectionVisible":                  func(string, trips.Trip) bool { return true },
				"tripSectionEnabled":                  func(string, trips.Trip) bool { return true },
				"sidebarWidgetVisible":                func(string, trips.Trip) bool { return true },
				"tripMobileFabHasItems":               trips.TripMobileFabHasItems,
				"tripDesktopCalendarFlyoutHasActions": trips.TripDesktopCalendarFlyoutHasActions,
				"tripExpenseQuickAddVisible":          trips.TripExpenseQuickAddVisible,
				"effectiveDistanceUnit": func(trip trips.Trip, settings trips.AppSettings) string {
					return trips.EffectiveDistanceUnit(&trip, settings)
				},
				"tripMainSectionLabel":            func(s string) string { return s },
				"tripSidebarWidgetLabel":          func(s string) string { return s },
				"tripMainSectionVisibilityIcon":   trips.MainSectionVisibilityIcon,
				"tripSidebarWidgetVisibilityIcon": trips.SidebarWidgetVisibilityIcon,
				"googleMapsSearchURL": func(lat, lng float64, hint string) string {
					return ""
				},
				"googleMapsDirectionsURL":    func(fromLat, fromLng, toLat, toLng float64, mode string) string { return "" },
				"itineraryDayGoogleMapsURL":  func([]itineraryItemView) string { return "" },
				"joinItineraryLocalDateTime": stubJoinItineraryLocalDateTime,
				"locationLineBeforeComma":    func(s string) string { return s },
				"itineraryNotesDisplay":      func(s string) string { return s },
				"isImageWebPath":             func(string) bool { return true },
				"itineraryGeocodeQuery":      func(any) string { return "" },
				"profileInitial": func(u trips.User) string {
					p := trips.UserProfile{DisplayName: u.DisplayName, Username: u.Username, Email: u.Email}
					return p.InitialForAvatar()
				},
				"profileAvatarURL": func(u trips.User) string { return "" },
				"sub":              func(a, b int) int { return a - b },
				"dict": func(values ...any) (map[string]any, error) {
					if len(values)%2 != 0 {
						return nil, fmt.Errorf("dict: expected even number of arguments")
					}
					m := make(map[string]any, len(values)/2)
					for i := 0; i < len(values); i += 2 {
						k, ok := values[i].(string)
						if !ok {
							return nil, fmt.Errorf("dict: key at %d must be string", i)
						}
						m[k] = values[i+1]
					}
					return m, nil
				},
				"tabEffectivePaidBy": func(e trips.Expense, ownerID string) string {
					return trips.EffectivePaidBy(e, ownerID)
				},
				"tabSettlementParticipantKey": trips.TabSettlementParticipantKey,
				"tabSplitModeShort": func(mode string) string {
					switch strings.ToLower(strings.TrimSpace(mode)) {
					case trips.TabSplitEqual, "":
						return "Equal"
					case trips.TabSplitExact:
						return "Exact"
					case trips.TabSplitPercent:
						return "Percent"
					case trips.TabSplitShares:
						return "Shares"
					default:
						return mode
					}
				},
				"add":  func(a, b int) int { return a + b },
				"addF": func(a, b float64) float64 { return a + b },
				"mod": func(a, b int) int {
					if b == 0 {
						return 0
					}
					return a % b
				},
				"guestInitial":  trips.GuestInitialFromDisplayName,
				"tabPayerThumb": tabPayerThumb,
				"tabAvatarURL": func(s string) string {
					s = strings.TrimSpace(s)
					if s == "" {
						return ""
					}
					if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
						return s
					}
					if strings.HasPrefix(s, "/") {
						return s
					}
					return "/" + s
				},
				"tabSplitMethodBadgeClass": func(mode string) string {
					switch strings.ToLower(strings.TrimSpace(mode)) {
					case trips.TabSplitEqual, "":
						return "tab-split-method-badge--equal"
					default:
						return "tab-split-method-badge--neutral"
					}
				},
				"tabTabQueryString": func(balanceView, q, tabCat string) string {
					v := url.Values{}
					bv := strings.ToLower(strings.TrimSpace(balanceView))
					if bv == "debts" {
						v.Set("balance_view", "debts")
					}
					if strings.TrimSpace(q) != "" {
						v.Set("q", strings.TrimSpace(q))
					}
					if strings.TrimSpace(tabCat) != "" {
						v.Set("tab_cat", strings.TrimSpace(tabCat))
					}
					s := v.Encode()
					if s == "" {
						return ""
					}
					return "?" + s
				},
				"urlQueryEscape": func(s string) string { return url.QueryEscape(s) },
			}).
			ParseGlob(filepath.Join(root, "web", "templates", "*.html")),
	)

	card := dashboardTripCard{
		Trip: trips.Trip{
			ID:                     "t1",
			Name:                   "Test Trip",
			Description:            "Fun",
			StartDate:              "2026-04-01",
			EndDate:                "2026-04-10",
			CurrencySymbol:         "$",
			UICollaborationEnabled: true,
		},
		BudgetTotal:           100,
		BudgetPercent:         50,
		StatusLabel:           "Upcoming",
		StatusSlug:            "upcoming",
		TripSubtitle:          "General",
		HasValidSchedule:      true,
		ScheduleDurationLabel: "10 Days",
		DashboardListLayout:   false,
		DashboardCSRF:         "csrf-test",
		SiteDateSettings: trips.AppSettings{
			DefaultUIDateFormat: "dmy",
		},
	}

	data := map[string]any{
		"ActiveTripCards":    []dashboardTripCard{card},
		"SharedTripCards":    []dashboardTripCard(nil),
		"DraftTripCards":     []dashboardTripCard(nil),
		"CompletedTripCards": []dashboardTripCard(nil),
		"ArchivedTripCards":  []dashboardTripCard(nil),
		"Settings": trips.AppSettings{
			AppTitle:              "App",
			TripDashboardHeading:  "TD",
			DefaultCurrencyName:   "USD",
			DefaultCurrencySymbol: "$",
			ThemePreference:       "light",
			DashboardTripLayout:   "grid",
		},
		"TravelStats":             trips.TravelStats{MilesDisplay: "0"},
		"TravelDistanceDisplay":   "0 km",
		"HomeDistanceUnit":        "km",
		"DashboardListLayout":     false,
		"HeroPatternClass":        "",
		"HeroImageURL":            "",
		"Saved":                   false,
		"HasError":                false,
		"ErrorText":               "",
		"CSRFToken":               "test-csrf",
		"CurrentUser":             trips.User{DisplayName: "Test", Username: "test"},
		"SidebarNavActive":        "home",
		"SidebarInProgressTrips":  []sidebarInProgressTrip{{ID: "live", Name: "Live Trip", DateRange: "Jan 1 – 7"}},
		"SidebarTripID":           "",
		"TripID":                  "",
		"NotificationUnreadCount": 0,
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "home.html", data); err != nil {
		t.Fatalf("execute: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `class="trip-card"`) {
		t.Fatalf("expected trip-card in output, got len=%d snippet=%q", len(out), truncate(out, 400))
	}

	card.Party = []trips.UserProfile{
		{DisplayName: "One"},
		{DisplayName: "Two"},
		{DisplayName: "Three"},
		{DisplayName: "Four"},
	}
	card.ViewerIsOwner = true
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
	if !strings.Contains(out, `trip-list-mobile-party`) {
		t.Fatalf("expected mobile list party strip in output")
	}
	if strings.Count(out, ">+1<") < 2 {
		t.Fatalf("expected +1 on desktop and mobile list party (4 members, cap 3), snippet=%q", truncate(out, 900))
	}
}

func TestTripMembersPanelOverflowChip(t *testing.T) {
	root := findModuleRoot(t)
	tmpl := ht.Must(
		ht.New("").
			Funcs(ht.FuncMap{
				"add":          func(a, b int) int { return a + b },
				"sub":          func(a, b int) int { return a - b },
				"guestInitial": trips.GuestInitialFromDisplayName,
			}).
			ParseFiles(filepath.Join(root, "web", "templates", "trip_members_panel.html")),
	)
	data := map[string]any{
		"Details": trips.TripDetails{
			Trip: trips.Trip{ID: "t1", OwnerUserID: "o1", UICollaborationEnabled: true},
		},
		"TripAccess": trips.TripAccess{IsOwner: true},
		"CSRFToken":  "csrf-test",
		"Party": []trips.UserProfile{
			{ID: "o1", DisplayName: "Owner"},
			{ID: "u2", DisplayName: "Two"},
			{ID: "u3", DisplayName: "Three"},
			{ID: "u4", DisplayName: "Four"},
			{ID: "u5", DisplayName: "Five"},
			{ID: "u6", DisplayName: "Six"},
		},
		"PendingInvites": []trips.TripInvitePending(nil),
		"TripGuests":     []trips.TripGuest(nil),
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "tripMembersPanel", data); err != nil {
		t.Fatalf("execute tripMembersPanel: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `trip-party-avatar--more`) || !strings.Contains(out, ">+1<") {
		t.Fatalf("expected +1 overflow for 6 party members (cap 5), got: %s", truncate(out, 700))
	}
}

func TestTripMembersPanelHiddenWhenSoloNonOwner(t *testing.T) {
	root := findModuleRoot(t)
	tmpl := ht.Must(
		ht.New("").
			Funcs(ht.FuncMap{
				"add":          func(a, b int) int { return a + b },
				"sub":          func(a, b int) int { return a - b },
				"guestInitial": trips.GuestInitialFromDisplayName,
			}).
			ParseFiles(filepath.Join(root, "web", "templates", "trip_members_panel.html")),
	)
	data := map[string]any{
		"Details": trips.TripDetails{
			Trip: trips.Trip{ID: "t1", OwnerUserID: "o1"},
		},
		"TripAccess":     trips.TripAccess{IsOwner: false},
		"CSRFToken":      "csrf-test",
		"Party":          []trips.UserProfile{{ID: "u1", DisplayName: "Solo"}},
		"PendingInvites": []trips.TripInvitePending(nil),
		"TripGuests":     []trips.TripGuest(nil),
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "tripMembersPanel", data); err != nil {
		t.Fatalf("execute tripMembersPanel: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "sidebar-trip-members-wrap") || strings.Contains(out, "Trip Members") {
		t.Fatalf("expected empty panel for solo non-owner, got: %s", truncate(out, 500))
	}
}

func TestTripMembersPanelInviteOnlyWhenSoloOwner(t *testing.T) {
	root := findModuleRoot(t)
	tmpl := ht.Must(
		ht.New("").
			Funcs(ht.FuncMap{
				"add":          func(a, b int) int { return a + b },
				"sub":          func(a, b int) int { return a - b },
				"guestInitial": trips.GuestInitialFromDisplayName,
			}).
			ParseFiles(filepath.Join(root, "web", "templates", "trip_members_panel.html")),
	)
	data := map[string]any{
		"Details": trips.TripDetails{
			Trip: trips.Trip{ID: "t1", OwnerUserID: "o1", UICollaborationEnabled: true},
		},
		"TripAccess":     trips.TripAccess{IsOwner: true},
		"CSRFToken":      "csrf-test",
		"Party":          []trips.UserProfile{{ID: "o1", DisplayName: "Owner"}},
		"PendingInvites": []trips.TripInvitePending(nil),
		"TripGuests":     []trips.TripGuest(nil),
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "tripMembersPanel", data); err != nil {
		t.Fatalf("execute tripMembersPanel: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `data-trip-invite-methods`) {
		t.Fatalf("expected invite block for solo owner, got: %s", truncate(out, 500))
	}
	if strings.Contains(out, "sidebar-trip-members-label") || strings.Contains(out, ">Trip Members<") {
		t.Fatalf("did not expect Trip Members strip for solo owner, got: %s", truncate(out, 500))
	}
	if !strings.Contains(out, "sidebar-trip-members-wrap--invite-only") {
		t.Fatalf("expected invite-only modifier class, got: %s", truncate(out, 500))
	}
}

// htmlTemplateSmokeFuncs is shared by template smoke tests that ParseGlob all HTML (about, trip notes, etc.).
func htmlTemplateSmokeFuncs() ht.FuncMap {
	return ht.FuncMap{
		"hasPrefix":             strings.HasPrefix,
		"trimSpace":             strings.TrimSpace,
		"keepNoteBodyPreview":   keepNoteBodyPreview,
		"keepNoteColorInPicker": keepNoteColorInPicker,
		"formatDateTime":        func(s string) string { return s },
		"formatTripUIDate":      func(any, trips.AppSettings, string) string { return "d" },
		"formatTripDateTime":    func(_ trips.Trip, _ trips.AppSettings, s string) string { return s },
		"formatTripClock":       func(_ trips.Trip, s string) string { return s },
		"formatTripDateRange":   func(any, trips.AppSettings, string, string) string { return "a–b" },
		"formatTripDateShort":   func(any, trips.AppSettings, string, string) string { return "Jan 1 – 7" },
		"siteUIDateIsMDY":       func(trips.AppSettings) bool { return false },
		"effectiveUIDateIsMDY":  func(trips.Trip, trips.AppSettings) bool { return false },
		"effectiveUITimeIs24h":  func(trips.Trip) bool { return false },
		"formatTripMoney":       func(f float64) string { return fmt.Sprintf("%.0f", f) },
		"humanFileSize":         func(_ int64) string { return "1 MB" },
		"abbrevMoney":           func(sym string, f float64) string { return sym + fmt.Sprintf("%.2f", f) },
		"expenseCategoryStyle":  func(s string) string { return "" },
		"expenseCategoryIcon":   func(s string) string { return "" },
		"listContains": func(list []string, s string) bool {
			for _, x := range list {
				if x == s {
					return true
				}
			}
			return false
		},
		"mainSectionVisible":                  func(string, trips.Trip) bool { return true },
		"tripSectionEnabled":                  func(string, trips.Trip) bool { return true },
		"sidebarWidgetVisible":                func(string, trips.Trip) bool { return true },
		"tripMobileFabHasItems":               trips.TripMobileFabHasItems,
		"tripDesktopCalendarFlyoutHasActions": trips.TripDesktopCalendarFlyoutHasActions,
		"tripExpenseQuickAddVisible":          trips.TripExpenseQuickAddVisible,
		"effectiveDistanceUnit": func(trip trips.Trip, settings trips.AppSettings) string {
			return trips.EffectiveDistanceUnit(&trip, settings)
		},
		"tripMainSectionLabel":            func(s string) string { return s },
		"tripSidebarWidgetLabel":          func(s string) string { return s },
		"tripMainSectionVisibilityIcon":   trips.MainSectionVisibilityIcon,
		"tripSidebarWidgetVisibilityIcon": trips.SidebarWidgetVisibilityIcon,
		"googleMapsSearchURL":             func(lat, lng float64, hint string) string { return "" },
		"googleMapsDirectionsURL":         func(fromLat, fromLng, toLat, toLng float64, mode string) string { return "" },
		"itineraryDayGoogleMapsURL":       func([]itineraryItemView) string { return "" },
		"joinItineraryLocalDateTime":      stubJoinItineraryLocalDateTime,
		"locationLineBeforeComma":         func(s string) string { return s },
		"itineraryNotesDisplay":           func(s string) string { return s },
		"isImageWebPath":                  func(string) bool { return true },
		"itineraryGeocodeQuery":           func(any) string { return "" },
		"profileInitial": func(u trips.User) string {
			p := trips.UserProfile{DisplayName: u.DisplayName, Username: u.Username, Email: u.Email}
			return p.InitialForAvatar()
		},
		"profileAvatarURL": func(u trips.User) string { return "" },
		"sub":              func(a, b int) int { return a - b },
		"dict": func(values ...any) (map[string]any, error) {
			if len(values)%2 != 0 {
				return nil, fmt.Errorf("dict: expected even number of arguments")
			}
			m := make(map[string]any, len(values)/2)
			for i := 0; i < len(values); i += 2 {
				k, ok := values[i].(string)
				if !ok {
					return nil, fmt.Errorf("dict: key at %d must be string", i)
				}
				m[k] = values[i+1]
			}
			return m, nil
		},
		"tabEffectivePaidBy": func(e trips.Expense, ownerID string) string {
			return trips.EffectivePaidBy(e, ownerID)
		},
		"tabSettlementParticipantKey": trips.TabSettlementParticipantKey,
		"tabSplitModeShort": func(mode string) string {
			switch strings.ToLower(strings.TrimSpace(mode)) {
			case trips.TabSplitEqual, "":
				return "Equal"
			case trips.TabSplitExact:
				return "Exact"
			case trips.TabSplitPercent:
				return "Percent"
			case trips.TabSplitShares:
				return "Shares"
			default:
				return mode
			}
		},
		"add":  func(a, b int) int { return a + b },
		"addF": func(a, b float64) float64 { return a + b },
		"mod": func(a, b int) int {
			if b == 0 {
				return 0
			}
			return a % b
		},
		"guestInitial":  trips.GuestInitialFromDisplayName,
		"tabPayerThumb": tabPayerThumb,
		"tabAvatarURL": func(s string) string {
			s = strings.TrimSpace(s)
			if s == "" {
				return ""
			}
			if strings.HasPrefix(s, "http://") || strings.HasPrefix(s, "https://") {
				return s
			}
			if strings.HasPrefix(s, "/") {
				return s
			}
			return "/" + s
		},
		"tabSplitMethodBadgeClass": func(mode string) string {
			switch strings.ToLower(strings.TrimSpace(mode)) {
			case trips.TabSplitEqual, "":
				return "tab-split-method-badge--equal"
			default:
				return "tab-split-method-badge--neutral"
			}
		},
		"tabTabQueryString": func(balanceView, q, tabCat string) string {
			v := url.Values{}
			bv := strings.ToLower(strings.TrimSpace(balanceView))
			if bv == "debts" {
				v.Set("balance_view", "debts")
			}
			if strings.TrimSpace(q) != "" {
				v.Set("q", strings.TrimSpace(q))
			}
			if strings.TrimSpace(tabCat) != "" {
				v.Set("tab_cat", strings.TrimSpace(tabCat))
			}
			s := v.Encode()
			if s == "" {
				return ""
			}
			return "?" + s
		},
		"urlQueryEscape": func(s string) string { return url.QueryEscape(s) },
	}
}

func TestAboutPageTemplateRenders(t *testing.T) {
	root := findModuleRoot(t)
	tmpl := ht.Must(
		ht.New("").
			Funcs(htmlTemplateSmokeFuncs()).
			ParseGlob(filepath.Join(root, "web", "templates", "*.html")),
	)
	data := map[string]any{
		"Settings": trips.AppSettings{
			AppTitle:             "App",
			ThemePreference:      "light",
			TripDashboardHeading: "Trips",
		},
		"ClearThemeOverride":      false,
		"CSRFToken":               "test-csrf",
		"CurrentUser":             trips.User{DisplayName: "Test"},
		"AppVersion":              "9.9.9",
		"ChangelogHTML":           ht.HTML("<p class=\"t\">Note</p>"),
		"SidebarNavActive":        "about",
		"SidebarInProgressTrips":  []sidebarInProgressTrip(nil),
		"SidebarTripID":           "",
		"TripID":                  "",
		"NotificationUnreadCount": 0,
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

func TestTripDocumentRowTemplateRenders(t *testing.T) {
	root := findModuleRoot(t)
	tmpl := ht.Must(
		ht.New("").
			Funcs(htmlTemplateSmokeFuncs()).
			ParseGlob(filepath.Join(root, "web", "templates", "*.html")),
	)
	row := tripDocumentRow{
		ID:           "d-smoke",
		Section:      "general",
		FileKind:     "pdf",
		FileTypeIcon: "picture_as_pdf",
		Category:     "General Documents",
		ItemName:     "Smoke Trip",
		FileName:     "doc.pdf",
		DisplayName:  "",
		FilePath:     "/uploads/trips/t1/doc.pdf",
		FileSize:     2048,
		UploadedAt:   time.Date(2026, 4, 2, 0, 0, 0, 0, time.UTC),
		SearchText:   "smoke doc general",
	}
	data := map[string]any{
		"Details": trips.TripDetails{
			Trip: trips.Trip{ID: "t1", Name: "Smoke Trip"},
		},
		"CSRFToken": "csrf-row",
		"D":         row,
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "trip_document_row", data); err != nil {
		t.Fatalf("execute trip_document_row: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `data-trip-doc-id="d-smoke"`) {
		t.Fatalf("expected data-trip-doc-id on row, snippet=%q", truncate(out, 400))
	}
	if !strings.Contains(out, `data-ajax-submit`) || !strings.Contains(out, `/documents/d-smoke/update`) {
		t.Fatalf("expected ajax rename form action, snippet=%q", truncate(out, 500))
	}
}

func TestTripNotesPageTemplateRenders(t *testing.T) {
	root := findModuleRoot(t)
	tmpl := ht.Must(
		ht.New("").
			Funcs(htmlTemplateSmokeFuncs()).
			ParseGlob(filepath.Join(root, "web", "templates", "*.html")),
	)

	now := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	trip := trips.Trip{
		ID:                     "t-notes-smoke",
		Name:                   "Notes Smoke Trip",
		OwnerUserID:            "owner-smoke",
		UIShowStay:             true,
		UIShowVehicle:          true,
		UIShowFlights:          true,
		UIShowSpends:           true,
		UIShowItinerary:        true,
		UIShowChecklist:        true,
		UIShowTheTab:           true,
		UIShowDocuments:        true,
		UICollaborationEnabled: true,
		UIDateFormat:           "mdy",
	}
	data := map[string]any{
		"Details": trips.TripDetails{
			Trip: trip,
		},
		"Settings": trips.AppSettings{
			AppTitle:              "REMÍ Test",
			ThemePreference:       "light",
			DefaultUIDateFormat:   "mdy",
			DefaultCurrencySymbol: "$",
		},
		"CSRFToken":          "test-csrf",
		"CurrentUser":        trips.User{DisplayName: "Tester", Username: "tester", Email: "t@example.com"},
		"CustomSidebarLinks": []trips.CustomSidebarLink(nil),
		"TripAccess":         trips.TripAccess{IsOwner: true, CanManage: true},
		"Party": []trips.UserProfile{
			{ID: "owner-smoke", DisplayName: "Owner"},
		},
		"PendingInvites":          []trips.TripInvitePending(nil),
		"TripGuests":              []trips.TripGuest(nil),
		"TabDepartedParticipants": []trips.DepartedTabParticipant(nil),
		"SidebarNavActive":        "notes",
		"NotificationUnreadCount": 0,
		"KeepView":                trips.KeepViewNotes,
		"KeepQuery":               "",
		"ReturnTo":                "/trips/t-notes-smoke/notes",
		"KeepNoteColors":          keepNotePickerColors,
		"ReminderCategories":      trips.ReminderChecklistCategories,
		"KeepMasonry": []KeepMasonryCard{
			{
				Kind: "note",
				Note: trips.TripNote{
					ID: "n1", TripID: trip.ID, Title: "Reminder", Body: "Pack light\n— layers.",
					Color: "sage", Pinned: true, CreatedAt: now, UpdatedAt: now,
				},
			},
			{
				Kind:     "checklist",
				Category: "Packing List",
				Items: []trips.ChecklistItem{
					{ID: "c1", TripID: trip.ID, Category: "Packing List", Text: "Passport", DueAt: "2026-06-01", Done: false, CreatedAt: now, UpdatedAt: now},
					{ID: "c2", TripID: trip.ID, Category: "Packing List", Text: "Charger", DueAt: "", Done: true, CreatedAt: now, UpdatedAt: now},
				},
			},
		},
		"KeepChecklistGroups": []KeepChecklistGroup(nil),
		"DayGroups":           []itineraryDayGroup{},
		"FabReturnTo":         "/trips/t-notes-smoke/notes",
		"ExpenseCategories":   trips.QuickExpenseCategories,
		"ChecklistCategories": trips.ReminderChecklistCategories,
		"CurrencySymbol":      "$",
		"CurrentUserID":       "owner-smoke",
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "trip_notes.html", data); err != nil {
		t.Fatalf("execute trip_notes.html: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `class="page-trip-notes"`) {
		t.Fatalf("expected page-trip-notes body class, snippet=%q", truncate(out, 500))
	}
	if !strings.Contains(out, `class="trip-keep-masonry`) {
		t.Fatalf("expected trip-keep-masonry grid, snippet=%q", truncate(out, 500))
	}
	if !strings.Contains(out, `reminder-checklist-card`) {
		t.Fatalf("expected checklist card in unified masonry, snippet=%q", truncate(out, 500))
	}
	if !strings.Contains(out, "Add Note") || !strings.Contains(out, "Add Checklist") {
		t.Fatalf("expected composer sections in output, snippet=%q", truncate(out, 600))
	}
	if !strings.Contains(out, "Reminder") || !strings.Contains(out, "Passport") {
		t.Fatalf("expected masonry card content, snippet=%q", truncate(out, 600))
	}
	if !strings.Contains(out, `trip-keep-util-link--active`) {
		t.Fatalf("expected active util nav state for Notes view, snippet=%q", truncate(out, 700))
	}
}

func TestTripKeepNotesBoardInnerTemplateRenders(t *testing.T) {
	root := findModuleRoot(t)
	tmpl := ht.Must(
		ht.New("").
			Funcs(htmlTemplateSmokeFuncs()).
			ParseGlob(filepath.Join(root, "web", "templates", "*.html")),
	)
	now := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	trip := trips.Trip{
		ID:          "t-frag",
		Name:        "Frag Trip",
		OwnerUserID: "u1",
		StartDate:   "2026-04-01",
		EndDate:     "2026-04-07",
	}
	longBody := strings.Repeat("a", 400)
	data := map[string]any{
		"Details":   trips.TripDetails{Trip: trip},
		"Settings":  trips.AppSettings{AppTitle: "App", DefaultUIDateFormat: "mdy"},
		"CSRFToken": "csrf-frag",
		"KeepView":  trips.KeepViewNotes,
		"KeepMasonry": []KeepMasonryCard{
			{Kind: "note", Note: trips.TripNote{ID: "n1", TripID: trip.ID, Title: "Live title", Body: "Hi", Color: "mint", CreatedAt: now, UpdatedAt: now}},
			{Kind: "note", Note: trips.TripNote{ID: "n-long", TripID: trip.ID, Title: "Long", Body: longBody, Color: "mint", CreatedAt: now, UpdatedAt: now}},
		},
		"KeepNoteColors":     keepNotePickerColors,
		"ReminderCategories": trips.ReminderChecklistCategories,
		"ReturnTo":           "/trips/t-frag/notes",
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "tripKeepNotesBoardInner", data); err != nil {
		t.Fatalf("execute tripKeepNotesBoardInner: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `data-trip-keep-board-root`) {
		t.Fatalf("expected board root marker, snippet=%q", truncate(out, 400))
	}
	if !strings.Contains(out, "Live title") {
		t.Fatalf("expected note title in fragment, snippet=%q", truncate(out, 400))
	}
	if !strings.Contains(out, `data-trip-keep-note-view-more`) || !strings.Contains(out, `trip-keep-note-full-n-long`) {
		t.Fatalf("expected view-more control for long note, snippet=%q", truncate(out, 600))
	}
}

func TestTripKeepDetailsPreviewInnerTemplateRenders(t *testing.T) {
	root := findModuleRoot(t)
	tmpl := ht.Must(
		ht.New("").
			Funcs(htmlTemplateSmokeFuncs()).
			ParseGlob(filepath.Join(root, "web", "templates", "*.html")),
	)
	now := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	trip := trips.Trip{
		ID:          "t-prev",
		Name:        "Preview Trip",
		OwnerUserID: "u1",
		StartDate:   "2026-04-01",
		EndDate:     "2026-04-07",
	}
	data := map[string]any{
		"Details":             trips.TripDetails{Trip: trip},
		"Settings":            trips.AppSettings{AppTitle: "App", DefaultUIDateFormat: "mdy"},
		"CSRFToken":           "csrf-prev",
		"ChecklistCategories": trips.ReminderChecklistCategories,
		"KeepNoteColors":      keepNotePickerColors,
		"TripKeepPreview": []KeepMasonryCard{
			{Kind: "note", Note: trips.TripNote{ID: "n1", TripID: trip.ID, Title: "Preview Note", Body: "B", CreatedAt: now, UpdatedAt: now}},
			{
				Kind:     "checklist",
				Category: "Gear",
				Items: []trips.ChecklistItem{
					{ID: "ci1", TripID: trip.ID, Category: "Gear", Text: "Tent", Done: false, CreatedAt: now, UpdatedAt: now},
				},
			},
		},
	}
	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "tripKeepDetailsPreviewInner", data); err != nil {
		t.Fatalf("execute tripKeepDetailsPreviewInner: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, `trip-details-keep-preview`) {
		t.Fatalf("expected preview class, snippet=%q", truncate(out, 400))
	}
	if !strings.Contains(out, "Preview Note") {
		t.Fatalf("expected note in preview fragment, snippet=%q", truncate(out, 400))
	}
	if !strings.Contains(out, "Tent") || !strings.Contains(out, `name="csrf_token"`) {
		t.Fatalf("expected checklist row with csrf on toggle form, snippet=%q", truncate(out, 700))
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
