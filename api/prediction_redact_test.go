package api

import (
	"strings"
	"testing"
)

func TestRedactPreviewWireJSON(t *testing.T) {
	wire := `{"signature":"0xabcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890","salt":"1"}`
	out := redactPreviewWireJSON(wire)
	if strings.Contains(out, "0xabcdef1234567890abcdef1234567890") {
		t.Fatalf("signature not redacted: %s", out)
	}
	if !strings.Contains(out, "redacted") {
		t.Fatalf("expected redacted marker: %s", out)
	}
}
