package trips

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
)

// Participant key prefixes for users (registered) and trip guests.
const (
	TabParticipantUserPrefix  = "user:"
	TabParticipantGuestPrefix = "guest:"
)

// Tab split modes stored on expenses and trip defaults.
const (
	TabSplitEqual     = "equal"
	TabSplitExact     = "exact"
	TabSplitPercent   = "percent"
	TabSplitShares    = "shares"
	TabSplitModeEmpty = ""
)

// TabSplitPayload is stored in Expense.SplitJSON and trip TabDefaultSplitJSON.
type TabSplitPayload struct {
	Participants []string           `json:"participants"`
	Weights      map[string]float64 `json:"weights,omitempty"`
}

// TabTransfer is one directed payment in a simplified settlement plan.
type TabTransfer struct {
	FromKey string
	ToKey   string
	Amount  float64
}

// TabBalanceView is used for net vs gross debt UI.
type TabBalanceView struct {
	NetByKey      map[string]float64
	TotalOwedOut  float64 // sum of max(0, -net) — total debt magnitude
	TotalOwedIn   float64 // sum of max(0, net)
	Simplified    []TabTransfer
	LegacyExpense bool
}

func ParticipantKeyUser(userID string) string {
	return TabParticipantUserPrefix + strings.TrimSpace(userID)
}

func ParticipantKeyGuest(guestID string) string {
	return TabParticipantGuestPrefix + strings.TrimSpace(guestID)
}

func ParseParticipantKey(key string) (kind string, id string, ok bool) {
	key = strings.TrimSpace(key)
	switch {
	case strings.HasPrefix(key, TabParticipantUserPrefix):
		return "user", strings.TrimPrefix(key, TabParticipantUserPrefix), true
	case strings.HasPrefix(key, TabParticipantGuestPrefix):
		return "guest", strings.TrimPrefix(key, TabParticipantGuestPrefix), true
	default:
		return "", "", false
	}
}

// TabSettlementParticipantKey normalizes a value stored or submitted for tab settlement payer/payee.
// Accepts "user:id", "guest:id", or a bare user id (legacy) and returns the canonical participant key.
func TabSettlementParticipantKey(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	if kind, id, ok := ParseParticipantKey(s); ok {
		id = strings.TrimSpace(id)
		if id == "" {
			return ""
		}
		switch kind {
		case "user":
			return ParticipantKeyUser(id)
		case "guest":
			return ParticipantKeyGuest(id)
		}
	}
	return ParticipantKeyUser(s)
}

// NormalizeTabSplitPayload validates and returns a copy with consistent participants.
func NormalizeTabSplitPayload(mode string, amount float64, rawJSON string, allowedKeys map[string]struct{}) (TabSplitPayload, error) {
	mode = strings.TrimSpace(strings.ToLower(mode))
	if mode == TabSplitModeEmpty {
		mode = TabSplitEqual
	}
	var p TabSplitPayload
	if strings.TrimSpace(rawJSON) != "" {
		if err := json.Unmarshal([]byte(rawJSON), &p); err != nil {
			return p, errors.New("invalid split data")
		}
	}
	// Deduplicate participants preserving order
	seen := map[string]struct{}{}
	var parts []string
	for _, k := range p.Participants {
		k = strings.TrimSpace(k)
		if k == "" {
			continue
		}
		if allowedKeys != nil {
			if _, ok := allowedKeys[k]; !ok {
				return p, fmt.Errorf("unknown participant %q", k)
			}
		}
		if _, ok := seen[k]; ok {
			continue
		}
		seen[k] = struct{}{}
		parts = append(parts, k)
	}
	p.Participants = parts
	if len(p.Participants) == 0 {
		return p, errors.New("choose at least one person in the split")
	}
	switch mode {
	case TabSplitEqual:
		return p, nil
	case TabSplitExact:
		if p.Weights == nil {
			return p, errors.New("exact split needs an amount per person")
		}
		var sum float64
		for _, k := range p.Participants {
			w := p.Weights[k]
			if w < 0 || math.IsNaN(w) {
				return p, errors.New("invalid exact amount")
			}
			sum += w
		}
		if math.Abs(sum-amount) > 0.02 && amount > 0 {
			return p, errors.New("exact amounts must add up to the expense total")
		}
		return p, nil
	case TabSplitPercent:
		if p.Weights == nil {
			return p, errors.New("percent split needs a percentage per person")
		}
		var sum float64
		for _, k := range p.Participants {
			w := p.Weights[k]
			if w < 0 || math.IsNaN(w) {
				return p, errors.New("invalid percentage")
			}
			sum += w
		}
		if math.Abs(sum-100) > 0.05 && sum > 0 {
			return p, errors.New("percentages must add up to 100")
		}
		return p, nil
	case TabSplitShares:
		if p.Weights == nil {
			return p, errors.New("shares split needs a share count per person")
		}
		for _, k := range p.Participants {
			w := p.Weights[k]
			if w <= 0 || math.IsNaN(w) {
				return p, errors.New("each share count must be greater than zero")
			}
		}
		return p, nil
	default:
		return p, errors.New("unknown split method")
	}
}

// mergeParticipantKeysFromExpense adds keys referenced by stored split JSON and effective payer
// so historical expenses stay valid after someone leaves the trip.
func mergeParticipantKeysFromExpense(e Expense, tripOwnerUserID string, allowed map[string]struct{}) {
	if allowed == nil {
		return
	}
	if k := strings.TrimSpace(EffectivePaidBy(e, tripOwnerUserID)); k != "" {
		allowed[k] = struct{}{}
	}
	raw := strings.TrimSpace(e.SplitJSON)
	if raw == "" {
		return
	}
	var p TabSplitPayload
	if err := json.Unmarshal([]byte(raw), &p); err != nil {
		return
	}
	for _, k := range p.Participants {
		k = strings.TrimSpace(k)
		if k != "" {
			allowed[k] = struct{}{}
		}
	}
	for k := range p.Weights {
		k = strings.TrimSpace(k)
		if k != "" {
			allowed[k] = struct{}{}
		}
	}
}

// SharesForExpense returns each participant's owed share for one expense (FromTab only).
func SharesForExpense(e Expense, partyUserIDs []string, guestIDs []string, tripOwnerUserID string) (map[string]float64, error) {
	if !e.FromTab || e.Amount <= 0 {
		return map[string]float64{}, nil
	}
	allowed := map[string]struct{}{}
	for _, uid := range partyUserIDs {
		allowed[ParticipantKeyUser(uid)] = struct{}{}
	}
	for _, gid := range guestIDs {
		allowed[ParticipantKeyGuest(gid)] = struct{}{}
	}
	mergeParticipantKeysFromExpense(e, tripOwnerUserID, allowed)
	mode := strings.TrimSpace(strings.ToLower(e.SplitMode))
	if mode == "" && strings.TrimSpace(e.SplitJSON) == "" {
		// Legacy: equal among registered trip members only.
		out := map[string]float64{}
		if len(partyUserIDs) == 0 {
			return out, nil
		}
		share := e.Amount / float64(len(partyUserIDs))
		for _, uid := range partyUserIDs {
			out[ParticipantKeyUser(uid)] = roundMoney(share)
		}
		return fixRoundingDrift(out, e.Amount), nil
	}
	payload, err := NormalizeTabSplitPayload(mode, e.Amount, e.SplitJSON, allowed)
	if err != nil {
		return nil, err
	}
	mode = strings.TrimSpace(strings.ToLower(e.SplitMode))
	if mode == "" {
		mode = TabSplitEqual
	}
	switch mode {
	case TabSplitEqual:
		n := float64(len(payload.Participants))
		share := e.Amount / n
		out := map[string]float64{}
		for _, k := range payload.Participants {
			out[k] = roundMoney(share)
		}
		return fixRoundingDrift(out, e.Amount), nil
	case TabSplitExact:
		out := map[string]float64{}
		for _, k := range payload.Participants {
			out[k] = roundMoney(payload.Weights[k])
		}
		return fixRoundingDrift(out, e.Amount), nil
	case TabSplitPercent:
		out := map[string]float64{}
		for _, k := range payload.Participants {
			out[k] = roundMoney(e.Amount * payload.Weights[k] / 100)
		}
		return fixRoundingDrift(out, e.Amount), nil
	case TabSplitShares:
		var totalW float64
		for _, k := range payload.Participants {
			totalW += payload.Weights[k]
		}
		out := map[string]float64{}
		for _, k := range payload.Participants {
			out[k] = roundMoney(e.Amount * payload.Weights[k] / totalW)
		}
		return fixRoundingDrift(out, e.Amount), nil
	default:
		return nil, errors.New("unknown split method")
	}
}

func roundMoney(x float64) float64 {
	return math.Round(x*100) / 100
}

func fixRoundingDrift(m map[string]float64, target float64) map[string]float64 {
	var sum float64
	for _, v := range m {
		sum += v
	}
	drift := roundMoney(target - sum)
	if math.Abs(drift) < 0.001 {
		return m
	}
	// Add drift to lexicographically last key so totals match.
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	if len(keys) == 0 {
		return m
	}
	last := keys[len(keys)-1]
	m[last] = roundMoney(m[last] + drift)
	return m
}

// EffectivePaidBy returns PaidBy or legacy fallback to trip owner as user:key.
func EffectivePaidBy(e Expense, tripOwnerUserID string) string {
	if s := strings.TrimSpace(e.PaidBy); s != "" {
		return s
	}
	if tripOwnerUserID != "" {
		return ParticipantKeyUser(tripOwnerUserID)
	}
	return ""
}

// TabLedger builds running balances from tab expenses and settlements (members and guests).
func TabLedger(tabExpenses []Expense, partyUserIDs []string, guestIDs []string, settlements []TabSettlement, tripOwnerUserID string) (map[string]float64, error) {
	bal := map[string]float64{}
	for _, e := range tabExpenses {
		if !e.FromTab {
			continue
		}
		shares, err := SharesForExpense(e, partyUserIDs, guestIDs, tripOwnerUserID)
		if err != nil {
			return nil, err
		}
		payer := EffectivePaidBy(e, tripOwnerUserID)
		if payer == "" {
			continue
		}
		bal[payer] = roundMoney(bal[payer] + e.Amount)
		for k, sh := range shares {
			bal[k] = roundMoney(bal[k] - sh)
		}
	}
	for _, s := range settlements {
		pk := TabSettlementParticipantKey(s.PayerUserID)
		qk := TabSettlementParticipantKey(s.PayeeUserID)
		if pk == "" || qk == "" {
			continue
		}
		bal[pk] = roundMoney(bal[pk] + s.Amount)
		bal[qk] = roundMoney(bal[qk] - s.Amount)
	}
	return bal, nil
}

func moneyToCents(x float64) int64 {
	return int64(math.Round(roundMoney(x) * 100))
}

func centsToMoney(c int64) float64 {
	return roundMoney(float64(c) / 100)
}

func absInt64(x int64) int64 {
	if x < 0 {
		return -x
	}
	return x
}

// netBalancesToCentsNormalized converts net balances to integer cents and fixes any 1¢ sum drift
// on the lexicographically last key so total debt cents equals total credit cents (same basis as SimplifyDebts).
func netBalancesToCentsNormalized(net map[string]float64) (bal map[string]int64, keys []string) {
	bal = make(map[string]int64, len(net))
	keys = make([]string, 0, len(net))
	for k, v := range net {
		bal[k] = moneyToCents(v)
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var sum int64
	for _, k := range keys {
		sum += bal[k]
	}
	if sum != 0 && len(keys) > 0 {
		last := keys[len(keys)-1]
		bal[last] -= sum
	}
	return bal, keys
}

// SimplifyDebts runs greedy pairwise settlement to minimize transfer count.
// All arithmetic uses integer cents so suggested payments sum exactly to net balances (no float drift).
func SimplifyDebts(net map[string]float64, eps float64) []TabTransfer {
	if eps <= 0 {
		eps = 0.01
	}
	epsCents := int64(math.Round(eps * 100))
	if epsCents < 1 {
		epsCents = 1
	}
	bal, keys := netBalancesToCentsNormalized(net)

	type slot struct {
		key string
		c   int64 // negative = debtor, positive = creditor (cents)
	}
	debtors := make([]slot, 0)
	creditors := make([]slot, 0)
	for _, k := range keys {
		c := bal[k]
		if absInt64(c) < epsCents {
			continue
		}
		if c < 0 {
			debtors = append(debtors, slot{k, c})
		} else {
			creditors = append(creditors, slot{k, c})
		}
	}
	sort.Slice(debtors, func(i, j int) bool { return debtors[i].c < debtors[j].c })
	sort.Slice(creditors, func(i, j int) bool { return creditors[i].c > creditors[j].c })

	var out []TabTransfer
	di, ci := 0, 0
	for di < len(debtors) && ci < len(creditors) {
		dOwe := -debtors[di].c
		cGet := creditors[ci].c
		x := dOwe
		if cGet < x {
			x = cGet
		}
		if x < epsCents {
			break
		}
		out = append(out, TabTransfer{
			FromKey: debtors[di].key,
			ToKey:   creditors[ci].key,
			Amount:  centsToMoney(x),
		})
		debtors[di].c += x
		creditors[ci].c -= x
		if absInt64(debtors[di].c) < epsCents {
			di++
		}
		if absInt64(creditors[ci].c) < epsCents {
			ci++
		}
	}
	return out
}

// TabDebtTotals returns magnitudes for "total debt" views (same cent normalization as SimplifyDebts).
func TabDebtTotals(net map[string]float64) (owedOut, owedIn float64) {
	bal, keys := netBalancesToCentsNormalized(net)
	var outC, inC int64
	for _, k := range keys {
		c := bal[k]
		if c < 0 {
			outC += -c
		} else if c > 0 {
			inC += c
		}
	}
	return centsToMoney(outC), centsToMoney(inC)
}
