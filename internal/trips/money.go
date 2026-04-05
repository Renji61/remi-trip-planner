package trips

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
)

var ErrInvalidMoneyAmount = errors.New("invalid money amount")

// MoneyFromCents converts minor units to a display-ready decimal amount.
func MoneyFromCents(cents int64) float64 {
	return float64(cents) / 100
}

// MoneyToCentsFloat converts a legacy decimal amount to minor units.
func MoneyToCentsFloat(v float64) int64 {
	return moneyToCents(v)
}

// ParseMoneyToCents parses a decimal money string into minor units.
func ParseMoneyToCents(raw string) (int64, error) {
	s := strings.TrimSpace(strings.ReplaceAll(raw, ",", ""))
	if s == "" {
		return 0, nil
	}
	sign := int64(1)
	if strings.HasPrefix(s, "+") {
		s = strings.TrimPrefix(s, "+")
	}
	if strings.HasPrefix(s, "-") {
		sign = -1
		s = strings.TrimPrefix(s, "-")
	}
	if s == "" {
		return 0, ErrInvalidMoneyAmount
	}
	parts := strings.Split(s, ".")
	if len(parts) > 2 {
		return 0, ErrInvalidMoneyAmount
	}
	whole := parts[0]
	if whole == "" {
		whole = "0"
	}
	for _, r := range whole {
		if r < '0' || r > '9' {
			return 0, ErrInvalidMoneyAmount
		}
	}
	var frac string
	if len(parts) == 2 {
		frac = parts[1]
		if len(frac) > 2 {
			return 0, fmt.Errorf("%w: use at most two decimal places", ErrInvalidMoneyAmount)
		}
		for _, r := range frac {
			if r < '0' || r > '9' {
				return 0, ErrInvalidMoneyAmount
			}
		}
	}
	for len(frac) < 2 {
		frac += "0"
	}
	wholeUnits, err := strconv.ParseInt(whole, 10, 64)
	if err != nil {
		return 0, ErrInvalidMoneyAmount
	}
	fracUnits, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, ErrInvalidMoneyAmount
	}
	return sign * (wholeUnits*100 + fracUnits), nil
}

func SetExpenseAmountCents(e *Expense, cents int64) {
	if e == nil {
		return
	}
	e.AmountCents = cents
	e.Amount = MoneyFromCents(cents)
}

func SetExpenseAmountFloat(e *Expense, amount float64) {
	if e == nil {
		return
	}
	SetExpenseAmountCents(e, MoneyToCentsFloat(amount))
}

func SetTripBudgetCapCents(t *Trip, cents int64) {
	if t == nil {
		return
	}
	t.BudgetCapCents = cents
	t.BudgetCap = MoneyFromCents(cents)
}

func SetTripBudgetCapFloat(t *Trip, amount float64) {
	if t == nil {
		return
	}
	SetTripBudgetCapCents(t, MoneyToCentsFloat(amount))
}

func SetItineraryEstCostCents(i *ItineraryItem, cents int64) {
	if i == nil {
		return
	}
	i.EstCostCents = cents
	i.EstCost = MoneyFromCents(cents)
}

func SetItineraryEstCostFloat(i *ItineraryItem, amount float64) {
	if i == nil {
		return
	}
	SetItineraryEstCostCents(i, MoneyToCentsFloat(amount))
}

func SetLodgingCostCents(l *Lodging, cents int64) {
	if l == nil {
		return
	}
	l.CostCents = cents
	l.Cost = MoneyFromCents(cents)
}

func SetLodgingCostFloat(l *Lodging, amount float64) {
	if l == nil {
		return
	}
	SetLodgingCostCents(l, MoneyToCentsFloat(amount))
}

func SetVehicleCostCents(v *VehicleRental, cents int64) {
	if v == nil {
		return
	}
	v.CostCents = cents
	v.Cost = MoneyFromCents(cents)
}

func SetVehicleCostFloat(v *VehicleRental, amount float64) {
	if v == nil {
		return
	}
	SetVehicleCostCents(v, MoneyToCentsFloat(amount))
}

func SetVehicleInsuranceCostCents(v *VehicleRental, cents int64) {
	if v == nil {
		return
	}
	v.InsuranceCostCents = cents
	v.InsuranceCost = MoneyFromCents(cents)
}

func SetVehicleInsuranceCostFloat(v *VehicleRental, amount float64) {
	if v == nil {
		return
	}
	SetVehicleInsuranceCostCents(v, MoneyToCentsFloat(amount))
}

func SetFlightCostCents(f *Flight, cents int64) {
	if f == nil {
		return
	}
	f.CostCents = cents
	f.Cost = MoneyFromCents(cents)
}

func SetFlightCostFloat(f *Flight, amount float64) {
	if f == nil {
		return
	}
	SetFlightCostCents(f, MoneyToCentsFloat(amount))
}

func SetTabSettlementAmountCents(s *TabSettlement, cents int64) {
	if s == nil {
		return
	}
	s.AmountCents = cents
	s.Amount = MoneyFromCents(cents)
}

func SetTabSettlementAmountFloat(s *TabSettlement, amount float64) {
	if s == nil {
		return
	}
	SetTabSettlementAmountCents(s, MoneyToCentsFloat(amount))
}
