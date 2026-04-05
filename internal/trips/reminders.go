package trips

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"
)

// ItineraryItemStartLocal combines trip start date, day number, and optional HH:MM (empty → 09:00).
func ItineraryItemStartLocal(trip Trip, item ItineraryItem) (time.Time, bool) {
	start, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(trip.StartDate), time.Local)
	if err != nil {
		return time.Time{}, false
	}
	day := start.AddDate(0, 0, item.DayNumber-1)
	h, m := 9, 0
	st := strings.TrimSpace(item.StartTime)
	if st != "" {
		parts := strings.SplitN(st, ":", 3)
		if len(parts) >= 2 {
			h, _ = strconv.Atoi(parts[0])
			m, _ = strconv.Atoi(parts[1])
		}
	}
	return time.Date(day.Year(), day.Month(), day.Day(), h, m, 0, 0, time.Local), true
}

func parseLocalDateTime(layout, value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation(layout, value, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func endOfLocalDay(d time.Time) time.Time {
	y, m, day := d.Date()
	return time.Date(y, m, day, 23, 59, 59, 999999999, d.Location())
}

func reminderFireEligible(fireAt, relevanceEnd time.Time, now time.Time) bool {
	if now.Before(fireAt) {
		return false
	}
	if now.Sub(fireAt) > 48*time.Hour {
		return false
	}
	if relevanceEnd.Before(now.Add(-30 * time.Minute)) {
		return false
	}
	return true
}

func (s *Service) tripNotificationRecipients(ctx context.Context, trip Trip) ([]string, error) {
	seen := map[string]struct{}{}
	if id := strings.TrimSpace(trip.OwnerUserID); id != "" {
		seen[id] = struct{}{}
	}
	ids, err := s.repo.ListActiveTripMemberUserIDs(ctx, trip.ID)
	if err != nil {
		return nil, err
	}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id != "" {
			seen[id] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for id := range seen {
		out = append(out, id)
	}
	return out, nil
}

func (s *Service) tryNotify(ctx context.Context, userID, tripID, title, body, href, kind, dedupe string, now time.Time) error {
	if userID == "" || dedupe == "" {
		return nil
	}
	_, err := s.repo.InsertAppNotification(ctx, AppNotification{
		UserID:    userID,
		TripID:    tripID,
		Title:     title,
		Body:      body,
		Href:      href,
		Kind:      kind,
		DedupeKey: dedupe,
	})
	return err
}

// RunReminderTick evaluates due reminders for active trips and inserts in-app notifications (deduped per user).
func (s *Service) RunReminderTick(ctx context.Context, now time.Time) error {
	if now.IsZero() {
		now = time.Now()
	}
	tripIDs, err := s.repo.ListTripIDsForReminderScan(ctx)
	if err != nil {
		return err
	}
	for _, tid := range tripIDs {
		trip, err := s.repo.GetTrip(ctx, tid)
		if err != nil || !TripEligibleForInAppNotifications(trip, now) {
			continue
		}
		recipients, err := s.tripNotificationRecipients(ctx, trip)
		if err != nil || len(recipients) == 0 {
			continue
		}
		hrefBase := "/trips/" + trip.ID

		lodgings, _ := s.repo.ListLodgings(ctx, tid)
		flights, _ := s.repo.ListFlights(ctx, tid)
		vehicles, _ := s.repo.ListVehicleRentals(ctx, tid)
		expenses, _ := s.repo.ListExpenses(ctx, tid)
		checklist, _ := s.repo.ListChecklistItems(ctx, tid)
		items, _ := s.repo.ListItineraryItems(ctx, tid)
		customReminders, _ := s.repo.ListItineraryCustomRemindersByTrip(ctx, tid)

		for _, l := range lodgings {
			if ci, ok := parseLocalDateTime("2006-01-02T15:04", l.CheckInAt); ok {
				for _, off := range []int{24 * 60, 60} {
					fire := ci.Add(-time.Duration(off) * time.Minute)
					if !reminderFireEligible(fire, ci, now) {
						continue
					}
					dedupe := fmt.Sprintf("lodging:%s:checkin:%s:pre%d", l.ID, ci.UTC().Format(time.RFC3339Nano), off)
					title := "Accommodation check-in"
					body := fmt.Sprintf("%s — check-in %s", l.Name, ci.Format("Mon Jan 2, 15:04"))
					for _, uid := range recipients {
						_ = s.tryNotify(ctx, uid, tid, title, body, hrefBase+"/accommodation", "check_in", dedupe, now)
					}
				}
			}
			if co, ok := parseLocalDateTime("2006-01-02T15:04", l.CheckOutAt); ok {
				fire := co.Add(-time.Hour)
				if !reminderFireEligible(fire, co, now) {
					continue
				}
				dedupe := fmt.Sprintf("lodging:%s:checkout:%s:pre60", l.ID, co.UTC().Format(time.RFC3339Nano))
				title := "Accommodation check-out"
				body := fmt.Sprintf("%s — check-out %s", l.Name, co.Format("Mon Jan 2, 15:04"))
				for _, uid := range recipients {
					_ = s.tryNotify(ctx, uid, tid, title, body, hrefBase+"/accommodation", "check_out", dedupe, now)
				}
			}
		}

		for _, f := range flights {
			if dep, ok := parseLocalDateTime("2006-01-02T15:04", f.DepartAt); ok {
				for _, off := range []int{24 * 60, 180, 45} {
					fire := dep.Add(-time.Duration(off) * time.Minute)
					if !reminderFireEligible(fire, dep, now) {
						continue
					}
					dedupe := fmt.Sprintf("flight:%s:dep:%s:pre%d", f.ID, dep.UTC().Format(time.RFC3339Nano), off)
					title := "Flight reminder"
					if off <= 45 {
						title = "Boarding / departure soon"
					}
					body := formatFlightReminderNotificationBody(trip, f, dep)
					for _, uid := range recipients {
						_ = s.tryNotify(ctx, uid, tid, title, body, hrefBase+"/flights", "boarding", dedupe, now)
					}
				}
			}
		}

		for _, v := range vehicles {
			if pu, ok := parseLocalDateTime("2006-01-02T15:04", v.PickUpAt); ok {
				fire := pu.Add(-time.Hour)
				if reminderFireEligible(fire, pu, now) {
					dedupe := fmt.Sprintf("veh:%s:pu:%s:pre60", v.ID, pu.UTC().Format(time.RFC3339Nano))
					for _, uid := range recipients {
						_ = s.tryNotify(ctx, uid, tid, "Vehicle pick-up", "Rental pick-up "+pu.Format("Mon Jan 2, 15:04"), hrefBase+"/vehicle-rental", "check_in", dedupe, now)
					}
				}
			}
			if dr, ok := parseLocalDateTime("2006-01-02T15:04", v.DropOffAt); ok {
				fire := dr.Add(-time.Hour)
				if reminderFireEligible(fire, dr, now) {
					dedupe := fmt.Sprintf("veh:%s:do:%s:pre60", v.ID, dr.UTC().Format(time.RFC3339Nano))
					for _, uid := range recipients {
						_ = s.tryNotify(ctx, uid, tid, "Vehicle drop-off", "Return rental "+dr.Format("Mon Jan 2, 15:04"), hrefBase+"/vehicle-rental", "check_out", dedupe, now)
					}
				}
			}
		}

		for _, e := range expenses {
			due := strings.TrimSpace(e.DueAt)
			if due == "" {
				continue
			}
			dueDay, err := time.ParseInLocation("2006-01-02", due, time.Local)
			if err != nil {
				continue
			}
			for _, slot := range []struct {
				label string
				at    time.Time
			}{
				{"pay1d", time.Date(dueDay.Year(), dueDay.Month(), dueDay.Day(), 9, 0, 0, 0, time.Local).AddDate(0, 0, -1)},
				{"pay0d", time.Date(dueDay.Year(), dueDay.Month(), dueDay.Day(), 9, 0, 0, 0, time.Local)},
			} {
				end := endOfLocalDay(dueDay)
				if !reminderFireEligible(slot.at, end, now) {
					continue
				}
				dedupe := fmt.Sprintf("expense:%s:due:%s:%s", e.ID, due, slot.label)
				title := "Payment due"
				body := fmt.Sprintf("%s — %s%s due %s", e.Category, defaultSym(trip.CurrencySymbol), formatMoneyShort(e.Amount), due)
				if strings.TrimSpace(e.Title) != "" {
					body = fmt.Sprintf("%s — %s", e.Title, body)
				}
				for _, uid := range recipients {
					_ = s.tryNotify(ctx, uid, tid, title, body, hrefBase+"/expenses", "payment_due", dedupe, now)
				}
			}
		}

		for _, c := range checklist {
			if c.Done {
				continue
			}
			due := strings.TrimSpace(c.DueAt)
			if due == "" {
				continue
			}
			dueDay, err := time.ParseInLocation("2006-01-02", due, time.Local)
			if err != nil {
				continue
			}
			cat := strings.TrimSpace(c.Category)
			if cat != "Travel Documents" && !strings.Contains(strings.ToLower(c.Text), "visa") && !strings.Contains(strings.ToLower(cat), "document") {
				continue
			}
			for _, days := range []int{7, 2, 0} {
				d := dueDay.AddDate(0, 0, -days)
				fire := time.Date(d.Year(), d.Month(), d.Day(), 9, 0, 0, 0, time.Local)
				if !reminderFireEligible(fire, endOfLocalDay(dueDay), now) {
					continue
				}
				dedupe := fmt.Sprintf("checklist:%s:due:%s:pre%d", c.ID, due, days)
				title := "Travel document deadline"
				body := c.Text + " — due " + due
				for _, uid := range recipients {
					_ = s.tryNotify(ctx, uid, tid, title, body, hrefBase+"/notes?view=reminders", "visa_doc", dedupe, now)
				}
			}
		}

		byItem := map[string][]ItineraryCustomReminder{}
		for _, cr := range customReminders {
			byItem[cr.ItineraryItemID] = append(byItem[cr.ItineraryItemID], cr)
		}
		for _, it := range items {
			start, ok := ItineraryItemStartLocal(trip, it)
			if !ok {
				continue
			}
			for _, cr := range byItem[it.ID] {
				if cr.MinutesBeforeStart < 0 {
					continue
				}
				fire := start.Add(-time.Duration(cr.MinutesBeforeStart) * time.Minute)
				if !reminderFireEligible(fire, start.Add(2*time.Hour), now) {
					continue
				}
				dedupe := fmt.Sprintf("itin:%s:custom:%d:%s", it.ID, cr.MinutesBeforeStart, strings.TrimSpace(cr.Label))
				title := "Itinerary reminder"
				if strings.TrimSpace(cr.Label) != "" {
					title = cr.Label
				}
				body := fmt.Sprintf("%s — %s at %s", it.Title, it.Location, start.Format("Mon Jan 2, 15:04"))
				for _, uid := range recipients {
					_ = s.tryNotify(ctx, uid, tid, title, body, hrefBase, "custom_itinerary", dedupe, now)
				}
			}
		}
	}
	return nil
}

func defaultSym(sym string) string {
	if strings.TrimSpace(sym) == "" {
		return "$"
	}
	return sym
}

func formatMoneyShort(f float64) string {
	return fmt.Sprintf("%.2f", f)
}

// StartReminderScheduler runs RunReminderTick every minute until ctx is cancelled.
func (s *Service) StartReminderScheduler(ctx context.Context) {
	go func() {
		t := time.NewTicker(time.Minute)
		defer t.Stop()
		// Run once shortly after startup
		_ = s.RunReminderTick(context.Background(), time.Now())
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				_ = s.RunReminderTick(context.Background(), time.Now())
			}
		}
	}()
}

func (s *Service) CountUnreadNotifications(ctx context.Context, userID string) (int, error) {
	if strings.TrimSpace(userID) == "" {
		return 0, nil
	}
	raw, err := s.repo.ListUnreadAppNotifications(ctx, userID, 2000)
	if err != nil {
		return 0, err
	}
	visible, err := s.filterAppNotificationsForEligibleTrips(ctx, raw, time.Now())
	if err != nil {
		return 0, err
	}
	return len(visible), nil
}

func (s *Service) ListNotificationsForUser(ctx context.Context, userID string, limit int) ([]AppNotification, error) {
	if limit <= 0 {
		limit = 50
	}
	fetch := limit * 5
	if fetch < 150 {
		fetch = 150
	}
	if fetch > 500 {
		fetch = 500
	}
	raw, err := s.repo.ListUnreadAppNotifications(ctx, userID, fetch)
	if err != nil {
		return nil, err
	}
	visible, err := s.filterAppNotificationsForEligibleTrips(ctx, raw, time.Now())
	if err != nil {
		return nil, err
	}
	if len(visible) > limit {
		visible = visible[:limit]
	}
	return visible, nil
}

func (s *Service) filterAppNotificationsForEligibleTrips(ctx context.Context, list []AppNotification, now time.Time) ([]AppNotification, error) {
	if len(list) == 0 {
		return nil, nil
	}
	seen := make(map[string]struct{}, len(list))
	for _, n := range list {
		if tid := strings.TrimSpace(n.TripID); tid != "" {
			seen[tid] = struct{}{}
		}
	}
	tripByID := make(map[string]Trip, len(seen))
	for tid := range seen {
		t, err := s.repo.GetTrip(ctx, tid)
		if err != nil {
			continue
		}
		tripByID[tid] = t
	}
	var out []AppNotification
	for _, n := range list {
		t, ok := tripByID[n.TripID]
		if !ok || !TripEligibleForInAppNotifications(t, now) {
			continue
		}
		n2 := n
		n2.Title = AppNotificationTitleWithTrip(t.Name, n.Title)
		out = append(out, n2)
	}
	return out, nil
}

func (s *Service) MarkNotificationRead(ctx context.Context, userID, notificationID string) error {
	return s.repo.MarkAppNotificationRead(ctx, userID, notificationID)
}

func (s *Service) MarkAllNotificationsRead(ctx context.Context, userID string) error {
	return s.repo.MarkAllAppNotificationsRead(ctx, userID)
}

func (s *Service) ReplaceItineraryCustomReminders(ctx context.Context, tripID, itemID string, rows []ItineraryCustomReminder) error {
	return s.repo.ReplaceItineraryItemCustomReminders(ctx, tripID, itemID, rows)
}

func (s *Service) ListItineraryCustomRemindersForTrip(ctx context.Context, tripID string) ([]ItineraryCustomReminder, error) {
	return s.repo.ListItineraryCustomRemindersByTrip(ctx, tripID)
}
