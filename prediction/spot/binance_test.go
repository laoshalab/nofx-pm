package spot

import "testing"

func TestNormalizeSymbol(t *testing.T) {
	if got := NormalizeSymbol("btc"); got != "BTCUSDT" {
		t.Fatalf("got %q", got)
	}
	if got := NormalizeSymbol("ETHUSDT"); got != "ETHUSDT" {
		t.Fatalf("got %q", got)
	}
}
