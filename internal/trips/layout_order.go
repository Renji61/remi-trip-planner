package trips

import (
	"strings"
)

// Main trip page section keys (hero + edit panel stay fixed above these).
const (
	MainSectionMap       = "map"
	MainSectionItinerary = "itinerary"
	MainSectionSpends    = "spends"
	MainSectionChecklist = "checklist"
	MainSectionStay      = "stay"
	MainSectionVehicle   = "vehicle"
	MainSectionFlights   = "flights"
)

// DefaultMainSectionOrder matches the historical trip page layout.
var DefaultMainSectionOrder = []string{
	MainSectionMap,
	MainSectionItinerary,
	MainSectionSpends,
	MainSectionChecklist,
	MainSectionStay,
	MainSectionVehicle,
	MainSectionFlights,
}

var mainSectionSet = map[string]struct{}{
	MainSectionMap:       {},
	MainSectionItinerary: {},
	MainSectionSpends:    {},
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
	SidebarAddChecklist = "checklist"
)

var DefaultSidebarWidgetOrder = []string{
	SidebarAddStop,
	SidebarBudget,
	SidebarQuickSpends,
	SidebarAddChecklist,
}

var sidebarWidgetSet = map[string]struct{}{
	SidebarAddStop:      {},
	SidebarBudget:       {},
	SidebarQuickSpends:  {},
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

// MainSectionVisible returns whether a normalized main section should render for this trip.
func MainSectionVisible(key string, t Trip) bool {
	switch key {
	case MainSectionSpends:
		return t.UIShowSpends
	case MainSectionStay:
		return t.UIShowStay
	case MainSectionVehicle:
		return t.UIShowVehicle
	case MainSectionFlights:
		return t.UIShowFlights
	default:
		return true
	}
}

// SidebarWidgetVisible returns whether a sidebar widget should render.
func SidebarWidgetVisible(key string, t Trip) bool {
	switch key {
	case SidebarBudget, SidebarQuickSpends:
		return t.UIShowSpends
	default:
		return true
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
		return "Spends"
	case MainSectionChecklist:
		return "Reminder Checklist"
	case MainSectionStay:
		return "Stay"
	case MainSectionVehicle:
		return "Vehicle"
	case MainSectionFlights:
		return "Flights"
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
		return "Total Budgeted Cost"
	case SidebarQuickSpends:
		return "Quick Spends"
	case SidebarAddChecklist:
		return "Add to Checklist"
	default:
		return key
	}
}
