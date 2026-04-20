package httpapp

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"remi-trip-planner/internal/trips"
)

// globalKeepImportRow is one selectable row on the trip import page.
type globalKeepImportRow struct {
	Kind            string // "note" | "checklist"
	ID              string
	Label           string
	Sub             string
	AlreadyImported bool
}

func (a *app) tripNotesImportPage(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	uid := CurrentUserID(r.Context())
	details, err := a.tripService.GetTripDetailsVisible(r.Context(), tripID, uid)
	if err != nil {
		if tripForbiddenOrMissing(err) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}
	if a.redirectIfTripSectionDisabled(w, r, details.Trip, "checklist") {
		return
	}

	notes, err := a.tripService.ListGlobalKeepNotesByUser(r.Context(), uid)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	templates, err := a.tripService.ListGlobalChecklistTemplatesByUser(r.Context(), uid)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	importedNotes, err := a.tripService.ListGlobalKeepImportedIDs(r.Context(), tripID, trips.GlobalKeepImportNote)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	importedCh, err := a.tripService.ListGlobalKeepImportedIDs(r.Context(), tripID, trips.GlobalKeepImportChecklist)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	noteImported := make(map[string]bool, len(importedNotes))
	for _, id := range importedNotes {
		noteImported[id] = true
	}
	chImported := make(map[string]bool, len(importedCh))
	for _, id := range importedCh {
		chImported[id] = true
	}

	var rows []globalKeepImportRow
	for _, n := range notes {
		title := strings.TrimSpace(n.Title)
		if title == "" {
			title = "Untitled note"
		}
		rows = append(rows, globalKeepImportRow{
			Kind:            "note",
			ID:              n.ID,
			Label:           title,
			Sub:             "Note",
			AlreadyImported: noteImported[n.ID],
		})
	}
	for _, t := range templates {
		cat := strings.TrimSpace(t.Category)
		if cat == "" {
			cat = "Checklist"
		}
		nLines := len(t.Lines)
		sub := fmt.Sprintf("Checklist · %d %s", nLines, pluralizeSimple(nLines, "item", "items"))
		rows = append(rows, globalKeepImportRow{
			Kind:            "checklist",
			ID:              t.ID,
			Label:           cat,
			Sub:             sub,
			AlreadyImported: chImported[t.ID],
		})
	}

	settings, err := a.tripService.MergedSettingsForUI(r.Context(), uid)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	pageData := map[string]any{
		"Details":               details,
		"Settings":              settings,
		"CSRFToken":             CSRFToken(r.Context()),
		"GlobalKeepImportRows":  rows,
		"GlobalKeepImportEmpty": len(rows) == 0,
		"ReturnTo":              tripNotesReturnURL(tripID, trips.KeepViewNotes),
	}
	a.mergeTripSidebarContext(r.Context(), r, tripID, details, pageData, "notes")
	if err := a.mergeTripFabFlyoutContext(r.Context(), tripID, details, pageData, "notes"); err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	_ = a.templates.ExecuteTemplate(w, "trip_notes_import.html", pageData)
}

func pluralizeSimple(n int, one, many string) string {
	if n == 1 {
		return one
	}
	return many
}

func (a *app) tripNotesImportSubmit(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	uid := CurrentUserID(r.Context())
	if err := r.ParseForm(); err != nil {
		http.Error(w, "invalid form", http.StatusBadRequest)
		return
	}
	var noteIDs []string
	for _, v := range r.Form["import_note"] {
		if id := strings.TrimSpace(v); id != "" {
			noteIDs = append(noteIDs, id)
		}
	}
	var chIDs []string
	for _, v := range r.Form["import_checklist"] {
		if id := strings.TrimSpace(v); id != "" {
			chIDs = append(chIDs, id)
		}
	}
	if _, err := a.tripService.ImportGlobalKeepIntoTrip(r.Context(), uid, tripID, noteIDs, chIDs); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	ret := strings.TrimSpace(r.FormValue("return_to"))
	if ret != "" && isSafeReturnForTrip(ret, tripID) {
		http.Redirect(w, r, ret, http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/trips/"+tripID+"/notes/import", http.StatusSeeOther)
}
