package trips

import (
	"context"
	"errors"
	"time"
)

type Trip struct {
	ID          string
	Name        string
	Description string
	StartDate   string
	EndDate     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type ItineraryItem struct {
	ID          string
	TripID      string
	DayNumber   int
	OrderIndex  int
	Title       string
	Notes       string
	Location    string
	Latitude    float64
	Longitude   float64
	EstCost     float64
	StartTime   string
	EndTime     string
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Expense struct {
	ID        string
	TripID    string
	Category  string
	Amount    float64
	Notes     string
	SpentOn   string
	CreatedAt time.Time
}

type ChecklistItem struct {
	ID        string
	TripID    string
	Text      string
	Done      bool
	CreatedAt time.Time
}

type Change struct {
	ID        int64
	TripID    string
	Entity    string
	EntityID  string
	Operation string
	ChangedAt time.Time
	Payload   string
}

type TripDetails struct {
	Trip      Trip
	Itinerary []ItineraryItem
	Expenses  []Expense
	Checklist []ChecklistItem
}

type Repository interface {
	CreateTrip(ctx context.Context, t Trip) error
	ListTrips(ctx context.Context) ([]Trip, error)
	GetTrip(ctx context.Context, tripID string) (Trip, error)
	AddItineraryItem(ctx context.Context, item ItineraryItem) error
	ListItineraryItems(ctx context.Context, tripID string) ([]ItineraryItem, error)
	AddExpense(ctx context.Context, expense Expense) error
	ListExpenses(ctx context.Context, tripID string) ([]Expense, error)
	AddChecklistItem(ctx context.Context, item ChecklistItem) error
	ToggleChecklistItem(ctx context.Context, itemID string, done bool) error
	ListChecklistItems(ctx context.Context, tripID string) ([]ChecklistItem, error)
	ListChanges(ctx context.Context, tripID, since string) ([]Change, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) CreateTrip(ctx context.Context, t Trip) error {
	if t.Name == "" {
		return errors.New("trip name is required")
	}
	return s.repo.CreateTrip(ctx, t)
}

func (s *Service) ListTrips(ctx context.Context) ([]Trip, error) {
	return s.repo.ListTrips(ctx)
}

func (s *Service) GetTripDetails(ctx context.Context, tripID string) (TripDetails, error) {
	trip, err := s.repo.GetTrip(ctx, tripID)
	if err != nil {
		return TripDetails{}, err
	}
	itinerary, err := s.repo.ListItineraryItems(ctx, tripID)
	if err != nil {
		return TripDetails{}, err
	}
	expenses, err := s.repo.ListExpenses(ctx, tripID)
	if err != nil {
		return TripDetails{}, err
	}
	checklist, err := s.repo.ListChecklistItems(ctx, tripID)
	if err != nil {
		return TripDetails{}, err
	}
	return TripDetails{
		Trip:      trip,
		Itinerary: itinerary,
		Expenses:  expenses,
		Checklist: checklist,
	}, nil
}

func (s *Service) AddItineraryItem(ctx context.Context, item ItineraryItem) error {
	if item.TripID == "" || item.Title == "" {
		return errors.New("trip and title are required")
	}
	if item.DayNumber < 1 {
		item.DayNumber = 1
	}
	return s.repo.AddItineraryItem(ctx, item)
}

func (s *Service) AddExpense(ctx context.Context, expense Expense) error {
	if expense.TripID == "" || expense.Amount < 0 {
		return errors.New("invalid expense")
	}
	return s.repo.AddExpense(ctx, expense)
}

func (s *Service) AddChecklistItem(ctx context.Context, item ChecklistItem) error {
	if item.TripID == "" || item.Text == "" {
		return errors.New("invalid checklist item")
	}
	return s.repo.AddChecklistItem(ctx, item)
}

func (s *Service) ToggleChecklistItem(ctx context.Context, itemID string, done bool) error {
	return s.repo.ToggleChecklistItem(ctx, itemID, done)
}

func (s *Service) ListChanges(ctx context.Context, tripID, since string) ([]Change, error) {
	return s.repo.ListChanges(ctx, tripID, since)
}
