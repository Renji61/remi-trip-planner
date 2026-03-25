package trips

import (
	"context"
	"database/sql"
	"errors"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

type Trip struct {
	ID             string
	Name           string
	Description    string
	StartDate      string
	EndDate        string
	CoverImage     string
	CurrencyName   string
	CurrencySymbol string
	IsArchived     bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type ItineraryItem struct {
	ID        string
	TripID    string
	DayNumber int
	Title     string
	Notes     string
	Location  string
	Latitude  float64
	Longitude float64
	EstCost   float64
	StartTime string
	EndTime   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Expense struct {
	ID        string
	TripID    string
	Category  string
	Amount    float64
	Notes     string
	SpentOn   string
	PaymentMethod string
	LodgingID string
	CreatedAt time.Time
}

// ExpenseCategoryAccommodation is used for accommodation-synced expenses and matches the Accommodation quick-expense option.
const ExpenseCategoryAccommodation = "Accommodation"

// QuickExpenseCategories are standard manual expense categories (Quick Expense and edit dropdowns), sorted A–Z.
var QuickExpenseCategories = []string{
	"Airfare",
	"Car Rental",
	ExpenseCategoryAccommodation,
	"Transportation",
	"Food & Dining",
	"Groceries",
	"Activities",
	"Shopping",
	"Miscellaneous",
	"Visa & Documentation",
	"Insurance",
	"Parking & Toll",
	"Fuel",
	"Connectivity",
	"Tips & Gratuities",
}

func init() {
	sort.Strings(QuickExpenseCategories)
}

const (
	itineraryAccommodationCheckInPrefix  = "Accommodation check-in: "
	itineraryAccommodationCheckOutPrefix = "Accommodation check-out: "
)

const (
	legacyItineraryCheckInPrefix  = "Hotel Check-in: "
	legacyItineraryCheckOutPrefix = "Hotel Check-out: "
)

// AccommodationItineraryCheckInTitle is the itinerary row title for an accommodation check-in stop.
func AccommodationItineraryCheckInTitle(propertyName string) string {
	return itineraryAccommodationCheckInPrefix + propertyName
}

// AccommodationItineraryCheckOutTitle is the itinerary row title for an accommodation check-out stop.
func AccommodationItineraryCheckOutTitle(propertyName string) string {
	return itineraryAccommodationCheckOutPrefix + propertyName
}

func addAccommodationItineraryTitleKeys(m map[string]struct{}, name string) {
	if name == "" {
		return
	}
	m[itineraryAccommodationCheckInPrefix+name] = struct{}{}
	m[itineraryAccommodationCheckOutPrefix+name] = struct{}{}
	m[legacyItineraryCheckInPrefix+name] = struct{}{}
	m[legacyItineraryCheckOutPrefix+name] = struct{}{}
}

func accommodationNameFromItineraryTitle(title string) (name string, ok bool) {
	switch {
	case strings.HasPrefix(title, itineraryAccommodationCheckInPrefix):
		return strings.TrimPrefix(title, itineraryAccommodationCheckInPrefix), true
	case strings.HasPrefix(title, itineraryAccommodationCheckOutPrefix):
		return strings.TrimPrefix(title, itineraryAccommodationCheckOutPrefix), true
	case strings.HasPrefix(title, legacyItineraryCheckInPrefix):
		return strings.TrimPrefix(title, legacyItineraryCheckInPrefix), true
	case strings.HasPrefix(title, legacyItineraryCheckOutPrefix):
		return strings.TrimPrefix(title, legacyItineraryCheckOutPrefix), true
	default:
		return "", false
	}
}

type ChecklistItem struct {
	ID        string
	TripID    string
	Category  string
	Text      string
	Done      bool
	CreatedAt time.Time
}

// ReminderChecklistCategories are the available categories for Add to Checklist.
var ReminderChecklistCategories = []string{
	"Travel Documents",
	"Health & Safety",
	"Bookings & Reservations",
	"Packing List",
	"Home Preparation",
	"Finance & Budgeting",
	"Navigation & Maps",
	"Electronics & Connectivity",
	"Itinerary Planning",
	"Transit & Arrival",
}

type Lodging struct {
	ID                  string
	TripID              string
	Name                string
	Address             string
	CheckInAt           string
	CheckOutAt          string
	BookingConfirmation string
	Cost                float64
	Notes               string
	AttachmentPath      string
	CheckInItineraryID  string
	CheckOutItineraryID string
	CreatedAt           time.Time
}

type VehicleRental struct {
	ID                  string
	TripID              string
	PickUpLocation      string
	VehicleDetail       string
	PickUpAt            string
	DropOffAt           string
	BookingConfirmation string
	Notes               string
	VehicleImagePath    string
	Cost                float64
	InsuranceCost       float64
	PayAtPickUp         bool
	PickUpItineraryID   string
	DropOffItineraryID  string
	RentalExpenseID     string
	InsuranceExpenseID  string
	CreatedAt           time.Time
}

type Flight struct {
	ID                  string
	TripID              string
	FlightName          string
	FlightNumber        string
	DepartAirport       string
	ArriveAirport       string
	DepartAt            string
	ArriveAt            string
	BookingConfirmation string
	Notes               string
	DocumentPath        string
	Cost                float64
	DepartItineraryID   string
	ArriveItineraryID   string
	ExpenseID           string
	CreatedAt           time.Time
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

type AppSettings struct {
	AppTitle              string
	DefaultCurrencyName   string
	DefaultCurrencySymbol string
	MapDefaultLatitude    float64
	MapDefaultLongitude   float64
	MapDefaultZoom        int
	EnableLocationLookup  bool
}

type TripDetails struct {
	Trip      Trip
	Itinerary []ItineraryItem
	Expenses  []Expense
	Checklist []ChecklistItem
	Lodgings  []Lodging
	Vehicles  []VehicleRental
	Flights   []Flight
}

type Repository interface {
	CreateTrip(ctx context.Context, t Trip) error
	ListTrips(ctx context.Context) ([]Trip, error)
	GetTrip(ctx context.Context, tripID string) (Trip, error)
	UpdateTrip(ctx context.Context, t Trip) error
	ArchiveTrip(ctx context.Context, tripID string) error
	DeleteTrip(ctx context.Context, tripID string) error
	AddItineraryItem(ctx context.Context, item ItineraryItem) error
	UpdateItineraryItem(ctx context.Context, item ItineraryItem) (rowsAffected int64, err error)
	DeleteItineraryItem(ctx context.Context, tripID, itemID string) error
	ListItineraryItems(ctx context.Context, tripID string) ([]ItineraryItem, error)
	AddExpense(ctx context.Context, expense Expense) error
	UpdateExpense(ctx context.Context, expense Expense) error
	DeleteExpense(ctx context.Context, tripID, expenseID string) error
	GetExpense(ctx context.Context, tripID, expenseID string) (Expense, error)
	GetExpenseByLodgingID(ctx context.Context, tripID, lodgingID string) (Expense, error)
	DeleteExpenseByLodgingID(ctx context.Context, tripID, lodgingID string) error
	ListExpenses(ctx context.Context, tripID string) ([]Expense, error)
	AddChecklistItem(ctx context.Context, item ChecklistItem) error
	GetChecklistItem(ctx context.Context, itemID string) (ChecklistItem, error)
	UpdateChecklistItem(ctx context.Context, item ChecklistItem) error
	DeleteChecklistItem(ctx context.Context, itemID string) error
	ToggleChecklistItem(ctx context.Context, itemID string, done bool) error
	ListChecklistItems(ctx context.Context, tripID string) ([]ChecklistItem, error)
	AddLodging(ctx context.Context, item Lodging) error
	UpdateLodging(ctx context.Context, item Lodging) error
	DeleteLodging(ctx context.Context, tripID, lodgingID string) error
	ListLodgings(ctx context.Context, tripID string) ([]Lodging, error)
	GetLodging(ctx context.Context, tripID, lodgingID string) (Lodging, error)
	AddVehicleRental(ctx context.Context, item VehicleRental) error
	UpdateVehicleRental(ctx context.Context, item VehicleRental) error
	DeleteVehicleRental(ctx context.Context, tripID, rentalID string) error
	ListVehicleRentals(ctx context.Context, tripID string) ([]VehicleRental, error)
	GetVehicleRental(ctx context.Context, tripID, rentalID string) (VehicleRental, error)
	AddFlight(ctx context.Context, item Flight) error
	UpdateFlight(ctx context.Context, item Flight) error
	DeleteFlight(ctx context.Context, tripID, flightID string) error
	ListFlights(ctx context.Context, tripID string) ([]Flight, error)
	GetFlight(ctx context.Context, tripID, flightID string) (Flight, error)
	ListChanges(ctx context.Context, tripID, since string) ([]Change, error)
	GetAppSettings(ctx context.Context) (AppSettings, error)
	SaveAppSettings(ctx context.Context, settings AppSettings) error
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

func (s *Service) GetTrip(ctx context.Context, tripID string) (Trip, error) {
	if tripID == "" {
		return Trip{}, errors.New("trip id is required")
	}
	return s.repo.GetTrip(ctx, tripID)
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
	lodgings, err := s.repo.ListLodgings(ctx, tripID)
	if err != nil {
		return TripDetails{}, err
	}
	vehicles, err := s.repo.ListVehicleRentals(ctx, tripID)
	if err != nil {
		return TripDetails{}, err
	}
	flights, err := s.repo.ListFlights(ctx, tripID)
	if err != nil {
		return TripDetails{}, err
	}
	return TripDetails{
		Trip:      trip,
		Itinerary: itinerary,
		Expenses:  expenses,
		Checklist: checklist,
		Lodgings:  lodgings,
		Vehicles:  vehicles,
		Flights:   flights,
	}, nil
}

func (s *Service) UpdateTrip(ctx context.Context, t Trip) error {
	if t.ID == "" || t.Name == "" {
		return errors.New("trip id and name are required")
	}
	current, err := s.repo.GetTrip(ctx, t.ID)
	if err != nil {
		return err
	}
	if current.IsArchived {
		return errors.New("archived trips are read-only")
	}
	return s.repo.UpdateTrip(ctx, t)
}

func (s *Service) ArchiveTrip(ctx context.Context, tripID string) error {
	if tripID == "" {
		return errors.New("trip id is required")
	}
	return s.repo.ArchiveTrip(ctx, tripID)
}

func (s *Service) DeleteTrip(ctx context.Context, tripID string) error {
	if tripID == "" {
		return errors.New("trip id is required")
	}
	return s.repo.DeleteTrip(ctx, tripID)
}

func (s *Service) AddItineraryItem(ctx context.Context, item ItineraryItem) error {
	if item.TripID == "" || item.Title == "" {
		return errors.New("trip and title are required")
	}
	trip, err := s.repo.GetTrip(ctx, item.TripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	if item.DayNumber < 1 {
		item.DayNumber = 1
	}
	return s.repo.AddItineraryItem(ctx, item)
}

// SyncLodgingItinerary keeps itinerary check-in/out rows in sync with a lodging entry: it removes
// stray duplicate hotel lines, recreates the pair when IDs are missing or stale, and returns
// lodging with CheckInItineraryID / CheckOutItineraryID set for persistence.
func (s *Service) SyncLodgingItinerary(ctx context.Context, trip Trip, lodging Lodging, previousName string,
	checkInDay int, checkInTime string, checkOutDay int, checkOutTime string,
	checkInNotes, checkOutNotes string,
) (Lodging, error) {
	if lodging.TripID == "" || lodging.Name == "" {
		return lodging, errors.New("invalid accommodation entry")
	}
	if trip.IsArchived {
		return lodging, errors.New("archived trips are read-only")
	}
	items, err := s.repo.ListItineraryItems(ctx, lodging.TripID)
	if err != nil {
		return lodging, err
	}
	exists := make(map[string]struct{}, len(items))
	for _, it := range items {
		exists[it.ID] = struct{}{}
	}

	ciID := lodging.CheckInItineraryID
	coID := lodging.CheckOutItineraryID
	if ciID == "" || coID == "" {
		li, lo := findLodgingItineraryPairFromItems(items, lodging.TripID, previousName)
		if ciID == "" {
			ciID = li
		}
		if coID == "" {
			coID = lo
		}
	}
	if ciID == "" || coID == "" {
		li, lo := findLodgingItineraryPairFromItems(items, lodging.TripID, lodging.Name)
		if ciID == "" {
			ciID = li
		}
		if coID == "" {
			coID = lo
		}
	}
	if ciID != "" {
		if _, ok := exists[ciID]; !ok {
			ciID = ""
		}
	}
	if coID != "" {
		if _, ok := exists[coID]; !ok {
			coID = ""
		}
	}

	titlesToClean := map[string]struct{}{}
	addName := func(n string) { addAccommodationItineraryTitleKeys(titlesToClean, n) }
	addName(previousName)
	addName(lodging.Name)

	keep := map[string]struct{}{}
	if ciID != "" {
		keep[ciID] = struct{}{}
	}
	if coID != "" {
		keep[coID] = struct{}{}
	}
	for _, it := range items {
		if _, match := titlesToClean[it.Title]; !match {
			continue
		}
		if _, protected := keep[it.ID]; protected {
			continue
		}
		_ = s.repo.DeleteItineraryItem(ctx, lodging.TripID, it.ID)
	}

	checkInItem := ItineraryItem{
		ID:        ciID,
		TripID:    lodging.TripID,
		DayNumber: checkInDay,
		Title:     AccommodationItineraryCheckInTitle(lodging.Name),
		Location:  lodging.Address,
		Notes:     checkInNotes,
		EstCost:   lodging.Cost,
		StartTime: checkInTime,
		EndTime:   checkInTime,
	}
	checkOutItem := ItineraryItem{
		ID:        coID,
		TripID:    lodging.TripID,
		DayNumber: checkOutDay,
		Title:     AccommodationItineraryCheckOutTitle(lodging.Name),
		Location:  lodging.Address,
		Notes:     checkOutNotes,
		EstCost:   lodging.Cost,
		StartTime: checkOutTime,
		EndTime:   checkOutTime,
	}

	if ciID != "" && coID != "" {
		n1, err1 := s.repo.UpdateItineraryItem(ctx, checkInItem)
		if err1 != nil {
			return lodging, err1
		}
		n2, err2 := s.repo.UpdateItineraryItem(ctx, checkOutItem)
		if err2 != nil {
			return lodging, err2
		}
		if n1 > 0 && n2 > 0 {
			lodging.CheckInItineraryID = ciID
			lodging.CheckOutItineraryID = coID
			return lodging, nil
		}
	}

	if ciID != "" {
		_ = s.repo.DeleteItineraryItem(ctx, lodging.TripID, ciID)
	}
	if coID != "" {
		_ = s.repo.DeleteItineraryItem(ctx, lodging.TripID, coID)
	}

	newCI := uuid.NewString()
	newCO := uuid.NewString()
	newCheckIn := ItineraryItem{
		ID: newCI, TripID: lodging.TripID, DayNumber: checkInDay,
		Title: AccommodationItineraryCheckInTitle(lodging.Name), Location: lodging.Address,
		Notes: checkInNotes, EstCost: lodging.Cost,
		StartTime: checkInTime, EndTime: checkInTime,
	}
	newCheckOut := ItineraryItem{
		ID: newCO, TripID: lodging.TripID, DayNumber: checkOutDay,
		Title: AccommodationItineraryCheckOutTitle(lodging.Name), Location: lodging.Address,
		Notes: checkOutNotes, EstCost: lodging.Cost,
		StartTime: checkOutTime, EndTime: checkOutTime,
	}
	if err := s.repo.AddItineraryItem(ctx, newCheckIn); err != nil {
		return lodging, err
	}
	if err := s.repo.AddItineraryItem(ctx, newCheckOut); err != nil {
		return lodging, err
	}
	lodging.CheckInItineraryID = newCI
	lodging.CheckOutItineraryID = newCO
	return lodging, nil
}

func findLodgingItineraryPairFromItems(items []ItineraryItem, tripID, lodgingName string) (checkInID, checkOutID string) {
	if lodgingName == "" {
		return "", ""
	}
	inTitles := map[string]struct{}{
		itineraryAccommodationCheckInPrefix + lodgingName: {},
		legacyItineraryCheckInPrefix + lodgingName:        {},
	}
	outTitles := map[string]struct{}{
		itineraryAccommodationCheckOutPrefix + lodgingName: {},
		legacyItineraryCheckOutPrefix + lodgingName:        {},
	}
	for _, it := range items {
		if it.TripID != tripID {
			continue
		}
		if _, ok := inTitles[it.Title]; ok && checkInID == "" {
			checkInID = it.ID
		}
		if _, ok := outTitles[it.Title]; ok && checkOutID == "" {
			checkOutID = it.ID
		}
	}
	return checkInID, checkOutID
}

func (s *Service) AddExpense(ctx context.Context, expense Expense) error {
	if expense.TripID == "" || expense.Amount < 0 {
		return errors.New("invalid expense")
	}
	trip, err := s.repo.GetTrip(ctx, expense.TripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	return s.repo.AddExpense(ctx, expense)
}

func (s *Service) UpdateItineraryItem(ctx context.Context, item ItineraryItem) error {
	if item.TripID == "" || item.ID == "" || item.Title == "" {
		return errors.New("invalid itinerary item")
	}
	trip, err := s.repo.GetTrip(ctx, item.TripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	n, err := s.repo.UpdateItineraryItem(ctx, item)
	if err != nil {
		return err
	}
	if n == 0 {
		return errors.New("itinerary item not found")
	}
	return nil
}

func (s *Service) DeleteItineraryItem(ctx context.Context, tripID, itemID string) error {
	if tripID == "" || itemID == "" {
		return errors.New("invalid itinerary item")
	}
	trip, err := s.repo.GetTrip(ctx, tripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	if err := s.ClearLodgingItineraryRefs(ctx, tripID, itemID); err != nil {
		return err
	}
	return s.repo.DeleteItineraryItem(ctx, tripID, itemID)
}

// ClearLodgingItineraryRefs clears check-in/out itinerary id fields on lodging rows that point at itemID.
func (s *Service) ClearLodgingItineraryRefs(ctx context.Context, tripID, itemID string) error {
	lodgings, err := s.repo.ListLodgings(ctx, tripID)
	if err != nil {
		return err
	}
	for _, l := range lodgings {
		changed := false
		updated := l
		if l.CheckInItineraryID == itemID {
			updated.CheckInItineraryID = ""
			changed = true
		}
		if l.CheckOutItineraryID == itemID {
			updated.CheckOutItineraryID = ""
			changed = true
		}
		if changed {
			if err := s.repo.UpdateLodging(ctx, updated); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *Service) UpdateExpense(ctx context.Context, expense Expense) error {
	if expense.TripID == "" || expense.ID == "" || expense.Amount < 0 {
		return errors.New("invalid expense")
	}
	prev, err := s.repo.GetExpense(ctx, expense.TripID, expense.ID)
	if err != nil {
		return err
	}
	if prev.LodgingID != "" {
		return errors.New("this expense is tied to Accommodation — change the booking there")
	}
	vehicles, err := s.repo.ListVehicleRentals(ctx, expense.TripID)
	if err != nil {
		return err
	}
	for _, v := range vehicles {
		if v.RentalExpenseID == prev.ID || v.InsuranceExpenseID == prev.ID {
			return errors.New("this expense is tied to Vehicle Rental — change the booking there")
		}
	}
	flights, err := s.repo.ListFlights(ctx, expense.TripID)
	if err != nil {
		return err
	}
	for _, f := range flights {
		if f.ExpenseID == prev.ID {
			return errors.New("this expense is tied to Flights — change the booking there")
		}
	}
	trip, err := s.repo.GetTrip(ctx, expense.TripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	expense.LodgingID = prev.LodgingID
	return s.repo.UpdateExpense(ctx, expense)
}

func (s *Service) DeleteExpense(ctx context.Context, tripID, expenseID string) error {
	if tripID == "" || expenseID == "" {
		return errors.New("invalid expense")
	}
	prev, err := s.repo.GetExpense(ctx, tripID, expenseID)
	if err != nil {
		return err
	}
	if prev.LodgingID != "" {
		return errors.New("remove the stay under Accommodation to delete this expense")
	}
	vehicles, err := s.repo.ListVehicleRentals(ctx, tripID)
	if err != nil {
		return err
	}
	for _, v := range vehicles {
		if v.RentalExpenseID == prev.ID || v.InsuranceExpenseID == prev.ID {
			return errors.New("remove this booking under Vehicle Rental to delete these expenses")
		}
	}
	flights, err := s.repo.ListFlights(ctx, tripID)
	if err != nil {
		return err
	}
	for _, f := range flights {
		if f.ExpenseID == prev.ID {
			return errors.New("remove this booking under Flights to delete this expense")
		}
	}
	trip, err := s.repo.GetTrip(ctx, tripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	return s.repo.DeleteExpense(ctx, tripID, expenseID)
}

func (s *Service) AddChecklistItem(ctx context.Context, item ChecklistItem) error {
	if item.TripID == "" || item.Text == "" {
		return errors.New("invalid checklist item")
	}
	if strings.TrimSpace(item.Category) == "" {
		item.Category = "Packing List"
	}
	trip, err := s.repo.GetTrip(ctx, item.TripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	return s.repo.AddChecklistItem(ctx, item)
}

func (s *Service) ToggleChecklistItem(ctx context.Context, itemID string, done bool) error {
	return s.repo.ToggleChecklistItem(ctx, itemID, done)
}

func (s *Service) UpdateChecklistItem(ctx context.Context, item ChecklistItem) error {
	if item.ID == "" || strings.TrimSpace(item.Text) == "" {
		return errors.New("invalid checklist item")
	}
	existing, err := s.repo.GetChecklistItem(ctx, item.ID)
	if err != nil {
		return err
	}
	trip, err := s.repo.GetTrip(ctx, existing.TripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	item.TripID = existing.TripID
	item.Text = strings.TrimSpace(item.Text)
	item.Category = strings.TrimSpace(item.Category)
	if item.Category == "" {
		item.Category = "Packing List"
	}
	item.Done = existing.Done
	item.CreatedAt = existing.CreatedAt
	return s.repo.UpdateChecklistItem(ctx, item)
}

func (s *Service) DeleteChecklistItem(ctx context.Context, itemID string) error {
	if strings.TrimSpace(itemID) == "" {
		return errors.New("invalid checklist item")
	}
	existing, err := s.repo.GetChecklistItem(ctx, itemID)
	if err != nil {
		return err
	}
	trip, err := s.repo.GetTrip(ctx, existing.TripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	return s.repo.DeleteChecklistItem(ctx, itemID)
}

func (s *Service) AddLodging(ctx context.Context, item Lodging) error {
	if item.TripID == "" || item.Name == "" {
		return errors.New("trip and accommodation name are required")
	}
	trip, err := s.repo.GetTrip(ctx, item.TripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	if err := s.repo.AddLodging(ctx, item); err != nil {
		return err
	}
	return s.SyncExpenseForLodging(ctx, item)
}

func (s *Service) UpdateLodging(ctx context.Context, item Lodging) error {
	if item.TripID == "" || item.ID == "" || item.Name == "" {
		return errors.New("invalid accommodation entry")
	}
	trip, err := s.repo.GetTrip(ctx, item.TripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	if err := s.repo.UpdateLodging(ctx, item); err != nil {
		return err
	}
	return s.SyncExpenseForLodging(ctx, item)
}

// SyncExpenseForLodging upserts an expense row tied to this accommodation booking (category Accommodation).
func (s *Service) SyncExpenseForLodging(ctx context.Context, l Lodging) error {
	if l.TripID == "" || l.ID == "" {
		return nil
	}
	trip, err := s.repo.GetTrip(ctx, l.TripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	existing, err := s.repo.GetExpenseByLodgingID(ctx, l.TripID, l.ID)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	notes := lodgingExpenseNotes(l)
	spentOn := lodgingExpenseSpentOn(l)
	if errors.Is(err, sql.ErrNoRows) {
		return s.repo.AddExpense(ctx, Expense{
			ID:        uuid.NewString(),
			TripID:    l.TripID,
			Category:  ExpenseCategoryAccommodation,
			Amount:    l.Cost,
			Notes:     notes,
			SpentOn:   spentOn,
			LodgingID: l.ID,
		})
	}
	existing.Category = ExpenseCategoryAccommodation
	existing.Amount = l.Cost
	existing.Notes = notes
	existing.SpentOn = spentOn
	existing.LodgingID = l.ID
	return s.repo.UpdateExpense(ctx, existing)
}

const (
	itineraryVehiclePickUpPrefix  = "Vehicle pick-up: "
	itineraryVehicleDropOffPrefix = "Vehicle drop-off: "
)

func VehicleRentalItineraryPickUpTitle(location string) string {
	return itineraryVehiclePickUpPrefix + location
}

func VehicleRentalItineraryDropOffTitle(location string) string {
	return itineraryVehicleDropOffPrefix + location
}

func vehicleLocationFromItineraryTitle(title string) (location string, ok bool) {
	switch {
	case strings.HasPrefix(title, itineraryVehiclePickUpPrefix):
		return strings.TrimPrefix(title, itineraryVehiclePickUpPrefix), true
	case strings.HasPrefix(title, itineraryVehicleDropOffPrefix):
		return strings.TrimPrefix(title, itineraryVehicleDropOffPrefix), true
	default:
		return "", false
	}
}

const (
	itineraryFlightDepartPrefix = "Flight departure: "
	itineraryFlightArrivePrefix = "Flight arrival: "
)

func FlightItineraryDepartTitle(flightLabel string) string {
	return itineraryFlightDepartPrefix + flightLabel
}

func FlightItineraryArriveTitle(flightLabel string) string {
	return itineraryFlightArrivePrefix + flightLabel
}

func flightLabelValue(flightName, flightNumber string) string {
	name := strings.TrimSpace(flightName)
	number := strings.TrimSpace(flightNumber)
	switch {
	case name != "" && number != "":
		return name + " (" + number + ")"
	case name != "":
		return name
	case number != "":
		return number
	default:
		return "Flight"
	}
}

func flightLabelFromItineraryTitle(title string) (label string, ok bool) {
	switch {
	case strings.HasPrefix(title, itineraryFlightDepartPrefix):
		return strings.TrimPrefix(title, itineraryFlightDepartPrefix), true
	case strings.HasPrefix(title, itineraryFlightArrivePrefix):
		return strings.TrimPrefix(title, itineraryFlightArrivePrefix), true
	default:
		return "", false
	}
}

func (s *Service) AddVehicleRental(ctx context.Context, item VehicleRental) error {
	if item.TripID == "" || item.PickUpLocation == "" {
		return errors.New("trip and pick up location are required")
	}
	trip, err := s.repo.GetTrip(ctx, item.TripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	if err := s.repo.AddVehicleRental(ctx, item); err != nil {
		return err
	}
	return s.SyncExpenseForVehicleRental(ctx, item)
}

func (s *Service) UpdateVehicleRental(ctx context.Context, item VehicleRental) error {
	if item.TripID == "" || item.ID == "" || item.PickUpLocation == "" {
		return errors.New("invalid vehicle rental entry")
	}
	trip, err := s.repo.GetTrip(ctx, item.TripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	if err := s.repo.UpdateVehicleRental(ctx, item); err != nil {
		return err
	}
	return s.SyncExpenseForVehicleRental(ctx, item)
}

func (s *Service) GetVehicleRental(ctx context.Context, tripID, rentalID string) (VehicleRental, error) {
	if tripID == "" || rentalID == "" {
		return VehicleRental{}, errors.New("invalid vehicle rental entry")
	}
	return s.repo.GetVehicleRental(ctx, tripID, rentalID)
}

func (s *Service) DeleteVehicleRental(ctx context.Context, tripID, rentalID string) error {
	if tripID == "" || rentalID == "" {
		return errors.New("invalid vehicle rental entry")
	}
	trip, err := s.repo.GetTrip(ctx, tripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	rental, err := s.repo.GetVehicleRental(ctx, tripID, rentalID)
	if err != nil {
		return err
	}
	if rental.PickUpItineraryID != "" {
		_ = s.repo.DeleteItineraryItem(ctx, tripID, rental.PickUpItineraryID)
	}
	if rental.DropOffItineraryID != "" {
		_ = s.repo.DeleteItineraryItem(ctx, tripID, rental.DropOffItineraryID)
	}
	if rental.RentalExpenseID != "" {
		_ = s.repo.DeleteExpense(ctx, tripID, rental.RentalExpenseID)
	}
	if rental.InsuranceExpenseID != "" {
		_ = s.repo.DeleteExpense(ctx, tripID, rental.InsuranceExpenseID)
	}
	return s.repo.DeleteVehicleRental(ctx, tripID, rentalID)
}

func (s *Service) AddFlight(ctx context.Context, item Flight) error {
	if item.TripID == "" || strings.TrimSpace(item.DepartAirport) == "" || strings.TrimSpace(item.ArriveAirport) == "" {
		return errors.New("trip and airport details are required")
	}
	trip, err := s.repo.GetTrip(ctx, item.TripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	if err := s.repo.AddFlight(ctx, item); err != nil {
		return err
	}
	return s.SyncExpenseForFlight(ctx, item)
}

func (s *Service) UpdateFlight(ctx context.Context, item Flight) error {
	if item.TripID == "" || item.ID == "" || strings.TrimSpace(item.DepartAirport) == "" || strings.TrimSpace(item.ArriveAirport) == "" {
		return errors.New("invalid flight entry")
	}
	trip, err := s.repo.GetTrip(ctx, item.TripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	if err := s.repo.UpdateFlight(ctx, item); err != nil {
		return err
	}
	return s.SyncExpenseForFlight(ctx, item)
}

func (s *Service) GetFlight(ctx context.Context, tripID, flightID string) (Flight, error) {
	if tripID == "" || flightID == "" {
		return Flight{}, errors.New("invalid flight entry")
	}
	return s.repo.GetFlight(ctx, tripID, flightID)
}

func (s *Service) DeleteFlight(ctx context.Context, tripID, flightID string) error {
	if tripID == "" || flightID == "" {
		return errors.New("invalid flight entry")
	}
	trip, err := s.repo.GetTrip(ctx, tripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	flight, err := s.repo.GetFlight(ctx, tripID, flightID)
	if err != nil {
		return err
	}
	if flight.DepartItineraryID != "" {
		_ = s.repo.DeleteItineraryItem(ctx, tripID, flight.DepartItineraryID)
	}
	if flight.ArriveItineraryID != "" {
		_ = s.repo.DeleteItineraryItem(ctx, tripID, flight.ArriveItineraryID)
	}
	if flight.ExpenseID != "" {
		_ = s.repo.DeleteExpense(ctx, tripID, flight.ExpenseID)
	}
	return s.repo.DeleteFlight(ctx, tripID, flightID)
}

func (s *Service) SyncExpenseForFlight(ctx context.Context, f Flight) error {
	if f.TripID == "" || f.ID == "" {
		return nil
	}
	trip, err := s.repo.GetTrip(ctx, f.TripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	notes := flightExpenseNotes(f)
	spentOn := flightExpenseSpentOn(f)
	if f.ExpenseID != "" {
		existing, err := s.repo.GetExpense(ctx, f.TripID, f.ExpenseID)
		if err == nil {
			existing.Category = "Airfare"
			existing.Amount = f.Cost
			existing.Notes = notes
			existing.SpentOn = spentOn
			existing.LodgingID = ""
			f.ExpenseID = existing.ID
			if err := s.repo.UpdateExpense(ctx, existing); err != nil {
				return err
			}
			return s.repo.UpdateFlight(ctx, f)
		}
	}
	id := uuid.NewString()
	if err := s.repo.AddExpense(ctx, Expense{
		ID:       id,
		TripID:   f.TripID,
		Category: "Airfare",
		Amount:   f.Cost,
		Notes:    notes,
		SpentOn:  spentOn,
	}); err != nil {
		return err
	}
	f.ExpenseID = id
	return s.repo.UpdateFlight(ctx, f)
}

func (s *Service) SyncExpenseForVehicleRental(ctx context.Context, v VehicleRental) error {
	if v.TripID == "" || v.ID == "" {
		return nil
	}
	trip, err := s.repo.GetTrip(ctx, v.TripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}

	upsert := func(expenseID string, amount float64, kind string) (string, error) {
		notes := vehicleExpenseNotes(v, kind)
		spentOn := vehicleExpenseSpentOn(v)
		if expenseID != "" {
			existing, err := s.repo.GetExpense(ctx, v.TripID, expenseID)
			if err == nil {
				existing.Category = "Car Rental"
				existing.Amount = amount
				existing.Notes = notes
				existing.SpentOn = spentOn
				existing.LodgingID = ""
				return existing.ID, s.repo.UpdateExpense(ctx, existing)
			}
		}
		id := uuid.NewString()
		return id, s.repo.AddExpense(ctx, Expense{
			ID:       id,
			TripID:   v.TripID,
			Category: "Car Rental",
			Amount:   amount,
			Notes:    notes,
			SpentOn:  spentOn,
		})
	}

	rentalExpenseID, err := upsert(v.RentalExpenseID, v.Cost, "Rental Cost")
	if err != nil {
		return err
	}
	insuranceExpenseID, err := upsert(v.InsuranceExpenseID, v.InsuranceCost, "Insurance Cost")
	if err != nil {
		return err
	}
	v.RentalExpenseID = rentalExpenseID
	v.InsuranceExpenseID = insuranceExpenseID
	return s.repo.UpdateVehicleRental(ctx, v)
}

func vehicleExpenseNotes(v VehicleRental, kind string) string {
	parts := []string{"Vehicle Rental", kind, v.PickUpLocation}
	if v.VehicleDetail != "" {
		parts = append(parts, "Vehicle: "+v.VehicleDetail)
	}
	if v.BookingConfirmation != "" {
		parts = append(parts, "Booking: "+v.BookingConfirmation)
	}
	if v.Notes != "" {
		parts = append(parts, v.Notes)
	}
	return strings.Join(parts, " · ")
}

func vehicleExpenseSpentOn(v VehicleRental) string {
	target := v.PickUpAt
	if v.PayAtPickUp {
		target = v.DropOffAt
	}
	if target == "" {
		return ""
	}
	t, err := time.Parse("2006-01-02T15:04", target)
	if err != nil {
		return ""
	}
	return t.Format("2006-01-02")
}

func VehicleRentalByItineraryItemID(vehicles []VehicleRental, items []ItineraryItem) map[string]VehicleRental {
	out := make(map[string]VehicleRental)
	for _, v := range vehicles {
		if v.PickUpItineraryID != "" {
			out[v.PickUpItineraryID] = v
		}
		if v.DropOffItineraryID != "" {
			out[v.DropOffItineraryID] = v
		}
	}
	for _, it := range items {
		if _, ok := out[it.ID]; ok {
			continue
		}
		titleValue, ok := vehicleLocationFromItineraryTitle(it.Title)
		if !ok {
			continue
		}
		for _, v := range vehicles {
			if v.PickUpLocation == titleValue || v.VehicleDetail == titleValue {
				out[it.ID] = v
				break
			}
		}
	}
	return out
}

func VehicleRentalByExpenseID(vehicles []VehicleRental) map[string]VehicleRental {
	out := make(map[string]VehicleRental)
	for _, v := range vehicles {
		if v.RentalExpenseID != "" {
			out[v.RentalExpenseID] = v
		}
		if v.InsuranceExpenseID != "" {
			out[v.InsuranceExpenseID] = v
		}
	}
	return out
}

func FlightByItineraryItemID(flights []Flight, items []ItineraryItem) map[string]Flight {
	out := make(map[string]Flight)
	for _, f := range flights {
		if f.DepartItineraryID != "" {
			out[f.DepartItineraryID] = f
		}
		if f.ArriveItineraryID != "" {
			out[f.ArriveItineraryID] = f
		}
	}
	for _, it := range items {
		if _, ok := out[it.ID]; ok {
			continue
		}
		label, ok := flightLabelFromItineraryTitle(it.Title)
		if !ok {
			continue
		}
		for _, f := range flights {
			if flightLabelValue(f.FlightName, f.FlightNumber) == label {
				out[it.ID] = f
				break
			}
		}
	}
	return out
}

func FlightByExpenseID(flights []Flight) map[string]Flight {
	out := make(map[string]Flight)
	for _, f := range flights {
		if f.ExpenseID != "" {
			out[f.ExpenseID] = f
		}
	}
	return out
}

func lodgingExpenseNotes(l Lodging) string {
	parts := []string{l.Name}
	if l.BookingConfirmation != "" {
		parts = append(parts, "Booking: "+l.BookingConfirmation)
	}
	if l.Notes != "" {
		parts = append(parts, l.Notes)
	}
	return strings.Join(parts, " · ")
}

func lodgingExpenseSpentOn(l Lodging) string {
	if l.CheckInAt == "" {
		return ""
	}
	t, err := time.Parse("2006-01-02T15:04", l.CheckInAt)
	if err != nil {
		return ""
	}
	return t.Format("2006-01-02")
}

func flightExpenseNotes(f Flight) string {
	parts := []string{"Flight"}
	label := flightLabelValue(f.FlightName, f.FlightNumber)
	if label != "" {
		parts = append(parts, label)
	}
	if f.DepartAirport != "" {
		parts = append(parts, "From: "+f.DepartAirport)
	}
	if f.ArriveAirport != "" {
		parts = append(parts, "To: "+f.ArriveAirport)
	}
	if f.BookingConfirmation != "" {
		parts = append(parts, "Booking: "+f.BookingConfirmation)
	}
	if f.Notes != "" {
		parts = append(parts, f.Notes)
	}
	return strings.Join(parts, " · ")
}

func flightExpenseSpentOn(f Flight) string {
	if f.DepartAt == "" {
		return ""
	}
	t, err := time.Parse("2006-01-02T15:04", f.DepartAt)
	if err != nil {
		return ""
	}
	return t.Format("2006-01-02")
}

func (s *Service) DeleteLodging(ctx context.Context, tripID, lodgingID string) error {
	if tripID == "" || lodgingID == "" {
		return errors.New("invalid accommodation entry")
	}
	trip, err := s.repo.GetTrip(ctx, tripID)
	if err != nil {
		return err
	}
	if trip.IsArchived {
		return errors.New("archived trips are read-only")
	}
	lodging, err := s.repo.GetLodging(ctx, tripID, lodgingID)
	if err != nil {
		return err
	}
	ci, co := lodging.CheckInItineraryID, lodging.CheckOutItineraryID
	if ci == "" || co == "" {
		ci2, co2 := s.findLodgingItineraryPair(ctx, tripID, lodging.Name)
		if ci == "" {
			ci = ci2
		}
		if co == "" {
			co = co2
		}
	}
	if ci != "" {
		_ = s.repo.DeleteItineraryItem(ctx, tripID, ci)
	}
	if co != "" {
		_ = s.repo.DeleteItineraryItem(ctx, tripID, co)
	}
	_ = s.deleteStrayLodgingItineraryByName(ctx, tripID, lodging.Name)
	_ = s.repo.DeleteExpenseByLodgingID(ctx, tripID, lodgingID)
	return s.repo.DeleteLodging(ctx, tripID, lodgingID)
}

func (s *Service) deleteStrayLodgingItineraryByName(ctx context.Context, tripID, name string) error {
	if name == "" {
		return nil
	}
	items, err := s.repo.ListItineraryItems(ctx, tripID)
	if err != nil {
		return err
	}
	want := map[string]struct{}{}
	addAccommodationItineraryTitleKeys(want, name)
	for _, it := range items {
		if _, match := want[it.Title]; match {
			_ = s.repo.DeleteItineraryItem(ctx, tripID, it.ID)
		}
	}
	return nil
}

func (s *Service) findLodgingItineraryPair(ctx context.Context, tripID, lodgingName string) (checkInID, checkOutID string) {
	items, err := s.repo.ListItineraryItems(ctx, tripID)
	if err != nil {
		return "", ""
	}
	return findLodgingItineraryPairFromItems(items, tripID, lodgingName)
}

func (s *Service) GetLodging(ctx context.Context, tripID, lodgingID string) (Lodging, error) {
	if tripID == "" || lodgingID == "" {
		return Lodging{}, errors.New("invalid accommodation entry")
	}
	return s.repo.GetLodging(ctx, tripID, lodgingID)
}

func (s *Service) ListChanges(ctx context.Context, tripID, since string) ([]Change, error) {
	return s.repo.ListChanges(ctx, tripID, since)
}

func (s *Service) GetAppSettings(ctx context.Context) (AppSettings, error) {
	settings, err := s.repo.GetAppSettings(ctx)
	if err != nil {
		return AppSettings{}, err
	}
	if settings.AppTitle == "" {
		settings.AppTitle = "REMI Trip Planner"
	}
	if settings.DefaultCurrencyName == "" {
		settings.DefaultCurrencyName = "USD"
	}
	if settings.DefaultCurrencySymbol == "" {
		settings.DefaultCurrencySymbol = "$"
	}
	if settings.MapDefaultZoom < 1 {
		settings.MapDefaultZoom = 6
	}
	return settings, nil
}

func (s *Service) SaveAppSettings(ctx context.Context, settings AppSettings) error {
	if settings.AppTitle == "" {
		return errors.New("app title is required")
	}
	if settings.DefaultCurrencyName == "" {
		settings.DefaultCurrencyName = "USD"
	}
	if settings.DefaultCurrencySymbol == "" {
		settings.DefaultCurrencySymbol = "$"
	}
	if settings.MapDefaultZoom < 1 || settings.MapDefaultZoom > 20 {
		settings.MapDefaultZoom = 6
	}
	return s.repo.SaveAppSettings(ctx, settings)
}

// MatchLodgingByHotelTitle matches accommodation check-in/out itinerary titles to a lodging row.
func MatchLodgingByHotelTitle(it ItineraryItem, lodgings []Lodging) (Lodging, bool) {
	name, ok := accommodationNameFromItineraryTitle(it.Title)
	if !ok {
		return Lodging{}, false
	}
	for _, l := range lodgings {
		if l.Name == name {
			return l, true
		}
	}
	return Lodging{}, false
}

// LodgingByItineraryItemID maps itinerary item ids for lodging stops to their Lodging row.
func LodgingByItineraryItemID(lodgings []Lodging, items []ItineraryItem) map[string]Lodging {
	out := make(map[string]Lodging)
	for _, l := range lodgings {
		if l.CheckInItineraryID != "" {
			out[l.CheckInItineraryID] = l
		}
		if l.CheckOutItineraryID != "" {
			out[l.CheckOutItineraryID] = l
		}
	}
	for _, it := range items {
		if _, ok := out[it.ID]; ok {
			continue
		}
		if l, ok := MatchLodgingByHotelTitle(it, lodgings); ok {
			out[it.ID] = l
		}
	}
	return out
}
