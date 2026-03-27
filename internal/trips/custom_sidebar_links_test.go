package trips

import (
	"testing"
)

func TestParseCustomSidebarLinksJSON(t *testing.T) {
	raw := `[{"label":"A","url":"https://a.example"},{"label":"B","url":"http://b.example"}]`
	got := ParseCustomSidebarLinksJSON(raw)
	if len(got) != 2 || got[0].Label != "A" || got[1].URL != "http://b.example" {
		t.Fatalf("got %#v", got)
	}
}

func TestParseCustomSidebarLinksJSON_maxThree(t *testing.T) {
	raw := `[{"label":"1","url":"https://1.example"},{"label":"2","url":"https://2.example"},{"label":"3","url":"https://3.example"},{"label":"4","url":"https://4.example"}]`
	got := ParseCustomSidebarLinksJSON(raw)
	if len(got) != 3 {
		t.Fatalf("len %d", len(got))
	}
}

func TestParseCustomSidebarLinksJSON_rejectNonHTTP(t *testing.T) {
	raw := `[{"label":"x","url":"javascript:alert(1)"}]`
	got := ParseCustomSidebarLinksJSON(raw)
	if len(got) != 0 {
		t.Fatalf("got %#v", got)
	}
}

func TestCustomSidebarLinksFromForm(t *testing.T) {
	vals := map[string]string{
		"custom_link_1_label": "One",
		"custom_link_1_url":   "https://one.example",
		"custom_link_2_label": "",
		"custom_link_2_url":   "",
		"custom_link_3_label": "Three",
		"custom_link_3_url":   "https://three.example",
	}
	got, err := CustomSidebarLinksFromForm("3,1", func(k string) string { return vals[k] })
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Label != "Three" || got[1].Label != "One" {
		t.Fatalf("got %#v", got)
	}
}

func TestDefaultAppSettings(t *testing.T) {
	d := DefaultAppSettings()
	if d.AppTitle == "" || d.MapDefaultZoom == 0 {
		t.Fatalf("%+v", d)
	}
}

func TestApplyDefaultTripUIPresets(t *testing.T) {
	tr := Trip{
		UIShowStay:           false,
		UIMainSectionOrder:   "map",
		UICustomSidebarLinks: `[{"label":"x","url":"https://x.com"}]`,
	}
	ApplyDefaultTripUIPresets(&tr)
	if !tr.UIShowStay || tr.UIMainSectionOrder != "" || tr.UICustomSidebarLinks != "" {
		t.Fatalf("%+v", tr)
	}
}
