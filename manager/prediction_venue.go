package manager

import (
	"fmt"
	"strings"

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
	cfg.PrivateKey = effectivePrivateKey(row)
	if strings.TrimSpace(row.ProxyAddress) != "" {
		cfg.ProxyAddress = row.ProxyAddress
	}
	cfg.SignatureType = polymarket.ResolveSignatureType(row.SignatureType, cfg.ProxyAddress)
	return polymarket.NewClient(cfg)
}

func effectivePrivateKey(row *store.PredictionTraderDB) string {
	if row == nil {
		return polymarket.EnvPrivateKey()
	}
	if k := row.PrivateKey.String(); k != "" {
		return k
	}
	return polymarket.EnvPrivateKey()
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
		if effectivePrivateKey(row) == "" {
			return fmt.Errorf("private key required for live trading (configure trader key or POLYMARKET_PRIVATE_KEY)")
		}
	case store.PredictionTradingModePreview:
		if effectivePrivateKey(row) == "" {
			return fmt.Errorf("private key required for preview signing (configure trader key or POLYMARKET_PRIVATE_KEY)")
		}
	}
	return nil
}
