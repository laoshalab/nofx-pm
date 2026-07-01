package manager

import (
	"os"
	"testing"

	"nofx/crypto"
	"nofx/store"
)

func TestEffectivePrivateKey_FallsBackToEnv(t *testing.T) {
	t.Setenv("POLYMARKET_PRIVATE_KEY", "0xabc")
	row := &store.PredictionTraderDB{}
	if got := effectivePrivateKey(row); got != "0xabc" {
		t.Fatalf("expected env key, got %q", got)
	}
	row.PrivateKey = crypto.EncryptedString("0xtrader")
	if got := effectivePrivateKey(row); got != "0xtrader" {
		t.Fatalf("expected trader key, got %q", got)
	}
	os.Unsetenv("POLYMARKET_PRIVATE_KEY")
}

func TestValidatePredictionTraderRow_LiveWithEnvKey(t *testing.T) {
	t.Setenv("POLYMARKET_PRIVATE_KEY", "0xabc")
	row := &store.PredictionTraderDB{TradingMode: store.PredictionTradingModeLive}
	if err := ValidatePredictionTraderRow(row); err != nil {
		t.Fatalf("expected ok with env key: %v", err)
	}
	os.Unsetenv("POLYMARKET_PRIVATE_KEY")
	row = &store.PredictionTraderDB{TradingMode: store.PredictionTradingModeLive}
	if err := ValidatePredictionTraderRow(row); err == nil {
		t.Fatal("expected error without key")
	}
}
