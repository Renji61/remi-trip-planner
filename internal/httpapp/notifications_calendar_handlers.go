package httpapp

import (
	"html"
	"io"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"remi-trip-planner/internal/trips"
)

func (a *app) calendarFeedICS(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	key := strings.TrimSpace(r.URL.Query().Get("k"))
	if tripID == "" || key == "" {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	if !a.tripService.CalendarFeedMatches(r.Context(), tripID, key) {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}
	data, err := a.tripService.BuildTripICSBytes(r.Context(), tripID)
	if err != nil {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Cache-Control", "private, max-age=300")
	_, _ = w.Write(data)
}

func (a *app) tripCalendarFeedRegenerate(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	if !a.requireTripOwner(w, r, tripID) {
		return
	}
	uid := CurrentUserID(r.Context())
	plain, err := a.tripService.RegenerateCalendarFeedToken(r.Context(), tripID, uid)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	u := *r.URL
	u.Scheme = ""
	u.Host = ""
	u.Path = "/calendar/feed/" + tripID + ".ics"
	q := u.Query()
	q.Set("k", plain)
	u.RawQuery = q.Encode()
	feedPath := u.String()
	if !strings.HasPrefix(feedPath, "/") {
		feedPath = "/" + feedPath
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = io.WriteString(w, `<!doctype html><html><head><meta charset="utf-8"><title>Calendar feed</title></head><body style="font-family:system-ui,sans-serif;max-width:42rem;margin:2rem auto;padding:0 1rem;">`)
	_, _ = io.WriteString(w, `<h1>Calendar subscription link</h1><p>Copy this URL into Google Calendar (<strong>From URL</strong>) or Apple Calendar (<strong>New calendar subscription</strong>):</p>`)
	_, _ = io.WriteString(w, `<p><input readonly style="width:100%;padding:0.5rem" value="`+html.EscapeString(absoluteURLForPublicStatic(r, feedPath))+`"></p>`)
	_, _ = io.WriteString(w, `<p class="muted" style="color:#64748b;font-size:0.9rem">Anyone with this link can view trip times on the calendar. Regenerate the link in trip settings if it is leaked.</p>`)
	_, _ = io.WriteString(w, `<p><a href="/trips/`+html.EscapeString(tripID)+`/settings">Back to trip settings</a></p></body></html>`)
}

func (a *app) notificationsPage(w http.ResponseWriter, r *http.Request) {
	uid := CurrentUserID(r.Context())
	settings, err := a.tripService.MergedSettingsForUI(r.Context(), uid)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	list, err := a.tripService.ListNotificationsForUser(r.Context(), uid, 100)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	for i := range list {
		list[i].Body = trips.ShortenFlightNotificationBody(list[i].Body)
	}
	data := map[string]any{
		"Settings":         settings,
		"CSRFToken":        CSRFToken(r.Context()),
		"Notifications":    list,
		"SidebarNavActive": "notifications",
	}
	if err := a.mergeDashboardShell(r.Context(), uid, "notifications", "", data); err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	_ = a.templates.ExecuteTemplate(w, "notifications.html", data)
}

func (a *app) notificationsMarkAllRead(w http.ResponseWriter, r *http.Request) {
	uid := CurrentUserID(r.Context())
	_ = a.tripService.MarkAllNotificationsRead(r.Context(), uid)
	http.Redirect(w, r, "/notifications", http.StatusSeeOther)
}

func (a *app) notificationsMarkOneRead(w http.ResponseWriter, r *http.Request) {
	uid := CurrentUserID(r.Context())
	id := chi.URLParam(r, "notificationID")
	_ = a.tripService.MarkNotificationRead(r.Context(), uid, id)
	http.Redirect(w, r, "/notifications", http.StatusSeeOther)
}
