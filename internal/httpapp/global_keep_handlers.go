package httpapp

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-chi/chi/v5"

	"remi-trip-planner/internal/trips"
)

func globalKeepNoteMatchesQuery(q string, n trips.GlobalKeepNote) bool {
	q = strings.TrimSpace(strings.ToLower(q))
	if q == "" {
		return true
	}
	if strings.Contains(strings.ToLower(n.Title), q) || strings.Contains(strings.ToLower(n.Body), q) {
		return true
	}
	return false
}

func globalKeepChecklistMatchesQuery(q string, t trips.GlobalChecklistTemplate) bool {
	q = strings.TrimSpace(strings.ToLower(q))
	if q == "" {
		return true
	}
	if strings.Contains(strings.ToLower(t.Category), q) {
		return true
	}
	for _, line := range t.Lines {
		if strings.Contains(strings.ToLower(line), q) {
			return true
		}
	}
	return false
}

func filterGlobalKeepLibrary(q string, notes []trips.GlobalKeepNote, templates []trips.GlobalChecklistTemplate) ([]trips.GlobalKeepNote, []trips.GlobalChecklistTemplate) {
	var outN []trips.GlobalKeepNote
	for _, n := range notes {
		if !globalKeepNoteMatchesQuery(q, n) {
			continue
		}
		outN = append(outN, n)
	}
	var outT []trips.GlobalChecklistTemplate
	for _, t := range templates {
		if !globalKeepChecklistMatchesQuery(q, t) {
			continue
		}
		outT = append(outT, t)
	}
	return outN, outT
}

func globalKeepReturnURL(view, q string) string {
	base := "/notes-checklists"
	v := trips.NormalizeKeepView(view)
	q = strings.TrimSpace(q)
	params := url.Values{}
	if v != trips.KeepViewNotes {
		params.Set("view", v)
	}
	if q != "" {
		params.Set("q", q)
	}
	if enc := params.Encode(); enc != "" {
		return base + "?" + enc
	}
	return base
}

func (a *app) globalNotesChecklistsPage(w http.ResponseWriter, r *http.Request) {
	uid := CurrentUserID(r.Context())
	if uid == "" {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	settings, err := a.tripService.MergedSettingsForUI(r.Context(), uid)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	view := trips.NormalizeKeepView(r.URL.Query().Get("view"))
	if view == trips.KeepViewReminders {
		view = trips.KeepViewNotes
	}
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	notes, err := a.tripService.ListGlobalKeepNotesForKeepView(r.Context(), uid, view)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	templates, err := a.tripService.ListGlobalChecklistTemplatesForKeepView(r.Context(), uid, view)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	fNotes, fTemplates := filterGlobalKeepLibrary(q, notes, templates)

	data := map[string]any{
		"Settings":                 settings,
		"CSRFToken":                CSRFToken(r.Context()),
		"GlobalKeepNotes":          fNotes,
		"GlobalChecklistTemplates": fTemplates,
		"KeepNoteColors":           keepNotePickerColors,
		"ReminderCategories":       trips.ReminderChecklistCategories,
		"KeepView":                 view,
		"KeepQuery":                q,
		"GlobalDistanceUnit":       trips.EffectiveDistanceUnit(nil, settings),
		"ReturnTo":                 globalKeepReturnURL(view, q),
	}
	if err := a.mergeDashboardShell(r.Context(), uid, "notes-checklists", "", data); err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	_ = a.templates.ExecuteTemplate(w, "global_notes_checklists.html", data)
}

func (a *app) globalKeepNoteCreate(w http.ResponseWriter, r *http.Request) {
	uid := CurrentUserID(r.Context())
	if uid == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	_ = r.ParseForm()
	title := strings.TrimSpace(r.FormValue("title"))
	body := strings.TrimSpace(r.FormValue("body"))
	color := normalizeKeepNoteColor(r.FormValue("color"))
	if err := a.tripService.AddGlobalKeepNote(r.Context(), trips.GlobalKeepNote{
		UserID: uid,
		Title:  title,
		Body:   body,
		Color:  color,
		DueAt:  "",
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/notes-checklists", http.StatusSeeOther)
}

func (a *app) globalKeepChecklistCreate(w http.ResponseWriter, r *http.Request) {
	uid := CurrentUserID(r.Context())
	if uid == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	_ = r.ParseForm()
	category := strings.TrimSpace(r.FormValue("category"))
	if category == "" {
		category = "Packing List"
	}
	var lines []string
	itemsJSON := strings.TrimSpace(r.FormValue("items_json"))
	if itemsJSON != "" {
		if err := json.Unmarshal([]byte(itemsJSON), &lines); err != nil {
			http.Error(w, "invalid checklist items payload", http.StatusBadRequest)
			return
		}
	} else {
		raw := strings.TrimSpace(r.FormValue("lines"))
		for _, line := range strings.Split(raw, "\n") {
			if t := strings.TrimSpace(line); t != "" {
				lines = append(lines, t)
			}
		}
	}
	if err := a.tripService.AddGlobalChecklistTemplate(r.Context(), trips.GlobalChecklistTemplate{
		UserID:   uid,
		Category: category,
		DueAt:    "",
	}, lines); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, "/notes-checklists", http.StatusSeeOther)
}

func (a *app) globalKeepNoteIntent(w http.ResponseWriter, r *http.Request) {
	uid := CurrentUserID(r.Context())
	if uid == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	noteID := chi.URLParam(r, "noteID")
	_ = r.ParseForm()
	intent := strings.TrimSpace(strings.ToLower(r.FormValue("intent")))
	n, err := a.tripService.GetGlobalKeepNote(r.Context(), uid, noteID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "note not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch intent {
	case "pin":
		n.Pinned = true
	case "unpin":
		n.Pinned = false
	case "archive":
		n.Archived, n.Trashed = true, false
	case "trash":
		n.Archived, n.Trashed = false, true
	case "restore":
		n.Archived, n.Trashed = false, false
	case "purge":
		if !n.Trashed {
			http.Error(w, "only trashed notes can be deleted permanently", http.StatusBadRequest)
			return
		}
		if err := a.tripService.DeleteGlobalKeepNote(r.Context(), uid, noteID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if ret := strings.TrimSpace(r.FormValue("return_to")); ret != "" && isSafeReturnForGlobalKeep(ret) {
			http.Redirect(w, r, ret, http.StatusSeeOther)
		} else {
			http.Redirect(w, r, globalKeepReturnURL(trips.KeepViewTrash, ""), http.StatusSeeOther)
		}
		return
	default:
		http.Error(w, "unknown intent", http.StatusBadRequest)
		return
	}
	if err := a.tripService.UpdateGlobalKeepNote(r.Context(), n); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if ret := strings.TrimSpace(r.FormValue("return_to")); ret != "" && isSafeReturnForGlobalKeep(ret) {
		http.Redirect(w, r, ret, http.StatusSeeOther)
		return
	}
	redir := globalKeepReturnURL(trips.KeepViewNotes, "")
	switch intent {
	case "archive":
		redir = globalKeepReturnURL(trips.KeepViewArchive, "")
	case "trash":
		redir = globalKeepReturnURL(trips.KeepViewTrash, "")
	}
	http.Redirect(w, r, redir, http.StatusSeeOther)
}

func (a *app) globalKeepChecklistIntent(w http.ResponseWriter, r *http.Request) {
	uid := CurrentUserID(r.Context())
	if uid == "" {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	templateID := chi.URLParam(r, "templateID")
	_ = r.ParseForm()
	intent := strings.TrimSpace(strings.ToLower(r.FormValue("intent")))
	tpl, err := a.tripService.GetGlobalChecklistTemplate(r.Context(), uid, templateID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "checklist not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	switch intent {
	case "pin":
		tpl.Pinned = true
	case "unpin":
		tpl.Pinned = false
	case "archive":
		tpl.Archived, tpl.Trashed = true, false
	case "trash":
		tpl.Archived, tpl.Trashed = false, true
	case "restore":
		tpl.Archived, tpl.Trashed = false, false
	case "purge":
		if !tpl.Trashed {
			http.Error(w, "only trashed checklists can be deleted permanently", http.StatusBadRequest)
			return
		}
		if err := a.tripService.DeleteGlobalChecklistTemplate(r.Context(), uid, templateID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if ret := strings.TrimSpace(r.FormValue("return_to")); ret != "" && isSafeReturnForGlobalKeep(ret) {
			http.Redirect(w, r, ret, http.StatusSeeOther)
		} else {
			http.Redirect(w, r, globalKeepReturnURL(trips.KeepViewTrash, ""), http.StatusSeeOther)
		}
		return
	default:
		http.Error(w, "unknown intent", http.StatusBadRequest)
		return
	}
	if err := a.tripService.UpdateGlobalChecklistTemplate(r.Context(), tpl); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if ret := strings.TrimSpace(r.FormValue("return_to")); ret != "" && isSafeReturnForGlobalKeep(ret) {
		http.Redirect(w, r, ret, http.StatusSeeOther)
		return
	}
	redir := globalKeepReturnURL(trips.KeepViewNotes, "")
	switch intent {
	case "archive":
		redir = globalKeepReturnURL(trips.KeepViewArchive, "")
	case "trash":
		redir = globalKeepReturnURL(trips.KeepViewTrash, "")
	}
	http.Redirect(w, r, redir, http.StatusSeeOther)
}
