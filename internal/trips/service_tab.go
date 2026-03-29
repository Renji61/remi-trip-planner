package trips

import (
	"context"
	"errors"
	"strings"
)

func userProfileDisplayName(u User) string {
	p := UserProfile{ID: u.ID, Email: u.Email, Username: u.Username, DisplayName: u.DisplayName}
	return p.PublicDisplayName()
}

func (s *Service) ListTripGuests(ctx context.Context, tripID string) ([]TripGuest, error) {
	return s.repo.ListTripGuests(ctx, tripID)
}

func (s *Service) AddTripGuest(ctx context.Context, g TripGuest) error {
	if strings.TrimSpace(g.TripID) == "" || strings.TrimSpace(g.DisplayName) == "" {
		return errors.New("invalid guest")
	}
	trip, err := s.repo.GetTrip(ctx, g.TripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	g.DisplayName = strings.TrimSpace(g.DisplayName)
	return s.repo.AddTripGuest(ctx, g)
}

func (s *Service) GetTripGuest(ctx context.Context, tripID, guestID string) (TripGuest, error) {
	return s.repo.GetTripGuest(ctx, tripID, guestID)
}

func (s *Service) ListDepartedTabParticipants(ctx context.Context, tripID string) ([]DepartedTabParticipant, error) {
	return s.repo.ListDepartedTabParticipants(ctx, tripID)
}

func (s *Service) DeleteTripGuest(ctx context.Context, tripID, guestID string) error {
	trip, err := s.repo.GetTrip(ctx, tripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	g, gerr := s.repo.GetTripGuest(ctx, tripID, guestID)
	if gerr == nil {
		_ = s.repo.UpsertDepartedTabParticipant(ctx, tripID, ParticipantKeyGuest(guestID), strings.TrimSpace(g.DisplayName))
	}
	return s.repo.DeleteTripGuest(ctx, tripID, guestID)
}

func (s *Service) ListTabSettlements(ctx context.Context, tripID string) ([]TabSettlement, error) {
	return s.repo.ListTabSettlements(ctx, tripID)
}

func (s *Service) prepareTabSettlementParties(ctx context.Context, tripID string, st *TabSettlement) error {
	pk := TabSettlementParticipantKey(st.PayerUserID)
	qk := TabSettlementParticipantKey(st.PayeeUserID)
	if pk == "" || qk == "" {
		return errors.New("choose payer and payee")
	}
	if pk == qk {
		return errors.New("payer and payee must differ")
	}
	party, err := s.TripParty(ctx, tripID)
	if err != nil {
		return err
	}
	guests, err := s.ListTripGuests(ctx, tripID)
	if err != nil {
		return err
	}
	allowed := make(map[string]struct{}, len(party)+len(guests))
	for _, p := range party {
		allowed[ParticipantKeyUser(p.ID)] = struct{}{}
	}
	for _, g := range guests {
		allowed[ParticipantKeyGuest(g.ID)] = struct{}{}
	}
	departed, err := s.repo.ListDepartedTabParticipants(ctx, tripID)
	if err != nil {
		return err
	}
	for _, d := range departed {
		if k := strings.TrimSpace(d.ParticipantKey); k != "" {
			allowed[k] = struct{}{}
		}
	}
	if _, ok := allowed[pk]; !ok {
		return errors.New("payer is not on this trip")
	}
	if _, ok := allowed[qk]; !ok {
		return errors.New("payee is not on this trip")
	}
	st.PayerUserID, st.PayeeUserID = pk, qk
	return nil
}

func (s *Service) AddTabSettlement(ctx context.Context, st TabSettlement) error {
	if strings.TrimSpace(st.TripID) == "" || st.Amount <= 0 {
		return errors.New("invalid settlement")
	}
	trip, err := s.repo.GetTrip(ctx, st.TripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	if err := s.prepareTabSettlementParties(ctx, st.TripID, &st); err != nil {
		return err
	}
	if strings.TrimSpace(st.Method) == "" {
		st.Method = "Cash"
	}
	return s.repo.AddTabSettlement(ctx, st)
}

func (s *Service) UpdateTabSettlement(ctx context.Context, st TabSettlement) error {
	if strings.TrimSpace(st.ID) == "" || strings.TrimSpace(st.TripID) == "" || st.Amount <= 0 {
		return errors.New("invalid settlement")
	}
	trip, err := s.repo.GetTrip(ctx, st.TripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	if err := s.prepareTabSettlementParties(ctx, st.TripID, &st); err != nil {
		return err
	}
	if strings.TrimSpace(st.Method) == "" {
		st.Method = "Cash"
	}
	return s.repo.UpdateTabSettlement(ctx, st)
}

func (s *Service) DeleteTabSettlement(ctx context.Context, tripID, settlementID string) error {
	if strings.TrimSpace(tripID) == "" || strings.TrimSpace(settlementID) == "" {
		return errors.New("invalid settlement")
	}
	trip, err := s.repo.GetTrip(ctx, tripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	return s.repo.DeleteTabSettlement(ctx, tripID, settlementID)
}

func (s *Service) SearchTabExpenseIDs(ctx context.Context, tripID, query string) ([]string, error) {
	return s.repo.SearchTabExpenseIDs(ctx, tripID, query)
}

func (s *Service) SaveTripTabDefaults(ctx context.Context, tripID, mode, splitJSON string) error {
	trip, err := s.repo.GetTrip(ctx, tripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	return s.repo.UpdateTripTabDefaults(ctx, tripID, mode, splitJSON)
}
