package trips

import (
	"strings"
	"testing"
)

func TestNormalizeMainSectionOrder_emptyUsesDefault(t *testing.T) {
	got := NormalizeMainSectionOrder("")
	if len(got) != len(DefaultMainSectionOrder) {
		t.Fatalf("len %d want %d", len(got), len(DefaultMainSectionOrder))
	}
	for i := range DefaultMainSectionOrder {
		if got[i] != DefaultMainSectionOrder[i] {
			t.Fatalf("idx %d: %q want %q", i, got[i], DefaultMainSectionOrder[i])
		}
	}
}

func TestNormalizeMainSectionOrder_reorderAndStripUnknown(t *testing.T) {
	raw := "flights,map,bogus,itinerary"
	got := NormalizeMainSectionOrder(raw)
	if got[0] != MainSectionFlights || got[1] != MainSectionMap || got[2] != MainSectionItinerary {
		t.Fatalf("prefix: %v", got)
	}
	if containsString(got, "bogus") {
		t.Fatal("unknown token leaked")
	}
	// rest filled from defaults
	if len(got) != len(DefaultMainSectionOrder) {
		t.Fatalf("len got %d", len(got))
	}
}

func TestNormalizeSidebarWidgetOrder(t *testing.T) {
	got := NormalizeSidebarWidgetOrder("checklist,budget")
	if got[0] != SidebarAddChecklist || got[1] != SidebarBudget {
		t.Fatalf("got %v", got)
	}
	if len(got) != len(DefaultSidebarWidgetOrder) {
		t.Fatalf("len %d", len(got))
	}
}

func TestNormalizeSidebarWidgetOrder_tabTotalBeforeAddTab(t *testing.T) {
	got := NormalizeSidebarWidgetOrder("add_tab,checklist")
	var idxTab, idxAdd int
	idxTab = -1
	idxAdd = -1
	for i, k := range got {
		if k == SidebarTabTotal {
			idxTab = i
		}
		if k == SidebarAddTab {
			idxAdd = i
		}
	}
	if idxTab < 0 || idxAdd < 0 || idxTab != idxAdd-1 {
		t.Fatalf("want tab_total immediately before add_tab, got %v", got)
	}
}

func TestJoinMainSectionOrder(t *testing.T) {
	s := JoinMainSectionOrder([]string{"map", "itinerary"})
	if s != "map,itinerary" {
		t.Fatal(s)
	}
}

func TestMainSectionVisible(t *testing.T) {
	tr := Trip{
		UIShowSpends: false, UIShowStay: true,
		UIShowItinerary: true, UIShowChecklist: true,
	}
	if MainSectionVisible(MainSectionSpends, tr) {
		t.Fatal("spends should hide")
	}
	if !MainSectionVisible(MainSectionMap, tr) {
		t.Fatal("map always on")
	}
	tr2 := Trip{
		UIMainSectionHidden: "map", UIShowStay: true, UIShowSpends: true,
		UIShowItinerary: true, UIShowChecklist: true,
	}
	if MainSectionVisible(MainSectionMap, tr2) {
		t.Fatal("map hidden via UIMainSectionHidden")
	}
	if !MainSectionVisible(MainSectionStay, tr2) {
		t.Fatal("stay still visible")
	}
	trIt := Trip{UIShowItinerary: false, UIShowChecklist: true, UIShowStay: true, UIShowSpends: true}
	if MainSectionVisible(MainSectionItinerary, trIt) {
		t.Fatal("itinerary off should hide main block")
	}
}

func TestSidebarWidgetVisible(t *testing.T) {
	tr := Trip{UIShowSpends: false, UIShowItinerary: true, UIShowChecklist: true, UIShowTheTab: true}
	if SidebarWidgetVisible(SidebarBudget, tr) {
		t.Fatal("budget hidden without spends")
	}
	tr2 := Trip{UIShowSpends: true, UISidebarWidgetHidden: "budget", UIShowItinerary: true, UIShowChecklist: true}
	if SidebarWidgetVisible(SidebarBudget, tr2) {
		t.Fatal("budget hidden via UISidebarWidgetHidden")
	}
	if !SidebarWidgetVisible(SidebarAddStop, tr2) {
		t.Fatal("add_stop visible")
	}
	trNoIt := Trip{UIShowSpends: true, UIShowItinerary: false, UIShowChecklist: true}
	if SidebarWidgetVisible(SidebarAddStop, trNoIt) {
		t.Fatal("add_stop hidden when itinerary off")
	}
	if SidebarWidgetVisible(SidebarAddCommute, trNoIt) {
		t.Fatal("add_commute hidden when itinerary off")
	}
	trNoCk := Trip{UIShowSpends: true, UIShowItinerary: true, UIShowChecklist: false}
	if SidebarWidgetVisible(SidebarAddChecklist, trNoCk) {
		t.Fatal("sidebar checklist hidden when checklist section off")
	}
	trTab := Trip{UIShowSpends: true, UIShowTheTab: true, UIShowItinerary: true, UIShowChecklist: true}
	if !SidebarWidgetVisible(SidebarAddTab, trTab) {
		t.Fatal("add_tab visible when spends+tab on")
	}
	if SidebarWidgetVisible(SidebarAddTab, Trip{UIShowSpends: true, UIShowTheTab: false, UIShowItinerary: true, UIShowChecklist: true}) {
		t.Fatal("add_tab hidden when Group Expenses section off")
	}
	if !SidebarWidgetVisible(SidebarTabTotal, trTab) {
		t.Fatal("tab_total visible when spends+tab on")
	}
	if SidebarWidgetVisible(SidebarTabTotal, Trip{UIShowSpends: true, UIShowTheTab: false, UIShowItinerary: true, UIShowChecklist: true}) {
		t.Fatal("tab_total hidden when Group Expenses section off")
	}
	if SidebarWidgetVisible(SidebarTabTotal, Trip{UIShowSpends: false, UIShowTheTab: true, UIShowItinerary: true, UIShowChecklist: true}) {
		t.Fatal("tab_total hidden when spends off")
	}
}

func TestTripMobileFabHasItems(t *testing.T) {
	allOff := Trip{}
	if TripMobileFabHasItems(allOff) {
		t.Fatal("expected no FAB items when all sections off")
	}
	notesOnly := Trip{UIShowChecklist: true}
	if !TripMobileFabHasItems(notesOnly) {
		t.Fatal("notes section alone should show FAB")
	}
	stayOnly := Trip{UIShowStay: true}
	if !TripMobileFabHasItems(stayOnly) {
		t.Fatal("stay alone should show FAB")
	}
	docsOnly := Trip{UIShowDocuments: true}
	if !TripMobileFabHasItems(docsOnly) {
		t.Fatal("trip documents alone should show FAB")
	}
	itinHiddenStop := Trip{UIShowItinerary: true, UISidebarWidgetHidden: "add_stop,add_commute"}
	if TripMobileFabHasItems(itinHiddenStop) {
		t.Fatal("add_stop and add_commute hidden: no itinerary FAB entries")
	}
	itinStopHiddenCommuteOn := Trip{UIShowItinerary: true, UISidebarWidgetHidden: "add_stop"}
	if !TripMobileFabHasItems(itinStopHiddenCommuteOn) {
		t.Fatal("add_commute still visible should keep FAB")
	}
	checklistWidgetHidden := Trip{UIShowChecklist: true, UISidebarWidgetHidden: "checklist"}
	if TripMobileFabHasItems(checklistWidgetHidden) {
		t.Fatal("checklist widget hidden and no Notes FAB link: expect no FAB")
	}
}

func TestTripDesktopCalendarFlyoutHasActions(t *testing.T) {
	if TripDesktopCalendarFlyoutHasActions(Trip{}) {
		t.Fatal("expected no flyout actions")
	}
	tr := Trip{UIShowItinerary: true, UIShowStay: true}
	if !TripDesktopCalendarFlyoutHasActions(tr) {
		t.Fatal("itinerary+stay should enable flyout")
	}
	spendsOnly := Trip{UIShowItinerary: false, UIShowSpends: true}
	if TripDesktopCalendarFlyoutHasActions(spendsOnly) {
		t.Fatal("quick spends alone should not enable calendar flyout (no flyout buttons)")
	}
	tabOnly := Trip{UIShowItinerary: false, UIShowSpends: true, UIShowTheTab: true}
	if TripDesktopCalendarFlyoutHasActions(tabOnly) {
		t.Fatal("the tab + spends without itinerary sections should not enable calendar flyout")
	}
	checklistOnly := Trip{UIShowChecklist: true, UIShowItinerary: false}
	if TripDesktopCalendarFlyoutHasActions(checklistOnly) {
		t.Fatal("checklist-only visibility should not enable calendar flyout")
	}
	commuteOnly := Trip{UIShowItinerary: true, UISidebarWidgetHidden: "add_stop"}
	if !TripDesktopCalendarFlyoutHasActions(commuteOnly) {
		t.Fatal("add_commute visible without add_stop should still enable calendar flyout")
	}
}

func TestDefaultMainSectionOrder_tripDetailsSequence(t *testing.T) {
	wantPrefix := []string{
		MainSectionMap,
		MainSectionItinerary,
		MainSectionFlights,
		MainSectionStay,
		MainSectionVehicle,
		MainSectionChecklist,
		MainSectionSpends,
		MainSectionTheTab,
	}
	if len(DefaultMainSectionOrder) != len(wantPrefix) {
		t.Fatalf("len %d want %d", len(DefaultMainSectionOrder), len(wantPrefix))
	}
	for i := range wantPrefix {
		if DefaultMainSectionOrder[i] != wantPrefix[i] {
			t.Fatalf("idx %d: %q want %q", i, DefaultMainSectionOrder[i], wantPrefix[i])
		}
	}
}

func TestSidebarWidgetLabel_renames(t *testing.T) {
	if got := SidebarWidgetLabel(SidebarAddStop); got != "Add Stop" {
		t.Fatalf("add_stop: %q", got)
	}
	if got := SidebarWidgetLabel(SidebarAddChecklist); got != "Add Note / Checklist" {
		t.Fatalf("checklist: %q", got)
	}
	if got := SidebarWidgetLabel(SidebarAddCommute); got != "Add commute leg" {
		t.Fatalf("add_commute: %q", got)
	}
}

func TestMainSectionTheTabVisibility(t *testing.T) {
	tr := Trip{UIShowTheTab: true, UIShowSpends: true}
	if !MainSectionVisible(MainSectionTheTab, tr) {
		t.Fatal("the_tab visible")
	}
	if MainSectionVisible(MainSectionTheTab, Trip{UIShowTheTab: false, UIShowSpends: true}) {
		t.Fatal("the_tab off hides")
	}
	if MainSectionVisible(MainSectionTheTab, Trip{UIShowTheTab: true, UIShowSpends: false}) {
		t.Fatal("spends off hides the_tab")
	}
}

func FuzzNormalizeMainSectionOrder(f *testing.F) {
	f.Add("map,itinerary,spends")
	f.Fuzz(func(t *testing.T, s string) {
		if len(s) > 2000 {
			t.Skip()
		}
		out := NormalizeMainSectionOrder(s)
		if len(out) != len(DefaultMainSectionOrder) {
			t.Fatalf("len")
		}
		joined := strings.Join(out, ",")
		out2 := NormalizeMainSectionOrder(joined)
		if strings.Join(out2, ",") != joined {
			t.Fatalf("idempotent: %q -> %q -> %q", s, joined, strings.Join(out2, ","))
		}
	})
}
