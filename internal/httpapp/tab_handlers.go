package httpapp

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"

	"remi-trip-planner/internal/trips"
)

const tabDateLayout = "2006-01-02"

// TabPayerThumb is avatar + name for Tab expense “paid by” cells.
type TabPayerThumb struct {
	Name       string
	Initial    string
	AvatarPath string
	IsGuest    bool
}

// TabSimplifyTransferRow is one suggested settlement in the Tab “simplified debts” UI.
type TabSimplifyTransferRow struct {
	FromName       string
	ToName         string
	FromKey        string
	ToKey          string
	FromInitial    string
	ToInitial      string
	FromAvatarPath string
	ToAvatarPath   string
	FromGuest      bool
	ToGuest        bool
	FromTone       int
	ToTone         int
	Amount         float64
}

func tabParticipantKeyTone(key string) int {
	var h uint32 = 2166136261
	for i := 0; i < len(key); i++ {
		h ^= uint32(key[i])
		h *= 16777619
	}
	return int(h % 3)
}

func buildTabSimplifyTransferRows(transfers []trips.TabTransfer, party []trips.UserProfile, guests []trips.TripGuest, departed []trips.DepartedTabParticipant) []TabSimplifyTransferRow {
	out := make([]TabSimplifyTransferRow, 0, len(transfers))
	for _, tr := range transfers {
		f := tabPayerThumb(party, guests, departed, tr.FromKey)
		tTo := tabPayerThumb(party, guests, departed, tr.ToKey)
		out = append(out, TabSimplifyTransferRow{
			FromName:       f.Name,
			ToName:         tTo.Name,
			FromKey:        tr.FromKey,
			ToKey:          tr.ToKey,
			FromInitial:    f.Initial,
			ToInitial:      tTo.Initial,
			FromAvatarPath: f.AvatarPath,
			ToAvatarPath:   tTo.AvatarPath,
			FromGuest:      f.IsGuest,
			ToGuest:        tTo.IsGuest,
			FromTone:       tabParticipantKeyTone(tr.FromKey),
			ToTone:         tabParticipantKeyTone(tr.ToKey),
			Amount:         tr.Amount,
		})
	}
	return out
}

func tabPayerThumb(party []trips.UserProfile, guests []trips.TripGuest, departed []trips.DepartedTabParticipant, paidKey string) TabPayerThumb {
	paidKey = strings.TrimSpace(paidKey)
	kind, id, ok := trips.ParseParticipantKey(paidKey)
	if !ok {
		return TabPayerThumb{Name: paidKey, Initial: "?"}
	}
	if kind == "user" {
		for _, u := range party {
			if u.ID == id {
				return TabPayerThumb{
					Name:       u.PublicDisplayName(),
					Initial:    u.InitialForAvatar(),
					AvatarPath: strings.TrimSpace(u.AvatarPath),
				}
			}
		}
	}
	if kind == "guest" {
		for _, g := range guests {
			if g.ID == id {
				return TabPayerThumb{
					Name:    g.DisplayName,
					Initial: trips.GuestInitialFromDisplayName(g.DisplayName),
					IsGuest: true,
				}
			}
		}
	}
	for _, d := range departed {
		if strings.TrimSpace(d.ParticipantKey) != paidKey {
			continue
		}
		name := strings.TrimSpace(d.DisplayName)
		if name == "" {
			name = paidKey
		}
		if kind == "guest" {
			return TabPayerThumb{
				Name:    name + " (guest) (Left trip)",
				Initial: trips.GuestInitialFromDisplayName(name),
				IsGuest: true,
			}
		}
		return TabPayerThumb{
			Name:    name + " (Left trip)",
			Initial: trips.GuestInitialFromDisplayName(name),
		}
	}
	return TabPayerThumb{Name: paidKey, Initial: "?"}
}

// TabTimeSeriesPoint is one day bucket for the “Spending over time” chart.
type TabTimeSeriesPoint struct {
	Label  string  `json:"label"` // e.g. "01-Apr" for chart axis
	Date   string  `json:"date"`  // YYYY-MM-DD (trip-local calendar day)
	Amount float64 `json:"amount"`
}

func tabSpendingOverTimeSeries(trip trips.Trip, effUIDate string, tab []trips.Expense) []TabTimeSeriesPoint {
	start, end, ok := tabTripDateRange(trip.StartDate, trip.EndDate, tab)
	if !ok {
		return nil
	}
	labelLayout := "02-Jan"
	if trips.UIDateIsMDY(effUIDate) {
		labelLayout = "Jan-02"
	}
	byDay := map[string]float64{}
	for _, e := range tab {
		d := strings.TrimSpace(e.SpentOn)
		if d == "" {
			continue
		}
		byDay[d] += e.Amount
	}
	var points []TabTimeSeriesPoint
	n := 0
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		if n >= 400 {
			break
		}
		key := d.Format(tabDateLayout)
		amt := byDay[key]
		points = append(points, TabTimeSeriesPoint{
			Label:  d.Format(labelLayout),
			Date:   key,
			Amount: math.Round(amt*100) / 100,
		})
		n++
	}
	return points
}

func tabTripDateRange(startStr, endStr string, tab []trips.Expense) (start, end time.Time, ok bool) {
	startStr = strings.TrimSpace(startStr)
	endStr = strings.TrimSpace(endStr)
	var err error
	if startStr != "" && endStr != "" {
		start, err = time.Parse(tabDateLayout, startStr)
		if err != nil {
			start = time.Time{}
		}
		end, err = time.Parse(tabDateLayout, endStr)
		if err != nil {
			end = time.Time{}
		}
		if !start.IsZero() && !end.IsZero() && !end.Before(start) {
			return start, end, true
		}
	}
	var dates []string
	for _, e := range tab {
		if strings.TrimSpace(e.SpentOn) != "" {
			dates = append(dates, strings.TrimSpace(e.SpentOn))
		}
	}
	if len(dates) == 0 {
		return time.Time{}, time.Time{}, false
	}
	sort.Strings(dates)
	start, err = time.Parse(tabDateLayout, dates[0])
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	end, err = time.Parse(tabDateLayout, dates[len(dates)-1])
	if err != nil {
		return time.Time{}, time.Time{}, false
	}
	return start, end, true
}

// TabChartRow is a single bar in Tab analytics (category, payer, or time bucket).
type TabChartRow struct {
	Label  string
	Amount float64
	Pct    float64
}

func participantLabelMap(party []trips.UserProfile, guests []trips.TripGuest, departed []trips.DepartedTabParticipant) map[string]string {
	m := make(map[string]string)
	for _, u := range party {
		m[trips.ParticipantKeyUser(u.ID)] = u.PublicDisplayName()
	}
	for _, g := range guests {
		m[trips.ParticipantKeyGuest(g.ID)] = g.DisplayName + " (guest)"
	}
	for _, d := range departed {
		k := strings.TrimSpace(d.ParticipantKey)
		if k == "" || m[k] != "" {
			continue
		}
		name := strings.TrimSpace(d.DisplayName)
		if name == "" {
			name = k
		}
		if kind, _, ok := trips.ParseParticipantKey(k); ok && kind == "guest" {
			m[k] = name + " (guest) (Left trip)"
		} else {
			m[k] = name + " (Left trip)"
		}
	}
	return m
}

func tabCategoryChartRows(tab []trips.Expense) []TabChartRow {
	m := map[string]float64{}
	for _, e := range tab {
		cat := strings.TrimSpace(e.Category)
		if cat == "" {
			cat = "General"
		}
		m[cat] += e.Amount
	}
	return mapToChartRows(m, func(k string) string { return k })
}

func tabPayerChartRows(tab []trips.Expense, ownerID string, labels map[string]string) []TabChartRow {
	m := map[string]float64{}
	for _, e := range tab {
		k := trips.EffectivePaidBy(e, ownerID)
		m[k] += e.Amount
	}
	return mapToChartRows(m, func(k string) string {
		if lb := labels[k]; lb != "" {
			return lb
		}
		return k
	})
}

func tabTimeChartRows(trip trips.Trip, effUIDate string, tab []trips.Expense) []TabChartRow {
	m := map[string]float64{}
	for _, e := range tab {
		d := strings.TrimSpace(e.SpentOn)
		if d == "" {
			d = "(no date)"
		}
		m[d] += e.Amount
	}
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var max float64
	for _, v := range m {
		if v > max {
			max = v
		}
	}
	layout := trips.UIDateNumericLayout(effUIDate)
	var rows []TabChartRow
	for _, k := range keys {
		pct := 0.0
		if max > 0 {
			pct = m[k] / max * 100
		}
		label := k
		if k != "(no date)" {
			if _, err := time.Parse(tabDateLayout, k); err == nil {
				label = trips.FormatISODate(k, layout)
			}
		}
		rows = append(rows, TabChartRow{Label: label, Amount: m[k], Pct: pct})
	}
	return rows
}

func mapToChartRows(values map[string]float64, labelFn func(string) string) []TabChartRow {
	var max float64
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	keys := make([]string, 0, len(values))
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var rows []TabChartRow
	for _, k := range keys {
		pct := 0.0
		if max > 0 {
			pct = values[k] / max * 100
		}
		rows = append(rows, TabChartRow{Label: labelFn(k), Amount: values[k], Pct: pct})
	}
	return rows
}

func sumTabExpenseAmounts(tab []trips.Expense) float64 {
	var s float64
	for _, e := range tab {
		s += e.Amount
	}
	return math.Round(s*100) / 100
}

func sortTabExpensesNewestFirst(list []trips.Expense) {
	sort.Slice(list, func(i, j int) bool {
		di, dj := list[i].SpentOn, list[j].SpentOn
		if di != "" && dj != "" && di != dj {
			return di > dj
		}
		if di != "" && dj == "" {
			return true
		}
		if di == "" && dj != "" {
			return false
		}
		return list[i].CreatedAt.After(list[j].CreatedAt)
	})
}

func tabYourShareCents(uid string, tab []trips.Expense, partyIDs, guestIDs []string, tripOwnerUserID string) float64 {
	me := trips.ParticipantKeyUser(uid)
	var sum float64
	for _, e := range tab {
		sh, err := trips.SharesForExpense(e, partyIDs, guestIDs, tripOwnerUserID)
		if err != nil {
			continue
		}
		sum += sh[me]
	}
	return math.Round(sum*100) / 100
}

func buildEqualSplitJSON(party []trips.UserProfile, guests []trips.TripGuest) string {
	var p trips.TabSplitPayload
	for _, u := range party {
		p.Participants = append(p.Participants, trips.ParticipantKeyUser(u.ID))
	}
	for _, g := range guests {
		p.Participants = append(p.Participants, trips.ParticipantKeyGuest(g.ID))
	}
	if len(p.Participants) == 0 {
		return `{"participants":[]}`
	}
	b, _ := json.Marshal(p)
	return string(b)
}

func tabAllowedParticipantKeys(party []trips.UserProfile, guests []trips.TripGuest, departed []trips.DepartedTabParticipant) map[string]struct{} {
	m := make(map[string]struct{})
	for _, u := range party {
		m[trips.ParticipantKeyUser(u.ID)] = struct{}{}
	}
	for _, g := range guests {
		m[trips.ParticipantKeyGuest(g.ID)] = struct{}{}
	}
	for _, d := range departed {
		if k := strings.TrimSpace(d.ParticipantKey); k != "" {
			m[k] = struct{}{}
		}
	}
	return m
}

func (a *app) resolveTabSplitForRequest(ctx context.Context, tripID string, trip trips.Trip, amount float64, r *http.Request) (splitMode, splitJSON string, err error) {
	splitMode = strings.TrimSpace(r.FormValue("split_mode"))
	splitJSON = strings.TrimSpace(r.FormValue("split_json"))
	party, _ := a.tripService.TripParty(ctx, tripID)
	guests, _ := a.tripService.ListTripGuests(ctx, tripID)
	departed, _ := a.tripService.ListDepartedTabParticipants(ctx, tripID)
	allowed := tabAllowedParticipantKeys(party, guests, departed)
	quickEqualAll := strings.TrimSpace(strings.ToLower(r.FormValue("quick_tab_equal_all"))) == "1" ||
		strings.TrimSpace(strings.ToLower(r.FormValue("quick_tab_equal_all"))) == "true"

	if quickEqualAll && splitMode == "" && splitJSON == "" {
		// Quick add on Trip Details always starts from equal split across all current members + guests.
		splitMode = trips.TabSplitEqual
		splitJSON = buildEqualSplitJSON(party, guests)
	} else if splitMode == "" && splitJSON == "" {
		if strings.TrimSpace(trip.TabDefaultSplitMode) != "" && strings.TrimSpace(trip.TabDefaultSplitJSON) != "" {
			splitMode = trip.TabDefaultSplitMode
			splitJSON = trip.TabDefaultSplitJSON
			_, err = trips.NormalizeTabSplitPayload(splitMode, amount, splitJSON, allowed)
			if err != nil {
				splitMode = trips.TabSplitEqual
				splitJSON = buildEqualSplitJSON(party, guests)
			}
		} else {
			splitMode = trips.TabSplitEqual
			splitJSON = buildEqualSplitJSON(party, guests)
		}
	} else if splitJSON == "" {
		splitMode = trips.TabSplitEqual
		splitJSON = buildEqualSplitJSON(party, guests)
	}
	if splitMode == "" {
		splitMode = trips.TabSplitEqual
	}
	_, err = trips.NormalizeTabSplitPayload(splitMode, amount, splitJSON, allowed)
	return splitMode, splitJSON, err
}

func (a *app) parseTabExpenseFields(ctx context.Context, tripID string, trip trips.Trip, amount float64, fromTab bool, r *http.Request) (title, paidBy, splitMode, splitJSON string, err error) {
	if !fromTab {
		return "", "", "", "", nil
	}
	title = strings.TrimSpace(r.FormValue("title"))
	paidBy = strings.TrimSpace(r.FormValue("paid_by"))
	if paidBy == "" {
		paidBy = trips.ParticipantKeyUser(CurrentUserID(ctx))
	}
	party, _ := a.tripService.TripParty(ctx, tripID)
	guests, _ := a.tripService.ListTripGuests(ctx, tripID)
	departed, _ := a.tripService.ListDepartedTabParticipants(ctx, tripID)
	allowed := tabAllowedParticipantKeys(party, guests, departed)
	if _, ok := allowed[paidBy]; !ok {
		return "", "", "", "", errors.New("choose who paid from the trip members or guests")
	}
	splitMode, splitJSON, err = a.resolveTabSplitForRequest(ctx, tripID, trip, amount, r)
	return title, paidBy, splitMode, splitJSON, err
}

func (a *app) addTripGuest(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	_ = r.ParseForm()
	name := strings.TrimSpace(r.FormValue("display_name"))
	if name == "" {
		http.Error(w, "name is required", http.StatusBadRequest)
		return
	}
	if err := a.tripService.AddTripGuest(r.Context(), trips.TripGuest{TripID: tripID, DisplayName: name}); err != nil {
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
	http.Redirect(w, r, "/trips/"+tripID+"/settings", http.StatusSeeOther)
}

func (a *app) deleteTripGuest(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	guestID := chi.URLParam(r, "guestID")
	_ = r.ParseForm()
	if err := a.tripService.DeleteTripGuest(r.Context(), tripID, guestID); err != nil {
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
	http.Redirect(w, r, "/trips/"+tripID+"/settings", http.StatusSeeOther)
}

func (a *app) addTabSettlement(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	_ = r.ParseForm()
	amount, _ := strconv.ParseFloat(strings.TrimSpace(r.FormValue("amount")), 64)
	if amount <= 0 {
		http.Error(w, "invalid amount", http.StatusBadRequest)
		return
	}
	st := trips.TabSettlement{
		TripID:      tripID,
		PayerUserID: strings.TrimSpace(r.FormValue("payer_user_id")),
		PayeeUserID: strings.TrimSpace(r.FormValue("payee_user_id")),
		Amount:      amount,
		Method:      strings.TrimSpace(r.FormValue("method")),
		SettledOn:   strings.TrimSpace(r.FormValue("settled_on")),
		Notes:       strings.TrimSpace(r.FormValue("notes")),
	}
	if err := a.tripService.AddTabSettlement(r.Context(), st); err != nil {
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
	http.Redirect(w, r, "/trips/"+tripID+"/group-expenses", http.StatusSeeOther)
}

func (a *app) updateTabSettlement(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	settlementID := chi.URLParam(r, "settlementID")
	_ = r.ParseForm()
	tripRow, err := a.tripService.GetTrip(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if CurrentUserID(r.Context()) != tripRow.OwnerUserID {
		http.Error(w, "only the trip owner can edit settlements", http.StatusForbidden)
		return
	}
	amount, _ := strconv.ParseFloat(strings.TrimSpace(r.FormValue("amount")), 64)
	st := trips.TabSettlement{
		ID:          settlementID,
		TripID:      tripID,
		PayerUserID: strings.TrimSpace(r.FormValue("payer_user_id")),
		PayeeUserID: strings.TrimSpace(r.FormValue("payee_user_id")),
		Amount:      amount,
		Method:      strings.TrimSpace(r.FormValue("method")),
		SettledOn:   strings.TrimSpace(r.FormValue("settled_on")),
		Notes:       strings.TrimSpace(r.FormValue("notes")),
	}
	if err := a.tripService.UpdateTabSettlement(r.Context(), st); err != nil {
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
	http.Redirect(w, r, "/trips/"+tripID+"/group-expenses", http.StatusSeeOther)
}

func (a *app) deleteTabSettlement(w http.ResponseWriter, r *http.Request) {
	tripID := chi.URLParam(r, "tripID")
	settlementID := chi.URLParam(r, "settlementID")
	_ = r.ParseForm()
	tripRow, err := a.tripService.GetTrip(r.Context(), tripID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/", http.StatusSeeOther)
			return
		}
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	if CurrentUserID(r.Context()) != tripRow.OwnerUserID {
		http.Error(w, "only the trip owner can delete settlements", http.StatusForbidden)
		return
	}
	if err := a.tripService.DeleteTabSettlement(r.Context(), tripID, settlementID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			http.Redirect(w, r, "/trips/"+tripID+"/group-expenses", http.StatusSeeOther)
			return
		}
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
	http.Redirect(w, r, "/trips/"+tripID+"/group-expenses", http.StatusSeeOther)
}

// TabSettlementListRow is one row in the Recent settlements list on Group Expenses.
type TabSettlementListRow struct {
	SettlementID  string
	PrimaryLine   string
	SecondaryLine string
	AmountDisplay string
	IconClass     string
	AmountClass   string
	CanManage     bool
	PayerUserID   string
	PayeeUserID   string
	Amount        float64
	Method        string
	SettledOn     string
	Notes         string
}

func formatTabSettlementWhen(created time.Time, settledOnDate string) string {
	if !created.IsZero() {
		t := created.Local()
		now := time.Now().Local()
		tDay := time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
		nDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		diff := int(nDay.Sub(tDay) / (24 * time.Hour))
		clock := t.Format("3:04 PM")
		switch {
		case diff == 0:
			return "Today, " + clock
		case diff == 1:
			return "Yesterday, " + clock
		case diff >= 2 && diff < 7:
			return t.Format("Mon") + ", " + clock
		default:
			if t.Year() == now.Year() {
				return t.Format("Jan 2") + ", " + clock
			}
			return t.Format("Jan 2, 2006") + ", " + clock
		}
	}
	s := strings.TrimSpace(settledOnDate)
	if s == "" {
		return ""
	}
	if tt, err := time.ParseInLocation(tabDateLayout, s, time.Local); err == nil {
		return tt.Format("Jan 2, 2006")
	}
	return s
}

func buildTabSettlementRows(settlements []trips.TabSettlement, labels map[string]string, currentUserID, ownerUserID, currencySymbol string, tripArchived bool) []TabSettlementListRow {
	out := make([]TabSettlementListRow, 0, len(settlements))
	sym := strings.TrimSpace(currencySymbol)
	if sym == "" {
		sym = "$"
	}
	canManage := !tripArchived && currentUserID != "" && currentUserID == ownerUserID
	meKey := trips.ParticipantKeyUser(currentUserID)
	for _, s := range settlements {
		pk := trips.TabSettlementParticipantKey(s.PayerUserID)
		yk := trips.TabSettlementParticipantKey(s.PayeeUserID)
		payerName := strings.TrimSpace(labels[pk])
		payeeName := strings.TrimSpace(labels[yk])
		if payerName == "" {
			payerName = "Member"
		}
		if payeeName == "" {
			payeeName = "Member"
		}
		var primary, iconClass, amtClass, amountDisp string
		amt := s.Amount
		switch {
		case currentUserID != "" && yk == meKey:
			primary = payerName + " paid You"
			iconClass = "tab-settlement-icon--in"
			amtClass = "tab-settlement-amt--in"
			amountDisp = "+" + sym + fmt.Sprintf("%.2f", amt)
		case currentUserID != "" && pk == meKey:
			primary = "You paid " + payeeName
			iconClass = "tab-settlement-icon--out"
			amtClass = "tab-settlement-amt--out"
			amountDisp = "-" + sym + fmt.Sprintf("%.2f", amt)
		default:
			primary = payerName + " paid " + payeeName
			iconClass = "tab-settlement-icon--neutral"
			amtClass = "tab-settlement-amt--neutral"
			amountDisp = sym + fmt.Sprintf("%.2f", amt)
		}
		method := strings.TrimSpace(s.Method)
		if method == "" {
			method = "Settlement"
		}
		when := formatTabSettlementWhen(s.CreatedAt, s.SettledOn)
		sub := method
		if when != "" {
			sub = method + " • " + when
		}
		out = append(out, TabSettlementListRow{
			SettlementID:  s.ID,
			PrimaryLine:   primary,
			SecondaryLine: sub,
			AmountDisplay: amountDisp,
			IconClass:     iconClass,
			AmountClass:   amtClass,
			CanManage:     canManage,
			PayerUserID:   s.PayerUserID,
			PayeeUserID:   s.PayeeUserID,
			Amount:        s.Amount,
			Method:        s.Method,
			SettledOn:     s.SettledOn,
			Notes:         s.Notes,
		})
	}
	return out
}
