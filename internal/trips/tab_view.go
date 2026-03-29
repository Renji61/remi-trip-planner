package trips

import (
	"fmt"
	"strings"
)

// TabBalanceParticipantView is one row in the Tab "net balance" card list.
type TabBalanceParticipantView struct {
	DisplayName string
	Role        string // "Owner", "Member", or "Guest"
	IsGuest     bool
	Net         float64
	AvatarPath  string // profile image path for members; empty for guests
	Initial     string // letter(s) for avatar fallback
	NetClass    string // tab-net--neg | tab-net--pos | tab-net--zero (UI)
	NetDisplay  string // currency symbol already applied
}

// GuestInitialFromDisplayName returns one letter for guest avatar chips.
func GuestInitialFromDisplayName(name string) string {
	return UserProfile{DisplayName: strings.TrimSpace(name)}.InitialForAvatar()
}

const tabNetEpsilon = 0.02

// TabNetDisplay formats net balance for Tab UI (owe / owed / even).
func TabNetDisplay(currencySymbol string, net float64) (netClass, display string) {
	sym := strings.TrimSpace(currencySymbol)
	if sym == "" {
		sym = "$"
	}
	switch {
	case net < -tabNetEpsilon:
		return "tab-net--neg", fmt.Sprintf("-%s%.2f", sym, -net)
	case net > tabNetEpsilon:
		return "tab-net--pos", fmt.Sprintf("+%s%.2f", sym, net)
	default:
		return "tab-net--zero", fmt.Sprintf("%s0.00", sym)
	}
}
