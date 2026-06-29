package polymarket

import (
	"testing"

	"github.com/ethereum/go-ethereum/common"
)

func TestParseConditionID(t *testing.T) {
	h, err := parseConditionID("0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef")
	if err != nil {
		t.Fatal(err)
	}
	if h == (common.Hash{}) {
		t.Fatal("expected non-zero hash")
	}
}

func TestBuildRedeemCallStandard(t *testing.T) {
	cid := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	call, err := buildRedeemCall(cid, false, false)
	if err != nil {
		t.Fatal(err)
	}
	if call.To != ctfCollateralAddr {
		t.Fatalf("target=%s want ctf collateral adapter", call.To.Hex())
	}
	if len(call.Data) < 4 {
		t.Fatal("expected calldata")
	}
}

func TestBuildRedeemCallNegRisk(t *testing.T) {
	cid := "0x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	call, err := buildRedeemCall(cid, true, false)
	if err != nil {
		t.Fatal(err)
	}
	if call.To != negRiskCollateralAddr {
		t.Fatalf("target=%s want neg risk adapter", call.To.Hex())
	}
}

func TestDeriveSafeAndProxy(t *testing.T) {
	owner := common.HexToAddress("0x1111111111111111111111111111111111111111")
	safe := deriveSafeAddress(owner)
	proxy := deriveProxyAddress(owner)
	if safe == proxy {
		t.Fatal("safe and proxy should differ")
	}
	if safe == (common.Address{}) || proxy == (common.Address{}) {
		t.Fatal("expected derived addresses")
	}
}

func TestBuildBuilderHMACDeterministic(t *testing.T) {
	secret := "dGVzdA==" // base64 "test"
	sig1 := buildBuilderHMAC(secret, 1700000000, "POST", "/submit", `{"a":1}`)
	sig2 := buildBuilderHMAC(secret, 1700000000, "POST", "/submit", `{"a":1}`)
	if sig1 != sig2 || sig1 == "" {
		t.Fatalf("hmac mismatch: %q", sig1)
	}
}

func TestLiveRedeemConfigured(t *testing.T) {
	if !LiveRedeemConfigured(Config{}) {
		t.Fatal("expected true with default Polygon RPC fallback")
	}
	cfg := Config{
		BuilderAPIKey:     "key",
		BuilderSecret:     "dGVzdA==",
		BuilderPassphrase: "pass",
	}
	if !LiveRedeemConfigured(cfg) {
		t.Fatal("expected true with builder creds")
	}
}

func TestNeedsRelayer(t *testing.T) {
	c := &Client{cfg: Config{SignatureType: 0}}
	if c.needsRelayer() {
		t.Fatal("EOA without proxy should not need relayer")
	}
	c.cfg.ProxyAddress = "0x1234567890123456789012345678901234567890"
	if !c.needsRelayer() {
		t.Fatal("proxy address should need relayer")
	}
}
