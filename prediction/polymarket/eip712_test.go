package polymarket

import (
	"testing"

	"github.com/ethereum/go-ethereum/crypto"
)

func TestSignClobAuthDeterministic(t *testing.T) {
	// Test vector key (never use in production)
	key, err := crypto.HexToECDSA("0123456789012345678901234567890123456789012345678901234567890123")
	if err != nil {
		t.Fatal(err)
	}
	w := &wallet{privateKey: key, address: crypto.PubkeyToAddress(key.PublicKey)}
	sig, err := signClobAuth(w, 137, "1700000000", 0)
	if err != nil {
		t.Fatal(err)
	}
	if len(sig) != 132 { // 0x + 130 hex chars
		t.Fatalf("unexpected sig len %d: %s", len(sig), sig)
	}
}

func TestBuildOrderAmounts(t *testing.T) {
	maker, taker, side := buildOrderAmounts("BUY", 0.5, 10)
	if side != 0 {
		t.Fatalf("side %d", side)
	}
	if maker.Int64() != 5_000_000 || taker.Int64() != 10_000_000 {
		t.Fatalf("BUY amounts maker=%s taker=%s", maker, taker)
	}

	maker, taker, side = buildOrderAmounts("SELL", 0.5, 10)
	if side != 1 {
		t.Fatalf("SELL side %d", side)
	}
	if maker.Int64() != 10_000_000 || taker.Int64() != 5_000_000 {
		t.Fatalf("SELL amounts maker=%s taker=%s", maker, taker)
	}
}
