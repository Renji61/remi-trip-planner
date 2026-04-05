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
	MainSectionStay      = "stay"
	MainSectionVehicle   = "vehicle"
	MainSectionFlights   = "flights"
	MainSectionTheTab    = "the_tab"
)

// DefaultMainSectionOrder is the main-column order on Trip Details (below the hero).
var DefaultMainSectionOrder = []string{
	MainSectionMap,
	MainSectionItinerary,
	MainSectionFlights,
	MainSectionStay,
	MainSectionVehicle,
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
	MainSectionStay:      {},
	MainSectionVehicle:   {},
	MainSectionFlights:   {},
}

// Sidebar widget keys (desktop/tablet right column).
const (
	SidebarAddStop      = "add_stop"
	SidebarBudget       = "budget"
	SidebarQuickSpends  = "quick_spends"
	SidebarTabTotal     = "tab_total"
	SidebarAddTab       = "add_tab"
	SidebarAddChecklist = "checklist"
)

var DefaultSidebarWidgetOrder = []string{
	SidebarAddStop,
	SidebarAddChecklist,
	SidebarQuickSpends,
	SidebarTabTotal,
	SidebarAddTab,
	SidebarBudget,
}

var sidebarWidgetSet = map[string]struct{}{
	SidebarAddStop:      {},
	SidebarBudget:       {},
	SidebarQuickSpends:  {},
	SidebarTabTotal:     {},
	SidebarAddTab:       {},
	SidebarAddChecklist: {},
}

// NormalizeMainSectionOrder parses a comma-separated saved order, drops unknown
// tokens, dedupes, then appends any missing keys in default order.
func NormalizeMainSectionOrder(raw string) []string {
	return normalizeOrder(raw, mainSectionSet, DefaultMainSectionOrder)
}

// NormalizeSidebarWidgetOrder is like NormalizeMainSectionOrder for sidebar keys.
func NormalizeSidebarWidgetOrder(raw string) []string {
	out := normalizeOrder(raw, sidebarWidgetSet, DefaultSidebarWidgetOrder)
	return ensureSidebarTabTotalBeforeAddTab(out)
}

// ensureSidebarTabTotalBeforeAddTab keeps the group-expense summary tile directly above
// the add form when both appear in the sidebar order (including legacy saved orders).
func ensureSidebarTabTotalBeforeAddTab(keys []string) []string {
	hasAdd := false
	for _, k := range keys {
		if k == SidebarAddTab {
			hasAdd = true
			break
		}
	}
	if !hasAdd {
		return keys
	}
	without := make([]string, 0, len(keys))
	for _, k := range keys {
		if k != SidebarTabTotal {
			without = append(without, k)
		}
	}
	out := make([]string, 0, len(without)+1)
	for _, k := range without {
		if k == SidebarAddTab {
			out = append(out, SidebarTabTotal)
		}
		out = append(out, k)
	}
	return out
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
	if !t.UIShowItinerary && key == SidebarAddStop {
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
	if t.UIShowItinerary && SidebarWidgetVisible(SidebarAddStop, t) {
		return true
	}
	if t.UIShowSpends && SidebarWidgetVisible(SidebarQuickSpends, t) {
		return true
	}
	if t.SectionEnabledTheTab() && SidebarWidgetVisible(SidebarAddTab, t) {
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

// TripDesktopCalendarFlyoutHasActions is true when the itinerary calendar “quick add” flyout
// would show at least one button (trip.html desktop-calendar-flyout).
func TripDesktopCalendarFlyoutHasActions(t Trip) bool {
	if t.UIShowItinerary && SidebarWidgetVisible(SidebarAddStop, t) {
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
	case SidebarBudget:
		return "Budget Limit"
	case SidebarQuickSpends:
		return "Add Expense"
	case SidebarTabTotal:
		return "Group Expense"
	case SidebarAddTab:
		return "Add Group Expense"
	case SidebarAddChecklist:
		return "Add Note / Checklist"
	default:
		return key
	}
}
