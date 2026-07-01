package polymarket

import "testing"

func TestResolveSignatureType(t *testing.T) {
	if got := ResolveSignatureType(1, ""); got != 1 {
		t.Fatalf("row wins: got %d", got)
	}
	if got := ResolveSignatureType(0, "0xabc"); got != 2 {
		t.Fatalf("proxy implies Safe: got %d", got)
	}
	if got := ResolveSignatureType(0, ""); got != 0 {
		t.Fatalf("EOA default: got %d", got)
	}
}

func TestEnvSignatureTypeMerge(t *testing.T) {
	t.Setenv("POLYMARKET_SIGNATURE_TYPE", "2")
	st, ok := EnvSignatureType()
	if !ok || st != 2 {
		t.Fatalf("env parse: ok=%v st=%d", ok, st)
	}
	cfg := MergeEnvConfig(Config{})
	if cfg.SignatureType != 2 {
		t.Fatalf("merge env sig: got %d", cfg.SignatureType)
	}
	if got := ResolveSignatureType(0, ""); got != 2 {
		t.Fatalf("resolve env: got %d", got)
	}
	t.Setenv("POLYMARKET_SIGNATURE_TYPE", "")
}

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
