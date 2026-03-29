package trips

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"testing"
)

func TestTabDebtTotals_matchesSimplifiedSum(t *testing.T) {
	net := map[string]float64{
		"user:a": -10.01,
		"user:b": -9.99,
		"user:c": 20.0,
	}
	out, in := TabDebtTotals(net)
	tr := SimplifyDebts(net, 0.01)
	var sum float64
	for _, x := range tr {
		sum = roundMoney(sum + x.Amount)
	}
	if absInt64(moneyToCents(out)-moneyToCents(in)) > 1 {
		t.Fatalf("out %v vs in %v", out, in)
	}
	if absInt64(moneyToCents(sum)-moneyToCents(out)) > 1 {
		t.Fatalf("transfer sum %v vs owed out %v transfers=%+v", sum, out, tr)
	}
}

func TestSimplifyDebts_conservesCents(t *testing.T) {
	net := map[string]float64{
		"user:a": -10.01,
		"user:b": -9.99,
		"user:c": 20.0,
	}
	tr := SimplifyDebts(net, 0.01)
	// Simulate applying transfers on cent balances.
	bal := map[string]int64{}
	for k, v := range net {
		bal[k] = moneyToCents(v)
	}
	var sk int64
	for _, c := range bal {
		sk += c
	}
	if sk != 0 && len(bal) > 0 {
		var ks []string
		for k := range bal {
			ks = append(ks, k)
		}
		sort.Strings(ks)
		bal[ks[len(ks)-1]] -= sk
	}
	for _, x := range tr {
		bal[x.FromKey] += moneyToCents(x.Amount)
		bal[x.ToKey] -= moneyToCents(x.Amount)
	}
	for k, c := range bal {
		if absInt64(c) >= 1 {
			t.Fatalf("non-zero remain %s=%d after transfers %+v", k, c, tr)
		}
	}
}

func TestSimplifyDebts_chain(t *testing.T) {
	// A owes 10 to B, B owes 10 to C → net: A:-10, B:0, C:+10 → A pays C 10
	net := map[string]float64{
		"user:a": -10,
		"user:b": 0,
		"user:c": 10,
	}
	tr := SimplifyDebts(net, 0.01)
	if len(tr) != 1 {
		t.Fatalf("expected 1 transfer, got %+v", tr)
	}
	if tr[0].FromKey != "user:a" || tr[0].ToKey != "user:c" || math.Abs(tr[0].Amount-10) > 0.02 {
		t.Fatalf("unexpected transfer %+v", tr[0])
	}
}

func TestSharesEqual(t *testing.T) {
	e := Expense{
		FromTab:   true,
		Amount:    100,
		SplitMode: TabSplitEqual,
		SplitJSON: `{"participants":["user:u1","user:u2"]}`,
	}
	sh, err := SharesForExpense(e, []string{"u1", "u2", "u3"}, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(sh["user:u1"]-50) > 0.02 || math.Abs(sh["user:u2"]-50) > 0.02 {
		t.Fatalf("shares %+v", sh)
	}
}

func TestSharesForExpense_guestRemovedFromTripStillParsesSplit(t *testing.T) {
	e := Expense{
		FromTab:   true,
		Amount:    100,
		SplitMode: TabSplitEqual,
		SplitJSON: `{"participants":["user:a","guest:gone"]}`,
		PaidBy:    "user:a",
	}
	sh, err := SharesForExpense(e, []string{"a"}, nil, "a")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(sh["user:a"]-50) > 0.02 || math.Abs(sh["guest:gone"]-50) > 0.02 {
		t.Fatalf("shares %+v", sh)
	}
}

func TestTabLedger_settlement(t *testing.T) {
	ex := []Expense{
		{FromTab: true, Amount: 100, PaidBy: "user:a", SplitMode: TabSplitEqual, SplitJSON: `{"participants":["user:a","user:b"]}`},
	}
	sett := []TabSettlement{{PayerUserID: "b", PayeeUserID: "a", Amount: 50}}
	bal, err := TabLedger(ex, []string{"a", "b"}, nil, sett, "a")
	if err != nil {
		t.Fatal(err)
	}
	// a paid 100, owes 50 share → net +50 before settlement; b owes -50; settlement b pays a 50 → b:0, a:0 approx
	if math.Abs(bal["user:a"]) > 0.05 || math.Abs(bal["user:b"]) > 0.05 {
		t.Fatalf("balance %+v", bal)
	}
}

func TestTabSettlementParticipantKey(t *testing.T) {
	if got := TabSettlementParticipantKey("abc"); got != "user:abc" {
		t.Fatalf("legacy bare id: got %q", got)
	}
	if got := TabSettlementParticipantKey("user:abc"); got != "user:abc" {
		t.Fatalf("explicit user: got %q", got)
	}
	if got := TabSettlementParticipantKey("guest:g1"); got != "guest:g1" {
		t.Fatalf("guest: got %q", got)
	}
	if TabSettlementParticipantKey("") != "" {
		t.Fatal("empty")
	}
}

func TestTabLedger_settlement_guest(t *testing.T) {
	ex := []Expense{
		{FromTab: true, Amount: 90, PaidBy: "user:owner", SplitMode: TabSplitEqual, SplitJSON: `{"participants":["user:owner","guest:guest1"]}`},
	}
	sett := []TabSettlement{{PayerUserID: "guest:guest1", PayeeUserID: "user:owner", Amount: 45}}
	bal, err := TabLedger(ex, []string{"owner"}, []string{"guest1"}, sett, "owner")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(bal["user:owner"]) > 0.05 || math.Abs(bal["guest:guest1"]) > 0.05 {
		t.Fatalf("balance %+v", bal)
	}
}

func TestNormalizeTabSplitPayload_percent(t *testing.T) {
	allowed := map[string]struct{}{"user:x": {}, "user:y": {}}
	raw := `{"participants":["user:x","user:y"],"weights":{"user:x":40,"user:y":60}}`
	_, err := NormalizeTabSplitPayload(TabSplitPercent, 0, raw, allowed)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTabSplitPayload_roundtrip(t *testing.T) {
	p := TabSplitPayload{
		Participants: []string{"user:1", "guest:g1"},
		Weights:      map[string]float64{"user:1": 2, "guest:g1": 1},
	}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var out TabSplitPayload
	if err := json.Unmarshal(b, &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Participants) != 2 {
		t.Fatal(out)
	}
}

// tenMixedParty is 5 registered users + 5 guests (10 participants).
func tenMixedParty() (partyUserIDs, guestIDs []string, keys []string) {
	partyUserIDs = []string{"u0", "u1", "u2", "u3", "u4"}
	guestIDs = []string{"g0", "g1", "g2", "g3", "g4"}
	keys = make([]string, 0, 10)
	for _, id := range partyUserIDs {
		keys = append(keys, ParticipantKeyUser(id))
	}
	for _, id := range guestIDs {
		keys = append(keys, ParticipantKeyGuest(id))
	}
	return partyUserIDs, guestIDs, keys
}

func sumShareMap(m map[string]float64) float64 {
	var s float64
	for _, v := range m {
		s += v
	}
	return math.Round(s*100) / 100
}

func tabNetSumCents(net map[string]float64) int64 {
	var t int64
	for _, v := range net {
		t += moneyToCents(v)
	}
	return t
}

func TestSharesForExpense_nonTab_returnsEmpty(t *testing.T) {
	party, guests, _ := tenMixedParty()
	e := Expense{FromTab: false, Amount: 99.99, Category: "General"}
	sh, err := SharesForExpense(e, party, guests, "u0")
	if err != nil {
		t.Fatal(err)
	}
	if len(sh) != 0 {
		t.Fatalf("expected empty shares, got %+v", sh)
	}
}

func TestSharesForExpense_legacyEqual_tenUsers(t *testing.T) {
	party := make([]string, 10)
	for i := range party {
		party[i] = fmt.Sprintf("m%d", i)
	}
	e := Expense{FromTab: true, Amount: 100, SplitMode: "", SplitJSON: ""}
	sh, err := SharesForExpense(e, party, nil, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(sh) != 10 {
		t.Fatalf("want 10 keys, got %d %+v", len(sh), sh)
	}
	want := 10.0
	for _, id := range party {
		k := ParticipantKeyUser(id)
		if math.Abs(sh[k]-want) > 0.02 {
			t.Fatalf("key %s got %v want ~%v", k, sh[k], want)
		}
	}
	if math.Abs(sumShareMap(sh)-100) > 0.02 {
		t.Fatalf("sum %v want 100", sumShareMap(sh))
	}
}

func TestSharesForExpense_equal_tenMixed_conservesTotal(t *testing.T) {
	party, guests, keys := tenMixedParty()
	p := TabSplitPayload{Participants: keys}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	e := Expense{FromTab: true, Amount: 100.03, SplitMode: TabSplitEqual, SplitJSON: string(raw)}
	sh, err := SharesForExpense(e, party, guests, "u0")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(sumShareMap(sh)-100.03) > 0.02 {
		t.Fatalf("equal 10-way sum got %v want 100.03 %+v", sumShareMap(sh), sh)
	}
}

func TestSharesForExpense_exact_tenMixed_conservesTotal(t *testing.T) {
	party, guests, keys := tenMixedParty()
	amounts := []float64{12.34, 8.76, 7.11, 9.89, 10.00, 11.11, 6.66, 13.33, 10.40, 10.40}
	var sum float64
	for _, a := range amounts {
		sum += a
	}
	if math.Abs(sum-100) > 0.001 {
		t.Fatalf("setup: amounts sum to %v", sum)
	}
	w := map[string]float64{}
	for i, k := range keys {
		w[k] = amounts[i]
	}
	p := TabSplitPayload{Participants: keys, Weights: w}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	e := Expense{FromTab: true, Amount: 100, SplitMode: TabSplitExact, SplitJSON: string(raw)}
	sh, err := SharesForExpense(e, party, guests, "u0")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(sumShareMap(sh)-100) > 0.02 {
		t.Fatalf("exact sum got %v %+v", sumShareMap(sh), sh)
	}
	for _, k := range keys {
		if math.Abs(sh[k]-w[k]) > 0.02 {
			t.Fatalf("key %s got %v want %v", k, sh[k], w[k])
		}
	}
}

func TestSharesForExpense_percent_tenMixed_conservesTotal(t *testing.T) {
	party, guests, keys := tenMixedParty()
	w := map[string]float64{}
	for _, k := range keys {
		w[k] = 10
	}
	p := TabSplitPayload{Participants: keys, Weights: w}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	e := Expense{FromTab: true, Amount: 200.01, SplitMode: TabSplitPercent, SplitJSON: string(raw)}
	sh, err := SharesForExpense(e, party, guests, "u0")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(sumShareMap(sh)-200.01) > 0.02 {
		t.Fatalf("percent sum got %v want 200.01 %+v", sumShareMap(sh), sh)
	}
}

func TestSharesForExpense_shares_tenMixed_equalWeights(t *testing.T) {
	party, guests, keys := tenMixedParty()
	w := map[string]float64{}
	for _, k := range keys {
		w[k] = 1
	}
	p := TabSplitPayload{Participants: keys, Weights: w}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	e := Expense{FromTab: true, Amount: 1000, SplitMode: TabSplitShares, SplitJSON: string(raw)}
	sh, err := SharesForExpense(e, party, guests, "u0")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(sumShareMap(sh)-1000) > 0.02 {
		t.Fatalf("shares sum got %v %+v", sumShareMap(sh), sh)
	}
	want := 100.0
	for _, k := range keys {
		if math.Abs(sh[k]-want) > 0.02 {
			t.Fatalf("key %s got %v want ~%v", k, sh[k], want)
		}
	}
}

func TestSharesForExpense_shares_tenMixed_weighted(t *testing.T) {
	party, guests, keys := tenMixedParty()
	w := map[string]float64{}
	for i, k := range keys {
		w[k] = float64(i + 1)
	}
	p := TabSplitPayload{Participants: keys, Weights: w}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	e := Expense{FromTab: true, Amount: 55, SplitMode: TabSplitShares, SplitJSON: string(raw)}
	sh, err := SharesForExpense(e, party, guests, "u0")
	if err != nil {
		t.Fatal(err)
	}
	if math.Abs(sumShareMap(sh)-55) > 0.02 {
		t.Fatalf("weighted shares sum got %v", sumShareMap(sh))
	}
	for i, k := range keys {
		if math.Abs(sh[k]-float64(i+1)) > 0.02 {
			t.Fatalf("key %s got %v want %d", k, sh[k], i+1)
		}
	}
}

func TestTabLedger_tenMixed_equalSplit_conservesCents(t *testing.T) {
	party, guests, keys := tenMixedParty()
	p := TabSplitPayload{Participants: keys}
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	ex := []Expense{{
		FromTab:   true,
		Amount:    999.99,
		PaidBy:    ParticipantKeyUser("u0"),
		SplitMode: TabSplitEqual,
		SplitJSON: string(raw),
	}}
	net, err := TabLedger(ex, party, guests, nil, "u0")
	if err != nil {
		t.Fatal(err)
	}
	if tabNetSumCents(net) != 0 {
		t.Fatalf("ledger should sum to 0 cents, got %d net=%+v", tabNetSumCents(net), net)
	}
}
