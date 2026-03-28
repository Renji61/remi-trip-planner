package httpapp

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"io"
	"log"
	"math"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/uuid"

	"remi-trip-planner/internal/trips"
)

type Dependencies struct {
	TripService *trips.Service
}

type app struct {
	tripService *trips.Service
	templates   *template.Template
	staticDir   string
}

func tripForbiddenOrMissing(err error) bool {
	return errors.Is(err, sql.ErrNoRows) || errors.Is(err, trips.ErrTripAccessDenied)
}

// geocodeLocation calls the Nominatim API to resolve a location string into
// geographic coordinates. Returns (0, 0) silently when geocoding fails so that
// callers can treat missing coordinates gracefully.
func geocodeLocation(ctx context.Context, query string) (lat, lng float64) {
	if query == "" {
		return 0, 0
	}
	reqURL := fmt.Sprintf(
		"https://nominatim.openstreetmap.org/search?q=%s&format=jsonv2&limit=1",
		url.QueryEscape(query),
	)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return 0, 0
	}
	req.Header.Set("User-Agent", "REMI-Trip-Planner/1.0")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0
	}
	defer resp.Body.Close()
	var results []struct {
		Lat string `json:"lat"`
		Lon string `json:"lon"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&results); err != nil || len(results) == 0 {
		return 0, 0
	}
	lat, _ = strconv.ParseFloat(results[0].Lat, 64)
	lng, _ = strconv.ParseFloat(results[0].Lon, 64)
	return lat, lng
}

type itineraryItemView struct {
	Item    trips.ItineraryItem
	Lodging trips.Lodging
	Vehicle trips.VehicleRental
	Flight  trips.Flight
}

type itineraryDayGroup struct {
	DayNumber      int
	DateLabel      string
	DayDescription string
	Items          []itineraryItemView
}

type expenseDayGroup struct {
	DayNumber int
	DateLabel string
	Items     []trips.Expense
}

type checklistCategoryGroup struct {
	Category string
	Items    []trips.ChecklistItem
}

// dashboardTripCard is a trip plus derived fields for the home dashboard grid.
type dashboardTripCard struct {
	trips.Trip
	BudgetTotal           float64
	BudgetPercent         int
	StatusLabel           string
	StatusSlug            string
	TripSubtitle          string
	HasValidSchedule      bool
	ScheduleDurationLabel string
	// DashboardListLayout mirrors settings; required inside {{define "dashboardTripCard"}} where $ is the card, not the page root.
	DashboardListLayout bool
	Party               []trips.UserProfile
	ActiveCollaborators int
	ViewerIsOwner       bool
	HasSharedIcon       bool
}

type dashboardBudgetRollup struct {
	Spent     float64
	Allocated float64
	Percent   int
}

type budgetCategoryGroupView struct {
	ID   string
	Name string
	Icon string
	// DonutStyle selects which CSS stroke color to use on the donut.
	DonutStyle string
	// DonutStroke uses the same base color as the category icon.
	DonutStroke string
	// IconStyle matches the existing expense category icon color scheme.
	IconStyle    string
	Amount       float64
	PercentInt   int
	ExpenseCount int

	// Donut rendering (viewbox 0..36 with circumference ~100).
	DonutDashArrayA int
	DonutDashArrayB int
	DonutDashOffset int
}

type budgetTransactionRowView struct {
	ExpenseID     string
	DateLabel     string
	CategoryName  string
	CategoryIcon  string
	CategoryStyle string
	Description   string
	Method        string
	Amount        float64
	SpentOn       string
	NotesRaw      string
	LodgingID     string
	VehicleLocked bool
	FlightLocked  bool
	CanEdit       bool
}

// noStoreNonStaticGET prevents proxies and browsers from caching HTML/API responses;
// static assets under /static/ are unaffected.
func noStoreNonStaticGET(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && !strings.HasPrefix(r.URL.Path, "/static/") {
			w.Header().Set("Cache-Control", "no-store")
		}
		next.ServeHTTP(w, r)
	})
}

func NewRouter(deps Dependencies) http.Handler {
	tmpl := template.Must(
		template.New("").
			Funcs(template.FuncMap{
				"formatDateTime":       formatDateTimeDisplay,
				"formatTripDateTime":   formatTripDateTime,
				"formatTripClock":      formatTripClock,
				"formatUIDate":         formatUIDate,
				"formatTripDateRange":  formatTripDateRangeEn,
				"formatTripDateShort":  formatTripDateShortRange,
				"formatTripMoney":      formatTripMoney,
				"expenseCategoryStyle": expenseCategoryStyle,
				"expenseCategoryIcon":  expenseCategoryIcon,
				"listContains":         listContainsString,
				"hasPrefix":            strings.HasPrefix,
				"mainSectionVisible": func(key string, trip trips.Trip) bool {
					return trips.MainSectionVisible(key, trip)
				},
				"tripSectionEnabled": func(key string, trip trips.Trip) bool {
					switch key {
					case trips.MainSectionItinerary:
						return trip.UIShowItinerary
					case trips.MainSectionChecklist:
						return trip.UIShowChecklist
					case trips.MainSectionStay:
						return trip.UIShowStay
					case trips.MainSectionVehicle:
						return trip.UIShowVehicle
					case trips.MainSectionFlights:
						return trip.UIShowFlights
					case trips.MainSectionSpends:
						return trip.UIShowSpends
					default:
						return true
					}
				},
				"sidebarWidgetVisible": func(key string, trip trips.Trip) bool {
					return trips.SidebarWidgetVisible(key, trip)
				},
				"tripMainSectionLabel":            trips.MainSectionLabel,
				"tripSidebarWidgetLabel":          trips.SidebarWidgetLabel,
				"tripMainSectionVisibilityIcon":   trips.MainSectionVisibilityIcon,
				"tripSidebarWidgetVisibilityIcon": trips.SidebarWidgetVisibilityIcon,
				"googleMapsSearchURL":             googleMapsSearchURL,
				"locationLineBeforeComma":         locationLineBeforeComma,
				"itineraryGeocodeQuery":           itineraryGeocodeQuery,
				"abbrevMoney":                     abbrevMoney,
				"profileInitial": func(u trips.User) string {
					p := trips.UserProfile{DisplayName: u.DisplayName, Username: u.Username, Email: u.Email}
					return p.InitialForAvatar()
				},
				"profileAvatarURL": func(u trips.User) string {
					s := strings.TrimSpace(u.AvatarPath)
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
				"sub": func(a, b int) int { return a - b },
			}).
			ParseGlob("web/templates/*.html"),
	)
	a := &app{
		tripService: deps.TripService,
		templates:   tmpl,
		staticDir:   filepath.Join("web", "static"),
	}

	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.Heartbeat("/healthz"))
	r.Use(noStoreNonStaticGET)
	r.Use(a.withSession)

	r.Get("/setup", a.setupPage)
	r.Post("/setup", a.setupSubmit)
	r.Get("/login", a.loginPage)
	r.Post("/login", a.loginSubmit)
	r.Get("/register", a.registerPage)
	r.Post("/register", a.registerSubmit)
	r.Post("/logout", a.logout)
	r.Get("/verify-email", a.verifyEmailPage)
	r.Get("/invites/accept", a.inviteAcceptPage)

	r.Group(func(r chi.Router) {
		r.Use(a.requireRegisteredUser)
		r.Use(a.verifyCSRF)
		r.Post("/invites/accept", a.inviteAcceptSubmit)
		r.Get("/", a.homePage)
		r.Get("/profile", a.profilePage)
		r.Post("/profile", a.profileSave)
		r.Post("/profile/password", a.profilePassword)
		r.Post("/profile/resend-verify", a.profileResendVerify)
		r.Get("/settings", a.settingsPage)
		r.Post("/settings", a.saveSettings)
		r.Post("/settings/reset-all", a.resetAllSiteSettings)
		r.Post("/settings/theme", a.saveThemeQuick)
		r.Post("/trips", a.createTrip)

		r.Route("/trips/{tripID}", func(r chi.Router) {
			r.Use(a.tripIDAccessMiddleware)
			r.Get("/", a.tripPage)
			r.Get("/settings", a.tripSettingsPage)
			r.Post("/reset-ui", a.resetTripUIPresets)
			r.Get("/budget", a.budgetPage)
			r.Get("/budget/transactions", a.budgetTransactionsRows)
			r.Get("/budget/export", a.exportBudgetReport)
			r.Post("/update", a.updateTrip)
			r.Post("/archive", a.archiveTrip)
			r.Post("/delete", a.deleteTrip)
			r.Post("/itinerary", a.addItineraryItem)
			r.Post("/days/{dayNumber}/label", a.saveTripDayLabel)
			r.Post("/itinerary/{itemID}/update", a.updateItineraryItem)
			r.Post("/itinerary/{itemID}/delete", a.deleteItineraryItem)
			r.Get("/accommodation", a.accommodationPage)
			r.Get("/vehicle-rental", a.vehicleRentalPage)
			r.Get("/flights", a.flightsPage)
			r.Post("/accommodation/{lodgingID}/update", a.updateLodging)
			r.Post("/accommodation/{lodgingID}/delete", a.deleteLodging)
			r.Post("/accommodation", a.addLodging)
			r.Post("/vehicle-rental/{rentalID}/update", a.updateVehicleRental)
			r.Post("/vehicle-rental/{rentalID}/delete", a.deleteVehicleRental)
			r.Post("/vehicle-rental", a.addVehicleRental)
			r.Post("/flights/{flightID}/update", a.updateFlight)
			r.Post("/flights/{flightID}/delete", a.deleteFlight)
			r.Post("/flights", a.addFlight)
			r.Post("/lodging/{lodgingID}/update", a.updateLodging)
			r.Post("/lodging/{lodgingID}/delete", a.deleteLodging)
			r.Post("/lodging", a.addLodging)
			r.Get("/lodging", a.redirectLegacyLodgingPath)
			r.Post("/expenses", a.addExpense)
			r.Post("/expenses/{expenseID}/update", a.updateExpense)
			r.Post("/expenses/{expenseID}/delete", a.deleteExpense)
			r.Post("/checklist", a.addChecklistItem)
			r.Post("/invite", a.tripInviteCollaborator)
			r.Post("/invite-link", a.tripCreateInviteLink)
			r.Post("/members/remove", a.tripRemoveMember)
			r.Post("/invites/revoke", a.tripRevokeInvite)
			r.Post("/leave", a.tripLeaveCollaboration)
			r.Post("/stop-sharing", a.tripStopSharing)
			r.Post("/hide-archived", a.tripHideArchived)
		})

		r.Post("/checklist/{itemID}/update", a.updateChecklistItem)
		r.Post("/checklist/{itemID}/delete", a.deleteChecklistItem)
		r.Post("/checklist/{itemID}/toggle", a.toggleChecklistItem)

		r.Route("/api/v1/trips/{tripID}", func(r chi.Router) {
			r.Use(a.tripIDAccessMiddleware)
			r.Get("/changes", a.listChanges)
			r.Post("/sync", a.syncChanges)
		})
	})

	r.Get("/manifest.webmanifest", func(w http.ResponseWriter, r *http.Request) {
		data, _ := os.ReadFile(filepath.Join(a.staticDir, "manifest.webmanifest"))
		w.Header().Set("Content-Type", "application/manifest+json")
		_, _ = w.Write(data)
	})
	r.Get("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		data, _ := os.ReadFile(filepath.Join(a.staticDir, "sw.js"))
		w.Header().Set("Content-Type", "text/javascript")
		w.Header().Set("Cache-Control", "no-cache, must-revalidate")
		_, _ = w.Write(data)
	})
	staticFS := http.FileServer(http.Dir(a.staticDir))
	r.Handle("/static/*", http.StripPrefix("/static/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		p := r.URL.Path
		if strings.HasSuffix(p, ".js") || strings.HasSuffix(p, ".css") {
			w.Header().Set("Cache-Control", "no-cache, must-revalidate")
		}
		staticFS.ServeHTTP(w, r)
	})))

	return r
}

func (a *app) homePage(w http.ResponseWriter, r *http.Request) {
	uid := CurrentUserID(r.Context())
	list, err := a.tripService.ListVisibleTrips(r.Context(), uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	settings, err := a.tripService.MergedSettingsForUI(r.Context(), uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	allTotals, err := a.tripService.SumExpensesByTrip(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	visibleIDs := make(map[string]struct{}, len(list))
	for _, t := range list {
		visibleIDs[t.ID] = struct{}{}
	}
	expenseTotals := make(map[string]float64)
	for id, v := range allTotals {
		if _, ok := visibleIDs[id]; ok {
			expenseTotals[id] = v
		}
	}
	travelStats, err := a.tripService.ComputeTravelStats(r.Context(), list)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	rollups := a.loadDashboardBudgetRollups(r.Context(), uid, list)
	listLayout := settings.DashboardTripLayout == "list"
	sortKey := settings.DashboardTripSort

	var ownerTrips, sharedTrips []trips.Trip
	for _, t := range list {
		if t.OwnerUserID == uid {
			ownerTrips = append(ownerTrips, t)
		} else {
			sharedTrips = append(sharedTrips, t)
		}
	}
	activeO, draftO, archO := buildDashboardTripGroups(ownerTrips, expenseTotals, rollups, listLayout, time.Now())
	activeS, draftS, archS := buildDashboardTripGroups(sharedTrips, expenseTotals, rollups, listLayout, time.Now())
	sortDashboardCards(activeO, sortKey)
	sortDashboardCards(draftO, sortKey)
	sortDashboardCards(archO, sortKey)
	sortDashboardCards(activeS, sortKey)
	sortDashboardCards(draftS, sortKey)
	sortDashboardCards(archS, sortKey)

	draftMerged := append(append([]dashboardTripCard{}, draftO...), draftS...)
	sortDashboardCards(draftMerged, sortKey)
	archMerged := append(append([]dashboardTripCard{}, archO...), archS...)
	sortDashboardCards(archMerged, sortKey)

	enrichParty := func(cards []dashboardTripCard) {
		for i := range cards {
			n, _ := a.tripService.TripCollaboratorCount(r.Context(), cards[i].ID)
			cards[i].ActiveCollaborators = n
			cards[i].ViewerIsOwner = cards[i].OwnerUserID == uid
			cards[i].HasSharedIcon = cards[i].ViewerIsOwner && n > 0
			cards[i].Party, _ = a.tripService.TripParty(r.Context(), cards[i].ID)
		}
	}
	enrichParty(activeO)
	enrichParty(draftO)
	enrichParty(archO)
	enrichParty(activeS)
	enrichParty(draftS)
	enrichParty(archS)
	enrichParty(draftMerged)
	enrichParty(archMerged)

	heroPatternClass := ""
	heroImageURL := ""
	switch bg := settings.DashboardHeroBackground; {
	case strings.HasPrefix(bg, "pattern:"):
		heroPatternClass = "dashboard-hero-adventure--pattern-" + strings.TrimPrefix(bg, "pattern:")
	case strings.HasPrefix(bg, "https://"):
		heroImageURL = bg
	}

	_ = a.templates.ExecuteTemplate(w, "home.html", map[string]any{
		"ActiveTripCards":     activeO,
		"SharedTripCards":     activeS,
		"DraftTripCards":      draftMerged,
		"ArchivedTripCards":   archMerged,
		"Settings":            settings,
		"TravelStats":         travelStats,
		"CSRFToken":           CSRFToken(r.Context()),
		"CurrentUser":         CurrentUser(r.Context()),
		"Saved":               r.URL.Query().Get("saved") == "1",
		"HasError":            false,
		"ErrorText":           "",
		"DashboardListLayout": settings.DashboardTripLayout == "list",
		"HeroPatternClass":    heroPatternClass,
		"HeroImageURL":        heroImageURL,
	})
}

// tripScheduleBounds returns calendar start/end when both dates parse and end is on or after start.
func tripScheduleBounds(t trips.Trip) (startD, endD time.Time, ok bool) {
	start, err1 := time.Parse("2006-01-02", strings.TrimSpace(t.StartDate))
	end, err2 := time.Parse("2006-01-02", strings.TrimSpace(t.EndDate))
	if err1 != nil || err2 != nil {
		return time.Time{}, time.Time{}, false
	}
	loc := time.Local
	startD = time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, loc)
	endD = time.Date(end.Year(), end.Month(), end.Day(), 0, 0, 0, 0, loc)
	if endD.Before(startD) {
		return time.Time{}, time.Time{}, false
	}
	return startD, endD, true
}

func tripHasValidSchedule(t trips.Trip) bool {
	_, _, ok := tripScheduleBounds(t)
	return ok
}

func tripInclusiveDayCount(t trips.Trip) int {
	startD, endD, ok := tripScheduleBounds(t)
	if !ok {
		return 0
	}
	n := int(endD.Sub(startD).Hours()/24) + 1
	if n < 1 {
		return 1
	}
	return n
}

func dashboardTripSubtitle(desc string) string {
	s := strings.TrimSpace(desc)
	if s == "" {
		return "General"
	}
	const max = 56
	r := []rune(s)
	if len(r) <= max {
		return s
	}
	return string(r[:max]) + "…"
}

// budgetRollupFromDetails matches budget page logic: spend excludes pending pay-at-pickup vehicle expenses,
// allocation is itinerary + bookings + non-booking expenses, percent capped at 100 for the bar.
func budgetRollupFromDetails(details trips.TripDetails) (spent, allocated float64, pct int) {
	now := time.Now()
	pendingExpenseIDs := map[string]struct{}{}
	for _, v := range details.Vehicles {
		if !v.PayAtPickUp || strings.TrimSpace(v.DropOffAt) == "" {
			continue
		}
		dropOffAt, err := time.Parse("2006-01-02T15:04", v.DropOffAt)
		if err != nil || !now.Before(dropOffAt) {
			continue
		}
		if v.RentalExpenseID != "" {
			pendingExpenseIDs[v.RentalExpenseID] = struct{}{}
		}
		if v.InsuranceExpenseID != "" {
			pendingExpenseIDs[v.InsuranceExpenseID] = struct{}{}
		}
	}
	totalSpent := 0.0
	for _, e := range details.Expenses {
		if _, isPending := pendingExpenseIDs[e.ID]; isPending {
			continue
		}
		totalSpent += e.Amount
	}
	if totalSpent < 0 {
		totalSpent = 0
	}
	vehicleByExpenseID := trips.VehicleRentalByExpenseID(details.Vehicles)
	flightByExpenseID := trips.FlightByExpenseID(details.Flights)
	nonLodgingExpenses := 0.0
	for _, e := range details.Expenses {
		if e.LodgingID == "" && vehicleByExpenseID[e.ID].ID == "" && flightByExpenseID[e.ID].ID == "" {
			nonLodgingExpenses += e.Amount
		}
	}
	totalBudgeted := computeTotalBudgeted(details.Itinerary, details.Lodgings, details.Vehicles, details.Flights) + nonLodgingExpenses
	budgetProgress := 0.0
	if totalBudgeted > 0 {
		budgetProgress = (totalSpent / totalBudgeted) * 100
		if budgetProgress > 100 {
			budgetProgress = 100
		}
	} else if totalSpent > 0 {
		budgetProgress = 100
	}
	return totalSpent, totalBudgeted, int(budgetProgress + 0.5)
}

func (a *app) loadDashboardBudgetRollups(ctx context.Context, userID string, list []trips.Trip) map[string]dashboardBudgetRollup {
	out := make(map[string]dashboardBudgetRollup, len(list))
	for _, t := range list {
		det, err := a.tripService.GetTripDetailsVisible(ctx, t.ID, userID)
		if err != nil {
			continue
		}
		spent, alloc, pct := budgetRollupFromDetails(det)
		out[t.ID] = dashboardBudgetRollup{Spent: spent, Allocated: alloc, Percent: pct}
	}
	return out
}

func buildDashboardTripGroups(list []trips.Trip, totals map[string]float64, rollups map[string]dashboardBudgetRollup, dashboardListLayout bool, now time.Time) (active, draft, archived []dashboardTripCard) {
	if totals == nil {
		totals = map[string]float64{}
	}
	if rollups == nil {
		rollups = map[string]dashboardBudgetRollup{}
	}
	for _, t := range list {
		label, slug := tripDashboardStatus(t, now)
		rollup, hasRollup := rollups[t.ID]
		spent := totals[t.ID]
		pct := 0
		if hasRollup {
			spent = rollup.Spent
			pct = rollup.Percent
		}
		hasSched := tripHasValidSchedule(t)
		nDays := tripInclusiveDayCount(t)
		durLabel := ""
		if nDays == 1 {
			durLabel = "1 Day"
		} else if nDays > 1 {
			durLabel = fmt.Sprintf("%d Days", nDays)
		}
		c := dashboardTripCard{
			Trip:                  t,
			BudgetTotal:           spent,
			BudgetPercent:         pct,
			StatusLabel:           label,
			StatusSlug:            slug,
			TripSubtitle:          dashboardTripSubtitle(t.Description),
			HasValidSchedule:      hasSched,
			ScheduleDurationLabel: durLabel,
			DashboardListLayout:   dashboardListLayout,
		}
		switch {
		case t.IsArchived:
			archived = append(archived, c)
		case !tripHasValidSchedule(t):
			draft = append(draft, c)
		default:
			active = append(active, c)
		}
	}
	return active, draft, archived
}

func statusSortRank(slug string) int {
	switch slug {
	case "draft":
		return 0
	case "upcoming":
		return 1
	case "in-progress":
		return 2
	case "completed":
		return 3
	case "archived":
		return 4
	default:
		return 9
	}
}

func parseTripStartForSort(t trips.Trip) (time.Time, bool) {
	s := strings.TrimSpace(t.StartDate)
	if s == "" {
		return time.Time{}, false
	}
	tm, err := time.Parse("2006-01-02", s)
	return tm, err == nil
}

func sortDashboardCards(cards []dashboardTripCard, sortKey string) {
	switch sortKey {
	case "start_date":
		sort.Slice(cards, func(i, j int) bool {
			ti, okI := parseTripStartForSort(cards[i].Trip)
			tj, okJ := parseTripStartForSort(cards[j].Trip)
			if okI != okJ {
				return okI
			}
			if !okI {
				return strings.ToLower(strings.TrimSpace(cards[i].Name)) < strings.ToLower(strings.TrimSpace(cards[j].Name))
			}
			if !ti.Equal(tj) {
				return ti.Before(tj)
			}
			return strings.ToLower(strings.TrimSpace(cards[i].Name)) < strings.ToLower(strings.TrimSpace(cards[j].Name))
		})
	case "updated":
		sort.Slice(cards, func(i, j int) bool {
			ui, uj := cards[i].UpdatedAt, cards[j].UpdatedAt
			if !ui.Equal(uj) {
				return ui.After(uj)
			}
			return strings.ToLower(strings.TrimSpace(cards[i].Name)) < strings.ToLower(strings.TrimSpace(cards[j].Name))
		})
	case "status":
		sort.Slice(cards, func(i, j int) bool {
			ri, rj := statusSortRank(cards[i].StatusSlug), statusSortRank(cards[j].StatusSlug)
			if ri != rj {
				return ri < rj
			}
			return strings.ToLower(strings.TrimSpace(cards[i].Name)) < strings.ToLower(strings.TrimSpace(cards[j].Name))
		})
	default:
		sort.Slice(cards, func(i, j int) bool {
			return strings.ToLower(strings.TrimSpace(cards[i].Name)) < strings.ToLower(strings.TrimSpace(cards[j].Name))
		})
	}
}

func tripDashboardStatus(t trips.Trip, now time.Time) (label, slug string) {
	if t.IsArchived {
		return "Archived", "archived"
	}
	startD, endD, ok := tripScheduleBounds(t)
	if !ok {
		return "Draft Trip", "draft"
	}
	loc := time.Local
	nowD := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, loc)
	switch {
	case nowD.Before(startD):
		return "Upcoming", "upcoming"
	case nowD.After(endD):
		return "Completed", "completed"
	default:
		return "In progress", "in-progress"
	}
}

func (a *app) settingsPage(w http.ResponseWriter, r *http.Request) {
	uid := CurrentUserID(r.Context())
	settings, err := a.tripService.MergedSettingsForUI(r.Context(), uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	tripID := strings.TrimSpace(r.URL.Query().Get("trip_id"))
	_ = a.templates.ExecuteTemplate(w, "settings.html", map[string]any{
		"Settings":           settings,
		"CSRFToken":          CSRFToken(r.Context()),
		"Saved":              r.URL.Query().Get("saved") == "1",
		"Reset":              r.URL.Query().Get("reset") == "1",
		"ClearThemeOverride": r.URL.Query().Get("saved") == "1" || r.URL.Query().Get("reset") == "1",
		"TripID":             tripID,
	})
}

func (a *app) saveSettings(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	uid := CurrentUserID(r.Context())
	if uid == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := a.tripService.EnsureUserSettings(r.Context(), uid); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	mapLat, _ := strconv.ParseFloat(r.FormValue("map_default_latitude"), 64)
	mapLng, _ := strconv.ParseFloat(r.FormValue("map_default_longitude"), 64)
	mapZoom, _ := strconv.Atoi(r.FormValue("map_default_zoom"))
	enableLookup := r.FormValue("enable_location_lookup") == "true"

	heroBG := strings.TrimSpace(r.FormValue("dashboard_hero_background"))
	if mode := strings.TrimSpace(r.FormValue("dashboard_hero_background_mode")); mode != "" {
		if mode == "custom_url" {
			heroBG = strings.TrimSpace(r.FormValue("dashboard_hero_background_url"))
		} else {
			heroBG = mode
		}
	}

	app, err := a.tripService.GetAppSettings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	app.AppTitle = defaultIfEmpty(r.FormValue("app_title"), "REMI Trip Planner")
	app.MapDefaultLatitude = mapLat
	app.MapDefaultLongitude = mapLng
	app.MapDefaultZoom = mapZoom
	app.EnableLocationLookup = enableLookup
	if vals, ok := r.PostForm["site_registration_enabled"]; ok && len(vals) > 0 {
		app.RegistrationEnabled = vals[len(vals)-1] == "1"
	}
	if err := a.tripService.SaveAppSettings(r.Context(), app); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := a.tripService.SaveUserUISettings(r.Context(), uid, trips.UserSettings{
		UserID:                  uid,
		ThemePreference:         r.FormValue("theme_preference"),
		DashboardTripLayout:     r.FormValue("dashboard_trip_layout"),
		DashboardTripSort:       r.FormValue("dashboard_trip_sort"),
		DashboardHeroBackground: heroBG,
		TripDashboardHeading:    strings.TrimSpace(r.FormValue("trip_dashboard_heading")),
		DefaultCurrencyName:     defaultIfEmpty(r.FormValue("default_currency_name"), "USD"),
		DefaultCurrencySymbol:   defaultIfEmpty(r.FormValue("default_currency_symbol"), "$"),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	returnTo := strings.TrimSpace(r.FormValue("return_to"))
	if returnTo == "" || !strings.HasPrefix(returnTo, "/") {
		returnTo = "/settings"
	}
	joiner := "?"
	if strings.Contains(returnTo, "?") {
		joiner = "&"
	}
	http.Redirect(w, r, returnTo+joiner+"saved=1", http.StatusSeeOther)
}

func (a *app) resetAllSiteSettings(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	uid := CurrentUserID(r.Context())
	if uid == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if err := a.tripService.EnsureUserSettings(r.Context(), uid); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := a.tripService.ResetSiteSettingsToDefaults(r.Context()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := a.tripService.ResetUserUISettingsToDefaults(r.Context(), uid); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	returnTo := strings.TrimSpace(r.FormValue("return_to"))
	if returnTo == "" || !strings.HasPrefix(returnTo, "/") {
		returnTo = "/settings"
	}
	joiner := "?"
	if strings.Contains(returnTo, "?") {
		joiner = "&"
	}
	http.Redirect(w, r, returnTo+joiner+"reset=1", http.StatusSeeOther)
}

// saveThemeQuick updates only theme preference (header toggle). Expects POST theme_preference=light|dark|system.
func (a *app) saveThemeQuick(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	uid := CurrentUserID(r.Context())
	if uid == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	pref := strings.TrimSpace(r.FormValue("theme_preference"))
	if err := a.tripService.EnsureUserSettings(r.Context(), uid); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	merged, err := a.tripService.MergedSettingsForUI(r.Context(), uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := a.tripService.SaveUserUISettings(r.Context(), uid, trips.UserSettings{
		UserID:                  uid,
		ThemePreference:         pref,
		DashboardTripLayout:     merged.DashboardTripLayout,
		DashboardTripSort:       merged.DashboardTripSort,
		DashboardHeroBackground: merged.DashboardHeroBackground,
		TripDashboardHeading:    merged.TripDashboardHeading,
		DefaultCurrencyName:     merged.DefaultCurrencyName,
		DefaultCurrencySymbol:   merged.DefaultCurrencySymbol,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (a *app) createTrip(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	uid := CurrentUserID(r.Context())
	id, err := a.tripService.CreateTrip(r.Context(), trips.Trip{
		Name:           r.FormValue("name"),
		Description:    r.FormValue("description"),
		StartDate:      r.FormValue("start_date"),
		EndDate:        r.FormValue("end_date"),
		CurrencyName:   defaultIfEmpty(r.FormValue("currency_name"), "USD"),
		CurrencySymbol: defaultIfEmpty(r.FormValue("currency_symbol"), "$"),
		OwnerUserID:    uid,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips/"+id, http.StatusSeeOther)
}

func (a *app) tripPage(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("open")), "edit") {
		http.Redirect(w, r, "/trips/"+tripID+"/settings", http.StatusSeeOther)
		return
	}
	details, err := a.tripService.GetTripDetailsVisible(r.Context(), tripID, CurrentUserID(r.Context()))
	if err != nil {
		if tripForbiddenOrMissing(err) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	for _, l := range details.Lodgings {
		if err := a.tripService.SyncExpenseForLodging(r.Context(), l); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	for _, v := range details.Vehicles {
		if err := a.tripService.SyncExpenseForVehicleRental(r.Context(), v); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	for _, f := range details.Flights {
		if err := a.tripService.SyncExpenseForFlight(r.Context(), f); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	details, err = a.tripService.GetTripDetailsVisible(r.Context(), tripID, CurrentUserID(r.Context()))
	if err != nil {
		if tripForbiddenOrMissing(err) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	total := 0.0
	for _, e := range details.Expenses {
		total += e.Amount
	}
	now := time.Now()
	for _, v := range details.Vehicles {
		if !v.PayAtPickUp {
			continue
		}
		dropOffAt, err := time.Parse("2006-01-02T15:04", v.DropOffAt)
		if err != nil || !now.Before(dropOffAt) {
			continue
		}
		total -= v.Cost + v.InsuranceCost
	}
	if total < 0 {
		total = 0
	}
	var nonLodgingExpenses float64
	vehicleByExpenseID := trips.VehicleRentalByExpenseID(details.Vehicles)
	flightByExpenseID := trips.FlightByExpenseID(details.Flights)
	for _, e := range details.Expenses {
		if e.LodgingID == "" && vehicleByExpenseID[e.ID].ID == "" && flightByExpenseID[e.ID].ID == "" {
			nonLodgingExpenses += e.Amount
		}
	}
	totalBudgeted := computeTotalBudgeted(details.Itinerary, details.Lodgings, details.Vehicles, details.Flights) + nonLodgingExpenses
	budgetProgress := 0.0
	if totalBudgeted > 0 {
		budgetProgress = (total / totalBudgeted) * 100
		if budgetProgress > 100 {
			budgetProgress = 100
		}
	} else if total > 0 {
		budgetProgress = 100
	}
	dayLabels, err := a.tripService.GetTripDayLabels(r.Context(), tripID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	dayGroups := buildItineraryDayGroups(details.Trip.StartDate, details.Itinerary, details.Lodgings, details.Vehicles, details.Flights, dayLabels)
	expenseGroups := buildExpenseDayGroups(details.Trip.StartDate, details.Expenses)
	checklistCategoryGroups := buildChecklistCategoryGroups(details.Checklist, trips.ReminderChecklistCategories)
	uid := CurrentUserID(r.Context())
	settings, err := a.tripService.MergedSettingsForUI(r.Context(), uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	acc, _ := TripAccessFromContext(r.Context())
	party, _ := a.tripService.TripParty(r.Context(), tripID)
	pendingInvites, _ := a.tripService.ListPendingTripInvitesForTrip(r.Context(), tripID, uid)
	nCollab, _ := a.tripService.TripCollaboratorCount(r.Context(), tripID)
	inviteNotice := strings.TrimSpace(r.URL.Query().Get("invite_notice"))
	inviteEmail := strings.TrimSpace(r.URL.Query().Get("invite_email"))
	if inviteNotice != "sent" && inviteNotice != "added" {
		inviteNotice = ""
		inviteEmail = ""
	}
	archivedHidden, _ := a.tripService.IsArchivedTripHiddenOnDashboard(r.Context(), tripID, uid)
	currencySymbol := defaultIfEmpty(details.Trip.CurrencySymbol, "$")
	currencyName := defaultIfEmpty(details.Trip.CurrencyName, "USD")
	vehicleExpenseLocked := map[string]bool{}
	for expenseID := range vehicleByExpenseID {
		vehicleExpenseLocked[expenseID] = true
	}
	flightExpenseLocked := map[string]bool{}
	for expenseID := range flightByExpenseID {
		flightExpenseLocked[expenseID] = true
	}
	mainSectionOrder := trips.NormalizeMainSectionOrder(details.Trip.UIMainSectionOrder)
	sidebarWidgetOrder := trips.NormalizeSidebarWidgetOrder(details.Trip.UISidebarWidgetOrder)
	customSidebarLinks := trips.ParseCustomSidebarLinksJSON(details.Trip.UICustomSidebarLinks)
	pageData := map[string]any{
		"Details":                     details,
		"DayGroups":                   dayGroups,
		"ExpenseGroups":               expenseGroups,
		"Settings":                    settings,
		"CurrencySymbol":              currencySymbol,
		"CurrencyName":                currencyName,
		"TotalExpense":                total,
		"TotalBudgeted":               totalBudgeted,
		"BudgetProgress":              budgetProgress,
		"ExpenseCategories":           trips.QuickExpenseCategories,
		"ChecklistCategories":         trips.ReminderChecklistCategories,
		"ChecklistGroups":             checklistCategoryGroups,
		"VehicleExpenseLocked":        vehicleExpenseLocked,
		"FlightExpenseLocked":         flightExpenseLocked,
		"MainSectionOrder":            mainSectionOrder,
		"SidebarWidgetOrder":          sidebarWidgetOrder,
		"CustomSidebarLinks":          customSidebarLinks,
		"TripAccess":                  acc,
		"Party":                       party,
		"PendingInvites":              pendingInvites,
		"CollaboratorCount":           nCollab,
		"InviteNotice":                inviteNotice,
		"InviteNoticeEmail":           inviteEmail,
		"ArchivedHiddenFromDashboard": archivedHidden,
		"CSRFToken":                   CSRFToken(r.Context()),
		"SidebarNavActive":            "trip",
	}
	var buf bytes.Buffer
	if err := a.templates.ExecuteTemplate(&buf, "trip.html", pageData); err != nil {
		log.Printf("trip page template: %v", err)
		http.Error(w, "Could not render trip page.", http.StatusInternalServerError)
		return
	}
	_, _ = io.Copy(w, &buf)
}

func (a *app) tripSettingsPage(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	t, err := a.tripService.GetTrip(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	uid := CurrentUserID(r.Context())
	settings, err := a.tripService.MergedSettingsForUI(r.Context(), uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	currencySymbol := defaultIfEmpty(t.CurrencySymbol, "$")
	currencyName := defaultIfEmpty(t.CurrencyName, "USD")
	mainSectionOrder := trips.NormalizeMainSectionOrder(t.UIMainSectionOrder)
	sidebarWidgetOrder := trips.NormalizeSidebarWidgetOrder(t.UISidebarWidgetOrder)
	pageData := map[string]any{
		"Details":                   trips.TripDetails{Trip: t},
		"Settings":                  settings,
		"CSRFToken":                 CSRFToken(r.Context()),
		"CurrencySymbol":            currencySymbol,
		"CurrencyName":              currencyName,
		"MainSectionOrder":          mainSectionOrder,
		"SidebarWidgetOrder":        sidebarWidgetOrder,
		"UIMainSectionOrderValue":   trips.JoinMainSectionOrder(mainSectionOrder),
		"UISidebarWidgetOrderValue": trips.JoinSidebarWidgetOrder(sidebarWidgetOrder),
		"CustomLinkEditorSlots":     trips.CustomLinkEditorSlots(t.UICustomSidebarLinks),
		"Saved":                     r.URL.Query().Get("saved") == "1",
		"Reset":                     r.URL.Query().Get("reset") == "1",
		"HideSettingsNavOnMobile":   true,
	}
	a.mergeTripSidebarContext(r.Context(), r, tripID, trips.TripDetails{Trip: t}, pageData, "settings")
	var buf bytes.Buffer
	if err := a.templates.ExecuteTemplate(&buf, "trip_settings.html", pageData); err != nil {
		log.Printf("trip settings page template: %v", err)
		http.Error(w, "Could not render trip settings.", http.StatusInternalServerError)
		return
	}
	_, _ = io.Copy(w, &buf)
}

func (a *app) budgetPage(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	details, err := a.tripService.GetTripDetailsVisible(r.Context(), tripID, CurrentUserID(r.Context()))
	if err != nil {
		if tripForbiddenOrMissing(err) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if a.redirectIfTripSectionDisabled(w, r, details.Trip, "spends") {
		return
	}

	uid := CurrentUserID(r.Context())
	settings, err := a.tripService.MergedSettingsForUI(r.Context(), uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	currencySymbol := defaultIfEmpty(details.Trip.CurrencySymbol, "$")

	// Pending vehicle expenses are pay-at-pickup costs that should not be counted as "spent" yet.
	now := time.Now()
	pendingExpenseIDs := map[string]struct{}{}
	for _, v := range details.Vehicles {
		if !v.PayAtPickUp || strings.TrimSpace(v.DropOffAt) == "" {
			continue
		}
		dropOffAt, err := time.Parse("2006-01-02T15:04", v.DropOffAt)
		if err != nil || !now.Before(dropOffAt) {
			continue
		}
		if v.RentalExpenseID != "" {
			pendingExpenseIDs[v.RentalExpenseID] = struct{}{}
		}
		if v.InsuranceExpenseID != "" {
			pendingExpenseIDs[v.InsuranceExpenseID] = struct{}{}
		}
	}

	spentExpenses := make([]trips.Expense, 0, len(details.Expenses))
	totalSpent := 0.0
	for _, e := range details.Expenses {
		if _, isPending := pendingExpenseIDs[e.ID]; isPending {
			continue
		}
		spentExpenses = append(spentExpenses, e)
		totalSpent += e.Amount
	}
	if totalSpent < 0 {
		totalSpent = 0
	}

	// Budgeted cost uses itinerary-planned costs + manually entered, non-booking expenses.
	vehicleByExpenseID := trips.VehicleRentalByExpenseID(details.Vehicles)
	flightByExpenseID := trips.FlightByExpenseID(details.Flights)
	vehicleExpenseLocked := map[string]bool{}
	for id := range vehicleByExpenseID {
		vehicleExpenseLocked[id] = true
	}
	flightExpenseLocked := map[string]bool{}
	for id := range flightByExpenseID {
		flightExpenseLocked[id] = true
	}
	nonLodgingExpenses := 0.0
	for _, e := range details.Expenses {
		if e.LodgingID == "" && vehicleByExpenseID[e.ID].ID == "" && flightByExpenseID[e.ID].ID == "" {
			nonLodgingExpenses += e.Amount
		}
	}
	totalBudgeted := computeTotalBudgeted(details.Itinerary, details.Lodgings, details.Vehicles, details.Flights) + nonLodgingExpenses

	remaining := totalBudgeted - totalSpent
	if remaining < 0 {
		remaining = 0
	}

	budgetProgress := 0.0
	if totalBudgeted > 0 {
		budgetProgress = (totalSpent / totalBudgeted) * 100
		if budgetProgress > 100 {
			budgetProgress = 100
		}
	} else if totalSpent > 0 {
		budgetProgress = 100
	}

	tripDays := 1
	startDate, startErr := time.Parse("2006-01-02", details.Trip.StartDate)
	endDate, endErr := time.Parse("2006-01-02", details.Trip.EndDate)
	if startErr == nil && endErr == nil && !endDate.Before(startDate) {
		tripDays = int(endDate.Sub(startDate).Hours()/24) + 1
		if tripDays < 1 {
			tripDays = 1
		}
	}

	dailyAvgSpent := totalSpent / float64(tripDays)
	budgetTargetPerDay := 0.0
	if tripDays > 0 {
		budgetTargetPerDay = totalBudgeted / float64(tripDays)
	}

	dailyDeltaPct := 0.0
	if budgetTargetPerDay > 0 {
		dailyDeltaPct = ((dailyAvgSpent - budgetTargetPerDay) / budgetTargetPerDay) * 100
	}
	dailyDeltaPctAbs := dailyDeltaPct
	if dailyDeltaPctAbs < 0 {
		dailyDeltaPctAbs = -dailyDeltaPctAbs
	}
	dailyDeltaPctAbsInt := int(dailyDeltaPctAbs + 0.5)
	dailyOverTarget := dailyDeltaPct > 0

	type categoryAggregate struct {
		Name   string
		Amount float64
		Count  int
	}
	categoryTotals := map[string]*categoryAggregate{}
	for _, e := range spentExpenses {
		name := strings.TrimSpace(e.Category)
		if name == "" {
			name = "Uncategorized"
		}
		if _, ok := categoryTotals[name]; !ok {
			categoryTotals[name] = &categoryAggregate{Name: name}
		}
		categoryTotals[name].Amount += e.Amount
		categoryTotals[name].Count++
	}

	ranked := make([]categoryAggregate, 0, len(categoryTotals))
	for _, agg := range categoryTotals {
		ranked = append(ranked, *agg)
	}
	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].Amount == ranked[j].Amount {
			return ranked[i].Name < ranked[j].Name
		}
		return ranked[i].Amount > ranked[j].Amount
	})

	segments := make([]budgetCategoryGroupView, 0, 4)
	topLimit := 3
	if topLimit > len(ranked) {
		topLimit = len(ranked)
	}
	for i := 0; i < topLimit; i++ {
		agg := ranked[i]
		segments = append(segments, budgetCategoryGroupView{
			ID:           "top-" + strconv.Itoa(i+1),
			Name:         agg.Name,
			Icon:         expenseCategoryIcon(agg.Name),
			IconStyle:    expenseCategoryStyle(agg.Name),
			DonutStyle:   "rank-" + strconv.Itoa(i+1),
			DonutStroke:  expenseCategoryStrokeColor(agg.Name),
			Amount:       agg.Amount,
			ExpenseCount: agg.Count,
		})
	}

	if len(ranked) > topLimit {
		otherAmount := 0.0
		otherCount := 0
		for i := topLimit; i < len(ranked); i++ {
			otherAmount += ranked[i].Amount
			otherCount += ranked[i].Count
		}
		segments = append(segments, budgetCategoryGroupView{
			ID:           "other",
			Name:         "Other Expenses",
			Icon:         expenseCategoryIcon("Miscellaneous"),
			IconStyle:    expenseCategoryStyle("Miscellaneous"),
			DonutStyle:   "other",
			DonutStroke:  expenseCategoryStrokeColor("Miscellaneous"),
			Amount:       otherAmount,
			ExpenseCount: otherCount,
		})
	}
	if len(segments) == 0 {
		segments = append(segments, budgetCategoryGroupView{
			ID:           "other",
			Name:         "Other Expenses",
			Icon:         expenseCategoryIcon("Miscellaneous"),
			IconStyle:    expenseCategoryStyle("Miscellaneous"),
			DonutStyle:   "other",
			DonutStroke:  expenseCategoryStrokeColor("Miscellaneous"),
			Amount:       0,
			ExpenseCount: 0,
		})
	}

	// Donut percentages + dash offsets.
	if totalSpent > 0 {
		remainingPct := 100
		cumulativePct := 0
		for i := range segments {
			seg := segments[i]
			percent := (seg.Amount / totalSpent) * 100
			percentInt := int(percent + 0.5)
			if i == len(segments)-1 {
				percentInt = remainingPct
			}
			if percentInt < 0 {
				percentInt = 0
			}
			segments[i].PercentInt = percentInt
			segments[i].DonutDashArrayA = percentInt
			segments[i].DonutDashArrayB = 100 - percentInt
			segments[i].DonutDashOffset = -cumulativePct

			remainingPct -= percentInt
			cumulativePct += percentInt
		}
	} else {
		for i := range segments {
			segments[i].PercentInt = 0
			segments[i].DonutDashArrayA = 0
			segments[i].DonutDashArrayB = 100
			segments[i].DonutDashOffset = 0
		}
	}

	// Transaction history (date desc), but excluding pending pay-at-pickup expenses.
	transactions := make([]budgetTransactionRowView, 0, len(spentExpenses))
	// Use SpentOn first (ISO date strings sort lexicographically); fallback to CreatedAt.
	sort.Slice(spentExpenses, func(i, j int) bool {
		di := spentExpenses[i].SpentOn
		dj := spentExpenses[j].SpentOn
		if di != "" && dj != "" && di != dj {
			return di > dj
		}
		if di != "" && dj == "" {
			return true
		}
		if di == "" && dj != "" {
			return false
		}
		return spentExpenses[i].CreatedAt.After(spentExpenses[j].CreatedAt)
	})

	const initialLimit = 10
	totalTx := len(spentExpenses)
	limit := initialLimit
	if limit > totalTx {
		limit = totalTx
	}
	for i := 0; i < limit; i++ {
		e := spentExpenses[i]
		dateLabel := ""
		if strings.TrimSpace(e.SpentOn) != "" {
			dateLabel = formatUIDate(e.SpentOn)
		}
		desc := e.Notes
		if strings.TrimSpace(desc) == "" {
			desc = "—"
		}
		vLocked := vehicleExpenseLocked[e.ID]
		fLocked := flightExpenseLocked[e.ID]
		canEdit := !details.Trip.IsArchived && e.LodgingID == "" && !vLocked && !fLocked
		transactions = append(transactions, budgetTransactionRowView{
			ExpenseID:     e.ID,
			DateLabel:     dateLabel,
			CategoryName:  e.Category,
			CategoryIcon:  expenseCategoryIcon(e.Category),
			CategoryStyle: expenseCategoryStyle(e.Category),
			Description:   desc,
			Method:        defaultIfEmpty(e.PaymentMethod, "Cash"),
			Amount:        e.Amount,
			SpentOn:       e.SpentOn,
			NotesRaw:      e.Notes,
			LodgingID:     e.LodgingID,
			VehicleLocked: vLocked,
			FlightLocked:  fLocked,
			CanEdit:       canEdit,
		})
	}

	dailyTrendIcon := "trending_down"
	dailyTrendClass := "budget-trend-down"
	if dailyOverTarget {
		dailyTrendIcon = "trending_up"
		dailyTrendClass = "budget-trend-up"
	}

	usedPercentInt := int(budgetProgress + 0.5)
	if usedPercentInt > 100 {
		usedPercentInt = 100
	}
	remainingPercentInt := 100 - usedPercentInt
	if remainingPercentInt < 0 {
		remainingPercentInt = 0
	}

	canShowAll := totalTx > len(transactions)

	pageData := map[string]any{
		"Trip":                   details.Trip,
		"Settings":               settings,
		"CSRFToken":              CSRFToken(r.Context()),
		"CurrencySymbol":         currencySymbol,
		"ExpenseCategories":      trips.QuickExpenseCategories,
		"TotalSpent":             totalSpent,
		"TotalBudgeted":          totalBudgeted,
		"Remaining":              remaining,
		"BudgetProgress":         budgetProgress,
		"DailyAvgSpent":          dailyAvgSpent,
		"BudgetTargetPerDay":     budgetTargetPerDay,
		"DailyDeltaPctAbsInt":    dailyDeltaPctAbsInt,
		"DailyTrendIcon":         dailyTrendIcon,
		"DailyTrendClass":        dailyTrendClass,
		"RemainingPercentInt":    remainingPercentInt,
		"TripDays":               tripDays,
		"BudgetGroups":           segments,
		"Transactions":           transactions,
		"HasTransactions":        len(transactions) > 0,
		"CanShowAllTransactions": canShowAll,
		"BudgetInitialLimit":     initialLimit,
		"VehicleExpenseLocked":   vehicleExpenseLocked,
		"FlightExpenseLocked":    flightExpenseLocked,
	}
	a.mergeTripSidebarContext(r.Context(), r, tripID, details, pageData, "budget")
	_ = a.templates.ExecuteTemplate(w, "budget.html", pageData)
}

func (a *app) budgetTransactionsRows(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")

	// Offset/limit for pagination.
	offset := 0
	limit := 10
	if v := strings.TrimSpace(r.URL.Query().Get("offset")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	if v := strings.TrimSpace(r.URL.Query().Get("limit")); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}

	details, err := a.tripService.GetTripDetailsVisible(r.Context(), tripID, CurrentUserID(r.Context()))
	if err != nil {
		if tripForbiddenOrMissing(err) {
			http.Error(w, "trip not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Pending vehicle expenses are pay-at-pickup costs that should not be counted as "spent" yet.
	now := time.Now()
	pendingExpenseIDs := map[string]struct{}{}
	for _, v := range details.Vehicles {
		if !v.PayAtPickUp || strings.TrimSpace(v.DropOffAt) == "" {
			continue
		}
		dropOffAt, err := time.Parse("2006-01-02T15:04", v.DropOffAt)
		if err != nil || !now.Before(dropOffAt) {
			continue
		}
		if v.RentalExpenseID != "" {
			pendingExpenseIDs[v.RentalExpenseID] = struct{}{}
		}
		if v.InsuranceExpenseID != "" {
			pendingExpenseIDs[v.InsuranceExpenseID] = struct{}{}
		}
	}

	spentExpenses := make([]trips.Expense, 0, len(details.Expenses))
	for _, e := range details.Expenses {
		if _, isPending := pendingExpenseIDs[e.ID]; isPending {
			continue
		}
		spentExpenses = append(spentExpenses, e)
	}

	// Sort: newest first (SpentOn first, fallback CreatedAt).
	sort.Slice(spentExpenses, func(i, j int) bool {
		di := spentExpenses[i].SpentOn
		dj := spentExpenses[j].SpentOn
		if di != "" && dj != "" && di != dj {
			return di > dj
		}
		if di != "" && dj == "" {
			return true
		}
		if di == "" && dj != "" {
			return false
		}
		return spentExpenses[i].CreatedAt.After(spentExpenses[j].CreatedAt)
	})

	// Pagination window.
	start := offset
	if start > len(spentExpenses) {
		start = len(spentExpenses)
	}
	end := start + limit
	if end > len(spentExpenses) {
		end = len(spentExpenses)
	}
	window := spentExpenses[start:end]

	currencySymbol := defaultIfEmpty(details.Trip.CurrencySymbol, "$")
	vehicleByExpenseID := trips.VehicleRentalByExpenseID(details.Vehicles)
	flightByExpenseID := trips.FlightByExpenseID(details.Flights)
	vehicleExpenseLocked := map[string]bool{}
	for id := range vehicleByExpenseID {
		vehicleExpenseLocked[id] = true
	}
	flightExpenseLocked := map[string]bool{}
	for id := range flightByExpenseID {
		flightExpenseLocked[id] = true
	}

	transactions := make([]budgetTransactionRowView, 0, len(window))
	for _, e := range window {
		dateLabel := ""
		if strings.TrimSpace(e.SpentOn) != "" {
			dateLabel = formatUIDate(e.SpentOn)
		}
		desc := e.Notes
		if strings.TrimSpace(desc) == "" {
			desc = "—"
		}
		vLocked := vehicleExpenseLocked[e.ID]
		fLocked := flightExpenseLocked[e.ID]
		canEdit := !details.Trip.IsArchived && e.LodgingID == "" && !vLocked && !fLocked
		transactions = append(transactions, budgetTransactionRowView{
			ExpenseID:     e.ID,
			DateLabel:     dateLabel,
			CategoryName:  e.Category,
			CategoryIcon:  expenseCategoryIcon(e.Category),
			CategoryStyle: expenseCategoryStyle(e.Category),
			Description:   desc,
			Method:        defaultIfEmpty(e.PaymentMethod, "Cash"),
			Amount:        e.Amount,
			SpentOn:       e.SpentOn,
			NotesRaw:      e.Notes,
			LodgingID:     e.LodgingID,
			VehicleLocked: vLocked,
			FlightLocked:  fLocked,
			CanEdit:       canEdit,
		})
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = a.templates.ExecuteTemplate(w, "budget_transactions_rows", map[string]any{
		"Trip":                 details.Trip,
		"Details":              details,
		"CSRFToken":            CSRFToken(r.Context()),
		"CurrencySymbol":       currencySymbol,
		"ExpenseCategories":    trips.QuickExpenseCategories,
		"Transactions":         transactions,
		"VehicleExpenseLocked": vehicleExpenseLocked,
		"FlightExpenseLocked":  flightExpenseLocked,
	})
}

func (a *app) exportBudgetReport(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")

	format := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("format")))
	if format == "" {
		format = "csv"
	}
	if format != "csv" {
		http.Error(w, "unsupported export format", http.StatusBadRequest)
		return
	}

	details, err := a.tripService.GetTripDetailsVisible(r.Context(), tripID, CurrentUserID(r.Context()))
	if err != nil {
		if tripForbiddenOrMissing(err) {
			http.Error(w, "trip not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	currencySymbol := defaultIfEmpty(details.Trip.CurrencySymbol, "$")

	// Pending vehicle expenses are pay-at-pickup costs that should not be counted as "spent" yet.
	now := time.Now()
	pendingExpenseIDs := map[string]struct{}{}
	for _, v := range details.Vehicles {
		if !v.PayAtPickUp || strings.TrimSpace(v.DropOffAt) == "" {
			continue
		}
		dropOffAt, err := time.Parse("2006-01-02T15:04", v.DropOffAt)
		if err != nil || !now.Before(dropOffAt) {
			continue
		}
		if v.RentalExpenseID != "" {
			pendingExpenseIDs[v.RentalExpenseID] = struct{}{}
		}
		if v.InsuranceExpenseID != "" {
			pendingExpenseIDs[v.InsuranceExpenseID] = struct{}{}
		}
	}

	spentExpenses := make([]trips.Expense, 0, len(details.Expenses))
	for _, e := range details.Expenses {
		if _, isPending := pendingExpenseIDs[e.ID]; isPending {
			continue
		}
		spentExpenses = append(spentExpenses, e)
	}

	// Sort: newest first (SpentOn first, fallback CreatedAt).
	sort.Slice(spentExpenses, func(i, j int) bool {
		di := spentExpenses[i].SpentOn
		dj := spentExpenses[j].SpentOn
		if di != "" && dj != "" && di != dj {
			return di > dj
		}
		if di != "" && dj == "" {
			return true
		}
		if di == "" && dj != "" {
			return false
		}
		return spentExpenses[i].CreatedAt.After(spentExpenses[j].CreatedAt)
	})

	filename := "budget-report-" + tripID + "-" + time.Now().Format("2006-01-02") + ".csv"
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	w.Header().Set("X-Content-Type-Options", "nosniff")

	writer := csv.NewWriter(w)
	if err := writer.Write([]string{"Date", "Category", "Description", "Method", "Amount"}); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	for _, e := range spentExpenses {
		dateLabel := "--"
		if strings.TrimSpace(e.SpentOn) != "" {
			dateLabel = formatUIDate(e.SpentOn)
		}
		desc := e.Notes
		if strings.TrimSpace(desc) == "" {
			desc = "—"
		}
		method := defaultIfEmpty(e.PaymentMethod, "Cash")
		amountStr := currencySymbol + strconv.FormatFloat(e.Amount, 'f', 2, 64)

		if err := writer.Write([]string{dateLabel, e.Category, desc, method, amountStr}); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}

	writer.Flush()
	if err := writer.Error(); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

// computeTotalBudgeted sums planned costs: each lodging counts once (lodging.Cost), other
// itinerary lines use EstCost so hotel stays are not double-counted on check-in + check-out.
func computeTotalBudgeted(items []trips.ItineraryItem, lodgings []trips.Lodging, vehicles []trips.VehicleRental, flights []trips.Flight) float64 {
	byItem := trips.LodgingByItineraryItemID(lodgings, items)
	byVehicleItem := trips.VehicleRentalByItineraryItemID(vehicles, items)
	byFlightItem := trips.FlightByItineraryItemID(flights, items)
	seenLodging := map[string]struct{}{}
	seenVehicle := map[string]struct{}{}
	seenFlight := map[string]struct{}{}
	var sum float64
	for _, i := range items {
		if l, ok := byItem[i.ID]; ok && l.ID != "" {
			if _, seen := seenLodging[l.ID]; !seen {
				sum += l.Cost
				seenLodging[l.ID] = struct{}{}
			}
			continue
		}
		if v, ok := byVehicleItem[i.ID]; ok && v.ID != "" {
			if _, seen := seenVehicle[v.ID]; !seen {
				sum += v.Cost + v.InsuranceCost
				seenVehicle[v.ID] = struct{}{}
			}
			continue
		}
		if f, ok := byFlightItem[i.ID]; ok && f.ID != "" {
			if _, seen := seenFlight[f.ID]; !seen {
				sum += f.Cost
				seenFlight[f.ID] = struct{}{}
			}
			continue
		}
		sum += i.EstCost
	}
	return sum
}

// itineraryGeocodeQuery returns the best free-text location for client-side geocoding (itinerary connectors).
func itineraryGeocodeQuery(v itineraryItemView) string {
	if v.Lodging.ID != "" {
		if a := strings.TrimSpace(v.Lodging.Address); a != "" {
			return a
		}
	}
	if v.Vehicle.ID != "" {
		if p := strings.TrimSpace(v.Vehicle.PickUpLocation); p != "" {
			return p
		}
	}
	return strings.TrimSpace(v.Item.Location)
}

func buildItineraryDayGroups(startDate string, items []trips.ItineraryItem, lodgings []trips.Lodging, vehicles []trips.VehicleRental, flights []trips.Flight, dayLabels map[int]string) []itineraryDayGroup {
	groups := make([]itineraryDayGroup, 0)
	indexByDay := make(map[int]int)
	parsedStart, hasStart := time.Parse("2006-01-02", startDate)
	byItem := trips.LodgingByItineraryItemID(lodgings, items)
	byVehicleItem := trips.VehicleRentalByItineraryItemID(vehicles, items)
	byFlightItem := trips.FlightByItineraryItemID(flights, items)
	for _, item := range items {
		idx, exists := indexByDay[item.DayNumber]
		if !exists {
			dateLabel := ""
			if hasStart == nil {
				dateLabel = parsedStart.AddDate(0, 0, item.DayNumber-1).Format("2006-01-02")
			}
			groups = append(groups, itineraryDayGroup{
				DayNumber:      item.DayNumber,
				DateLabel:      dateLabel,
				DayDescription: strings.TrimSpace(dayLabels[item.DayNumber]),
				Items:          []itineraryItemView{},
			})
			idx = len(groups) - 1
			indexByDay[item.DayNumber] = idx
		}
		view := itineraryItemView{Item: item}
		if l, ok := byItem[item.ID]; ok {
			view.Lodging = l
		}
		if v, ok := byVehicleItem[item.ID]; ok {
			view.Vehicle = v
		}
		if f, ok := byFlightItem[item.ID]; ok {
			view.Flight = f
		}
		groups[idx].Items = append(groups[idx].Items, view)
	}
	for i := range groups {
		sort.SliceStable(groups[i].Items, func(a, b int) bool {
			left := groups[i].Items[a]
			right := groups[i].Items[b]
			leftMinutes, leftHas := itineraryTimeSortKey(left.Item.StartTime)
			rightMinutes, rightHas := itineraryTimeSortKey(right.Item.StartTime)
			if leftHas != rightHas {
				return leftHas
			}
			if leftHas && rightHas && leftMinutes != rightMinutes {
				return leftMinutes < rightMinutes
			}
			leftEnd, leftEndHas := itineraryTimeSortKey(left.Item.EndTime)
			rightEnd, rightEndHas := itineraryTimeSortKey(right.Item.EndTime)
			if leftEndHas != rightEndHas {
				return leftEndHas
			}
			if leftEndHas && rightEndHas && leftEnd != rightEnd {
				return leftEnd < rightEnd
			}
			return left.Item.CreatedAt.Before(right.Item.CreatedAt)
		})
	}
	return groups
}

func itineraryTimeSortKey(raw string) (minutes int, ok bool) {
	t := strings.TrimSpace(raw)
	if t == "" {
		return 0, false
	}
	parsed, err := time.Parse("15:04", t)
	if err != nil {
		return 0, false
	}
	return parsed.Hour()*60 + parsed.Minute(), true
}

func buildExpenseDayGroups(startDate string, expenses []trips.Expense) []expenseDayGroup {
	groupMap := make(map[string][]trips.Expense)
	for _, expense := range expenses {
		groupMap[expense.SpentOn] = append(groupMap[expense.SpentOn], expense)
	}

	keys := make([]string, 0, len(groupMap))
	for k := range groupMap {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		if keys[i] == "" {
			return false
		}
		if keys[j] == "" {
			return true
		}
		return keys[i] < keys[j]
	})

	start, startErr := time.Parse("2006-01-02", startDate)
	out := make([]expenseDayGroup, 0, len(keys))
	for _, key := range keys {
		dayNum := 0
		if key != "" && startErr == nil {
			if d, err := time.Parse("2006-01-02", key); err == nil {
				dayNum = int(d.Sub(start).Hours()/24) + 1
				if dayNum < 1 {
					dayNum = 0
				}
			}
		}
		out = append(out, expenseDayGroup{
			DayNumber: dayNum,
			DateLabel: key,
			Items:     groupMap[key],
		})
	}
	return out
}

func buildChecklistCategoryGroups(items []trips.ChecklistItem, orderedCategories []string) []checklistCategoryGroup {
	grouped := make(map[string][]trips.ChecklistItem)
	for _, item := range items {
		category := strings.TrimSpace(item.Category)
		if category == "" {
			category = "Packing List"
		}
		grouped[category] = append(grouped[category], item)
	}

	out := make([]checklistCategoryGroup, 0, len(grouped))
	seen := make(map[string]struct{}, len(grouped))
	for _, category := range orderedCategories {
		itemsForCategory := grouped[category]
		if len(itemsForCategory) == 0 {
			continue
		}
		out = append(out, checklistCategoryGroup{
			Category: category,
			Items:    itemsForCategory,
		})
		seen[category] = struct{}{}
	}
	for category, itemsForCategory := range grouped {
		if _, ok := seen[category]; ok {
			continue
		}
		out = append(out, checklistCategoryGroup{
			Category: category,
			Items:    itemsForCategory,
		})
	}
	return out
}

// redirectIfTripSectionDisabled sends the user back to the trip page when a section is turned off in trip settings.
func (a *app) redirectIfTripSectionDisabled(w http.ResponseWriter, r *http.Request, trip trips.Trip, section string) bool {
	switch section {
	case "stay":
		if !trip.SectionEnabledStay() {
			http.Redirect(w, r, "/trips/"+trip.ID, http.StatusSeeOther)
			return true
		}
	case "vehicle":
		if !trip.SectionEnabledVehicle() {
			http.Redirect(w, r, "/trips/"+trip.ID, http.StatusSeeOther)
			return true
		}
	case "flights":
		if !trip.SectionEnabledFlights() {
			http.Redirect(w, r, "/trips/"+trip.ID, http.StatusSeeOther)
			return true
		}
	case "spends":
		if !trip.SectionEnabledSpends() {
			http.Redirect(w, r, "/trips/"+trip.ID, http.StatusSeeOther)
			return true
		}
	case "itinerary":
		if !trip.SectionEnabledItinerary() {
			http.Redirect(w, r, "/trips/"+trip.ID, http.StatusSeeOther)
			return true
		}
	case "checklist":
		if !trip.SectionEnabledChecklist() {
			http.Redirect(w, r, "/trips/"+trip.ID, http.StatusSeeOther)
			return true
		}
	}
	return false
}

// formTriSectionOn reports whether a trip "show section" control was checked.
// Unchecked HTML checkboxes are omitted from the body; we pair each checkbox with
// a hidden "0" and may receive multiple values for the same name, so we treat the
// section as enabled if any submitted value is on/1/true (not only FormValue's first).
func formTriSectionOn(r *http.Request, key string) bool {
	for _, raw := range r.Form[key] {
		v := strings.TrimSpace(strings.ToLower(raw))
		if v == "on" || v == "1" || v == "true" || v == "yes" {
			return true
		}
	}
	return false
}

func (a *app) updateTrip(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	existing, err := a.tripService.GetTrip(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	existing.Name = r.FormValue("name")
	existing.Description = r.FormValue("description")
	existing.StartDate = r.FormValue("start_date")
	existing.EndDate = r.FormValue("end_date")
	existing.CoverImage = r.FormValue("cover_image_url")
	existing.CurrencyName = defaultIfEmpty(r.FormValue("currency_name"), "USD")
	existing.CurrencySymbol = defaultIfEmpty(r.FormValue("currency_symbol"), "$")
	existing.UIShowItinerary = formTriSectionOn(r, "ui_trip_section_itinerary")
	existing.UIShowChecklist = formTriSectionOn(r, "ui_trip_section_checklist")
	var mainHidden []string
	for _, k := range trips.DefaultMainSectionOrder {
		switch k {
		case trips.MainSectionStay:
			existing.UIShowStay = formTriSectionOn(r, "ui_trip_section_stay")
		case trips.MainSectionVehicle:
			existing.UIShowVehicle = formTriSectionOn(r, "ui_trip_section_vehicle")
		case trips.MainSectionFlights:
			existing.UIShowFlights = formTriSectionOn(r, "ui_trip_section_flights")
		case trips.MainSectionSpends:
			existing.UIShowSpends = formTriSectionOn(r, "ui_trip_section_spends")
		}
		visOn := formTriSectionOn(r, "ui_vis_main_"+k)
		if !visOn {
			mainHidden = append(mainHidden, k)
		}
	}
	existing.UIMainSectionHidden = strings.Join(mainHidden, ",")

	var sidebarHidden []string
	for _, k := range trips.DefaultSidebarWidgetOrder {
		if !formTriSectionOn(r, "ui_vis_sidebar_"+k) {
			sidebarHidden = append(sidebarHidden, k)
		}
	}
	existing.UISidebarWidgetHidden = strings.Join(sidebarHidden, ",")
	itExp := strings.ToLower(strings.TrimSpace(r.FormValue("ui_itinerary_expand")))
	if itExp != "all" && itExp != "none" {
		itExp = "first"
	}
	existing.UIItineraryExpand = itExp
	spExp := strings.ToLower(strings.TrimSpace(r.FormValue("ui_spends_expand")))
	if spExp != "all" && spExp != "none" {
		spExp = "first"
	}
	existing.UISpendsExpand = spExp
	tf := strings.ToLower(strings.TrimSpace(r.FormValue("ui_time_format")))
	if tf != "24h" {
		tf = "12h"
	}
	existing.UITimeFormat = tf
	existing.UILabelStay = strings.TrimSpace(r.FormValue("ui_label_stay"))
	existing.UILabelVehicle = strings.TrimSpace(r.FormValue("ui_label_vehicle"))
	existing.UILabelFlights = strings.TrimSpace(r.FormValue("ui_label_flights"))
	existing.UILabelSpends = strings.TrimSpace(r.FormValue("ui_label_spends"))
	existing.UIMainSectionOrder = trips.JoinMainSectionOrder(trips.NormalizeMainSectionOrder(r.FormValue("ui_main_section_order")))
	existing.UISidebarWidgetOrder = trips.JoinSidebarWidgetOrder(trips.NormalizeSidebarWidgetOrder(r.FormValue("ui_sidebar_widget_order")))
	existing.UIShowCustomLinks = formTriSectionOn(r, "ui_show_custom_links")
	customLinks, err := trips.CustomSidebarLinksFromForm(r.FormValue("ui_custom_link_slot_order"), func(k string) string { return r.FormValue(k) })
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	existing.UICustomSidebarLinks = trips.EncodeCustomSidebarLinksJSON(customLinks)

	err = a.tripService.UpdateTrip(r.Context(), existing)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if isAsyncRequest(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	if ret := strings.TrimSpace(r.FormValue("return_to")); ret != "" && isSafeReturnForTrip(ret, tripID) {
		http.Redirect(w, r, ret, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
}

func (a *app) resetTripUIPresets(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	if err := a.tripService.ResetTripUIPresets(r.Context(), tripID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	returnTo := strings.TrimSpace(r.FormValue("return_to"))
	if returnTo != "" && isSafeReturnForTrip(returnTo, tripID) {
		joiner := "?"
		if strings.Contains(returnTo, "?") {
			joiner = "&"
		}
		http.Redirect(w, r, returnTo+joiner+"reset=1", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID+"/settings?reset=1", http.StatusSeeOther)
}

func (a *app) archiveTrip(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	if acc, ok := TripAccessFromContext(r.Context()); !ok || !acc.IsOwner {
		http.Error(w, "only the owner can archive this trip", http.StatusForbidden)
		return
	}
	if err := a.tripService.ArchiveTrip(r.Context(), tripID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if isAsyncRequest(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
}

func (a *app) deleteTrip(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	if acc, ok := TripAccessFromContext(r.Context()); !ok || !acc.IsOwner {
		http.Error(w, "only the owner can delete this trip", http.StatusForbidden)
		return
	}
	if err := a.tripService.DeleteTrip(r.Context(), tripID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *app) addItineraryItem(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	_ = r.ParseForm()
	trip, err := a.tripService.GetTrip(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if a.redirectIfTripSectionDisabled(w, r, trip, "itinerary") {
		return
	}
	day, err := dayNumberFromDate(trip.StartDate, trip.EndDate, r.FormValue("itinerary_date"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	lat, _ := strconv.ParseFloat(r.FormValue("latitude"), 64)
	lng, _ := strconv.ParseFloat(r.FormValue("longitude"), 64)
	estCost, _ := strconv.ParseFloat(r.FormValue("est_cost"), 64)
	err = a.tripService.AddItineraryItem(r.Context(), trips.ItineraryItem{
		TripID:    tripID,
		DayNumber: day,
		Title:     r.FormValue("title"),
		Notes:     r.FormValue("notes"),
		Location:  r.FormValue("location"),
		Latitude:  lat,
		Longitude: lng,
		EstCost:   estCost,
		StartTime: r.FormValue("start_time"),
		EndTime:   r.FormValue("end_time"),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if isAsyncRequest(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
}

func (a *app) saveTripDayLabel(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	dayNumber, err := strconv.Atoi(chi.URLParam(r, "dayNumber"))
	if err != nil || dayNumber < 1 {
		http.Error(w, "invalid day number", http.StatusBadRequest)
		return
	}
	if strings.Contains(strings.ToLower(r.Header.Get("Content-Type")), "multipart/form-data") {
		_ = r.ParseMultipartForm(2 << 20)
	} else {
		_ = r.ParseForm()
	}
	trip, err := a.tripService.GetTrip(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if a.redirectIfTripSectionDisabled(w, r, trip, "itinerary") {
		return
	}
	label := strings.TrimSpace(r.FormValue("day_label"))
	if err := a.tripService.SaveTripDayLabel(r.Context(), tripID, dayNumber, label); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if isAsyncRequest(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
}

func (a *app) addExpense(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	_ = r.ParseForm()
	trip, err := a.tripService.GetTrip(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if a.redirectIfTripSectionDisabled(w, r, trip, "spends") {
		return
	}
	amount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)
	paymentMethod := strings.TrimSpace(r.FormValue("payment_method"))
	if paymentMethod == "" {
		paymentMethod = "Cash"
	}
	err = a.tripService.AddExpense(r.Context(), trips.Expense{
		TripID:        tripID,
		Category:      r.FormValue("category"),
		Amount:        amount,
		Notes:         r.FormValue("notes"),
		SpentOn:       r.FormValue("spent_on"),
		PaymentMethod: paymentMethod,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	next := strings.TrimSpace(r.FormValue("return_to"))
	if next != "" && isSafeReturnForTrip(next, tripID) {
		http.Redirect(w, r, next, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
}

func (a *app) updateItineraryItem(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	itemID := chi.URLParam(r, "itemID")
	_ = r.ParseForm()
	details, err := a.tripService.GetTripDetailsVisible(r.Context(), tripID, CurrentUserID(r.Context()))
	if err != nil {
		if tripForbiddenOrMissing(err) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	trip := details.Trip
	if a.redirectIfTripSectionDisabled(w, r, trip, "itinerary") {
		return
	}
	if l, ok := trips.LodgingByItineraryItemID(details.Lodgings, details.Itinerary)[itemID]; ok && l.ID != "" {
		http.Error(w, "This stop is linked to Accommodation. Use the accommodation form opened from Edit on this item.", http.StatusBadRequest)
		return
	}
	if v, ok := trips.VehicleRentalByItineraryItemID(details.Vehicles, details.Itinerary)[itemID]; ok && v.ID != "" {
		http.Error(w, "This stop is linked to Vehicle Rental. Use the vehicle rental form opened from Edit on this item.", http.StatusBadRequest)
		return
	}
	if f, ok := trips.FlightByItineraryItemID(details.Flights, details.Itinerary)[itemID]; ok && f.ID != "" {
		http.Error(w, "This stop is linked to Flights. Use the flight form opened from Edit on this item.", http.StatusBadRequest)
		return
	}
	day, err := dayNumberFromDate(trip.StartDate, trip.EndDate, r.FormValue("itinerary_date"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	estCost, _ := strconv.ParseFloat(r.FormValue("est_cost"), 64)
	err = a.tripService.UpdateItineraryItem(r.Context(), trips.ItineraryItem{
		ID:        itemID,
		TripID:    tripID,
		DayNumber: day,
		Title:     r.FormValue("title"),
		Location:  r.FormValue("location"),
		Notes:     r.FormValue("notes"),
		EstCost:   estCost,
		StartTime: r.FormValue("start_time"),
		EndTime:   r.FormValue("end_time"),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if isAsyncRequest(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
}

func (a *app) deleteItineraryItem(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	itemID := chi.URLParam(r, "itemID")
	details, err := a.tripService.GetTripDetailsVisible(r.Context(), tripID, CurrentUserID(r.Context()))
	if err != nil {
		if tripForbiddenOrMissing(err) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if a.redirectIfTripSectionDisabled(w, r, details.Trip, "itinerary") {
		return
	}
	if l, ok := trips.LodgingByItineraryItemID(details.Lodgings, details.Itinerary)[itemID]; ok && l.ID != "" {
		http.Error(w, "Remove this stay from Accommodation instead of deleting the itinerary line.", http.StatusBadRequest)
		return
	}
	if v, ok := trips.VehicleRentalByItineraryItemID(details.Vehicles, details.Itinerary)[itemID]; ok && v.ID != "" {
		http.Error(w, "Remove this booking from Vehicle Rental instead of deleting the itinerary line.", http.StatusBadRequest)
		return
	}
	if f, ok := trips.FlightByItineraryItemID(details.Flights, details.Itinerary)[itemID]; ok && f.ID != "" {
		http.Error(w, "Remove this booking from Flights instead of deleting the itinerary line.", http.StatusBadRequest)
		return
	}
	if err := a.tripService.DeleteItineraryItem(r.Context(), tripID, itemID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
}

func (a *app) updateExpense(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	expenseID := chi.URLParam(r, "expenseID")
	_ = r.ParseForm()
	amount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)
	paymentMethod := strings.TrimSpace(r.FormValue("payment_method"))
	if paymentMethod == "" {
		paymentMethod = "Cash"
	}
	err := a.tripService.UpdateExpense(r.Context(), trips.Expense{
		ID:            expenseID,
		TripID:        tripID,
		Category:      r.FormValue("category"),
		Amount:        amount,
		Notes:         r.FormValue("notes"),
		SpentOn:       r.FormValue("spent_on"),
		PaymentMethod: paymentMethod,
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if isAsyncRequest(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
}

func (a *app) deleteExpense(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	expenseID := chi.URLParam(r, "expenseID")
	if err := a.tripService.DeleteExpense(r.Context(), tripID, expenseID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
}

func (a *app) addChecklistItem(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	_ = r.ParseForm()
	trip, err := a.tripService.GetTrip(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if a.redirectIfTripSectionDisabled(w, r, trip, "checklist") {
		return
	}
	category := strings.TrimSpace(r.FormValue("category"))
	if category == "" {
		category = "Packing List"
	}
	itemsJSON := strings.TrimSpace(r.FormValue("items_json"))
	if itemsJSON != "" {
		var pendingItems []string
		if err := json.Unmarshal([]byte(itemsJSON), &pendingItems); err != nil {
			http.Error(w, "invalid checklist items payload", http.StatusBadRequest)
			return
		}
		for _, text := range pendingItems {
			trimmed := strings.TrimSpace(text)
			if trimmed == "" {
				continue
			}
			err := a.tripService.AddChecklistItem(r.Context(), trips.ChecklistItem{
				TripID:   tripID,
				Category: category,
				Text:     trimmed,
			})
			if err != nil {
				if errors.Is(err, sql.ErrNoRows) {
					http.Redirect(w, r, "/", http.StatusSeeOther)
					return
				}
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
	} else {
		err := a.tripService.AddChecklistItem(r.Context(), trips.ChecklistItem{
			TripID:   tripID,
			Category: category,
			Text:     strings.TrimSpace(r.FormValue("text")),
		})
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				http.Redirect(w, r, "/", http.StatusSeeOther)
				return
			}
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
}

func (a *app) redirectLegacyLodgingPath(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	http.Redirect(w, r, "/trips/"+tripID+"/accommodation", http.StatusMovedPermanently)
}

func (a *app) accommodationPage(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	details, err := a.tripService.GetTripDetailsVisible(r.Context(), tripID, CurrentUserID(r.Context()))
	if err != nil {
		if tripForbiddenOrMissing(err) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if a.redirectIfTripSectionDisabled(w, r, details.Trip, "stay") {
		return
	}
	uid := CurrentUserID(r.Context())
	settings, err := a.tripService.MergedSettingsForUI(r.Context(), uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	currencySymbol := defaultIfEmpty(details.Trip.CurrencySymbol, "$")
	pageData := map[string]any{
		"Details":        details,
		"Settings":       settings,
		"CSRFToken":      CSRFToken(r.Context()),
		"CurrencySymbol": currencySymbol,
	}
	a.mergeTripSidebarContext(r.Context(), r, tripID, details, pageData, "stay")
	_ = a.templates.ExecuteTemplate(w, "accommodation.html", pageData)
}

func (a *app) addLodging(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "invalid accommodation form", http.StatusBadRequest)
		return
	}
	trip, err := a.tripService.GetTrip(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if a.redirectIfTripSectionDisabled(w, r, trip, "stay") {
		return
	}

	checkInAt, checkInDate, checkInTime, err := parseDateTimeLocal(r.FormValue("check_in_at"))
	if err != nil {
		http.Error(w, "invalid check-in date/time", http.StatusBadRequest)
		return
	}
	checkOutAt, checkOutDate, checkOutTime, err := parseDateTimeLocal(r.FormValue("check_out_at"))
	if err != nil {
		http.Error(w, "invalid check-out date/time", http.StatusBadRequest)
		return
	}
	checkInDay, err := dayNumberFromDate(trip.StartDate, trip.EndDate, checkInDate)
	if err != nil {
		http.Error(w, "check-in date must be within trip dates", http.StatusBadRequest)
		return
	}
	checkOutDay, err := dayNumberFromDate(trip.StartDate, trip.EndDate, checkOutDate)
	if err != nil {
		http.Error(w, "check-out date must be within trip dates", http.StatusBadRequest)
		return
	}
	if checkOutAt.Before(checkInAt) {
		http.Error(w, "check-out must be after check-in", http.StatusBadRequest)
		return
	}

	cost, _ := strconv.ParseFloat(r.FormValue("cost"), 64)
	attachmentPath, err := storeBookingAttachment(r, "booking_attachment")
	if err != nil {
		http.Error(w, "failed to save booking attachment", http.StatusBadRequest)
		return
	}
	name := r.FormValue("name")
	address := r.FormValue("address")
	bookingNo := r.FormValue("booking_confirmation")
	notes := r.FormValue("notes")

	lodgingID := uuid.NewString()
	checkInItemID := uuid.NewString()
	checkOutItemID := uuid.NewString()
	checkInNotes := buildLodgingCheckInNotes(notes, bookingNo, attachmentPath)
	addrLat, addrLng := geocodeLocation(r.Context(), address)

	if err := a.tripService.AddItineraryItem(r.Context(), trips.ItineraryItem{
		ID:        checkInItemID,
		TripID:    tripID,
		DayNumber: checkInDay,
		Title:     trips.AccommodationItineraryCheckInTitle(name),
		Location:  address,
		Latitude:  addrLat,
		Longitude: addrLng,
		Notes:     checkInNotes,
		EstCost:   cost,
		StartTime: checkInTime,
		EndTime:   checkInTime,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := a.tripService.AddItineraryItem(r.Context(), trips.ItineraryItem{
		ID:        checkOutItemID,
		TripID:    tripID,
		DayNumber: checkOutDay,
		Title:     trips.AccommodationItineraryCheckOutTitle(name),
		Location:  address,
		Latitude:  addrLat,
		Longitude: addrLng,
		Notes:     defaultIfEmpty(notes, ""),
		EstCost:   cost,
		StartTime: checkOutTime,
		EndTime:   checkOutTime,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = a.tripService.AddLodging(r.Context(), trips.Lodging{
		ID:                  lodgingID,
		TripID:              tripID,
		Name:                name,
		Address:             address,
		CheckInAt:           checkInAt.Format("2006-01-02T15:04"),
		CheckOutAt:          checkOutAt.Format("2006-01-02T15:04"),
		BookingConfirmation: bookingNo,
		Cost:                cost,
		Notes:               notes,
		AttachmentPath:      attachmentPath,
		CheckInItineraryID:  checkInItemID,
		CheckOutItineraryID: checkOutItemID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if isAsyncRequest(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID+"/accommodation", http.StatusSeeOther)
}

func (a *app) updateLodging(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	lodgingID := chi.URLParam(r, "lodgingID")
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "invalid accommodation form", http.StatusBadRequest)
		return
	}
	existing, err := a.tripService.GetLodging(r.Context(), tripID, lodgingID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/trips/"+tripID+"/accommodation", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	trip, err := a.tripService.GetTrip(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	checkInAt, checkInDate, checkInTime, err := parseDateTimeLocal(r.FormValue("check_in_at"))
	if err != nil {
		http.Error(w, "invalid check-in date/time", http.StatusBadRequest)
		return
	}
	checkOutAt, checkOutDate, checkOutTime, err := parseDateTimeLocal(r.FormValue("check_out_at"))
	if err != nil {
		http.Error(w, "invalid check-out date/time", http.StatusBadRequest)
		return
	}
	checkInDay, err := dayNumberFromDate(trip.StartDate, trip.EndDate, checkInDate)
	if err != nil {
		http.Error(w, "check-in date must be within trip dates", http.StatusBadRequest)
		return
	}
	checkOutDay, err := dayNumberFromDate(trip.StartDate, trip.EndDate, checkOutDate)
	if err != nil {
		http.Error(w, "check-out date must be within trip dates", http.StatusBadRequest)
		return
	}
	if checkOutAt.Before(checkInAt) {
		http.Error(w, "check-out must be after check-in", http.StatusBadRequest)
		return
	}

	attachmentPath, err := storeBookingAttachment(r, "booking_attachment")
	if err != nil {
		http.Error(w, "failed to save booking attachment", http.StatusBadRequest)
		return
	}
	removeAttachment := r.FormValue("remove_attachment") == "true"
	if attachmentPath == "" {
		attachmentPath = r.FormValue("current_attachment_path")
	}
	if removeAttachment && r.FormValue("current_attachment_path") != "" && attachmentPath == r.FormValue("current_attachment_path") {
		_ = deleteUploadedFileByWebPath(attachmentPath)
		attachmentPath = ""
	}
	if attachmentPath != "" && r.FormValue("current_attachment_path") != "" && attachmentPath != r.FormValue("current_attachment_path") {
		_ = deleteUploadedFileByWebPath(r.FormValue("current_attachment_path"))
	}
	cost, _ := strconv.ParseFloat(r.FormValue("cost"), 64)
	name := r.FormValue("name")
	address := r.FormValue("address")
	bookingNo := r.FormValue("booking_confirmation")
	notes := r.FormValue("notes")
	addrLat, addrLng := geocodeLocation(r.Context(), address)

	checkInNotes := buildLodgingCheckInNotes(notes, bookingNo, attachmentPath)
	lodging := trips.Lodging{
		ID:                  lodgingID,
		TripID:              tripID,
		Name:                name,
		Address:             address,
		Latitude:            addrLat,
		Longitude:           addrLng,
		CheckInAt:           checkInAt.Format("2006-01-02T15:04"),
		CheckOutAt:          checkOutAt.Format("2006-01-02T15:04"),
		BookingConfirmation: bookingNo,
		Cost:                cost,
		Notes:               notes,
		AttachmentPath:      attachmentPath,
		CheckInItineraryID:  existing.CheckInItineraryID,
		CheckOutItineraryID: existing.CheckOutItineraryID,
	}
	lodging, err = a.tripService.SyncLodgingItinerary(r.Context(), trip, lodging, existing.Name,
		checkInDay, checkInTime, checkOutDay, checkOutTime, checkInNotes, defaultIfEmpty(notes, ""))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	err = a.tripService.UpdateLodging(r.Context(), lodging)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if isAsyncRequest(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	next := strings.TrimSpace(r.FormValue("return_to"))
	if next != "" && isSafeReturnForTrip(next, tripID) {
		http.Redirect(w, r, next, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID+"/accommodation", http.StatusSeeOther)
}

func (a *app) deleteLodging(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	lodgingID := chi.URLParam(r, "lodgingID")
	_ = r.ParseForm()
	if err := a.tripService.DeleteLodging(r.Context(), tripID, lodgingID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	next := strings.TrimSpace(r.FormValue("return_to"))
	if next != "" && isSafeReturnForTrip(next, tripID) {
		http.Redirect(w, r, next, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID+"/accommodation", http.StatusSeeOther)
}

func (a *app) vehicleRentalPage(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	details, err := a.tripService.GetTripDetailsVisible(r.Context(), tripID, CurrentUserID(r.Context()))
	if err != nil {
		if tripForbiddenOrMissing(err) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	for _, v := range details.Vehicles {
		if err := a.tripService.SyncExpenseForVehicleRental(r.Context(), v); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	details, err = a.tripService.GetTripDetailsVisible(r.Context(), tripID, CurrentUserID(r.Context()))
	if err != nil {
		if tripForbiddenOrMissing(err) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if a.redirectIfTripSectionDisabled(w, r, details.Trip, "vehicle") {
		return
	}
	uid := CurrentUserID(r.Context())
	settings, err := a.tripService.MergedSettingsForUI(r.Context(), uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	currencySymbol := defaultIfEmpty(details.Trip.CurrencySymbol, "$")
	pageData := map[string]any{
		"Details":        details,
		"Settings":       settings,
		"CSRFToken":      CSRFToken(r.Context()),
		"CurrencySymbol": currencySymbol,
	}
	a.mergeTripSidebarContext(r.Context(), r, tripID, details, pageData, "vehicle")
	_ = a.templates.ExecuteTemplate(w, "vehicle_rental.html", pageData)
}

func (a *app) addVehicleRental(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "invalid vehicle rental form", http.StatusBadRequest)
		return
	}
	trip, err := a.tripService.GetTrip(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if a.redirectIfTripSectionDisabled(w, r, trip, "vehicle") {
		return
	}
	pickUpAt, pickUpDate, pickUpTime, err := parseDateTimeLocal(r.FormValue("pick_up_at"))
	if err != nil {
		http.Error(w, "invalid pick up date/time", http.StatusBadRequest)
		return
	}
	dropOffAt, dropOffDate, dropOffTime, err := parseDateTimeLocal(r.FormValue("drop_off_at"))
	if err != nil {
		http.Error(w, "invalid drop off date/time", http.StatusBadRequest)
		return
	}
	pickUpDay, err := dayNumberFromDate(trip.StartDate, trip.EndDate, pickUpDate)
	if err != nil {
		http.Error(w, "pick up date must be within trip dates", http.StatusBadRequest)
		return
	}
	dropOffDay, err := dayNumberFromDate(trip.StartDate, trip.EndDate, dropOffDate)
	if err != nil {
		http.Error(w, "drop off date must be within trip dates", http.StatusBadRequest)
		return
	}
	if dropOffAt.Before(pickUpAt) {
		http.Error(w, "drop off must be after pick up", http.StatusBadRequest)
		return
	}
	cost, _ := strconv.ParseFloat(r.FormValue("cost"), 64)
	insuranceCost, _ := strconv.ParseFloat(r.FormValue("insurance_cost"), 64)
	totalCost := cost + insuranceCost
	location := r.FormValue("pick_up_location")
	vehicleDetail := r.FormValue("vehicle_detail")
	vehicleTitle := vehicleRentalTitleValue(vehicleDetail, location)
	booking := r.FormValue("booking_confirmation")
	notes := r.FormValue("notes")
	payAtPickUp := r.FormValue("pay_at_pick_up") == "true"
	vehicleImagePath, err := storeVehicleImage(r, "vehicle_image")
	if err != nil {
		http.Error(w, "failed to save vehicle image", http.StatusBadRequest)
		return
	}

	rentalID := uuid.NewString()
	pickUpItineraryID := uuid.NewString()
	dropOffItineraryID := uuid.NewString()
	pickUpNotes := buildVehicleItineraryNotes(notes, booking, payAtPickUp)
	dropOffNotes := defaultIfEmpty(notes, "")
	locLat, locLng := geocodeLocation(r.Context(), location)

	if err := a.tripService.AddItineraryItem(r.Context(), trips.ItineraryItem{
		ID:        pickUpItineraryID,
		TripID:    tripID,
		DayNumber: pickUpDay,
		Title:     trips.VehicleRentalItineraryPickUpTitle(vehicleTitle),
		Location:  location,
		Latitude:  locLat,
		Longitude: locLng,
		Notes:     pickUpNotes,
		EstCost:   totalCost,
		StartTime: pickUpTime,
		EndTime:   pickUpTime,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := a.tripService.AddItineraryItem(r.Context(), trips.ItineraryItem{
		ID:        dropOffItineraryID,
		TripID:    tripID,
		DayNumber: dropOffDay,
		Title:     trips.VehicleRentalItineraryDropOffTitle(vehicleTitle),
		Location:  location,
		Latitude:  locLat,
		Longitude: locLng,
		Notes:     dropOffNotes,
		EstCost:   totalCost,
		StartTime: dropOffTime,
		EndTime:   dropOffTime,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = a.tripService.AddVehicleRental(r.Context(), trips.VehicleRental{
		ID:                  rentalID,
		TripID:              tripID,
		PickUpLocation:      location,
		VehicleDetail:       vehicleDetail,
		PickUpAt:            pickUpAt.Format("2006-01-02T15:04"),
		DropOffAt:           dropOffAt.Format("2006-01-02T15:04"),
		BookingConfirmation: booking,
		Notes:               notes,
		VehicleImagePath:    vehicleImagePath,
		Cost:                cost,
		InsuranceCost:       insuranceCost,
		PayAtPickUp:         payAtPickUp,
		PickUpItineraryID:   pickUpItineraryID,
		DropOffItineraryID:  dropOffItineraryID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if isAsyncRequest(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID+"/vehicle-rental", http.StatusSeeOther)
}

func (a *app) updateVehicleRental(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	rentalID := chi.URLParam(r, "rentalID")
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "invalid vehicle rental form", http.StatusBadRequest)
		return
	}
	existing, err := a.tripService.GetVehicleRental(r.Context(), tripID, rentalID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/trips/"+tripID+"/vehicle-rental", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	trip, err := a.tripService.GetTrip(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	pickUpAt, pickUpDate, pickUpTime, err := parseDateTimeLocal(r.FormValue("pick_up_at"))
	if err != nil {
		http.Error(w, "invalid pick up date/time", http.StatusBadRequest)
		return
	}
	dropOffAt, dropOffDate, dropOffTime, err := parseDateTimeLocal(r.FormValue("drop_off_at"))
	if err != nil {
		http.Error(w, "invalid drop off date/time", http.StatusBadRequest)
		return
	}
	pickUpDay, err := dayNumberFromDate(trip.StartDate, trip.EndDate, pickUpDate)
	if err != nil {
		http.Error(w, "pick up date must be within trip dates", http.StatusBadRequest)
		return
	}
	dropOffDay, err := dayNumberFromDate(trip.StartDate, trip.EndDate, dropOffDate)
	if err != nil {
		http.Error(w, "drop off date must be within trip dates", http.StatusBadRequest)
		return
	}
	if dropOffAt.Before(pickUpAt) {
		http.Error(w, "drop off must be after pick up", http.StatusBadRequest)
		return
	}
	cost, _ := strconv.ParseFloat(r.FormValue("cost"), 64)
	insuranceCost, _ := strconv.ParseFloat(r.FormValue("insurance_cost"), 64)
	totalCost := cost + insuranceCost
	location := r.FormValue("pick_up_location")
	vehicleDetail := r.FormValue("vehicle_detail")
	vehicleTitle := vehicleRentalTitleValue(vehicleDetail, location)
	booking := r.FormValue("booking_confirmation")
	notes := r.FormValue("notes")
	payAtPickUp := r.FormValue("pay_at_pick_up") == "true"
	vehicleImagePath, err := storeVehicleImage(r, "vehicle_image")
	if err != nil {
		http.Error(w, "failed to save vehicle image", http.StatusBadRequest)
		return
	}
	removeImage := r.FormValue("remove_vehicle_image") == "true"
	if vehicleImagePath == "" {
		vehicleImagePath = r.FormValue("current_vehicle_image_path")
	}
	if removeImage && r.FormValue("current_vehicle_image_path") != "" && vehicleImagePath == r.FormValue("current_vehicle_image_path") {
		_ = deleteUploadedFileByWebPath(vehicleImagePath)
		vehicleImagePath = ""
	}
	if vehicleImagePath != "" && r.FormValue("current_vehicle_image_path") != "" && vehicleImagePath != r.FormValue("current_vehicle_image_path") {
		_ = deleteUploadedFileByWebPath(r.FormValue("current_vehicle_image_path"))
	}
	locLat, locLng := geocodeLocation(r.Context(), location)

	if err := a.tripService.UpdateItineraryItem(r.Context(), trips.ItineraryItem{
		ID:        existing.PickUpItineraryID,
		TripID:    tripID,
		DayNumber: pickUpDay,
		Title:     trips.VehicleRentalItineraryPickUpTitle(vehicleTitle),
		Location:  location,
		Latitude:  locLat,
		Longitude: locLng,
		Notes:     buildVehicleItineraryNotes(notes, booking, payAtPickUp),
		EstCost:   totalCost,
		StartTime: pickUpTime,
		EndTime:   pickUpTime,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := a.tripService.UpdateItineraryItem(r.Context(), trips.ItineraryItem{
		ID:        existing.DropOffItineraryID,
		TripID:    tripID,
		DayNumber: dropOffDay,
		Title:     trips.VehicleRentalItineraryDropOffTitle(vehicleTitle),
		Location:  location,
		Latitude:  locLat,
		Longitude: locLng,
		Notes:     defaultIfEmpty(notes, ""),
		EstCost:   totalCost,
		StartTime: dropOffTime,
		EndTime:   dropOffTime,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = a.tripService.UpdateVehicleRental(r.Context(), trips.VehicleRental{
		ID:                  rentalID,
		TripID:              tripID,
		PickUpLocation:      location,
		VehicleDetail:       vehicleDetail,
		PickUpAt:            pickUpAt.Format("2006-01-02T15:04"),
		DropOffAt:           dropOffAt.Format("2006-01-02T15:04"),
		BookingConfirmation: booking,
		Notes:               notes,
		VehicleImagePath:    vehicleImagePath,
		Cost:                cost,
		InsuranceCost:       insuranceCost,
		PayAtPickUp:         payAtPickUp,
		PickUpItineraryID:   existing.PickUpItineraryID,
		DropOffItineraryID:  existing.DropOffItineraryID,
		RentalExpenseID:     existing.RentalExpenseID,
		InsuranceExpenseID:  existing.InsuranceExpenseID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if isAsyncRequest(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	next := strings.TrimSpace(r.FormValue("return_to"))
	if next != "" && isSafeReturnForTrip(next, tripID) {
		http.Redirect(w, r, next, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID+"/vehicle-rental", http.StatusSeeOther)
}

func (a *app) deleteVehicleRental(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	rentalID := chi.URLParam(r, "rentalID")
	_ = r.ParseForm()
	existing, _ := a.tripService.GetVehicleRental(r.Context(), tripID, rentalID)
	if err := a.tripService.DeleteVehicleRental(r.Context(), tripID, rentalID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if existing.VehicleImagePath != "" {
		_ = deleteUploadedFileByWebPath(existing.VehicleImagePath)
	}
	next := strings.TrimSpace(r.FormValue("return_to"))
	if next != "" && isSafeReturnForTrip(next, tripID) {
		http.Redirect(w, r, next, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID+"/vehicle-rental", http.StatusSeeOther)
}

func (a *app) flightsPage(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	details, err := a.tripService.GetTripDetailsVisible(r.Context(), tripID, CurrentUserID(r.Context()))
	if err != nil {
		if tripForbiddenOrMissing(err) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	for _, f := range details.Flights {
		if err := a.tripService.SyncExpenseForFlight(r.Context(), f); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
	details, err = a.tripService.GetTripDetailsVisible(r.Context(), tripID, CurrentUserID(r.Context()))
	if err != nil {
		if tripForbiddenOrMissing(err) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if a.redirectIfTripSectionDisabled(w, r, details.Trip, "flights") {
		return
	}
	uid := CurrentUserID(r.Context())
	settings, err := a.tripService.MergedSettingsForUI(r.Context(), uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	currencySymbol := defaultIfEmpty(details.Trip.CurrencySymbol, "$")
	pageData := map[string]any{
		"Details":        details,
		"Settings":       settings,
		"CSRFToken":      CSRFToken(r.Context()),
		"CurrencySymbol": currencySymbol,
	}
	a.mergeTripSidebarContext(r.Context(), r, tripID, details, pageData, "flights")
	_ = a.templates.ExecuteTemplate(w, "flights.html", pageData)
}

func (a *app) addFlight(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "invalid flight form", http.StatusBadRequest)
		return
	}
	trip, err := a.tripService.GetTrip(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if a.redirectIfTripSectionDisabled(w, r, trip, "flights") {
		return
	}

	departAt, departDate, departTime, err := parseDateTimeLocal(r.FormValue("depart_at"))
	if err != nil {
		http.Error(w, "invalid departure date/time", http.StatusBadRequest)
		return
	}
	arriveAt, arriveDate, arriveTime, err := parseDateTimeLocal(r.FormValue("arrive_at"))
	if err != nil {
		http.Error(w, "invalid arrival date/time", http.StatusBadRequest)
		return
	}
	departDay, err := dayNumberFromDate(trip.StartDate, trip.EndDate, departDate)
	if err != nil {
		http.Error(w, "departure date must be within trip dates", http.StatusBadRequest)
		return
	}
	arriveDay, err := dayNumberFromDate(trip.StartDate, trip.EndDate, arriveDate)
	if err != nil {
		http.Error(w, "arrival date must be within trip dates", http.StatusBadRequest)
		return
	}
	if arriveAt.Before(departAt) {
		http.Error(w, "arrival must be after departure", http.StatusBadRequest)
		return
	}

	cost, _ := strconv.ParseFloat(r.FormValue("cost"), 64)
	documentPath, err := storeFlightDocument(r, "flight_document")
	if err != nil {
		http.Error(w, "failed to save flight document", http.StatusBadRequest)
		return
	}
	flightName := r.FormValue("flight_name")
	flightNumber := r.FormValue("flight_number")
	departAirport := r.FormValue("depart_airport")
	arriveAirport := r.FormValue("arrive_airport")
	booking := r.FormValue("booking_confirmation")
	notes := r.FormValue("notes")
	label := flightTitleValue(flightName, flightNumber)

	flightID := uuid.NewString()
	departItineraryID := uuid.NewString()
	arriveItineraryID := uuid.NewString()
	departNotes := buildFlightItineraryNotes(notes, booking)
	arriveNotes := defaultIfEmpty(notes, "")
	departLat, departLng := geocodeLocation(r.Context(), departAirport)
	arriveLat, arriveLng := geocodeLocation(r.Context(), arriveAirport)

	if err := a.tripService.AddItineraryItem(r.Context(), trips.ItineraryItem{
		ID:        departItineraryID,
		TripID:    tripID,
		DayNumber: departDay,
		Title:     trips.FlightItineraryDepartTitle(label),
		Location:  departAirport,
		Latitude:  departLat,
		Longitude: departLng,
		Notes:     departNotes,
		EstCost:   cost,
		StartTime: departTime,
		EndTime:   departTime,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := a.tripService.AddItineraryItem(r.Context(), trips.ItineraryItem{
		ID:        arriveItineraryID,
		TripID:    tripID,
		DayNumber: arriveDay,
		Title:     trips.FlightItineraryArriveTitle(label),
		Location:  arriveAirport,
		Latitude:  arriveLat,
		Longitude: arriveLng,
		Notes:     arriveNotes,
		EstCost:   cost,
		StartTime: arriveTime,
		EndTime:   arriveTime,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	if err := a.tripService.AddFlight(r.Context(), trips.Flight{
		ID:                  flightID,
		TripID:              tripID,
		FlightName:          flightName,
		FlightNumber:        flightNumber,
		DepartAirport:       departAirport,
		ArriveAirport:       arriveAirport,
		DepartAt:            departAt.Format("2006-01-02T15:04"),
		ArriveAt:            arriveAt.Format("2006-01-02T15:04"),
		BookingConfirmation: booking,
		Notes:               notes,
		DocumentPath:        documentPath,
		Cost:                cost,
		DepartItineraryID:   departItineraryID,
		ArriveItineraryID:   arriveItineraryID,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if isAsyncRequest(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID+"/flights", http.StatusSeeOther)
}

func (a *app) updateFlight(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	flightID := chi.URLParam(r, "flightID")
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "invalid flight form", http.StatusBadRequest)
		return
	}
	existing, err := a.tripService.GetFlight(r.Context(), tripID, flightID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/trips/"+tripID+"/flights", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	trip, err := a.tripService.GetTrip(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	departAt, departDate, departTime, err := parseDateTimeLocal(r.FormValue("depart_at"))
	if err != nil {
		http.Error(w, "invalid departure date/time", http.StatusBadRequest)
		return
	}
	arriveAt, arriveDate, arriveTime, err := parseDateTimeLocal(r.FormValue("arrive_at"))
	if err != nil {
		http.Error(w, "invalid arrival date/time", http.StatusBadRequest)
		return
	}
	departDay, err := dayNumberFromDate(trip.StartDate, trip.EndDate, departDate)
	if err != nil {
		http.Error(w, "departure date must be within trip dates", http.StatusBadRequest)
		return
	}
	arriveDay, err := dayNumberFromDate(trip.StartDate, trip.EndDate, arriveDate)
	if err != nil {
		http.Error(w, "arrival date must be within trip dates", http.StatusBadRequest)
		return
	}
	if arriveAt.Before(departAt) {
		http.Error(w, "arrival must be after departure", http.StatusBadRequest)
		return
	}

	documentPath, err := storeFlightDocument(r, "flight_document")
	if err != nil {
		http.Error(w, "failed to save flight document", http.StatusBadRequest)
		return
	}
	removeDocument := r.FormValue("remove_document") == "true"
	if documentPath == "" {
		documentPath = r.FormValue("current_document_path")
	}
	if removeDocument && r.FormValue("current_document_path") != "" && documentPath == r.FormValue("current_document_path") {
		_ = deleteUploadedFileByWebPath(documentPath)
		documentPath = ""
	}
	if documentPath != "" && r.FormValue("current_document_path") != "" && documentPath != r.FormValue("current_document_path") {
		_ = deleteUploadedFileByWebPath(r.FormValue("current_document_path"))
	}
	cost, _ := strconv.ParseFloat(r.FormValue("cost"), 64)
	flightName := r.FormValue("flight_name")
	flightNumber := r.FormValue("flight_number")
	departAirport := r.FormValue("depart_airport")
	arriveAirport := r.FormValue("arrive_airport")
	booking := r.FormValue("booking_confirmation")
	notes := r.FormValue("notes")
	label := flightTitleValue(flightName, flightNumber)
	departLat, departLng := geocodeLocation(r.Context(), departAirport)
	arriveLat, arriveLng := geocodeLocation(r.Context(), arriveAirport)

	if err := a.tripService.UpdateItineraryItem(r.Context(), trips.ItineraryItem{
		ID:        existing.DepartItineraryID,
		TripID:    tripID,
		DayNumber: departDay,
		Title:     trips.FlightItineraryDepartTitle(label),
		Location:  departAirport,
		Latitude:  departLat,
		Longitude: departLng,
		Notes:     buildFlightItineraryNotes(notes, booking),
		EstCost:   cost,
		StartTime: departTime,
		EndTime:   departTime,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if err := a.tripService.UpdateItineraryItem(r.Context(), trips.ItineraryItem{
		ID:        existing.ArriveItineraryID,
		TripID:    tripID,
		DayNumber: arriveDay,
		Title:     trips.FlightItineraryArriveTitle(label),
		Location:  arriveAirport,
		Latitude:  arriveLat,
		Longitude: arriveLng,
		Notes:     defaultIfEmpty(notes, ""),
		EstCost:   cost,
		StartTime: arriveTime,
		EndTime:   arriveTime,
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = a.tripService.UpdateFlight(r.Context(), trips.Flight{
		ID:                  flightID,
		TripID:              tripID,
		FlightName:          flightName,
		FlightNumber:        flightNumber,
		DepartAirport:       departAirport,
		ArriveAirport:       arriveAirport,
		DepartAt:            departAt.Format("2006-01-02T15:04"),
		ArriveAt:            arriveAt.Format("2006-01-02T15:04"),
		BookingConfirmation: booking,
		Notes:               notes,
		DocumentPath:        documentPath,
		Cost:                cost,
		DepartItineraryID:   existing.DepartItineraryID,
		ArriveItineraryID:   existing.ArriveItineraryID,
		ExpenseID:           existing.ExpenseID,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if isAsyncRequest(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	next := strings.TrimSpace(r.FormValue("return_to"))
	if next != "" && isSafeReturnForTrip(next, tripID) {
		http.Redirect(w, r, next, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID+"/flights", http.StatusSeeOther)
}

func (a *app) deleteFlight(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	flightID := chi.URLParam(r, "flightID")
	existing, _ := a.tripService.GetFlight(r.Context(), tripID, flightID)
	if err := a.tripService.DeleteFlight(r.Context(), tripID, flightID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if existing.DocumentPath != "" {
		_ = deleteUploadedFileByWebPath(existing.DocumentPath)
	}
	http.Redirect(w, r, "/trips/"+tripID+"/flights", http.StatusSeeOther)
}

func (a *app) toggleChecklistItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemID")
	_ = r.ParseForm()
	item, err := a.tripService.GetChecklistItem(r.Context(), itemID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	trip, err := a.tripService.GetTrip(r.Context(), item.TripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if a.redirectIfTripSectionDisabled(w, r, trip, "checklist") {
		return
	}
	done := r.FormValue("done") == "true"
	if err := a.tripService.ToggleChecklistItem(r.Context(), itemID, done); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if isAsyncRequest(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	back := r.Referer()
	if back == "" {
		back = "/"
	}
	http.Redirect(w, r, back, http.StatusSeeOther)
}

func (a *app) updateChecklistItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemID")
	_ = r.ParseForm()
	existing, err := a.tripService.GetChecklistItem(r.Context(), itemID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	trip, err := a.tripService.GetTrip(r.Context(), existing.TripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if a.redirectIfTripSectionDisabled(w, r, trip, "checklist") {
		return
	}
	category := strings.TrimSpace(r.FormValue("category"))
	if category == "" {
		category = "Packing List"
	}
	if err = a.tripService.UpdateChecklistItem(r.Context(), trips.ChecklistItem{
		ID:       itemID,
		Category: category,
		Text:     strings.TrimSpace(r.FormValue("text")),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if isAsyncRequest(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	back := r.Referer()
	if back == "" {
		back = "/"
	}
	http.Redirect(w, r, back, http.StatusSeeOther)
}

func (a *app) deleteChecklistItem(w http.ResponseWriter, r *http.Request) {
	itemID := chi.URLParam(r, "itemID")
	existing, err := a.tripService.GetChecklistItem(r.Context(), itemID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	trip, err := a.tripService.GetTrip(r.Context(), existing.TripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if a.redirectIfTripSectionDisabled(w, r, trip, "checklist") {
		return
	}
	if err := a.tripService.DeleteChecklistItem(r.Context(), itemID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if isAsyncRequest(r) {
		w.WriteHeader(http.StatusNoContent)
		return
	}
	back := r.Referer()
	if back == "" {
		back = "/"
	}
	http.Redirect(w, r, back, http.StatusSeeOther)
}

func (a *app) listChanges(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	since := r.URL.Query().Get("since")
	changes, err := a.tripService.ListChanges(r.Context(), tripID, since)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "trip not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"changes": changes,
	})
}

func (a *app) syncChanges(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	changes, _ := a.tripService.ListChanges(r.Context(), tripID, "")
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]any{
		"status":         "accepted",
		"trip_id":        tripID,
		"applied_count":  0,
		"server_changes": changes,
		"message":        "prototype sync endpoint; client writes can be queued and replayed using last-write-wins",
	})
}

// mergeTripSidebarContext adds Details, CustomSidebarLinks, TripAccess, Party, PendingInvites, and SidebarNavActive
// for shared trip sidebar templates (tripSidebarNav, tripMembersPanel).
func (a *app) mergeTripSidebarContext(ctx context.Context, r *http.Request, tripID string, details trips.TripDetails, into map[string]any, sidebarNavActive string) {
	uid := CurrentUserID(ctx)
	acc, _ := TripAccessFromContext(ctx)
	party, _ := a.tripService.TripParty(ctx, tripID)
	pendingInvites, _ := a.tripService.ListPendingTripInvitesForTrip(ctx, tripID, uid)
	customSidebarLinks := trips.ParseCustomSidebarLinksJSON(details.Trip.UICustomSidebarLinks)
	into["Details"] = details
	into["CustomSidebarLinks"] = customSidebarLinks
	into["TripAccess"] = acc
	into["Party"] = party
	into["PendingInvites"] = pendingInvites
	into["SidebarNavActive"] = sidebarNavActive
}

func defaultIfEmpty(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// isSafeReturnForTrip allows only relative paths for the same trip (no open redirects).
func isSafeReturnForTrip(raw string, tripID string) bool {
	if raw == "" || tripID == "" {
		return false
	}
	if !strings.HasPrefix(raw, "/") || strings.HasPrefix(raw, "//") {
		return false
	}
	base := "/trips/" + tripID
	return raw == base || strings.HasPrefix(raw, base+"/") || strings.HasPrefix(raw, base+"?")
}

func parseDateTimeLocal(raw string) (time.Time, string, string, error) {
	t, err := time.Parse("2006-01-02T15:04", raw)
	if err != nil {
		return time.Time{}, "", "", err
	}
	return t, t.Format("2006-01-02"), t.Format("15:04"), nil
}

func storeBookingAttachment(r *http.Request, field string) (string, error) {
	file, header, err := r.FormFile(field)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return "", nil
		}
		return "", err
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == "" {
		ext = ".bin"
	}
	name := strconv.FormatInt(time.Now().UnixNano(), 10) + ext
	targetDir := filepath.Join("web", "static", "uploads", "bookings")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}
	targetPath := filepath.Join(targetDir, name)
	dst, err := os.Create(targetPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		return "", err
	}
	return "/static/uploads/bookings/" + name, nil
}

func buildLodgingCheckInNotes(notes, bookingNo, attachmentPath string) string {
	checkInNotes := defaultIfEmpty(notes, "")
	if bookingNo != "" {
		if checkInNotes != "" {
			checkInNotes += " | "
		}
		checkInNotes += "Booking: " + bookingNo
	}
	if attachmentPath != "" {
		if checkInNotes != "" {
			checkInNotes += " | "
		}
		checkInNotes += "Attachment: " + attachmentPath
	}
	return checkInNotes
}

func buildVehicleItineraryNotes(notes, bookingNo string, payAtPickUp bool) string {
	out := defaultIfEmpty(notes, "")
	if bookingNo != "" {
		if out != "" {
			out += " | "
		}
		out += "Booking: " + bookingNo
	}
	if payAtPickUp {
		if out != "" {
			out += " | "
		}
		out += "Pay at pick up: Yes"
	}
	return out
}

func vehicleRentalTitleValue(vehicleDetail, pickUpLocation string) string {
	v := strings.TrimSpace(vehicleDetail)
	if v != "" {
		return v
	}
	return strings.TrimSpace(pickUpLocation)
}

func flightTitleValue(flightName, flightNumber string) string {
	name := strings.TrimSpace(flightName)
	number := strings.TrimSpace(flightNumber)
	switch {
	case name != "" && number != "":
		return name + " (" + number + ")"
	case name != "":
		return name
	case number != "":
		return number
	default:
		return "Flight"
	}
}

func buildFlightItineraryNotes(notes, bookingNo string) string {
	out := defaultIfEmpty(notes, "")
	if bookingNo != "" {
		if out != "" {
			out += " | "
		}
		out += "Booking: " + bookingNo
	}
	return out
}

func dayNumberFromDate(startDate, endDate, itineraryDate string) (int, error) {
	if itineraryDate == "" {
		return 0, errors.New("date is required")
	}
	start, err := time.Parse("2006-01-02", startDate)
	if err != nil {
		return 0, errors.New("trip start date is invalid")
	}
	end, err := time.Parse("2006-01-02", endDate)
	if err != nil {
		return 0, errors.New("trip end date is invalid")
	}
	selected, err := time.Parse("2006-01-02", itineraryDate)
	if err != nil {
		return 0, errors.New("invalid date")
	}
	if selected.Before(start) || selected.After(end) {
		return 0, errors.New("date must be within the trip start and end dates")
	}
	return int(selected.Sub(start).Hours()/24) + 1, nil
}

func storeVehicleImage(r *http.Request, field string) (string, error) {
	file, header, err := r.FormFile(field)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return "", nil
		}
		return "", err
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == "" {
		ext = ".jpg"
	}
	name := strconv.FormatInt(time.Now().UnixNano(), 10) + ext
	targetDir := filepath.Join("web", "static", "uploads", "vehicles")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}
	targetPath := filepath.Join(targetDir, name)
	dst, err := os.Create(targetPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		return "", err
	}
	return "/static/uploads/vehicles/" + name, nil
}

func storeFlightDocument(r *http.Request, field string) (string, error) {
	file, header, err := r.FormFile(field)
	if err != nil {
		if errors.Is(err, http.ErrMissingFile) {
			return "", nil
		}
		return "", err
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if ext == "" {
		ext = ".bin"
	}
	name := strconv.FormatInt(time.Now().UnixNano(), 10) + ext
	targetDir := filepath.Join("web", "static", "uploads", "flights")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		return "", err
	}
	targetPath := filepath.Join(targetDir, name)
	dst, err := os.Create(targetPath)
	if err != nil {
		return "", err
	}
	defer dst.Close()
	if _, err := io.Copy(dst, file); err != nil {
		return "", err
	}
	return "/static/uploads/flights/" + name, nil
}

func formatDateTimeDisplay(raw string) string {
	if raw == "" {
		return "--"
	}
	parsed, err := time.Parse("2006-01-02T15:04", raw)
	if err != nil {
		return raw
	}
	return parsed.Format("02-01-2006 | 03:04 PM")
}

// formatTripDateTime formats datetime-local values using the trip’s 12h/24h preference (trip detail pages only).
func formatTripDateTime(t trips.Trip, raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "--"
	}
	parsed, err := time.Parse("2006-01-02T15:04", raw)
	if err != nil {
		return raw
	}
	if trips.UITimeFormatIs24h(t.UITimeFormat) {
		return parsed.Format("02-01-2006 | 15:04")
	}
	return parsed.Format("02-01-2006 | 03:04 PM")
}

// formatTripClock formats stored time strings (e.g. itinerary start/end) using the trip’s clock preference.
func formatTripClock(t trips.Trip, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	var tt time.Time
	var err error
	for _, layout := range []string{"15:04:05", "15:04"} {
		tt, err = time.Parse(layout, raw)
		if err == nil {
			break
		}
	}
	if err != nil {
		return raw
	}
	if trips.UITimeFormatIs24h(t.UITimeFormat) {
		return tt.Format("15:04")
	}
	return tt.Format("3:04 PM")
}

// formatUIDate renders a stored YYYY-MM-DD value as DD-MM-YYYY for display. Unparseable input is returned unchanged.
func formatUIDate(iso string) string {
	s := strings.TrimSpace(iso)
	if s == "" {
		return ""
	}
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return iso
	}
	return t.Format("02-01-2006")
}

// formatTripDateRangeEn formats a trip span like "Dec 10 – Dec 18, 2023" for dashboard cards.
func formatTripDateRangeEn(startISO, endISO string) string {
	s := strings.TrimSpace(startISO)
	e := strings.TrimSpace(endISO)
	if s == "" && e == "" {
		return "Dates not set"
	}
	if s == "" {
		return formatUIDate(e)
	}
	if e == "" {
		return formatUIDate(s)
	}
	st, err1 := time.Parse("2006-01-02", s)
	en, err2 := time.Parse("2006-01-02", e)
	if err1 != nil || err2 != nil {
		return formatUIDate(s) + " – " + formatUIDate(e)
	}
	if st.Year() == en.Year() {
		return st.Format("Jan 2") + " – " + en.Format("Jan 2, 2006")
	}
	return st.Format("Jan 2, 2006") + " – " + en.Format("Jan 2, 2006")
}

// formatTripDateShortRange formats a compact span for mobile list cards, e.g. "Oct 12 – 18".
func formatTripDateShortRange(startISO, endISO string) string {
	s := strings.TrimSpace(startISO)
	e := strings.TrimSpace(endISO)
	if s == "" || e == "" {
		return ""
	}
	st, err1 := time.Parse("2006-01-02", s)
	en, err2 := time.Parse("2006-01-02", e)
	if err1 != nil || err2 != nil || en.Before(st) {
		return ""
	}
	if st.Year() == en.Year() && st.Month() == en.Month() {
		return st.Format("Jan 2") + " – " + strconv.Itoa(en.Day())
	}
	if st.Year() == en.Year() {
		return st.Format("Jan 2") + " – " + en.Format("Jan 2")
	}
	return st.Format("Jan 2, 2006") + " – " + en.Format("Jan 2, 2006")
}

func formatTripMoney(amount float64) string {
	return formatMoneyPlain(amount)
}

func formatMoneyPlain(amount float64) string {
	neg := amount < 0
	if neg {
		amount = -amount
	}
	cents := int64(math.Round(amount * 100))
	whole := cents / 100
	frac := cents % 100
	ws := formatInt64Commas(whole)
	var out string
	if frac == 0 {
		out = ws
	} else {
		out = ws + fmt.Sprintf(".%02d", frac)
	}
	if neg {
		return "-" + out
	}
	return out
}

func formatInt64Commas(n int64) string {
	neg := n < 0
	if neg {
		n = -n
	}
	s := strconv.FormatInt(n, 10)
	if len(s) <= 3 {
		if neg {
			return "-" + s
		}
		return s
	}
	var b strings.Builder
	if neg {
		b.WriteByte('-')
	}
	pref := len(s) % 3
	if pref == 0 {
		pref = 3
	}
	b.WriteString(s[:pref])
	for i := pref; i < len(s); i += 3 {
		b.WriteByte(',')
		b.WriteString(s[i : i+3])
	}
	return b.String()
}

func listContainsString(list []string, s string) bool {
	for _, x := range list {
		if x == s {
			return true
		}
	}
	return false
}

func expenseCategoryStyle(cat string) string {
	switch strings.TrimSpace(cat) {
	case "Airfare":
		return "airfare"
	case "Car Rental":
		return "car-rental"
	case "Accommodation":
		return "accommodation"
	case "Transportation":
		return "transportation"
	case "Food & Dining":
		return "food-dining"
	case "Groceries":
		return "groceries"
	case "Activities":
		return "activities"
	case "Shopping":
		return "shopping"
	case "Miscellaneous":
		return "misc"
	case "Visa & Documentation":
		return "visa-docs"
	case "Insurance":
		return "insurance"
	case "Parking & Toll":
		return "parking"
	case "Fuel":
		return "fuel"
	case "Connectivity":
		return "connectivity"
	case "Tips & Gratuities":
		return "tips"
	default:
		return "other"
	}
}

func expenseCategoryIcon(cat string) string {
	switch strings.TrimSpace(cat) {
	case "Airfare":
		return "flight"
	case "Car Rental":
		return "car_rental"
	case "Accommodation":
		return "holiday_village"
	case "Transportation":
		return "directions_transit"
	case "Food & Dining":
		return "restaurant"
	case "Groceries":
		return "local_grocery_store"
	case "Activities":
		return "local_activity"
	case "Shopping":
		return "shopping_bag"
	case "Miscellaneous":
		return "inventory_2"
	case "Visa & Documentation":
		return "badge"
	case "Insurance":
		return "shield_person"
	case "Parking & Toll":
		return "local_parking"
	case "Fuel":
		return "local_gas_station"
	case "Connectivity":
		return "wifi"
	case "Tips & Gratuities":
		return "savings"
	default:
		return "payments"
	}
}

func expenseCategoryStrokeColor(cat string) string {
	switch expenseCategoryStyle(cat) {
	case "airfare":
		return "#2563eb"
	case "car-rental":
		return "#7c3aed"
	case "accommodation":
		return "#0891b2"
	case "transportation":
		return "#4f46e5"
	case "food-dining":
		return "#ea580c"
	case "groceries":
		return "#65a30d"
	case "activities":
		return "#db2777"
	case "shopping":
		return "#ca8a04"
	case "misc":
		return "#64748b"
	case "visa-docs":
		return "#0f766e"
	case "insurance":
		return "#475569"
	case "parking":
		return "#78716c"
	case "fuel":
		return "#b45309"
	case "connectivity":
		return "#0284c7"
	case "tips":
		return "#c026d3"
	case "other":
		return "#94a3b8"
	default:
		return "#64748b"
	}
}

func deleteUploadedFileByWebPath(webPath string) error {
	clean := strings.TrimPrefix(webPath, "/")
	if clean == "" {
		return nil
	}
	if !strings.HasPrefix(clean, "static/uploads/bookings/") && !strings.HasPrefix(clean, "static/uploads/vehicles/") && !strings.HasPrefix(clean, "static/uploads/flights/") {
		return nil
	}
	target := filepath.Join("web", filepath.FromSlash(clean))
	if _, err := os.Stat(target); err == nil {
		return os.Remove(target)
	}
	return nil
}

func isAsyncRequest(r *http.Request) bool {
	if strings.EqualFold(strings.TrimSpace(r.Header.Get("X-Requested-With")), "XMLHttpRequest") {
		return true
	}
	if strings.Contains(strings.ToLower(r.Header.Get("Accept")), "application/json") {
		return true
	}
	if strings.TrimSpace(r.Header.Get("HX-Request")) == "true" {
		return true
	}
	return false
}
