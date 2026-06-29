package polymarket

import "testing"

func TestSignatureTypeForOrder(t *testing.T) {
	cfg := DefaultConfig()
	if got := cfg.SignatureTypeForOrder(); got != 0 {
		t.Fatalf("EOA without proxy: got %d want 0", got)
	}

	cfg.ProxyAddress = "0x1234567890123456789012345678901234567890"
	cfg.SignatureType = 2
	if got := cfg.SignatureTypeForOrder(); got != 2 {
		t.Fatalf("Safe with proxy: got %d want 2", got)
	}

	cfg.ProxyAddress = ""
	cfg.SignatureType = 2
	if got := cfg.SignatureTypeForOrder(); got != 0 {
		t.Fatalf("Safe without proxy must fall back to EOA: got %d", got)
	}
}
