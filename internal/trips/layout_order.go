package trips

import (
	"strings"
)

// Main trip page section keys (hero card and trip header/edit panel stay above these;
// Trip Map is the first key in the scrollable main column).
const (
	MainSectionMap       = "map"
	MainSectionItinerary = "itinerary"
	MainSectionSpends    = "spends"
	MainSectionChecklist = "checklist"
	MainSectionBookings  = "bookings"
	// MainSectionStay, MainSectionVehicle, MainSectionFlights are legacy keys merged into MainSectionBookings.
	MainSectionStay    = "stay"
	MainSectionVehicle = "vehicle"
	MainSectionFlights = "flights"
	MainSectionTheTab  = "the_tab"
)

// DefaultMainSectionOrder is the main-column order on Trip Details (below the hero).
var DefaultMainSectionOrder = []string{
	MainSectionMap,
	MainSectionItinerary,
	MainSectionBookings,
	MainSectionChecklist,
	MainSectionSpends,
	MainSectionTheTab,
}

var mainSectionSet = map[string]struct{}{
	MainSectionMap:       {},
	MainSectionItinerary: {},
	MainSectionSpends:    {},
	MainSectionTheTab:    {},
	MainSectionChecklist: {},
	MainSectionBookings:  {},
	MainSectionStay:      {},
	MainSectionVehicle:   {},
	MainSectionFlights:   {},
}

// Sidebar widget keys (desktop/tablet right column).
const (
	SidebarAddStop      = "add_stop"
	SidebarAddCommute   = "add_commute"
	SidebarBudget       = "budget"
	SidebarQuickSpends  = "quick_spends"
	SidebarTabTotal     = "tab_total"
	SidebarAddTab       = "add_tab"
	SidebarAddChecklist = "checklist"
)

// Trip Details right column: fixed set and order (legacy keys in stored CSV are ignored).
var DefaultSidebarWidgetOrder = []string{
	SidebarBudget,
	SidebarTabTotal,
	SidebarAddStop,
	SidebarAddChecklist,
}

var sidebarWidgetSet = map[string]struct{}{
	SidebarAddStop:      {},
	SidebarBudget:       {},
	SidebarTabTotal:     {},
	SidebarAddChecklist: {},
}

// PreprocessMainSectionOrderCSV merges legacy stay, vehicle, flights into a single
// "bookings" key (first position wins) so existing saved trip layouts keep working.
func PreprocessMainSectionOrderCSV(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return raw
	}
	parts := strings.Split(raw, ",")
	var out []string
	inserted := false
	for _, p := range parts {
		k := strings.ToLower(strings.TrimSpace(p))
		if k == "" {
			continue
		}
		if k == MainSectionStay || k == MainSectionVehicle || k == MainSectionFlights {
			if !inserted {
				out = append(out, MainSectionBookings)
				inserted = true
			}
			continue
		}
		if k == MainSectionBookings {
			if !inserted {
				out = append(out, MainSectionBookings)
				inserted = true
			}
			continue
		}
		out = append(out, k)
	}
	return strings.Join(out, ",")
}

// NormalizeMainSectionOrder parses a comma-separated saved order, drops unknown
// tokens, dedupes, then appends any missing keys in default order.
func NormalizeMainSectionOrder(raw string) []string {
	return normalizeOrder(PreprocessMainSectionOrderCSV(raw), mainSectionSet, DefaultMainSectionOrder)
}

// NormalizeSidebarWidgetOrder is like NormalizeMainSectionOrder for sidebar keys.
func NormalizeSidebarWidgetOrder(raw string) []string {
	return normalizeOrder(raw, sidebarWidgetSet, DefaultSidebarWidgetOrder)
}

// JoinMainSectionOrder encodes main section order for storage.
func JoinMainSectionOrder(keys []string) string {
	if len(keys) == 0 {
		return strings.Join(DefaultMainSectionOrder, ",")
	}
	return strings.Join(keys, ",")
}

// JoinSidebarWidgetOrder encodes sidebar widget order for storage.
func JoinSidebarWidgetOrder(keys []string) string {
	if len(keys) == 0 {
		return strings.Join(DefaultSidebarWidgetOrder, ",")
	}
	return strings.Join(keys, ",")
}

func normalizeOrder(raw string, valid map[string]struct{}, defaults []string) []string {
	var parts []string
	for _, tok := range strings.Split(raw, ",") {
		k := strings.ToLower(strings.TrimSpace(tok))
		if k == "" {
			continue
		}
		if _, ok := valid[k]; !ok {
			continue
		}
		if containsString(parts, k) {
			continue
		}
		parts = append(parts, k)
	}
	for _, d := range defaults {
		if !containsString(parts, d) {
			parts = append(parts, d)
		}
	}
	return parts
}

func containsString(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

func parseCommaKeySet(raw string) map[string]bool {
	m := make(map[string]bool)
	for _, tok := range strings.Split(raw, ",") {
		k := strings.ToLower(strings.TrimSpace(tok))
		if k != "" {
			m[k] = true
		}
	}
	return m
}

// MainSectionVisible returns whether a normalized main section should render for this trip.
func MainSectionVisible(key string, t Trip) bool {
	switch key {
	case MainSectionItinerary:
		if !t.UIShowItinerary {
			return false
		}
	case MainSectionChecklist:
		if !t.UIShowChecklist {
			return false
		}
	case MainSectionBookings:
		if !t.UIShowStay && !t.UIShowVehicle && !t.UIShowFlights {
			return false
		}
		h := parseCommaKeySet(t.UIMainSectionHidden)
		if h[MainSectionBookings] {
			return false
		}
		// Show unified stream if at least one enabled booking type is not hidden.
		if t.UIShowStay && !h[MainSectionStay] {
			return true
		}
		if t.UIShowVehicle && !h[MainSectionVehicle] {
			return true
		}
		if t.UIShowFlights && !h[MainSectionFlights] {
			return true
		}
		return false
	case MainSectionStay:
		if !t.UIShowStay {
			return false
		}
	case MainSectionVehicle:
		if !t.UIShowVehicle {
			return false
		}
	case MainSectionFlights:
		if !t.UIShowFlights {
			return false
		}
	case MainSectionSpends:
		if !t.UIShowSpends {
			return false
		}
	case MainSectionTheTab:
		if !t.UIShowTheTab {
			return false
		}
		if !t.UIShowSpends {
			return false
		}
	}
	if strings.TrimSpace(t.UIMainSectionHidden) != "" {
		return !parseCommaKeySet(t.UIMainSectionHidden)[key]
	}
	return true
}

// SidebarWidgetVisible returns whether a sidebar widget should render.
func SidebarWidgetVisible(key string, t Trip) bool {
	if !t.UIShowItinerary && (key == SidebarAddStop || key == SidebarAddCommute) {
		return false
	}
	if !t.UIShowChecklist && key == SidebarAddChecklist {
		return false
	}
	if !t.UIShowSpends && (key == SidebarBudget || key == SidebarQuickSpends || key == SidebarTabTotal || key == SidebarAddTab) {
		return false
	}
	if !t.UIShowTheTab && (key == SidebarTabTotal || key == SidebarAddTab) {
		return false
	}
	if strings.TrimSpace(t.UISidebarWidgetHidden) != "" {
		if parseCommaKeySet(t.UISidebarWidgetHidden)[key] {
			return false
		}
		return true
	}
	return true
}

// TripMobileFabHasItems is true when the trip FAB menu would list at least one action
// (must match web/templates/trip_mobile_fab_links.html).
func TripMobileFabHasItems(t Trip) bool {
	if t.UIShowItinerary && (SidebarWidgetVisible(SidebarAddStop, t) || SidebarWidgetVisible(SidebarAddCommute, t)) {
		return true
	}
	if TripExpenseQuickAddVisible(t) {
		return true
	}
	if t.UIShowStay {
		return true
	}
	if t.UIShowVehicle {
		return true
	}
	if t.UIShowFlights {
		return true
	}
	if SidebarWidgetVisible(SidebarAddChecklist, t) {
		return true
	}
	if t.UIShowDocuments {
		return true
	}
	return false
}

// TripExpenseQuickAddVisible is true when the unified Add expense entry (sidebar / mobile / calendar)
// should be offered on trip details (quick_spends widget and/or add_tab when The Tab is enabled).
func TripExpenseQuickAddVisible(t Trip) bool {
	if t.UIShowSpends && SidebarWidgetVisible(SidebarQuickSpends, t) {
		return true
	}
	if t.SectionEnabledTheTab() && SidebarWidgetVisible(SidebarAddTab, t) {
		return true
	}
	return false
}

// TripDesktopCalendarFlyoutHasActions is true when the itinerary calendar “quick add” flyout
// would show at least one button (trip.html desktop-calendar-flyout).
func TripDesktopCalendarFlyoutHasActions(t Trip) bool {
	if t.UIShowItinerary && (SidebarWidgetVisible(SidebarAddStop, t) || SidebarWidgetVisible(SidebarAddCommute, t)) {
		return true
	}
	if t.UIShowStay {
		return true
	}
	if t.UIShowVehicle {
		return true
	}
	if t.UIShowFlights {
		return true
	}
	return false
}

// MainSectionVisibilityIcon is a Material Symbols name for trip settings visibility rows.
func MainSectionVisibilityIcon(key string) string {
	switch key {
	case MainSectionMap:
		return "map"
	case MainSectionItinerary:
		return "route"
	case MainSectionSpends:
		return "payments"
	case MainSectionChecklist:
		return "checklist"
	case MainSectionStay:
		return "hotel"
	case MainSectionVehicle:
		return "directions_car"
	case MainSectionFlights:
		return "flight"
	case MainSectionBookings:
		return "confirmation_number"
	case MainSectionTheTab:
		return "tab"
	default:
		return "widgets"
	}
}

// SidebarWidgetVisibilityIcon is a Material Symbols name for sidebar visibility rows.
func SidebarWidgetVisibilityIcon(key string) string {
	switch key {
	case SidebarAddStop:
		return "pin_drop"
	case SidebarAddCommute:
		return "directions"
	case SidebarBudget:
		return "account_balance_wallet"
	case SidebarQuickSpends:
		return "receipt_long"
	case SidebarTabTotal:
		return "groups"
	case SidebarAddTab:
		return "post_add"
	case SidebarAddChecklist:
		return "playlist_add"
	default:
		return "widgets"
	}
}

// MainSectionLabel is a short title for trip settings reorder UI.
func MainSectionLabel(key string) string {
	switch key {
	case MainSectionMap:
		return "Trip Map"
	case MainSectionItinerary:
		return "Itinerary"
	case MainSectionSpends:
		return "Expenses"
	case MainSectionChecklist:
		return "Notes & Checklists"
	case MainSectionStay:
		return "Accommodation"
	case MainSectionVehicle:
		return "Vehicle Rental"
	case MainSectionFlights:
		return "Flights"
	case MainSectionBookings:
		return "Booking details"
	case MainSectionTheTab:
		return "Group Expenses"
	default:
		return key
	}
}

// SidebarWidgetLabel is a short title for sidebar reorder UI.
func SidebarWidgetLabel(key string) string {
	switch key {
	case SidebarAddStop:
		return "Add Stop"
	case SidebarAddCommute:
		return "Add commute leg"
	case SidebarBudget:
		return "Budget Limit"
	case SidebarQuickSpends:
		return "Add Expense"
	case SidebarTabTotal:
		return "Total Group Expense"
	case SidebarAddTab:
		return "Add Group Expense"
	case SidebarAddChecklist:
		return "Add Note / Checklist"
	default:
		return key
	}
}
