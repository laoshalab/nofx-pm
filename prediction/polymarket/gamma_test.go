package polymarket

import (
	"testing"
)

func TestParseTokenIDs(t *testing.T) {
	yes, no := parseTokenIDs(`["111","222"]`)
	if yes != "111" || no != "222" {
		t.Fatalf("json array: got %q %q", yes, no)
	}
	yes, no = parseTokenIDs("aaa,bbb")
	if yes != "aaa" || no != "bbb" {
		t.Fatalf("csv: got %q %q", yes, no)
	}
}

func TestDefaultConfig(t *testing.T) {
	c := DefaultConfig()
	if c.GammaURL == "" || c.ClobURL == "" {
		t.Fatal("expected default URLs")
	}
}
