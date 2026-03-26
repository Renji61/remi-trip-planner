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
	tr := Trip{UIShowSpends: false, UIShowStay: true}
	if MainSectionVisible(MainSectionSpends, tr) {
		t.Fatal("spends should hide")
	}
	if !MainSectionVisible(MainSectionMap, tr) {
		t.Fatal("map always on")
	}
}

func TestSidebarWidgetVisible(t *testing.T) {
	tr := Trip{UIShowSpends: false}
	if SidebarWidgetVisible(SidebarBudget, tr) {
		t.Fatal("budget hidden without spends")
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
