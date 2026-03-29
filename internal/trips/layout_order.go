package trips

import (
	"strings"
)

// Main trip page section keys (hero + edit panel remain fixed above these).
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

// DefaultMainSectionOrder matches the historical trip page layout.
var DefaultMainSectionOrder = []string{
	MainSectionMap,
	MainSectionItinerary,
	MainSectionSpends,
	MainSectionTheTab,
	MainSectionChecklist,
	MainSectionStay,
	MainSectionVehicle,
	MainSectionFlights,
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
	SidebarAddTab       = "add_tab"
	SidebarAddChecklist = "checklist"
)

var DefaultSidebarWidgetOrder = []string{
	SidebarAddStop,
	SidebarBudget,
	SidebarQuickSpends,
	SidebarAddTab,
	SidebarAddChecklist,
}

var sidebarWidgetSet = map[string]struct{}{
	SidebarAddStop:      {},
	SidebarBudget:       {},
	SidebarQuickSpends:  {},
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
	if !t.UIShowSpends && (key == SidebarBudget || key == SidebarQuickSpends || key == SidebarAddTab) {
		return false
	}
	if !t.UIShowTheTab && key == SidebarAddTab {
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
		return "Reminder Checklist"
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
		return "Add New Stop"
	case SidebarBudget:
		return "Budget Limit"
	case SidebarQuickSpends:
		return "Quick Expenses"
	case SidebarAddTab:
		return "Add to Group Expenses"
	case SidebarAddChecklist:
		return "Add to Checklist"
	default:
		return key
	}
}
