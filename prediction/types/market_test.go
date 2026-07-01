package types

import "testing"

func TestMarketWinningOutcome(t *testing.T) {
	m := Market{OutcomePrices: []float64{1, 0}}
	if m.WinningOutcome() != "YES" {
		t.Fatalf("expected YES, got %q", m.WinningOutcome())
	}
	m = Market{OutcomePrices: []float64{0.01, 0.99}}
	if m.WinningOutcome() != "NO" {
		t.Fatalf("expected NO, got %q", m.WinningOutcome())
	}
}

func TestMarketPayoutPerShare(t *testing.T) {
	m := Market{OutcomePrices: []float64{1, 0}}
	payout, ok := m.PayoutPerShare("YES")
	if !ok || payout != 1 {
		t.Fatalf("yes payout: ok=%v payout=%v", ok, payout)
	}
	payout, ok = m.PayoutPerShare("NO")
	if !ok || payout != 0 {
		t.Fatalf("no payout: ok=%v payout=%v", ok, payout)
	}
}
