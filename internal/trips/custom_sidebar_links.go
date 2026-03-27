package trips

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const MaxCustomSidebarLinks = 3

// CustomSidebarLink is a user-defined link shown in the trip page sidebar (desktop).
type CustomSidebarLink struct {
	Label string `json:"label"`
	URL   string `json:"url"`
}

// ParseCustomSidebarLinksJSON decodes stored JSON; invalid input yields nil.
func ParseCustomSidebarLinksJSON(raw string) []CustomSidebarLink {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}
	var out []CustomSidebarLink
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return NormalizeCustomSidebarLinksSlice(out)
}

// NormalizeCustomSidebarLinksSlice keeps at most MaxCustomSidebarLinks valid entries.
func NormalizeCustomSidebarLinksSlice(in []CustomSidebarLink) []CustomSidebarLink {
	var out []CustomSidebarLink
	for _, l := range in {
		l.Label = strings.TrimSpace(l.Label)
		l.URL = strings.TrimSpace(l.URL)
		if l.URL == "" {
			continue
		}
		if err := validateCustomSidebarURL(l.URL); err != nil {
			continue
		}
		if l.Label == "" {
			l.Label = l.URL
		}
		out = append(out, l)
		if len(out) >= MaxCustomSidebarLinks {
			break
		}
	}
	return out
}

func validateCustomSidebarURL(s string) error {
	u, err := url.Parse(s)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return fmt.Errorf("invalid url")
	}
	scheme := strings.ToLower(u.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("only http/https")
	}
	return nil
}

// EncodeCustomSidebarLinksJSON encodes links to JSON for storage.
func EncodeCustomSidebarLinksJSON(links []CustomSidebarLink) string {
	links = NormalizeCustomSidebarLinksSlice(links)
	if len(links) == 0 {
		return ""
	}
	b, err := json.Marshal(links)
	if err != nil {
		return ""
	}
	return string(b)
}

// CustomSidebarLinksFromForm reads ordered slot indices (e.g. "2,1,3") and custom_link_N_label/url fields.
func CustomSidebarLinksFromForm(slotOrder string, formValues func(key string) string) ([]CustomSidebarLink, error) {
	slotOrder = strings.TrimSpace(slotOrder)
	if slotOrder == "" {
		slotOrder = "1,2,3"
	}
	var links []CustomSidebarLink
	for _, part := range strings.Split(slotOrder, ",") {
		slot := strings.TrimSpace(part)
		if slot == "" {
			continue
		}
		label := strings.TrimSpace(formValues("custom_link_" + slot + "_label"))
		u := strings.TrimSpace(formValues("custom_link_" + slot + "_url"))
		if label == "" && u == "" {
			continue
		}
		if u == "" {
			return nil, fmt.Errorf("each custom link needs a valid http or https URL")
		}
		if err := validateCustomSidebarURL(u); err != nil {
			return nil, fmt.Errorf("custom link URL must be http or https")
		}
		if label == "" {
			label = u
		}
		links = append(links, CustomSidebarLink{Label: label, URL: u})
		if len(links) >= MaxCustomSidebarLinks {
			break
		}
	}
	return NormalizeCustomSidebarLinksSlice(links), nil
}

// DefaultCustomLinkSlotOrder is the default hidden field value for three slots.
const DefaultCustomLinkSlotOrder = "1,2,3"

// CustomLinkSlotsForTemplate returns up to 3 slots for trip settings (label/url by slot id 1..3).
func CustomLinkSlotsForTemplate(rawJSON string) [3]struct{ Label, URL string } {
	var slots [3]struct{ Label, URL string }
	links := ParseCustomSidebarLinksJSON(rawJSON)
	for i := range links {
		if i >= 3 {
			break
		}
		slots[i].Label = links[i].Label
		slots[i].URL = links[i].URL
	}
	return slots
}

// CustomLinkEditorSlot is one row in trip settings (reorder + inputs).
type CustomLinkEditorSlot struct {
	N     int
	Label string
	URL   string
}

// CustomLinkEditorSlots returns three editor rows from stored JSON (order is always 1..3 on load).
func CustomLinkEditorSlots(rawJSON string) []CustomLinkEditorSlot {
	s := CustomLinkSlotsForTemplate(rawJSON)
	return []CustomLinkEditorSlot{
		{1, s[0].Label, s[0].URL},
		{2, s[1].Label, s[1].URL},
		{3, s[2].Label, s[2].URL},
	}
}
