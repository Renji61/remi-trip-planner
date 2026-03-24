package httpapp

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

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

func NewRouter(deps Dependencies) http.Handler {
	tmpl := template.Must(template.ParseGlob("web/templates/*.html"))
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
	r.Post("/trips", a.createTrip)
	r.Get("/trips/{tripID}", a.tripPage)
	r.Post("/trips/{tripID}/itinerary", a.addItineraryItem)
	r.Post("/trips/{tripID}/expenses", a.addExpense)
	r.Post("/trips/{tripID}/checklist", a.addChecklistItem)
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
	_ = a.templates.ExecuteTemplate(w, "home.html", map[string]any{
		"Trips": list,
	})
}

func (a *app) createTrip(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	err := a.tripService.CreateTrip(r.Context(), trips.Trip{
		Name:        r.FormValue("name"),
		Description: r.FormValue("description"),
		StartDate:   r.FormValue("start_date"),
		EndDate:     r.FormValue("end_date"),
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
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	total := 0.0
	for _, e := range details.Expenses {
		total += e.Amount
	}
	_ = a.templates.ExecuteTemplate(w, "trip.html", map[string]any{
		"Details":      details,
		"TotalExpense": total,
	})
}

func (a *app) addItineraryItem(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	_ = r.ParseForm()
	day, _ := strconv.Atoi(r.FormValue("day_number"))
	orderIndex, _ := strconv.Atoi(r.FormValue("order_index"))
	lat, _ := strconv.ParseFloat(r.FormValue("latitude"), 64)
	lng, _ := strconv.ParseFloat(r.FormValue("longitude"), 64)
	estCost, _ := strconv.ParseFloat(r.FormValue("est_cost"), 64)
	err := a.tripService.AddItineraryItem(r.Context(), trips.ItineraryItem{
		TripID:     tripID,
		DayNumber:  day,
		OrderIndex: orderIndex,
		Title:      r.FormValue("title"),
		Notes:      r.FormValue("notes"),
		Location:   r.FormValue("location"),
		Latitude:   lat,
		Longitude:  lng,
		EstCost:    estCost,
		StartTime:  r.FormValue("start_time"),
		EndTime:    r.FormValue("end_time"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
}

func (a *app) addExpense(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	_ = r.ParseForm()
	amount, _ := strconv.ParseFloat(r.FormValue("amount"), 64)
	err := a.tripService.AddExpense(r.Context(), trips.Expense{
		TripID:   tripID,
		Category: r.FormValue("category"),
		Amount:   amount,
		Notes:    r.FormValue("notes"),
		SpentOn:  r.FormValue("spent_on"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
}

func (a *app) addChecklistItem(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	_ = r.ParseForm()
	err := a.tripService.AddChecklistItem(r.Context(), trips.ChecklistItem{
		TripID: tripID,
		Text:   r.FormValue("text"),
	})
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID, http.StatusSeeOther)
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

func (a *app) listChanges(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	since := r.URL.Query().Get("since")
	changes, err := a.tripService.ListChanges(r.Context(), tripID, since)
	if err != nil {
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
		"status":        "accepted",
		"trip_id":       tripID,
		"applied_count": 0,
		"server_changes": changes,
		"message":       "prototype sync endpoint; client writes can be queued and replayed using last-write-wins",
	})
}
