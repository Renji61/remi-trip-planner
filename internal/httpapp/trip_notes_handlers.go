package httpapp

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"

	"remi-trip-planner/internal/trips"
)

// keepNotePickerColors is the ordered list shown as swatches on the notes composer and edit form.
var keepNotePickerColors = []struct{ Key, Label string }{
	{"default", "No color"},
	{"coral", "Coral"},
	{"peach", "Peach"},
	{"butter", "Butter"},
	{"mint", "Mint"},
	{"teal", "Teal"},
	{"sky", "Sky"},
	{"lavender", "Lavender"},
	{"rose", "Rose"},
	{"sand", "Sand"},
}

var keepNoteColorAllowed = map[string]bool{
	"sage": true, "mist": true, "blush": true,
	"pearl": true, "steel": true,
}

func init() {
	for _, c := range keepNotePickerColors {
		keepNoteColorAllowed[c.Key] = true
	}
}

func normalizeKeepNoteColor(c string) string {
	c = strings.TrimSpace(strings.ToLower(c))
	if c == "" {
		return "default"
	}
	if keepNoteColorAllowed[c] {
		return c
	}
	return "default"
}

func keepNoteColorInPicker(c string) bool {
	c = strings.TrimSpace(strings.ToLower(c))
	for _, x := range keepNotePickerColors {
		if x.Key == c {
			return true
		}
	}
	return false
}

// keepNoteBodyPreviewMaxRunes is how much of a note body we show on cards before ellipsis + “View more”.
const keepNoteBodyPreviewMaxRunes = 351

// keepNoteBodyPreview returns template data for note excerpts: Short text and whether the full body is hidden behind “View more”.
func keepNoteBodyPreview(body string) map[string]any {
	r := []rune(body)
	if len(r) <= keepNoteBodyPreviewMaxRunes {
		return map[string]any{"Short": body, "Expanded": false}
	}
	return map[string]any{
		"Short":    string(r[:keepNoteBodyPreviewMaxRunes]) + "…",
		"Expanded": true,
	}
}

// KeepMasonryCard is one tile in the Notes & lists masonry grid.
type KeepMasonryCard struct {
	Kind           string // "note" | "checklist"
	Note           trips.TripNote
	Category       string
	Items          []trips.ChecklistItem
	CategoryPinned bool // grouped checklist category pinned on main board
}

// KeepChecklistGroup is one category column in the Notes & lists reminder checklist grid (non-reminder views).
type KeepChecklistGroup struct {
	Category string
	Items    []trips.ChecklistItem
}

type keepChecklistGroup struct {
	Category string
	Items    []trips.ChecklistItem
}

func tripNotesReturnURL(tripID, view string) string {
	base := "/trips/" + tripID + "/notes"
	v := trips.NormalizeKeepView(view)
	if v != trips.KeepViewNotes {
		return base + "?view=" + url.QueryEscape(v)
	}
	return base
}

func checklistItemActivityTime(it trips.ChecklistItem) time.Time {
	if !it.UpdatedAt.IsZero() {
		return it.UpdatedAt
	}
	return it.CreatedAt
}

func checklistGroupActivityTime(items []trips.ChecklistItem) time.Time {
	var t time.Time
	for _, it := range items {
		st := checklistItemActivityTime(it)
		if st.After(t) {
			t = st
		}
	}
	return t
}

func groupChecklistForKeep(items []trips.ChecklistItem) []keepChecklistGroup {
	order := make([]string, 0)
	byCat := make(map[string][]trips.ChecklistItem)
	for _, it := range items {
		cat := trips.NormalizeKeepChecklistCategory(it.Category)
		if _, ok := byCat[cat]; !ok {
			order = append(order, cat)
		}
		byCat[cat] = append(byCat[cat], it)
	}
	out := make([]keepChecklistGroup, 0, len(order))
	for _, cat := range order {
		out = append(out, keepChecklistGroup{Category: cat, Items: byCat[cat]})
	}
	return out
}

func reminderSingletonGroups(items []trips.ChecklistItem) []keepChecklistGroup {
	out := make([]keepChecklistGroup, 0, len(items))
	for _, it := range items {
		out = append(out, keepChecklistGroup{
			Category: strings.TrimSpace(it.Category),
			Items:    []trips.ChecklistItem{it},
		})
	}
	return out
}

func keepMatchesQuery(q string, note trips.TripNote, items []trips.ChecklistItem, cat string) bool {
	q = strings.TrimSpace(strings.ToLower(q))
	if q == "" {
		return true
	}
	if strings.Contains(strings.ToLower(note.Title), q) || strings.Contains(strings.ToLower(note.Body), q) {
		return true
	}
	if strings.Contains(strings.ToLower(strings.TrimSpace(note.DueAt)), q) {
		return true
	}
	if strings.Contains(strings.ToLower(cat), q) {
		return true
	}
	for _, it := range items {
		if strings.Contains(strings.ToLower(it.Text), q) || strings.Contains(strings.ToLower(it.Category), q) {
			return true
		}
	}
	return false
}

func buildKeepMasonry(view string, noteList []trips.TripNote, checklistItems []trips.ChecklistItem, q string, pinnedCats map[string]bool) []KeepMasonryCard {
	if view == trips.KeepViewReminders {
		return buildKeepMasonryReminders(noteList, checklistItems, q)
	}
	return buildKeepMasonryNotesAndChecklists(noteList, checklistItems, q, pinnedCats)
}

func parseYYYYMMDDLocal(s string) (time.Time, bool) {
	s = strings.TrimSpace(s)
	if len(s) < 10 {
		return time.Time{}, false
	}
	t, err := time.Parse("2006-01-02", s[:10])
	return t, err == nil
}

func buildKeepMasonryReminders(noteList []trips.TripNote, checklistItems []trips.ChecklistItem, q string) []KeepMasonryCard {
	groups := reminderSingletonGroups(checklistItems)
	type scored struct {
		card   KeepMasonryCard
		due    time.Time
		hasDue bool
		pinned bool
		sortAt time.Time
	}
	var row []scored
	for _, n := range noteList {
		if !keepMatchesQuery(q, n, nil, "") {
			continue
		}
		due, ok := parseYYYYMMDDLocal(n.DueAt)
		sk := n.UpdatedAt
		if sk.IsZero() {
			sk = n.CreatedAt
		}
		row = append(row, scored{
			card:   KeepMasonryCard{Kind: "note", Note: n},
			due:    due,
			hasDue: ok,
			pinned: n.Pinned,
			sortAt: sk,
		})
	}
	for _, g := range groups {
		if !keepMatchesQuery(q, trips.TripNote{}, g.Items, g.Category) {
			continue
		}
		due, ok := parseYYYYMMDDLocal(g.Items[0].DueAt)
		row = append(row, scored{
			card:   KeepMasonryCard{Kind: "checklist", Category: g.Category, Items: g.Items},
			due:    due,
			hasDue: ok,
			pinned: false,
			sortAt: checklistGroupActivityTime(g.Items),
		})
	}
	sort.SliceStable(row, func(i, j int) bool {
		if row[i].pinned != row[j].pinned {
			return row[i].pinned
		}
		if row[i].hasDue && row[j].hasDue && !row[i].due.Equal(row[j].due) {
			return row[i].due.Before(row[j].due)
		}
		if row[i].hasDue != row[j].hasDue {
			return row[i].hasDue
		}
		return row[i].sortAt.After(row[j].sortAt)
	})
	out := make([]KeepMasonryCard, len(row))
	for i := range row {
		out[i] = row[i].card
	}
	return out
}

func buildKeepMasonryNotesAndChecklists(noteList []trips.TripNote, checklistItems []trips.ChecklistItem, q string, pinnedCats map[string]bool) []KeepMasonryCard {
	if pinnedCats == nil {
		pinnedCats = map[string]bool{}
	}
	type scored struct {
		card   KeepMasonryCard
		sortAt time.Time
		pinned bool
	}
	var row []scored
	for _, n := range noteList {
		if !keepMatchesQuery(q, n, nil, "") {
			continue
		}
		sk := n.UpdatedAt
		if sk.IsZero() {
			sk = n.CreatedAt
		}
		row = append(row, scored{
			card:   KeepMasonryCard{Kind: "note", Note: n},
			sortAt: sk,
			pinned: n.Pinned,
		})
	}
	for _, g := range groupChecklistForKeep(checklistItems) {
		if !keepMatchesQuery(q, trips.TripNote{}, g.Items, g.Category) {
			continue
		}
		catKey := trips.NormalizeKeepChecklistCategory(g.Category)
		row = append(row, scored{
			card: KeepMasonryCard{
				Kind: "checklist", Category: g.Category, Items: g.Items,
				CategoryPinned: pinnedCats[catKey],
			},
			sortAt: checklistGroupActivityTime(g.Items),
			pinned: pinnedCats[catKey],
		})
	}
	sort.SliceStable(row, func(i, j int) bool {
		if row[i].pinned != row[j].pinned {
			return row[i].pinned
		}
		return row[i].sortAt.After(row[j].sortAt)
	})
	out := make([]KeepMasonryCard, len(row))
	for i := range row {
		out[i] = row[i].card
	}
	return out
}

func buildKeepChecklistGroupsForGrid(view string, checklistItems []trips.ChecklistItem, q string) []KeepChecklistGroup {
	if view == trips.KeepViewReminders {
		return nil
	}
	groups := groupChecklistForKeep(checklistItems)
	out := make([]KeepChecklistGroup, 0, len(groups))
	for _, g := range groups {
		if !keepMatchesQuery(q, trips.TripNote{}, g.Items, g.Category) {
			continue
		}
		out = append(out, KeepChecklistGroup{Category: g.Category, Items: g.Items})
	}
	return out
}

const tripDetailsKeepPreviewMaxFallback = 3

func noteActivityTime(n trips.TripNote) time.Time {
	if !n.UpdatedAt.IsZero() {
		return n.UpdatedAt
	}
	return n.CreatedAt
}

func keepMasonryCardActivity(c KeepMasonryCard) time.Time {
	if c.Kind == "note" {
		return noteActivityTime(c.Note)
	}
	return checklistGroupActivityTime(c.Items)
}

// categoryPinnedForTripDetails matches Keep board pin records to category labels used
// on the main trip column (including "" → "Packing List" vs Keep's "" → "Lists").
func categoryPinnedForTripDetails(displayCategory string, pinnedList []string) bool {
	dc := strings.TrimSpace(displayCategory)
	for _, raw := range pinnedList {
		p := strings.TrimSpace(raw)
		if p == dc {
			return true
		}
		if trips.NormalizeKeepChecklistCategory(p) == trips.NormalizeKeepChecklistCategory(dc) {
			return true
		}
		if dc == "Packing List" && trips.NormalizeKeepChecklistCategory(p) == "Lists" {
			return true
		}
		if trips.NormalizeKeepChecklistCategory(dc) == "Lists" && p == "Packing List" {
			return true
		}
	}
	return false
}

// buildTripDetailsKeepPreview selects cards for the Trip Details "Notes & Checklists" section:
// all pinned notes and pinned checklist categories; if none, the three most recently edited cards.
func buildTripDetailsKeepPreview(notes []trips.TripNote, allGroups []checklistCategoryGroup, pinnedList []string) []KeepMasonryCard {
	var pinnedNotes []trips.TripNote
	for _, n := range notes {
		if n.Pinned {
			pinnedNotes = append(pinnedNotes, n)
		}
	}
	var pinnedGroups []checklistCategoryGroup
	for _, g := range allGroups {
		if categoryPinnedForTripDetails(g.Category, pinnedList) {
			pinnedGroups = append(pinnedGroups, g)
		}
	}
	if len(pinnedNotes) > 0 || len(pinnedGroups) > 0 {
		out := make([]KeepMasonryCard, 0, len(pinnedNotes)+len(pinnedGroups))
		for _, n := range pinnedNotes {
			out = append(out, KeepMasonryCard{Kind: "note", Note: n})
		}
		for _, g := range pinnedGroups {
			out = append(out, KeepMasonryCard{
				Kind:           "checklist",
				Category:       g.Category,
				Items:          g.Items,
				CategoryPinned: true,
			})
		}
		sort.SliceStable(out, func(i, j int) bool {
			return keepMasonryCardActivity(out[i]).After(keepMasonryCardActivity(out[j]))
		})
		return out
	}
	type scored struct {
		card KeepMasonryCard
		at   time.Time
	}
	row := make([]scored, 0, len(notes)+len(allGroups))
	for _, n := range notes {
		row = append(row, scored{
			card: KeepMasonryCard{Kind: "note", Note: n},
			at:   noteActivityTime(n),
		})
	}
	for _, g := range allGroups {
		row = append(row, scored{
			card: KeepMasonryCard{Kind: "checklist", Category: g.Category, Items: g.Items},
			at:   checklistGroupActivityTime(g.Items),
		})
	}
	sort.SliceStable(row, func(i, j int) bool {
		return row[i].at.After(row[j].at)
	})
	limit := tripDetailsKeepPreviewMaxFallback
	if len(row) < limit {
		limit = len(row)
	}
	out := make([]KeepMasonryCard, 0, limit)
	for i := 0; i < limit; i++ {
		out = append(out, row[i].card)
	}
	return out
}

func (a *app) tripNotesPage(w http.ResponseWriter, r *http.Request) {
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
	trip := details.Trip
	if a.redirectIfTripSectionDisabled(w, r, trip, "checklist") {
		return
	}
	view := trips.NormalizeKeepView(r.URL.Query().Get("view"))
	q := strings.TrimSpace(r.URL.Query().Get("q"))

	notes, err := a.tripService.ListTripNotesForKeepView(r.Context(), tripID, view)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	checklistItems, err := a.tripService.ListChecklistItemsForKeepView(r.Context(), tripID, view)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}

	pinnedList, err := a.tripService.ListPinnedChecklistCategories(r.Context(), tripID)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	pinnedCats := make(map[string]bool, len(pinnedList))
	for _, c := range pinnedList {
		pinnedCats[c] = true
	}

	masonry := buildKeepMasonry(view, notes, checklistItems, q, pinnedCats)

	uid := CurrentUserID(r.Context())
	settings, err := a.tripService.MergedSettingsForUI(r.Context(), uid)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}

	pageData := map[string]any{
		"Details":             details,
		"Settings":            settings,
		"CSRFToken":           CSRFToken(r.Context()),
		"KeepView":            view,
		"KeepQuery":           q,
		"KeepMasonry":         masonry,
		"KeepChecklistGroups": []KeepChecklistGroup(nil),
		"KeepNoteColors":      keepNotePickerColors,
		"ReminderCategories":  trips.ReminderChecklistCategories,
		"ReturnTo":            tripNotesReturnURL(tripID, view),
	}
	a.mergeTripSidebarContext(r.Context(), r, tripID, details, pageData, "notes")
	_ = a.templates.ExecuteTemplate(w, "trip_notes.html", pageData)
}

func (a *app) tripNoteCreate(w http.ResponseWriter, r *http.Request) {
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
	title := strings.TrimSpace(r.FormValue("title"))
	body := strings.TrimSpace(r.FormValue("body"))
	color := normalizeKeepNoteColor(r.FormValue("color"))
	if title == "" && body == "" {
		http.Redirect(w, r, tripNotesReturnURL(tripID, trips.KeepViewNotes), http.StatusSeeOther)
		return
	}
	if err := a.tripService.AddTripNote(r.Context(), trips.TripNote{
		ID:     uuid.NewString(),
		TripID: tripID,
		Title:  title,
		Body:   body,
		Color:  color,
		DueAt:  strings.TrimSpace(r.FormValue("due_at")),
	}); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, tripNotesReturnURL(tripID, trips.KeepViewNotes), http.StatusSeeOther)
}

func (a *app) tripKeepChecklistCreate(w http.ResponseWriter, r *http.Request) {
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
	dueAt := strings.TrimSpace(r.FormValue("due_at"))
	itemsJSON := strings.TrimSpace(r.FormValue("items_json"))
	var toAdd []string
	if itemsJSON != "" {
		if err := json.Unmarshal([]byte(itemsJSON), &toAdd); err != nil {
			http.Error(w, "invalid checklist items payload", http.StatusBadRequest)
			return
		}
	} else {
		raw := strings.TrimSpace(r.FormValue("lines"))
		for _, line := range strings.Split(raw, "\n") {
			text := strings.TrimSpace(line)
			if text != "" {
				toAdd = append(toAdd, text)
			}
		}
	}
	if len(toAdd) == 0 {
		http.Redirect(w, r, tripNotesReturnURL(tripID, trips.KeepViewNotes), http.StatusSeeOther)
		return
	}
	for _, text := range toAdd {
		trimmed := strings.TrimSpace(text)
		if trimmed == "" {
			continue
		}
		if err := a.tripService.AddChecklistItem(r.Context(), trips.ChecklistItem{
			TripID:   tripID,
			Category: category,
			Text:     trimmed,
			DueAt:    dueAt,
		}); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
	}
	http.Redirect(w, r, tripNotesReturnURL(tripID, trips.KeepViewNotes), http.StatusSeeOther)
}

func (a *app) tripNoteUpdate(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	noteID := chi.URLParam(r, "noteID")
	_ = r.ParseForm()
	n, err := a.tripService.GetTripNote(r.Context(), noteID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "note not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if n.TripID != tripID {
		http.Error(w, "note not found", http.StatusNotFound)
		return
	}
	color := normalizeKeepNoteColor(r.FormValue("color"))
	n.Title = strings.TrimSpace(r.FormValue("title"))
	n.Body = strings.TrimSpace(r.FormValue("body"))
	n.Color = color
	n.DueAt = strings.TrimSpace(r.FormValue("due_at"))
	if err := a.tripService.UpdateTripNote(r.Context(), n); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, tripNotesReturnURL(tripID, trips.KeepViewNotes), http.StatusSeeOther)
}

func (a *app) tripNoteIntent(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	noteID := chi.URLParam(r, "noteID")
	_ = r.ParseForm()
	intent := strings.TrimSpace(strings.ToLower(r.FormValue("intent")))
	n, err := a.tripService.GetTripNote(r.Context(), noteID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Error(w, "note not found", http.StatusNotFound)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if n.TripID != tripID {
		http.Error(w, "note not found", http.StatusNotFound)
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
		if err := a.tripService.DeleteTripNote(r.Context(), tripID, noteID); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if ret := strings.TrimSpace(r.FormValue("return_to")); ret != "" && isSafeReturnForTrip(ret, tripID) {
			http.Redirect(w, r, ret, http.StatusSeeOther)
		} else {
			http.Redirect(w, r, tripNotesReturnURL(tripID, trips.KeepViewTrash), http.StatusSeeOther)
		}
		return
	default:
		http.Error(w, "unknown intent", http.StatusBadRequest)
		return
	}
	if err := a.tripService.UpdateTripNote(r.Context(), n); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if ret := strings.TrimSpace(r.FormValue("return_to")); ret != "" && isSafeReturnForTrip(ret, tripID) {
		http.Redirect(w, r, ret, http.StatusSeeOther)
		return
	}
	redir := tripNotesReturnURL(tripID, trips.KeepViewNotes)
	switch intent {
	case "archive":
		redir = tripNotesReturnURL(tripID, trips.KeepViewArchive)
	case "trash":
		redir = tripNotesReturnURL(tripID, trips.KeepViewTrash)
	}
	http.Redirect(w, r, redir, http.StatusSeeOther)
}

func (a *app) tripKeepChecklistCategoryPin(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	_ = r.ParseForm()
	intent := strings.TrimSpace(strings.ToLower(r.FormValue("intent")))
	rawCat := strings.TrimSpace(r.FormValue("category"))
	if rawCat == "" {
		http.Error(w, "category required", http.StatusBadRequest)
		return
	}
	category := trips.NormalizeKeepChecklistCategory(rawCat)
	trip, err := a.tripService.GetTrip(r.Context(), tripID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if a.redirectIfTripSectionDisabled(w, r, trip, "checklist") {
		return
	}
	var pin bool
	switch intent {
	case "pin":
		pin = true
	case "unpin":
		pin = false
	default:
		http.Error(w, "unknown intent", http.StatusBadRequest)
		return
	}
	if err := a.tripService.SetChecklistCategoryPinned(r.Context(), tripID, category, pin); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	redir := tripNotesReturnURL(tripID, trips.KeepViewNotes)
	if ret := strings.TrimSpace(r.FormValue("return_to")); ret != "" && isSafeReturnForTrip(ret, tripID) {
		redir = ret
	}
	http.Redirect(w, r, redir, http.StatusSeeOther)
}

func (a *app) tripKeepChecklistBatchIntent(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	_ = r.ParseForm()
	intent := strings.TrimSpace(strings.ToLower(r.FormValue("intent")))
	ids := r.Form["item_id"]
	if len(ids) == 0 {
		http.Error(w, "no items", http.StatusBadRequest)
		return
	}
	trip, err := a.tripService.GetTrip(r.Context(), tripID)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if a.redirectIfTripSectionDisabled(w, r, trip, "checklist") {
		return
	}

	normalizeIDs := func() []string {
		var out []string
		for _, rawID := range ids {
			id := strings.TrimSpace(rawID)
			if id != "" {
				out = append(out, id)
			}
		}
		return out
	}
	itemIDs := normalizeIDs()
	if len(itemIDs) == 0 {
		http.Error(w, "no items", http.StatusBadRequest)
		return
	}

	switch intent {
	case "purge":
		for _, itemID := range itemIDs {
			it, err := a.tripService.GetChecklistItem(r.Context(), itemID)
			if err != nil || it.TripID != tripID {
				http.Error(w, "invalid item", http.StatusBadRequest)
				return
			}
			if !it.Trashed {
				http.Error(w, "only trashed items can be deleted permanently", http.StatusBadRequest)
				return
			}
			if err := a.tripService.DeleteChecklistItem(r.Context(), itemID); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		if ret := strings.TrimSpace(r.FormValue("return_to")); ret != "" && isSafeReturnForTrip(ret, tripID) {
			http.Redirect(w, r, ret, http.StatusSeeOther)
		} else {
			http.Redirect(w, r, tripNotesReturnURL(tripID, trips.KeepViewTrash), http.StatusSeeOther)
		}
		return
	case "archive", "trash", "restore":
		var archived, trashed bool
		switch intent {
		case "archive":
			archived, trashed = true, false
		case "trash":
			archived, trashed = false, true
		case "restore":
			archived, trashed = false, false
		}
		for _, itemID := range itemIDs {
			it, err := a.tripService.GetChecklistItem(r.Context(), itemID)
			if err != nil || it.TripID != tripID {
				http.Error(w, "invalid item", http.StatusBadRequest)
				return
			}
			if err := a.tripService.UpdateChecklistItemKeepFlags(r.Context(), tripID, itemID, archived, trashed); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
		}
		if ret := strings.TrimSpace(r.FormValue("return_to")); ret != "" && isSafeReturnForTrip(ret, tripID) {
			http.Redirect(w, r, ret, http.StatusSeeOther)
			return
		}
		switch intent {
		case "archive":
			http.Redirect(w, r, tripNotesReturnURL(tripID, trips.KeepViewArchive), http.StatusSeeOther)
		case "trash":
			http.Redirect(w, r, tripNotesReturnURL(tripID, trips.KeepViewTrash), http.StatusSeeOther)
		default:
			http.Redirect(w, r, tripNotesReturnURL(tripID, trips.KeepViewNotes), http.StatusSeeOther)
		}
		return
	default:
		http.Error(w, "unknown intent", http.StatusBadRequest)
	}
}

func keepChangeAffectsKeepUI(entity string) bool {
	switch strings.TrimSpace(strings.ToLower(entity)) {
	case "trip_note", "checklist_item", "checklist_category_pin":
		return true
	default:
		return false
	}
}

func (a *app) tripKeepBoardFragment(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	uid := CurrentUserID(r.Context())
	details, err := a.tripService.GetTripDetailsVisible(r.Context(), tripID, uid)
	if err != nil {
		if tripForbiddenOrMissing(err) {
			http.NotFound(w, r)
			return
		}
		writeInternalServerError(w, r, err)
		return
	}
	if !details.Trip.SectionEnabledChecklist() {
		http.NotFound(w, r)
		return
	}
	view := trips.NormalizeKeepView(r.URL.Query().Get("view"))
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	notes, err := a.tripService.ListTripNotesForKeepView(r.Context(), tripID, view)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	checklistItems, err := a.tripService.ListChecklistItemsForKeepView(r.Context(), tripID, view)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	pinnedList, err := a.tripService.ListPinnedChecklistCategories(r.Context(), tripID)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	pinnedCats := make(map[string]bool, len(pinnedList))
	for _, c := range pinnedList {
		pinnedCats[c] = true
	}
	masonry := buildKeepMasonry(view, notes, checklistItems, q, pinnedCats)
	settings, err := a.tripService.MergedSettingsForUI(r.Context(), uid)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	pageData := map[string]any{
		"Details":            details,
		"Settings":           settings,
		"CSRFToken":          CSRFToken(r.Context()),
		"KeepView":           view,
		"KeepQuery":          q,
		"KeepMasonry":        masonry,
		"KeepNoteColors":     keepNotePickerColors,
		"ReminderCategories": trips.ReminderChecklistCategories,
		"ReturnTo":           tripNotesReturnURL(tripID, view),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := a.templates.ExecuteTemplate(w, "tripKeepNotesBoardInner", pageData); err != nil {
		writeInternalServerError(w, r, err)
	}
}

func (a *app) tripKeepDetailsPreviewFragment(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	uid := CurrentUserID(r.Context())
	details, err := a.tripService.GetTripDetailsVisible(r.Context(), tripID, uid)
	if err != nil {
		if tripForbiddenOrMissing(err) {
			http.NotFound(w, r)
			return
		}
		writeInternalServerError(w, r, err)
		return
	}
	if !details.Trip.SectionEnabledChecklist() {
		http.NotFound(w, r)
		return
	}
	checklistCategoryGroups := buildChecklistCategoryGroups(details.Checklist, trips.ReminderChecklistCategories)
	keepNotes, err := a.tripService.ListTripNotesForKeepView(r.Context(), tripID, trips.KeepViewNotes)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	pinnedChecklistCats, err := a.tripService.ListPinnedChecklistCategories(r.Context(), tripID)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	tripKeepPreview := buildTripDetailsKeepPreview(keepNotes, checklistCategoryGroups, pinnedChecklistCats)
	settings, err := a.tripService.MergedSettingsForUI(r.Context(), uid)
	if err != nil {
		writeInternalServerError(w, r, err)
		return
	}
	pageData := map[string]any{
		"Details":             details,
		"Settings":            settings,
		"CSRFToken":           CSRFToken(r.Context()),
		"TripKeepPreview":     tripKeepPreview,
		"ChecklistCategories": trips.ReminderChecklistCategories,
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if err := a.templates.ExecuteTemplate(w, "tripKeepDetailsPreviewInner", pageData); err != nil {
		writeInternalServerError(w, r, err)
	}
}
