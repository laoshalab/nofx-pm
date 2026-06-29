package store

import "testing"

func TestEffectiveTradingModeExplicitPreview(t *testing.T) {
	row := &PredictionTraderDB{
		TradingMode: PredictionTradingModePreview,
		PreviewMode: false, // drifted flag should not override explicit mode
	}
	if got := row.EffectiveTradingMode(); got != PredictionTradingModePreview {
		t.Fatalf("got %q want preview", got)
	}
}

func TestIsValidTradingMode(t *testing.T) {
	if !IsValidTradingMode(PredictionTradingModeSimulation) {
		t.Fatal("simulation should be valid")
	}
	if IsValidTradingMode("paper") {
		t.Fatal("unknown mode should be invalid")
	}
}
