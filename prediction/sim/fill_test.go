package sim

import "testing"

func TestApplySlippageBuy(t *testing.T) {
	got := applySlippage(0.50, 1, 50)
	want := 0.5025
	if got < want-1e-9 || got > want+1e-9 {
		t.Fatalf("applySlippage buy = %v want %v", got, want)
	}
}

func TestApplySlippageSell(t *testing.T) {
	got := applySlippage(0.50, -1, 50)
	want := 0.4975
	if got < want-1e-9 || got > want+1e-9 {
		t.Fatalf("applySlippage sell = %v want %v", got, want)
	}
}

func TestApplySlippageZeroBps(t *testing.T) {
	if applySlippage(0.42, 1, 0) != 0.42 {
		t.Fatal("zero bps should not change price")
	}
}
