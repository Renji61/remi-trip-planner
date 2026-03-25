package httpapp

import (
	"database/sql"
	"encoding/csv"
	"encoding/json"
	"errors"
	"html/template"
	"io"
	"net/http"
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

type itineraryItemView struct {
	Item    trips.ItineraryItem
	Lodging trips.Lodging
	Vehicle trips.VehicleRental
	Flight  trips.Flight
}

type itineraryDayGroup struct {
	DayNumber int
	DateLabel string
	Items     []itineraryItemView
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

type budgetCategoryGroupView struct {
	ID           string
	Name         string
	Icon         string
	// DonutStyle selects which CSS stroke color to use on the donut.
	DonutStyle string
	// DonutStroke uses the same base color as the category icon.
	DonutStroke string
	// IconStyle matches the existing expense category icon color scheme.
	IconStyle string
	Amount       float64
	PercentInt   int
	ExpenseCount int

	// Donut rendering (viewbox 0..36 with circumference ~100).
	DonutDashArrayA int
	DonutDashArrayB int
	DonutDashOffset int
}

type budgetTransactionRowView struct {
	DateLabel       string
	CategoryName    string
	CategoryIcon    string
	// CategoryStyle matches the existing expense category icon color scheme.
	CategoryStyle   string
	Description     string
	Method          string
	Amount          float64
}

func NewRouter(deps Dependencies) http.Handler {
	tmpl := template.Must(
		template.New("").
			Funcs(template.FuncMap{
				"formatDateTime":       formatDateTimeDisplay,
				"formatUIDate":         formatUIDate,
				"expenseCategoryStyle": expenseCategoryStyle,
				"expenseCategoryIcon":  expenseCategoryIcon,
				"listContains":         listContainsString,
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

	r.Get("/", a.homePage)
	r.Get("/settings", a.settingsPage)
	r.Post("/settings", a.saveSettings)
	r.Post("/trips", a.createTrip)
	r.Get("/trips/{tripID}", a.tripPage)
	r.Get("/trips/{tripID}/budget", a.budgetPage)
	r.Get("/trips/{tripID}/budget/transactions", a.budgetTransactionsRows)
	r.Get("/trips/{tripID}/budget/export", a.exportBudgetReport)
	r.Post("/trips/{tripID}/update", a.updateTrip)
	r.Post("/trips/{tripID}/archive", a.archiveTrip)
	r.Post("/trips/{tripID}/delete", a.deleteTrip)
	r.Post("/trips/{tripID}/itinerary", a.addItineraryItem)
	r.Post("/trips/{tripID}/itinerary/{itemID}/update", a.updateItineraryItem)
	r.Post("/trips/{tripID}/itinerary/{itemID}/delete", a.deleteItineraryItem)
	r.Get("/trips/{tripID}/accommodation", a.accommodationPage)
	r.Get("/trips/{tripID}/vehicle-rental", a.vehicleRentalPage)
	r.Get("/trips/{tripID}/flights", a.flightsPage)
	r.Post("/trips/{tripID}/accommodation/{lodgingID}/update", a.updateLodging)
	r.Post("/trips/{tripID}/accommodation/{lodgingID}/delete", a.deleteLodging)
	r.Post("/trips/{tripID}/accommodation", a.addLodging)
	r.Post("/trips/{tripID}/vehicle-rental/{rentalID}/update", a.updateVehicleRental)
	r.Post("/trips/{tripID}/vehicle-rental/{rentalID}/delete", a.deleteVehicleRental)
	r.Post("/trips/{tripID}/vehicle-rental", a.addVehicleRental)
	r.Post("/trips/{tripID}/flights/{flightID}/update", a.updateFlight)
	r.Post("/trips/{tripID}/flights/{flightID}/delete", a.deleteFlight)
	r.Post("/trips/{tripID}/flights", a.addFlight)
	r.Post("/trips/{tripID}/lodging/{lodgingID}/update", a.updateLodging)
	r.Post("/trips/{tripID}/lodging/{lodgingID}/delete", a.deleteLodging)
	r.Post("/trips/{tripID}/lodging", a.addLodging)
	r.Get("/trips/{tripID}/lodging", a.redirectLegacyLodgingPath)
	r.Post("/trips/{tripID}/expenses", a.addExpense)
	r.Post("/trips/{tripID}/expenses/{expenseID}/update", a.updateExpense)
	r.Post("/trips/{tripID}/expenses/{expenseID}/delete", a.deleteExpense)
	r.Post("/trips/{tripID}/checklist", a.addChecklistItem)
	r.Post("/checklist/{itemID}/update", a.updateChecklistItem)
	r.Post("/checklist/{itemID}/delete", a.deleteChecklistItem)
	r.Post("/checklist/{itemID}/toggle", a.toggleChecklistItem)

	r.Get("/api/v1/trips/{tripID}/changes", a.listChanges)
	r.Post("/api/v1/trips/{tripID}/sync", a.syncChanges)

	r.Get("/manifest.webmanifest", func(w http.ResponseWriter, r *http.Request) {
		data, _ := os.ReadFile(filepath.Join(a.staticDir, "manifest.webmanifest"))
		w.Header().Set("Content-Type", "application/manifest+json")
		_, _ = w.Write(data)
	})
	r.Get("/sw.js", func(w http.ResponseWriter, r *http.Request) {
		data, _ := os.ReadFile(filepath.Join(a.staticDir, "sw.js"))
		w.Header().Set("Content-Type", "text/javascript")
		_, _ = w.Write(data)
	})
	r.Handle("/static/*", http.StripPrefix("/static/", http.FileServer(http.Dir(a.staticDir))))

	return r
}

func (a *app) homePage(w http.ResponseWriter, r *http.Request) {
	list, err := a.tripService.ListTrips(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	settings, err := a.tripService.GetAppSettings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = a.templates.ExecuteTemplate(w, "home.html", map[string]any{
		"Trips":     list,
		"Settings":  settings,
		"Saved":     r.URL.Query().Get("saved") == "1",
		"HasError":  false,
		"ErrorText": "",
	})
}

func (a *app) settingsPage(w http.ResponseWriter, r *http.Request) {
	settings, err := a.tripService.GetAppSettings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = a.templates.ExecuteTemplate(w, "settings.html", map[string]any{
		"Settings": settings,
		"Saved":    r.URL.Query().Get("saved") == "1",
	})
}

func (a *app) saveSettings(w http.ResponseWriter, r *http.Request) {
	_ = r.ParseForm()
	mapLat, _ := strconv.ParseFloat(r.FormValue("map_default_latitude"), 64)
	mapLng, _ := strconv.ParseFloat(r.FormValue("map_default_longitude"), 64)
	mapZoom, _ := strconv.Atoi(r.FormValue("map_default_zoom"))
	enableLookup := r.FormValue("enable_location_lookup") == "true"

	err := a.tripService.SaveAppSettings(r.Context(), trips.AppSettings{
		AppTitle:              defaultIfEmpty(r.FormValue("app_title"), "REMI Trip Planner"),
		DefaultCurrencyName:   defaultIfEmpty(r.FormValue("default_currency_name"), "USD"),
		DefaultCurrencySymbol: defaultIfEmpty(r.FormValue("default_currency_symbol"), "$"),
		MapDefaultLatitude:    mapLat,
		MapDefaultLongitude:   mapLng,
		MapDefaultZoom:        mapZoom,
		EnableLocationLookup:  enableLookup,
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/settings?saved=1", http.StatusSeeOther)
}

func (a *app) createTrip(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	err := a.tripService.CreateTrip(r.Context(), trips.Trip{
		Name:           r.FormValue("name"),
		Description:    r.FormValue("description"),
		StartDate:      r.FormValue("start_date"),
		EndDate:        r.FormValue("end_date"),
		CurrencyName:   defaultIfEmpty(r.FormValue("currency_name"), "USD"),
		CurrencySymbol: defaultIfEmpty(r.FormValue("currency_symbol"), "$"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func (a *app) tripPage(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	details, err := a.tripService.GetTripDetails(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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
	details, err = a.tripService.GetTripDetails(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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
	dayGroups := buildItineraryDayGroups(details.Trip.StartDate, details.Itinerary, details.Lodgings, details.Vehicles, details.Flights)
	expenseGroups := buildExpenseDayGroups(details.Trip.StartDate, details.Expenses)
	checklistCategoryGroups := buildChecklistCategoryGroups(details.Checklist, trips.ReminderChecklistCategories)
	settings, err := a.tripService.GetAppSettings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
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
	_ = a.templates.ExecuteTemplate(w, "trip.html", map[string]any{
		"Details":              details,
		"DayGroups":            dayGroups,
		"ExpenseGroups":        expenseGroups,
		"Settings":             settings,
		"CurrencySymbol":       currencySymbol,
		"CurrencyName":         currencyName,
		"TotalExpense":         total,
		"TotalBudgeted":        totalBudgeted,
		"BudgetProgress":       budgetProgress,
		"ExpenseCategories":    trips.QuickExpenseCategories,
		"ChecklistCategories":  trips.ReminderChecklistCategories,
		"ChecklistGroups":      checklistCategoryGroups,
		"VehicleExpenseLocked": vehicleExpenseLocked,
		"FlightExpenseLocked":  flightExpenseLocked,
	})
}

func (a *app) budgetPage(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	details, err := a.tripService.GetTripDetails(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	settings, err := a.tripService.GetAppSettings(r.Context())
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
		transactions = append(transactions, budgetTransactionRowView{
			DateLabel:     dateLabel,
			CategoryName:  e.Category,
			CategoryIcon:  expenseCategoryIcon(e.Category),
			CategoryStyle: expenseCategoryStyle(e.Category),
			Description:   desc,
			Method:        defaultIfEmpty(e.PaymentMethod, "Cash"),
			Amount:        e.Amount,
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

	_ = a.templates.ExecuteTemplate(w, "budget.html", map[string]any{
		"Trip":                  details.Trip,
		"Settings":              settings,
		"CurrencySymbol":        currencySymbol,
		"ExpenseCategories":     trips.QuickExpenseCategories,
		"TotalSpent":            totalSpent,
		"TotalBudgeted":         totalBudgeted,
		"Remaining":             remaining,
		"BudgetProgress":        budgetProgress,
		"DailyAvgSpent":         dailyAvgSpent,
		"BudgetTargetPerDay":   budgetTargetPerDay,
		"DailyDeltaPctAbsInt":  dailyDeltaPctAbsInt,
		"DailyTrendIcon":       dailyTrendIcon,
		"DailyTrendClass":      dailyTrendClass,
		"RemainingPercentInt":  remainingPercentInt,
		"TripDays":              tripDays,
		"BudgetGroups":         segments,
		"Transactions":         transactions,
		"HasTransactions":     len(transactions) > 0,
		"CanShowAllTransactions": canShowAll,
		"BudgetInitialLimit":     initialLimit,
	})
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

	details, err := a.tripService.GetTripDetails(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "trip not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	settings, err := a.tripService.GetAppSettings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_ = settings // used only for template consistency

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
		transactions = append(transactions, budgetTransactionRowView{
			DateLabel:     dateLabel,
			CategoryName:  e.Category,
			CategoryIcon:  expenseCategoryIcon(e.Category),
			CategoryStyle: expenseCategoryStyle(e.Category),
			Description:   desc,
			Method:        defaultIfEmpty(e.PaymentMethod, "Cash"),
			Amount:        e.Amount,
		})
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_ = a.templates.ExecuteTemplate(w, "budget_transactions_rows.html", map[string]any{
		"CurrencySymbol": currencySymbol,
		"Transactions":   transactions,
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

	details, err := a.tripService.GetTripDetails(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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

func buildItineraryDayGroups(startDate string, items []trips.ItineraryItem, lodgings []trips.Lodging, vehicles []trips.VehicleRental, flights []trips.Flight) []itineraryDayGroup {
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
				DayNumber: item.DayNumber,
				DateLabel: dateLabel,
				Items:     []itineraryItemView{},
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

func (a *app) updateTrip(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	err := a.tripService.UpdateTrip(r.Context(), trips.Trip{
		ID:             tripID,
		Name:           r.FormValue("name"),
		Description:    r.FormValue("description"),
		StartDate:      r.FormValue("start_date"),
		EndDate:        r.FormValue("end_date"),
		CoverImage:     r.FormValue("cover_image_url"),
		CurrencyName:   defaultIfEmpty(r.FormValue("currency_name"), "USD"),
		CurrencySymbol: defaultIfEmpty(r.FormValue("currency_symbol"), "$"),
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
}

func (a *app) archiveTrip(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	if err := a.tripService.ArchiveTrip(r.Context(), tripID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
}

func (a *app) deleteTrip(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
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
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
}

func (a *app) addExpense(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	_ = r.ParseForm()
	amount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)
	paymentMethod := strings.TrimSpace(r.FormValue("payment_method"))
	if paymentMethod == "" {
		paymentMethod = "Cash"
	}
	err := a.tripService.AddExpense(r.Context(), trips.Expense{
		TripID:         tripID,
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
	details, err := a.tripService.GetTripDetails(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	trip := details.Trip
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
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
}

func (a *app) deleteItineraryItem(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	itemID := chi.URLParam(r, "itemID")
	details, err := a.tripService.GetTripDetails(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
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
	details, err := a.tripService.GetTripDetails(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	settings, err := a.tripService.GetAppSettings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	currencySymbol := defaultIfEmpty(details.Trip.CurrencySymbol, "$")
	_ = a.templates.ExecuteTemplate(w, "accommodation.html", map[string]any{
		"Details":        details,
		"Settings":       settings,
		"CurrencySymbol": currencySymbol,
	})
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

	if err := a.tripService.AddItineraryItem(r.Context(), trips.ItineraryItem{
		ID:        checkInItemID,
		TripID:    tripID,
		DayNumber: checkInDay,
		Title:     trips.AccommodationItineraryCheckInTitle(name),
		Location:  address,
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

	checkInNotes := buildLodgingCheckInNotes(notes, bookingNo, attachmentPath)
	lodging := trips.Lodging{
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
	if err := a.tripService.DeleteLodging(r.Context(), tripID, lodgingID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID+"/accommodation", http.StatusSeeOther)
}

func (a *app) vehicleRentalPage(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	details, err := a.tripService.GetTripDetails(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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
	details, err = a.tripService.GetTripDetails(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	settings, err := a.tripService.GetAppSettings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	currencySymbol := defaultIfEmpty(details.Trip.CurrencySymbol, "$")
	_ = a.templates.ExecuteTemplate(w, "vehicle_rental.html", map[string]any{
		"Details":        details,
		"Settings":       settings,
		"CurrencySymbol": currencySymbol,
	})
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

	if err := a.tripService.AddItineraryItem(r.Context(), trips.ItineraryItem{
		ID:        pickUpItineraryID,
		TripID:    tripID,
		DayNumber: pickUpDay,
		Title:     trips.VehicleRentalItineraryPickUpTitle(vehicleTitle),
		Location:  location,
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

	if err := a.tripService.UpdateItineraryItem(r.Context(), trips.ItineraryItem{
		ID:        existing.PickUpItineraryID,
		TripID:    tripID,
		DayNumber: pickUpDay,
		Title:     trips.VehicleRentalItineraryPickUpTitle(vehicleTitle),
		Location:  location,
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
	http.Redirect(w, r, "/trips/"+tripID+"/vehicle-rental", http.StatusSeeOther)
}

func (a *app) flightsPage(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	details, err := a.tripService.GetTripDetails(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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
	details, err = a.tripService.GetTripDetails(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	settings, err := a.tripService.GetAppSettings(r.Context())
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	currencySymbol := defaultIfEmpty(details.Trip.CurrencySymbol, "$")
	_ = a.templates.ExecuteTemplate(w, "flights.html", map[string]any{
		"Details":        details,
		"Settings":       settings,
		"CurrencySymbol": currencySymbol,
	})
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

	if err := a.tripService.AddItineraryItem(r.Context(), trips.ItineraryItem{
		ID:        departItineraryID,
		TripID:    tripID,
		DayNumber: departDay,
		Title:     trips.FlightItineraryDepartTitle(label),
		Location:  departAirport,
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

	if err := a.tripService.UpdateItineraryItem(r.Context(), trips.ItineraryItem{
		ID:        existing.DepartItineraryID,
		TripID:    tripID,
		DayNumber: departDay,
		Title:     trips.FlightItineraryDepartTitle(label),
		Location:  departAirport,
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
	done := r.FormValue("done") == "true"
	if err := a.tripService.ToggleChecklistItem(r.Context(), itemID, done); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
	category := strings.TrimSpace(r.FormValue("category"))
	if category == "" {
		category = "Packing List"
	}
	if err := a.tripService.UpdateChecklistItem(r.Context(), trips.ChecklistItem{
		ID:       itemID,
		Category: category,
		Text:     strings.TrimSpace(r.FormValue("text")),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
	if err := a.tripService.DeleteChecklistItem(r.Context(), itemID); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
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
