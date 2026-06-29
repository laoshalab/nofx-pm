package store

import "strings"

// EffectiveTradingMode returns normalized trading mode (simulation | preview | live).
func (t *PredictionTraderDB) EffectiveTradingMode() string {
	switch strings.TrimSpace(t.TradingMode) {
	case PredictionTradingModeSimulation, PredictionTradingModeLive, PredictionTradingModePreview:
		return t.TradingMode
	}
	// Legacy rows without trading_mode: infer from preview_mode flag.
	if !t.PreviewMode {
		return PredictionTradingModeLive
	}
	return PredictionTradingModePreview
}

// IsValidTradingMode reports whether tm is a supported trading mode string.
func IsValidTradingMode(tm string) bool {
	switch strings.TrimSpace(tm) {
	case "", PredictionTradingModeSimulation, PredictionTradingModePreview, PredictionTradingModeLive:
		return true
	default:
		return false
	}
}
