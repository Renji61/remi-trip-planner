package trips

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

const maxSyncOpsPerRequest = 200

// SyncApplyRequest is the JSON body for POST /api/v1/trips/{tripID}/sync.
type SyncApplyRequest struct {
	ClientID   string        `json:"client_id"`
	BaseCursor string        `json:"base_cursor"`
	Ops        []SyncOpInput `json:"ops"`
}

// SyncOpInput is one create/update/delete (or trip archive) operation.
type SyncOpInput struct {
	Entity    string          `json:"entity"`
	EntityID  string          `json:"entity_id"`
	Operation string          `json:"operation"`
	Payload   json.RawMessage `json:"payload"`
}

// SyncApplyResponse is returned after applying (or attempting) all operations.
type SyncApplyResponse struct {
	Status         string         `json:"status"`
	TripID         string         `json:"trip_id"`
	ClientID       string         `json:"client_id,omitempty"`
	AppliedCount   int            `json:"applied_count"`
	StaleBase      bool           `json:"stale_base"`
	LatestChangeID int64          `json:"latest_change_id"`
	Results        []SyncOpResult `json:"results"`
	ServerChanges  []Change       `json:"server_changes"`
}

// SyncOpResult reports success or a per-operation error (batch continues on error).
type SyncOpResult struct {
	Index int    `json:"index"`
	OK    bool   `json:"ok"`
	Code  string `json:"code,omitempty"`
	Error string `json:"error,omitempty"`
}

// ApplyTripSyncOps applies client operations with last-write-wins semantics (server wins on validation).
// acc must reflect the caller's trip access; trip delete/archive require IsOwner.
func (s *Service) ApplyTripSyncOps(ctx context.Context, tripID string, acc TripAccess, req SyncApplyRequest) (*SyncApplyResponse, error) {
	if strings.TrimSpace(tripID) == "" {
		return nil, errors.New("trip id required")
	}
	if len(req.Ops) > maxSyncOpsPerRequest {
		return nil, fmt.Errorf("too many ops (max %d)", maxSyncOpsPerRequest)
	}
	beforeID, err := s.repo.LatestChangeLogID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	baseNum, baseOK := parseSyncBaseCursor(req.BaseCursor)
	staleBase := baseOK && baseNum < beforeID

	results := make([]SyncOpResult, 0, len(req.Ops))
	applied := 0
	for i, op := range req.Ops {
		if err := s.applyOneSyncOp(ctx, tripID, acc, op); err != nil {
			res := SyncOpResult{Index: i, OK: false, Error: err.Error()}
			if IsConflictError(err) {
				res.Code = "conflict"
			}
			results = append(results, res)
			continue
		}
		applied++
		results = append(results, SyncOpResult{Index: i, OK: true})
	}

	latestID, err := s.repo.LatestChangeLogID(ctx, tripID)
	if err != nil {
		return nil, err
	}
	delta, err := s.repo.ListChangesAfterID(ctx, tripID, beforeID)
	if err != nil {
		return nil, err
	}

	status := "accepted"
	if len(req.Ops) > 0 {
		if applied == 0 {
			status = "rejected"
		} else if applied < len(req.Ops) {
			status = "partial"
		}
	}

	return &SyncApplyResponse{
		Status:         status,
		TripID:         tripID,
		ClientID:       strings.TrimSpace(req.ClientID),
		AppliedCount:   applied,
		StaleBase:      staleBase,
		LatestChangeID: latestID,
		Results:        results,
		ServerChanges:  delta,
	}, nil
}

func parseSyncBaseCursor(raw string) (int64, bool) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, false
	}
	for _, r := range s {
		if !unicode.IsDigit(r) {
			return 0, false
		}
	}
	n, err := strconv.ParseInt(s, 10, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func (s *Service) applyOneSyncOp(ctx context.Context, tripID string, acc TripAccess, op SyncOpInput) error {
	ent := strings.ToLower(strings.TrimSpace(op.Entity))
	oper := strings.ToLower(strings.TrimSpace(op.Operation))
	switch ent {
	case "trip":
		return s.syncApplyTrip(ctx, tripID, acc, oper, op)
	case "itinerary_item":
		return s.syncApplyItinerary(ctx, tripID, oper, op)
	case "expense":
		return s.syncApplyExpense(ctx, tripID, oper, op)
	case "checklist_item":
		return s.syncApplyChecklist(ctx, tripID, oper, op)
	case "trip_note":
		return s.syncApplyTripNote(ctx, tripID, oper, op)
	default:
		return fmt.Errorf("unknown entity %q", op.Entity)
	}
}

type tripSyncPatch struct {
	Name                   *string  `json:"name"`
	Description            *string  `json:"description"`
	StartDate              *string  `json:"start_date"`
	EndDate                *string  `json:"end_date"`
	CoverImage             *string  `json:"cover_image"`
	CurrencyName           *string  `json:"currency_name"`
	CurrencySymbol         *string  `json:"currency_symbol"`
	BudgetCap              *float64 `json:"budget_cap"`
	BudgetCapCents         *int64   `json:"budget_cap_cents"`
	HomeMapLatitude        *float64 `json:"home_map_latitude"`
	HomeMapLongitude       *float64 `json:"home_map_longitude"`
	HomeMapPlaceLabel      *string  `json:"home_map_place_label"`
	DistanceUnit           *string  `json:"distance_unit"`
	UIShowStay             *bool    `json:"ui_show_stay"`
	UIShowVehicle          *bool    `json:"ui_show_vehicle"`
	UIShowFlights          *bool    `json:"ui_show_flights"`
	UIShowSpends           *bool    `json:"ui_show_spends"`
	UIShowItinerary        *bool    `json:"ui_show_itinerary"`
	UIShowChecklist        *bool    `json:"ui_show_checklist"`
	UIShowTheTab           *bool    `json:"ui_show_the_tab"`
	UIShowDocuments        *bool    `json:"ui_show_documents"`
	UICollaborationEnabled *bool    `json:"ui_collaboration_enabled"`
	UIItineraryExpand      *string  `json:"ui_itinerary_expand"`
	UISpendsExpand         *string  `json:"ui_spends_expand"`
	UITimeFormat           *string  `json:"ui_time_format"`
	UIDateFormat           *string  `json:"ui_date_format"`
	UILabelStay            *string  `json:"ui_label_stay"`
	UILabelVehicle         *string  `json:"ui_label_vehicle"`
	UILabelFlights         *string  `json:"ui_label_flights"`
	UILabelSpends          *string  `json:"ui_label_spends"`
	UILabelGroupExpenses   *string  `json:"ui_label_group_expenses"`
	UIMainSectionOrder     *string  `json:"ui_main_section_order"`
	UISidebarWidgetOrder   *string  `json:"ui_sidebar_widget_order"`
	UIMainSectionHidden    *string  `json:"ui_main_section_hidden"`
	UISidebarWidgetHidden  *string  `json:"ui_sidebar_widget_hidden"`
	UIShowCustomLinks      *bool    `json:"ui_show_custom_links"`
	UICustomSidebarLinks   *string  `json:"ui_custom_sidebar_links"`
}

func (s *Service) syncApplyTrip(ctx context.Context, tripID string, acc TripAccess, oper string, op SyncOpInput) error {
	switch oper {
	case "create":
		return errors.New("trip create is not allowed on this endpoint")
	case "archive":
		if !acc.IsOwner {
			return ErrTripAccessDenied
		}
		return s.ArchiveTrip(ctx, tripID)
	case "delete":
		if !acc.IsOwner {
			return ErrTripAccessDenied
		}
		return s.repo.DeleteTrip(ctx, tripID)
	case "update":
		if len(op.Payload) == 0 {
			return errors.New("missing payload")
		}
		var patch tripSyncPatch
		if err := json.Unmarshal(op.Payload, &patch); err != nil {
			return fmt.Errorf("invalid trip payload: %w", err)
		}
		t, err := s.repo.GetTrip(ctx, tripID)
		if err != nil {
			return err
		}
		if patch.Name != nil {
			t.Name = strings.TrimSpace(*patch.Name)
		}
		if patch.Description != nil {
			t.Description = *patch.Description
		}
		if patch.StartDate != nil {
			t.StartDate = strings.TrimSpace(*patch.StartDate)
		}
		if patch.EndDate != nil {
			t.EndDate = strings.TrimSpace(*patch.EndDate)
		}
		if patch.CoverImage != nil {
			t.CoverImage = NormalizeTripCoverValue(strings.TrimSpace(*patch.CoverImage))
		}
		if patch.CurrencyName != nil {
			t.CurrencyName = strings.TrimSpace(*patch.CurrencyName)
		}
		if patch.CurrencySymbol != nil {
			t.CurrencySymbol = strings.TrimSpace(*patch.CurrencySymbol)
		}
		if patch.BudgetCapCents != nil && *patch.BudgetCapCents >= 0 {
			SetTripBudgetCapCents(&t, *patch.BudgetCapCents)
		} else if patch.BudgetCap != nil && *patch.BudgetCap >= 0 {
			SetTripBudgetCapFloat(&t, *patch.BudgetCap)
		}
		if patch.HomeMapLatitude != nil {
			t.HomeMapLatitude = *patch.HomeMapLatitude
		}
		if patch.HomeMapLongitude != nil {
			t.HomeMapLongitude = *patch.HomeMapLongitude
		}
		if patch.HomeMapPlaceLabel != nil {
			t.HomeMapPlaceLabel = strings.TrimSpace(*patch.HomeMapPlaceLabel)
		}
		if patch.DistanceUnit != nil {
			t.DistanceUnit = strings.TrimSpace(*patch.DistanceUnit)
		}
		if patch.UIShowStay != nil {
			t.UIShowStay = *patch.UIShowStay
		}
		if patch.UIShowVehicle != nil {
			t.UIShowVehicle = *patch.UIShowVehicle
		}
		if patch.UIShowFlights != nil {
			t.UIShowFlights = *patch.UIShowFlights
		}
		if patch.UIShowSpends != nil {
			t.UIShowSpends = *patch.UIShowSpends
		}
		if patch.UIShowItinerary != nil {
			t.UIShowItinerary = *patch.UIShowItinerary
		}
		if patch.UIShowChecklist != nil {
			t.UIShowChecklist = *patch.UIShowChecklist
		}
		if patch.UIShowTheTab != nil {
			t.UIShowTheTab = *patch.UIShowTheTab
		}
		if patch.UIShowDocuments != nil {
			t.UIShowDocuments = *patch.UIShowDocuments
		}
		if patch.UICollaborationEnabled != nil {
			t.UICollaborationEnabled = *patch.UICollaborationEnabled
		}
		if patch.UIItineraryExpand != nil {
			t.UIItineraryExpand = strings.TrimSpace(*patch.UIItineraryExpand)
		}
		if patch.UISpendsExpand != nil {
			t.UISpendsExpand = strings.TrimSpace(*patch.UISpendsExpand)
		}
		if patch.UITimeFormat != nil {
			t.UITimeFormat = strings.TrimSpace(*patch.UITimeFormat)
		}
		if patch.UIDateFormat != nil {
			t.UIDateFormat = NormalizeTripUIDateStorage(strings.TrimSpace(*patch.UIDateFormat))
		}
		if patch.UILabelStay != nil {
			t.UILabelStay = strings.TrimSpace(*patch.UILabelStay)
		}
		if patch.UILabelVehicle != nil {
			t.UILabelVehicle = strings.TrimSpace(*patch.UILabelVehicle)
		}
		if patch.UILabelFlights != nil {
			t.UILabelFlights = strings.TrimSpace(*patch.UILabelFlights)
		}
		if patch.UILabelSpends != nil {
			t.UILabelSpends = strings.TrimSpace(*patch.UILabelSpends)
		}
		if patch.UILabelGroupExpenses != nil {
			t.UILabelGroupExpenses = strings.TrimSpace(*patch.UILabelGroupExpenses)
		}
		if patch.UIMainSectionOrder != nil {
			t.UIMainSectionOrder = strings.TrimSpace(*patch.UIMainSectionOrder)
		}
		if patch.UISidebarWidgetOrder != nil {
			t.UISidebarWidgetOrder = strings.TrimSpace(*patch.UISidebarWidgetOrder)
		}
		if patch.UIMainSectionHidden != nil {
			t.UIMainSectionHidden = strings.TrimSpace(*patch.UIMainSectionHidden)
		}
		if patch.UISidebarWidgetHidden != nil {
			t.UISidebarWidgetHidden = strings.TrimSpace(*patch.UISidebarWidgetHidden)
		}
		if patch.UIShowCustomLinks != nil {
			t.UIShowCustomLinks = *patch.UIShowCustomLinks
		}
		if patch.UICustomSidebarLinks != nil {
			t.UICustomSidebarLinks = strings.TrimSpace(*patch.UICustomSidebarLinks)
		}
		if !t.UIShowSpends {
			t.UIShowTheTab = false
		}
		return s.UpdateTrip(ctx, t)
	default:
		return fmt.Errorf("unknown trip operation %q", op.Operation)
	}
}

type itinerarySyncPayload struct {
	ID                string  `json:"id"`
	DayNumber         int     `json:"day_number"`
	Title             string  `json:"title"`
	Notes             string  `json:"notes"`
	Location          string  `json:"location"`
	ImagePath         string  `json:"image_path"`
	Latitude          float64 `json:"latitude"`
	Longitude         float64 `json:"longitude"`
	EstCost           float64 `json:"est_cost"`
	EstCostCents      int64   `json:"est_cost_cents"`
	StartTime         string  `json:"start_time"`
	EndTime           string  `json:"end_time"`
	ExpectedUpdatedAt string  `json:"expected_updated_at"`
}

func (s *Service) syncApplyItinerary(ctx context.Context, tripID string, oper string, op SyncOpInput) error {
	switch oper {
	case "create":
		var p itinerarySyncPayload
		if err := json.Unmarshal(op.Payload, &p); err != nil {
			return fmt.Errorf("invalid itinerary payload: %w", err)
		}
		if strings.TrimSpace(p.Title) == "" {
			return errors.New("itinerary title required")
		}
		id := strings.TrimSpace(p.ID)
		if id == "" {
			id = strings.TrimSpace(op.EntityID)
		}
		if id == "" {
			id = uuid.NewString()
		}
		item := ItineraryItem{
			ID: id, TripID: tripID, DayNumber: p.DayNumber, Title: strings.TrimSpace(p.Title),
			Notes: p.Notes, Location: p.Location, ImagePath: p.ImagePath,
			Latitude: p.Latitude, Longitude: p.Longitude,
			StartTime: p.StartTime, EndTime: p.EndTime,
		}
		if p.EstCostCents != 0 || p.EstCost == 0 {
			SetItineraryEstCostCents(&item, p.EstCostCents)
		} else {
			SetItineraryEstCostFloat(&item, p.EstCost)
		}
		return s.AddItineraryItem(ctx, item)
	case "update":
		id := strings.TrimSpace(op.EntityID)
		if id == "" {
			var p itinerarySyncPayload
			if err := json.Unmarshal(op.Payload, &p); err != nil {
				return fmt.Errorf("invalid itinerary payload: %w", err)
			}
			id = strings.TrimSpace(p.ID)
		}
		if id == "" {
			return errors.New("itinerary entity_id or payload.id required")
		}
		items, err := s.repo.ListItineraryItems(ctx, tripID)
		if err != nil {
			return err
		}
		var cur ItineraryItem
		found := false
		for _, it := range items {
			if it.ID == id {
				cur = it
				found = true
				break
			}
		}
		if !found {
			return errors.New("itinerary item not found")
		}
		var p itinerarySyncPayload
		if err := json.Unmarshal(op.Payload, &p); err != nil {
			return fmt.Errorf("invalid itinerary payload: %w", err)
		}
		if strings.TrimSpace(p.Title) != "" {
			cur.Title = strings.TrimSpace(p.Title)
		}
		if p.DayNumber > 0 {
			cur.DayNumber = p.DayNumber
		}
		cur.Notes = p.Notes
		cur.Location = p.Location
		cur.ImagePath = p.ImagePath
		cur.Latitude = p.Latitude
		cur.Longitude = p.Longitude
		if p.EstCostCents != 0 || p.EstCost == 0 {
			SetItineraryEstCostCents(&cur, p.EstCostCents)
		} else {
			SetItineraryEstCostFloat(&cur, p.EstCost)
		}
		cur.StartTime = p.StartTime
		cur.EndTime = p.EndTime
		if ts := strings.TrimSpace(p.ExpectedUpdatedAt); ts != "" {
			expected, err := time.Parse(time.RFC3339Nano, ts)
			if err != nil {
				return fmt.Errorf("invalid itinerary expected_updated_at: %w", err)
			}
			cur.ExpectedUpdatedAt = expected
			cur.EnforceOptimisticLock = true
		}
		return s.UpdateItineraryItem(ctx, cur)
	case "delete":
		id := strings.TrimSpace(op.EntityID)
		if id == "" {
			return errors.New("itinerary entity_id required")
		}
		return s.DeleteItineraryItem(ctx, tripID, id)
	default:
		return fmt.Errorf("unknown itinerary_item operation %q", oper)
	}
}

type expenseSyncPayload struct {
	ID                string  `json:"id"`
	Category          string  `json:"category"`
	Amount            float64 `json:"amount"`
	AmountCents       int64   `json:"amount_cents"`
	Notes             string  `json:"notes"`
	SpentOn           string  `json:"spent_on"`
	PaymentMethod     string  `json:"payment_method"`
	Title             string  `json:"title"`
	PaidBy            string  `json:"paid_by"`
	SplitMode         string  `json:"split_mode"`
	SplitJSON         string  `json:"split_json"`
	FromTab           *bool   `json:"from_tab"`
	DueAt             string  `json:"due_at"`
	ExpectedUpdatedAt string  `json:"expected_updated_at"`
}

func (s *Service) syncApplyExpense(ctx context.Context, tripID string, oper string, op SyncOpInput) error {
	switch oper {
	case "create":
		var p expenseSyncPayload
		if err := json.Unmarshal(op.Payload, &p); err != nil {
			return fmt.Errorf("invalid expense payload: %w", err)
		}
		cat := strings.TrimSpace(p.Category)
		if cat == "" {
			cat = "Miscellaneous"
		}
		id := strings.TrimSpace(p.ID)
		if id == "" {
			id = strings.TrimSpace(op.EntityID)
		}
		if id == "" {
			id = uuid.NewString()
		}
		e := Expense{
			ID: id, TripID: tripID, Category: cat,
			Notes: p.Notes, SpentOn: p.SpentOn, PaymentMethod: p.PaymentMethod,
			Title: p.Title, PaidBy: p.PaidBy, SplitMode: p.SplitMode, SplitJSON: p.SplitJSON,
			DueAt: strings.TrimSpace(p.DueAt),
		}
		if p.AmountCents != 0 || p.Amount == 0 {
			SetExpenseAmountCents(&e, p.AmountCents)
		} else {
			SetExpenseAmountFloat(&e, p.Amount)
		}
		if p.FromTab != nil {
			e.FromTab = *p.FromTab
		}
		return s.AddExpense(ctx, e)
	case "update":
		id := strings.TrimSpace(op.EntityID)
		if id == "" {
			var p expenseSyncPayload
			if err := json.Unmarshal(op.Payload, &p); err != nil {
				return fmt.Errorf("invalid expense payload: %w", err)
			}
			id = strings.TrimSpace(p.ID)
		}
		if id == "" {
			return errors.New("expense entity_id or payload.id required")
		}
		prev, err := s.repo.GetExpense(ctx, tripID, id)
		if err != nil {
			return err
		}
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(op.Payload, &raw); err != nil {
			return fmt.Errorf("invalid expense payload: %w", err)
		}
		e := prev
		if v, ok := raw["category"]; ok {
			_ = json.Unmarshal(v, &e.Category)
			e.Category = strings.TrimSpace(e.Category)
		}
		if v, ok := raw["amount"]; ok {
			var amount float64
			_ = json.Unmarshal(v, &amount)
			SetExpenseAmountFloat(&e, amount)
		}
		if v, ok := raw["amount_cents"]; ok {
			var amountCents int64
			_ = json.Unmarshal(v, &amountCents)
			SetExpenseAmountCents(&e, amountCents)
		}
		if v, ok := raw["notes"]; ok {
			_ = json.Unmarshal(v, &e.Notes)
		}
		if v, ok := raw["spent_on"]; ok {
			_ = json.Unmarshal(v, &e.SpentOn)
		}
		if v, ok := raw["payment_method"]; ok {
			_ = json.Unmarshal(v, &e.PaymentMethod)
			e.PaymentMethod = strings.TrimSpace(e.PaymentMethod)
		}
		if v, ok := raw["title"]; ok {
			_ = json.Unmarshal(v, &e.Title)
		}
		if v, ok := raw["paid_by"]; ok {
			_ = json.Unmarshal(v, &e.PaidBy)
		}
		if v, ok := raw["split_mode"]; ok {
			_ = json.Unmarshal(v, &e.SplitMode)
		}
		if v, ok := raw["split_json"]; ok {
			_ = json.Unmarshal(v, &e.SplitJSON)
		}
		if v, ok := raw["from_tab"]; ok {
			_ = json.Unmarshal(v, &e.FromTab)
		}
		if v, ok := raw["due_at"]; ok {
			_ = json.Unmarshal(v, &e.DueAt)
			e.DueAt = strings.TrimSpace(e.DueAt)
		}
		if v, ok := raw["expected_updated_at"]; ok {
			var expectedRaw string
			_ = json.Unmarshal(v, &expectedRaw)
			expectedRaw = strings.TrimSpace(expectedRaw)
			if expectedRaw != "" {
				expectedAt, parseErr := time.Parse(time.RFC3339Nano, expectedRaw)
				if parseErr != nil {
					return errors.New("invalid expected_updated_at")
				}
				e.ExpectedUpdatedAt = expectedAt
				e.EnforceOptimisticLock = true
			}
		}
		return s.UpdateExpense(ctx, e)
	case "delete":
		id := strings.TrimSpace(op.EntityID)
		if id == "" {
			return errors.New("expense entity_id required")
		}
		return s.DeleteExpense(ctx, tripID, id)
	default:
		return fmt.Errorf("unknown expense operation %q", oper)
	}
}

type checklistSyncPayload struct {
	ID       string `json:"id"`
	Text     string `json:"text"`
	Category string `json:"category"`
	Done     *bool  `json:"done"`
	DueAt    string `json:"due_at"`
	Archived *bool  `json:"archived"`
	Trashed  *bool  `json:"trashed"`
}

func (s *Service) syncApplyChecklist(ctx context.Context, tripID string, oper string, op SyncOpInput) error {
	switch oper {
	case "create":
		var p checklistSyncPayload
		if err := json.Unmarshal(op.Payload, &p); err != nil {
			return fmt.Errorf("invalid checklist payload: %w", err)
		}
		if strings.TrimSpace(p.Text) == "" {
			return errors.New("checklist text required")
		}
		id := strings.TrimSpace(p.ID)
		if id == "" {
			id = strings.TrimSpace(op.EntityID)
		}
		if id == "" {
			id = uuid.NewString()
		}
		item := ChecklistItem{
			ID: id, TripID: tripID, Text: strings.TrimSpace(p.Text), Category: strings.TrimSpace(p.Category),
			DueAt: strings.TrimSpace(p.DueAt),
		}
		if p.Done != nil {
			item.Done = *p.Done
		}
		if p.Archived != nil {
			item.Archived = *p.Archived
		}
		if p.Trashed != nil {
			item.Trashed = *p.Trashed
		}
		return s.AddChecklistItem(ctx, item)
	case "update":
		id := strings.TrimSpace(op.EntityID)
		if id == "" {
			var p checklistSyncPayload
			if err := json.Unmarshal(op.Payload, &p); err != nil {
				return fmt.Errorf("invalid checklist payload: %w", err)
			}
			id = strings.TrimSpace(p.ID)
		}
		if id == "" {
			return errors.New("checklist entity_id or payload.id required")
		}
		existing, err := s.repo.GetChecklistItem(ctx, id)
		if err != nil {
			return err
		}
		if existing.TripID != tripID {
			return errors.New("checklist item not in this trip")
		}
		var p checklistSyncPayload
		if err := json.Unmarshal(op.Payload, &p); err != nil {
			return fmt.Errorf("invalid checklist payload: %w", err)
		}
		upd := existing
		needUpdate := false
		if strings.TrimSpace(p.Text) != "" {
			upd.Text = strings.TrimSpace(p.Text)
			needUpdate = true
		}
		if strings.TrimSpace(p.Category) != "" {
			upd.Category = strings.TrimSpace(p.Category)
			needUpdate = true
		}
		if strings.TrimSpace(p.DueAt) != "" {
			upd.DueAt = strings.TrimSpace(p.DueAt)
			needUpdate = true
		}
		if p.Archived != nil {
			upd.Archived = *p.Archived
			needUpdate = true
		}
		if p.Trashed != nil {
			upd.Trashed = *p.Trashed
			needUpdate = true
		}
		if needUpdate {
			if err := s.UpdateChecklistItem(ctx, upd); err != nil {
				return err
			}
		}
		if p.Done != nil && existing.Done != *p.Done {
			return s.ToggleChecklistItem(ctx, id, *p.Done)
		}
		return nil
	case "delete":
		id := strings.TrimSpace(op.EntityID)
		if id == "" {
			return errors.New("checklist entity_id required")
		}
		return s.DeleteChecklistItem(ctx, id)
	default:
		return fmt.Errorf("unknown checklist_item operation %q", oper)
	}
}

type tripNoteSyncPayload struct {
	ID       string  `json:"id"`
	Title    *string `json:"title"`
	Body     *string `json:"body"`
	Color    *string `json:"color"`
	DueAt    *string `json:"due_at"`
	Pinned   *bool   `json:"pinned"`
	Archived *bool   `json:"archived"`
	Trashed  *bool   `json:"trashed"`
}

func (s *Service) syncApplyTripNote(ctx context.Context, tripID string, oper string, op SyncOpInput) error {
	switch oper {
	case "create":
		var p tripNoteSyncPayload
		if err := json.Unmarshal(op.Payload, &p); err != nil {
			return fmt.Errorf("invalid trip_note payload: %w", err)
		}
		id := strings.TrimSpace(p.ID)
		if id == "" {
			id = strings.TrimSpace(op.EntityID)
		}
		if id == "" {
			id = uuid.NewString()
		}
		n := TripNote{ID: id, TripID: tripID}
		if p.Title != nil {
			n.Title = strings.TrimSpace(*p.Title)
		}
		if p.Body != nil {
			n.Body = strings.TrimSpace(*p.Body)
		}
		if p.Color != nil {
			n.Color = strings.TrimSpace(*p.Color)
		}
		if strings.TrimSpace(n.Color) == "" {
			n.Color = "default"
		}
		if p.Pinned != nil {
			n.Pinned = *p.Pinned
		}
		if p.Archived != nil {
			n.Archived = *p.Archived
		}
		if p.Trashed != nil {
			n.Trashed = *p.Trashed
		}
		if p.DueAt != nil {
			n.DueAt = strings.TrimSpace(*p.DueAt)
		}
		return s.AddTripNote(ctx, n)
	case "update":
		id := strings.TrimSpace(op.EntityID)
		if id == "" {
			var p tripNoteSyncPayload
			if err := json.Unmarshal(op.Payload, &p); err != nil {
				return fmt.Errorf("invalid trip_note payload: %w", err)
			}
			id = strings.TrimSpace(p.ID)
		}
		if id == "" {
			return errors.New("trip_note entity_id or payload.id required")
		}
		existing, err := s.repo.GetTripNote(ctx, id)
		if err != nil {
			return err
		}
		if existing.TripID != tripID {
			return errors.New("trip_note not in this trip")
		}
		var p tripNoteSyncPayload
		if err := json.Unmarshal(op.Payload, &p); err != nil {
			return fmt.Errorf("invalid trip_note payload: %w", err)
		}
		upd := existing
		changed := false
		if p.Title != nil {
			upd.Title = strings.TrimSpace(*p.Title)
			changed = true
		}
		if p.Body != nil {
			upd.Body = strings.TrimSpace(*p.Body)
			changed = true
		}
		if p.Color != nil {
			upd.Color = strings.TrimSpace(*p.Color)
			if upd.Color == "" {
				upd.Color = "default"
			}
			changed = true
		}
		if p.Pinned != nil {
			upd.Pinned = *p.Pinned
			changed = true
		}
		if p.Archived != nil {
			upd.Archived = *p.Archived
			changed = true
		}
		if p.Trashed != nil {
			upd.Trashed = *p.Trashed
			changed = true
		}
		if p.DueAt != nil {
			upd.DueAt = strings.TrimSpace(*p.DueAt)
			changed = true
		}
		if !changed {
			return nil
		}
		return s.UpdateTripNote(ctx, upd)
	case "delete":
		id := strings.TrimSpace(op.EntityID)
		if id == "" {
			return errors.New("trip_note entity_id required")
		}
		return s.DeleteTripNoteHard(ctx, tripID, id)
	default:
		return fmt.Errorf("unknown trip_note operation %q", oper)
	}
}
