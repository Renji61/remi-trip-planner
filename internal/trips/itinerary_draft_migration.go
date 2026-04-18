package trips

import (
	"context"
	"strings"
	"time"
)

func joinDayLabelParts(parts []string) string {
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return strings.Join(out, " · ")
}

// migrateDraftItineraryDayNumbersToTripWindow converts YYYYMMDD-encoded itinerary day_number
// values to 1-based indices relative to the trip start date. Day labels keyed by draft numbers
// are remapped the same way. Calendar dates outside the trip window are clamped to the
// nearest in-range day.
func (s *Service) migrateDraftItineraryDayNumbersToTripWindow(ctx context.Context, tripID, startDate, endDate string) error {
	startDate = strings.TrimSpace(startDate)
	endDate = strings.TrimSpace(endDate)
	items, err := s.repo.ListItineraryItems(ctx, tripID)
	if err != nil {
		return err
	}
	labels, err := s.repo.GetTripDayLabels(ctx, tripID)
	if err != nil {
		return err
	}

	targetDay := func(draftDay int) (int, bool, error) {
		iso, ok := CalendarDateFromDraftItineraryDayNumber(draftDay)
		if !ok {
			return 0, false, nil
		}
		cal, err := time.ParseInLocation("2006-01-02", iso, time.Local)
		if err != nil {
			return 0, false, nil
		}
		n, err := RelativeDayNumberInTripWindow(startDate, endDate, cal)
		if err != nil {
			return 0, false, err
		}
		return n, true, nil
	}

	for _, it := range items {
		newDay, ok, err := targetDay(it.DayNumber)
		if err != nil {
			return err
		}
		if !ok || newDay == it.DayNumber {
			continue
		}
		it.DayNumber = newDay
		it.EnforceOptimisticLock = false
		if err := s.UpdateItineraryItem(ctx, it); err != nil {
			return err
		}
	}

	labelParts := make(map[int][]string)
	for oldK, lab := range labels {
		lab = strings.TrimSpace(lab)
		if lab == "" {
			continue
		}
		if newK, ok, err := targetDay(oldK); err != nil {
			return err
		} else if ok {
			labelParts[newK] = append(labelParts[newK], lab)
		} else {
			labelParts[oldK] = append(labelParts[oldK], lab)
		}
	}

	for oldK := range labels {
		if err := s.repo.SaveTripDayLabel(ctx, tripID, oldK, ""); err != nil {
			return err
		}
	}
	for k, parts := range labelParts {
		j := joinDayLabelParts(parts)
		if j == "" {
			continue
		}
		if err := s.repo.SaveTripDayLabel(ctx, tripID, k, j); err != nil {
			return err
		}
	}
	return nil
}
