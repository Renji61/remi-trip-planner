package trips

import "strings"

// VehicleRentalMergedExpenseNotes is the description for a single spends row that combines
// rental and insurance amounts from the same vehicle booking.
func VehicleRentalMergedExpenseNotes(v VehicleRental) string {
	return vehicleExpenseNotes(v, "Rental & insurance")
}

// CollapseVehicleRentalExpenseDuplicates merges paired rental + insurance ledger rows from the
// same vehicle booking into one expense for display. Underlying stored expenses are unchanged;
// summed amounts match the previous two lines. Lodging and flights already use one expense per booking.
func CollapseVehicleRentalExpenseDuplicates(expenses []Expense, vehicles []VehicleRental) []Expense {
	if len(expenses) == 0 || len(vehicles) == 0 {
		return expenses
	}
	byID := make(map[string]Expense, len(expenses))
	for _, e := range expenses {
		byID[e.ID] = e
	}
	mergedByRentalID := make(map[string]Expense)
	skip := make(map[string]struct{})

	for _, v := range vehicles {
		rid := strings.TrimSpace(v.RentalExpenseID)
		iid := strings.TrimSpace(v.InsuranceExpenseID)
		if rid == "" || iid == "" || rid == iid {
			continue
		}
		ren, okR := byID[rid]
		ins, okI := byID[iid]
		if !okR || !okI {
			continue
		}
		m := ren
		m.Amount = ren.Amount + ins.Amount
		m.Notes = VehicleRentalMergedExpenseNotes(v)
		switch {
		case ren.SpentOn == "":
			m.SpentOn = ins.SpentOn
		case ins.SpentOn != "" && ins.SpentOn < ren.SpentOn:
			m.SpentOn = ins.SpentOn
		}
		mergedByRentalID[rid] = m
		skip[iid] = struct{}{}
	}

	if len(mergedByRentalID) == 0 {
		return expenses
	}

	out := make([]Expense, 0, len(expenses)-len(skip))
	for _, e := range expenses {
		if _, gone := skip[e.ID]; gone {
			continue
		}
		if merged, ok := mergedByRentalID[e.ID]; ok {
			out = append(out, merged)
			continue
		}
		out = append(out, e)
	}
	return out
}
