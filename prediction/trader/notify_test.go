package trader

import (
	"strings"
	"testing"

	"nofx/prediction/types"
)

func TestFormatExecutionTelegram(t *testing.T) {
	text := formatExecutionTelegram("Poly Crypto", types.ExecutionOutcome{
		MarketSlug: "btc-up",
		Action:     types.ActionBuyYes,
		Status:     "filled",
		FillUsd:    12.5,
		OrderID:    "ord-1",
		Message:    "filled $12.50",
	})
	if !strings.Contains(text, "Poly Crypto") || !strings.Contains(text, "btc-up") {
		t.Fatalf("unexpected: %s", text)
	}
}
