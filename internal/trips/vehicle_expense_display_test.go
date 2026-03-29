package trips

import (
	"testing"
)

func TestCollapseVehicleRentalExpenseDuplicates(t *testing.T) {
	v := VehicleRental{
		ID:                  "veh1",
		PickUpLocation:      "Airport",
		VehicleDetail:       "VW",
		BookingConfirmation: "B1",
		RentalExpenseID:     "e-rent",
		InsuranceExpenseID:  "e-ins",
	}
	expenses := []Expense{
		{ID: "e-rent", TripID: "t1", Category: "Car Rental", Amount: 2000, SpentOn: "2026-03-29", Notes: "old"},
		{ID: "e-ins", TripID: "t1", Category: "Car Rental", Amount: 0, SpentOn: "2026-03-29"},
		{ID: "e-other", TripID: "t1", Category: "Food & Dining", Amount: 10, SpentOn: "2026-03-28"},
	}
	out := CollapseVehicleRentalExpenseDuplicates(expenses, []VehicleRental{v})
	if len(out) != 2 {
		t.Fatalf("len=%d want 2", len(out))
	}
	var merged *Expense
	for i := range out {
		if out[i].ID == "e-rent" {
			merged = &out[i]
			break
		}
	}
	if merged == nil {
		t.Fatal("missing merged rental row")
	}
	if merged.Amount != 2000 {
		t.Fatalf("amount=%v want 2000", merged.Amount)
	}
	if merged.Notes == "" || merged.Notes == "old" {
		t.Fatalf("notes should be regenerated: %q", merged.Notes)
	}
}

func TestCollapseVehicleRentalExpenseDuplicates_noVehicles(t *testing.T) {
	expenses := []Expense{{ID: "a", Amount: 1}}
	out := CollapseVehicleRentalExpenseDuplicates(expenses, nil)
	if len(out) != 1 || out[0].ID != "a" {
		t.Fatalf("got %+v", out)
	}
}
