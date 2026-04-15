package httpapp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestExpensesPageAndFormsCopy validates personal/group expense UI copy on /expenses (budget.html),
// inline edits (budget_transactions_rows, trip.html, tab fragments), group expenses (the_tab), and JS.
func TestExpensesPageAndFormsCopy(t *testing.T) {
	root := findModuleRoot(t)

	budgetB, err := os.ReadFile(filepath.Join(root, "web", "templates", "budget.html"))
	if err != nil {
		t.Fatal(err)
	}
	budget := string(budgetB)

	rowsB, err := os.ReadFile(filepath.Join(root, "web", "templates", "budget_transactions_rows.html"))
	if err != nil {
		t.Fatal(err)
	}
	rows := string(rowsB)

	tripB, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip.html"))
	if err != nil {
		t.Fatal(err)
	}
	trip := string(tripB)

	unifiedB, err := os.ReadFile(filepath.Join(root, "web", "templates", "trip_unified_expense_form.html"))
	if err != nil {
		t.Fatal(err)
	}
	unifiedExpenseTpl := string(unifiedB)
	budgetAndUnified := budget + unifiedExpenseTpl
	tripAndUnified := trip + unifiedExpenseTpl

	tabB, err := os.ReadFile(filepath.Join(root, "web", "templates", "the_tab.html"))
	if err != nil {
		t.Fatal(err)
	}
	tab := string(tabB)

	fragB, err := os.ReadFile(filepath.Join(root, "web", "templates", "tab_expense_fragments.html"))
	if err != nil {
		t.Fatal(err)
	}
	frag := string(fragB)

	settleB, err := os.ReadFile(filepath.Join(root, "web", "templates", "tab_settlement_fragments.html"))
	if err != nil {
		t.Fatal(err)
	}
	settle := string(settleB)

	jsB, err := os.ReadFile(filepath.Join(root, "web", "static", "app.js"))
	if err != nil {
		t.Fatal(err)
	}
	js := string(jsB)

	budgetWant := []string{
		"This expense will be added to your trip total and exports.",
		`placeholder="e.g. Dinner at Ichiran or airport train"`,
		`<option value="" disabled selected>Select a category</option>`,
		`<option value="" disabled selected>Select a payment method</option>`,
		`placeholder="Add any notes"`,
		`>Showing all transactions</span>`,
		`this.textContent = 'Showing all transactions';`,
		`<h3 id="budget-mobile-expense-edit-title">Edit Expense</h3>`,
	}
	for _, s := range budgetWant {
		if !strings.Contains(budgetAndUnified, s) {
			t.Errorf("budget.html + trip_unified_expense_form.html missing %q", s)
		}
	}
	budgetAvoid := []string{
		"This expense will be added to your trip's spending total and export files.",
		"Dinner at Ichiran or Airport Train.",
		`<option value="" disabled selected>Select category</option>`,
		`<option value="" disabled selected>Select payment method</option>`,
		`placeholder="Optional notes"`,
		">All transactions shown</span>",
		`this.textContent = 'All transactions shown';`,
	}
	for _, s := range budgetAvoid {
		if strings.Contains(budget, s) {
			t.Errorf("budget.html should not contain %q", s)
		}
	}

	rowsWant := []string{
		`<h3>Edit Expense</h3>`,
		`Update details; changes sync with this trip.`,
		`placeholder="e.g. Dinner at Ichiran or airport train"`,
		`placeholder="Add any notes"`,
		`>Save Changes</button>`,
	}
	for _, s := range rowsWant {
		if !strings.Contains(rows, s) {
			t.Errorf("budget_transactions_rows.html missing %q", s)
		}
	}
	if strings.Contains(rows, "Update details below; changes sync to this trip.") {
		t.Error("budget_transactions_rows.html should use updated edit subtitle")
	}

	tripWant := []string{
		`<h3>Edit Expense</h3>`,
		`Update details; changes sync with this trip.`,
		`placeholder="e.g. Dinner at Ichiran or airport train"`,
		`placeholder="Add any notes"`,
		`<option value="" disabled selected>Select a category</option>`,
		`<option value="" disabled selected>Select a payment method</option>`,
		`>Save Changes</button>`,
		`<h3 id="tab-mobile-expense-edit-title">Edit Expense</h3>`,
		`Use <strong>Add expense</strong> above for personal spends and group splits.`,
		`View {{$.Details.Trip.GroupExpensesSectionTitle}}</a> to add or edit entries.`,
		`Add a spend without leaving this trip.`,
	}
	for _, s := range tripWant {
		if strings.Contains(s, "placeholder=") || strings.Contains(s, "<option value=\"\"") {
			if !strings.Contains(tripAndUnified, s) {
				t.Errorf("trip.html + trip_unified_expense_form.html (expenses) missing %q", s)
			}
			continue
		}
		if !strings.Contains(trip, s) {
			t.Errorf("trip.html (expenses) missing %q", s)
		}
	}
	tripAvoid := []string{
		`placeholder="e.g., Dinner at Ichiran."`,
		`placeholder="Optional notes"`,
		`placeholder="Boat transfer"`,
		`Update details below; changes sync to this trip.`,
		`<h3>Edit {{$.Details.Trip.SpendsSectionTitle}}</h3>`,
		`totals roll into {{$.Details.Trip.SpendsSectionTitle}}`,
		`Log a spend without leaving this trip.`,
		`>Open {{$.Details.Trip.GroupExpensesSectionTitle}}</a> to add or edit entries.`,
	}
	for _, s := range tripAvoid {
		if strings.Contains(trip, s) {
			t.Errorf("trip.html should not contain (expense legacy) %q", s)
		}
	}

	tabWant := []string{
		`Shared costs and balances. Totals are included in {{.Trip.SpendsSectionTitle}}.`,
		`View {{.Trip.SpendsSectionTitle}}</a>`,
		`>Add Expense</a>`,
		`>Save Settlement</a>`,
		`>Total Group Spending</span>`,
		`After splits and settlements</span>`,
		`>Balances &amp; Settlements</span>`,
		`>Pending Settlements</p>`,
		`>Total Owed</button>`,
		`Combined amount owed by net debtors</span>`,
		`Everyone is settled. Add expenses to see suggested payments.`,
		`Suggested payments are based on net balances and minimize the number of transfers.`,
		`aria-label="Add expenses and settlements"`,
		`>Add Expenses &amp; Settlements</h3>`,
		`<h3 id="tab-add-title">Add New Expense</h3>`,
		`<span class="tab-field-k">Title</span>`,
		`>Split Method</p>`,
		`Split equally among selected members.</p>`,
		`Enter an amount before configuring the split.</p>`,
		`<span class="tab-submit-label-desktop">Add Expense</span>`,
		`data-toast-message="Expense added."`,
		`Your last split method will be saved as default for this trip.`,
		`<h3 id="tab-settle-title">Save Settlement</h3>`,
		`Record a payment between members for this trip.`,
		`<span class="tab-field-k">Payment Method</span>`,
		`<span class="tab-submit-label-desktop">Save Settlement</span>`,
		`<h4 class="tab-recent-settlements-title">Recent Settlements</h4>`,
		`<p class="muted tab-settlement-empty">No settlements yet</p>`,
		`<option value="" disabled selected>Select a category</option>`,
		`<option value="" disabled selected>Select a payment method</option>`,
		`placeholder="Add any notes"`,
		`>Showing all transactions</span>`,
		`<h3 id="tab-mobile-expense-edit-title">Edit Expense</h3>`,
	}
	for _, s := range tabWant {
		if !strings.Contains(tab, s) {
			t.Errorf("the_tab.html missing %q", s)
		}
	}
	tabAvoid := []string{
		"Totals roll into",
		">Open {{.Trip.SpendsSectionTitle}}</a>",
		">Log Expense</a>",
		">Log Settlement</a>",
		">Total group spending</span>",
		"After splits &amp; recorded settlements",
		">Balances &amp; simplified debts</span>",
		">Settlements pending</p>",
		">Total owed / simplified</button>",
		"Combined amount owed by people who are net debtors",
		"Everyone is square, or add Tab expenses",
		`aria-label="Log expense and settlements"`,
		">Log expenses &amp; settlements</h3>",
		"<h3 id=\"tab-add-title\">Log new expense</h3>",
		">Expense title</span>",
		">Split configuration</p>",
		"Split equally among selected people.",
		"Enter an expense amount first, then configure the split.",
		`<span class="tab-submit-label-desktop">Log Expense</span>`,
		`data-toast-message="Expense logged."`,
		"<h3 id=\"tab-settle-title\">Record settlement</h3>",
		`<h3 id="tab-settle-title">Record Settlement</h3>`,
		"Record a payment from one person to another on this trip.",
		`<span class="tab-submit-label-desktop">Record settlement</span>`,
		`<span class="tab-submit-label-desktop">Record Settlement</span>`,
		">Record Settlement</a>",
		"Suggested payments below use the same balances",
		"Your last split method is saved as this trip's",
		"<h4 class=\"tab-recent-settlements-title\">Recent settlements</h4>",
		"No settlements recorded.",
		`<label class="tab-field-label"><span class="tab-field-k">Method</span>`,
	}
	for _, s := range tabAvoid {
		if strings.Contains(tab, s) {
			t.Errorf("the_tab.html should not contain %q", s)
		}
	}
	if strings.Contains(tab, "All transactions shown") {
		t.Error("the_tab.html should use Showing all transactions")
	}

	fragWant := []string{
		`<h3>Edit Expense</h3>`,
		`Update details; splits and balances will refresh.`,
		`placeholder="Add any notes"`,
		`>Save Changes</button>`,
		`Enter an amount before configuring the split.</p>`,
	}
	for _, s := range fragWant {
		if !strings.Contains(frag, s) {
			t.Errorf("tab_expense_fragments.html missing %q", s)
		}
	}
	if strings.Contains(frag, "Update the entry; splits and balances refresh.") {
		t.Error("tab_expense_fragments.html should use updated edit subtitle for Tab inline edit")
	}

	settleWant := []string{
		`<label>Payment Method`,
		`<button type="submit">Save Changes</button>`,
	}
	for _, s := range settleWant {
		if !strings.Contains(settle, s) {
			t.Errorf("tab_settlement_fragments.html missing %q", s)
		}
	}
	if strings.Contains(settle, "<label>Method") && !strings.Contains(settle, "<label>Payment Method") {
		t.Error("tab_settlement_fragments.html should label settlement method as Payment Method")
	}
	if strings.Contains(settle, `<button type="submit">Save</button>`) {
		t.Error("tab_settlement_fragments.html settlement edit should use Save Changes")
	}

	if !strings.Contains(js, `btn.textContent = "Showing all transactions";`) {
		t.Error(`app.js should set tab view-all button to "Showing all transactions"`)
	}
	if strings.Contains(js, `btn.textContent = "All transactions shown";`) {
		t.Error(`app.js should not use legacy "All transactions shown" for tab expenses`)
	}
	if !strings.Contains(js, `equal: "Split equally among selected members."`) {
		t.Error(`app.js tab split hint equal should use "selected members"`)
	}
	if !strings.Contains(js, `exact: "Enter each person’s share (must total the expense amount).",`) {
		t.Error(`app.js tab split hint exact should match updated copy`)
	}
	if !strings.Contains(js, `btn.setAttribute("title", "Enter an amount before configuring the split.");`) {
		t.Error(`app.js disabled split submit title should match updated amount warning`)
	}
	if strings.Contains(js, `Split equally among selected people.`) {
		t.Error(`app.js should not use legacy split hint "selected people"`)
	}
	if strings.Contains(js, "Enter each person’s share in dollars (must add up to the expense total).") {
		t.Error(`app.js should not use legacy exact-split hint`)
	}
	if !strings.Contains(js, `percent: "Percentages must total 100%.",`) {
		t.Error(`app.js tab split hint percent should use "must total" wording`)
	}
	if !strings.Contains(js, `shares: "Allocate costs using share units.",`) {
		t.Error(`app.js tab split hint shares should use share units wording`)
	}
	if strings.Contains(js, `percent: "Percentages should add up to 100%.",`) {
		t.Error(`app.js should not use legacy percent split hint`)
	}
	if strings.Contains(js, `shares: "Allocate costs using a proportional share system.",`) {
		t.Error(`app.js should not use legacy shares split hint`)
	}
}
