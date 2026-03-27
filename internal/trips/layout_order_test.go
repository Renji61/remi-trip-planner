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
	tr := Trip{UIShowSpends: false, UIShowItinerary: true, UIShowChecklist: true}
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
	trNoCk := Trip{UIShowSpends: true, UIShowItinerary: true, UIShowChecklist: false}
	if SidebarWidgetVisible(SidebarAddChecklist, trNoCk) {
		t.Fatal("sidebar checklist hidden when checklist section off")
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
