package manager

import (
	"fmt"

	"nofx/prediction/polymarket"
	"nofx/prediction/sim"
	"nofx/prediction/types"
	"nofx/store"
)

// BuildVenueForTrader creates the execution venue for a prediction trader row.
func BuildVenueForTrader(st *store.Store, row *store.PredictionTraderDB) types.PredictionVenue {
	switch row.EffectiveTradingMode() {
	case store.PredictionTradingModeSimulation:
		return buildSimVenue(st, row)
	default:
		return buildPolymarketVenue(row)
	}
}

func buildPolymarketVenue(row *store.PredictionTraderDB) types.PredictionVenue {
	cfg := polymarket.MergeEnvConfig(polymarket.DefaultConfig())
	cfg.PreviewMode = row.EffectiveTradingMode() != store.PredictionTradingModeLive
	cfg.ProxyAddress = row.ProxyAddress
	cfg.SignatureType = row.SignatureType
	cfg.PrivateKey = row.PrivateKey.String()
	return polymarket.NewClient(cfg)
}

func buildSimVenue(st *store.Store, row *store.PredictionTraderDB) types.PredictionVenue {
	simCfg := sim.DefaultConfig()
	_ = store.ParseSimConfigJSON(row.SimConfigJSON, &simCfg)

	pub := polymarket.NewClient(polymarket.DefaultConfig())
	ledger := sim.NewLedger(simCfg.InitialBalanceUsd)
	if st != nil {
		_ = sim.InitLedgerInStore(st, row.ID, simCfg)
		_ = sim.LoadLedgerFromStore(st, row.ID, ledger)
	}
	venue := sim.NewVenue(pub, simCfg, ledger)
	if st != nil {
		venue.BindStore(st, row.ID)
	}
	return venue
}

// BuildVenueForTraderLegacy is kept for callers without store access.
func BuildVenueForTraderLegacy(row *store.PredictionTraderDB) types.PredictionVenue {
	return BuildVenueForTrader(nil, row)
}

// ValidatePredictionTraderRow returns an error if the row cannot be loaded as a runtime trader.
func ValidatePredictionTraderRow(row *store.PredictionTraderDB) error {
	mode := row.EffectiveTradingMode()
	switch mode {
	case store.PredictionTradingModeLive:
		if row.PrivateKey.String() == "" {
			return fmt.Errorf("private key required for live trading")
		}
	case store.PredictionTradingModePreview:
		if row.PrivateKey.String() == "" {
			return fmt.Errorf("private key required for preview signing")
		}
	}
	return nil
}
