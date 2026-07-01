package risk

import (
	"testing"

	"nofx/prediction/types"
)

func TestFillDefaultsZeroRiskFields(t *testing.T) {
	cfg := FillDefaults(Config{})
	if cfg.MinPrice != 0.05 || cfg.MaxPrice != 0.95 {
		t.Fatalf("price band: min=%v max=%v", cfg.MinPrice, cfg.MaxPrice)
	}
	if cfg.MaxOpenMarkets != 10 {
		t.Fatalf("max open markets: %d", cfg.MaxOpenMarkets)
	}
}

func TestGateCheckDecision(t *testing.T) {
	g := NewGate(DefaultConfig())
	d := &types.PredictionDecision{
		MarketSlug: "test",
		Action:     types.ActionBuyYes,
		SizeUsd:    10,
		LimitPrice: 0.55,
		Confidence: 80,
		EdgePct:    3,
	}
	if err := g.CheckDecision(d); err != nil {
		t.Fatal(err)
	}
}

func TestGateRejectsLowConfidence(t *testing.T) {
	g := NewGate(DefaultConfig())
	d := &types.PredictionDecision{
		MarketSlug: "test",
		Action:     types.ActionBuyYes,
		SizeUsd:    10,
		LimitPrice: 0.55,
		Confidence: 50,
		EdgePct:    3,
	}
	if err := g.CheckDecision(d); err == nil {
		t.Fatal("expected confidence error")
	}
}

func TestGateHoldPasses(t *testing.T) {
	g := NewGate(DefaultConfig())
	if err := g.CheckDecision(&types.PredictionDecision{Action: types.ActionHold}); err != nil {
		t.Fatal(err)
	}
}

func TestGateOpenMarketsAllowsAddToExisting(t *testing.T) {
	g := NewGate(DefaultConfig())
	g.cfg.MaxOpenMarkets = 1
	g.SetPositions([]types.OutcomePosition{
		{MarketSlug: "existing", Shares: 10, MidPrice: 0.5},
	})
	d := &types.PredictionDecision{
		MarketSlug: "existing",
		Action:     types.ActionBuyYes,
		SizeUsd:    10,
		LimitPrice: 0.55,
		Confidence: 80,
		EdgePct:    3,
	}
	if err := g.CheckDecision(d); err != nil {
		t.Fatalf("add to existing market should pass open-markets gate: %v", err)
	}

	d.MarketSlug = "new-market"
	if err := g.CheckDecision(d); err == nil {
		t.Fatal("expected open markets error for new market")
	}
}

func TestGateMaxPositionMarketUsd(t *testing.T) {
	g := NewGate(DefaultConfig())
	g.cfg.MaxPositionMarketUsd = 100
	g.SetPositions([]types.OutcomePosition{
		{MarketSlug: "m1", Shares: 100, MidPrice: 0.9},
	})
	d := &types.PredictionDecision{
		MarketSlug: "m1",
		Action:     types.ActionBuyYes,
		SizeUsd:    20,
		LimitPrice: 0.55,
		Confidence: 80,
		EdgePct:    3,
	}
	if err := g.CheckDecision(d); err == nil {
		t.Fatal("expected market exposure error")
	}
}

func TestGateSellBypassesMaxOrderUsd(t *testing.T) {
	g := NewGate(DefaultConfig())
	g.cfg.MaxOrderUsd = 50
	g.SetPositions([]types.OutcomePosition{
		{TokenID: "t1", MarketSlug: "m1", Shares: 120, MidPrice: 0.55},
	})
	d := &types.PredictionDecision{
		MarketSlug: "m1",
		TokenID:    "t1",
		Action:     types.ActionSell,
		SizeUsd:    66,
		LimitPrice: 0.54,
	}
	if err := g.CheckDecision(d); err != nil {
		t.Fatalf("sell above max_order_usd should pass: %v", err)
	}
}

func TestGateSellRejectsBuyPriceBand(t *testing.T) {
	g := NewGate(DefaultConfig())
	g.SetPositions([]types.OutcomePosition{
		{TokenID: "t1", MarketSlug: "m1", Shares: 10, MidPrice: 0.02},
	})
	d := &types.PredictionDecision{
		MarketSlug: "m1",
		TokenID:    "t1",
		Action:     types.ActionSell,
		SizeUsd:    0.15,
		LimitPrice: 0.02,
	}
	if err := g.CheckDecision(d); err != nil {
		t.Fatalf("sell at low price should pass sell band: %v", err)
	}
	if err := g.CheckPrice(d.LimitPrice); err == nil {
		t.Fatal("buy band should reject 0.02")
	}
}

func TestGateReduceMarketExposure(t *testing.T) {
	g := NewGate(DefaultConfig())
	g.SetPositions([]types.OutcomePosition{
		{MarketSlug: "m1", Shares: 100, MidPrice: 0.5},
	})
	g.ReduceMarketExposure("m1", 30)
	if got := g.marketExposure["m1"]; got < 19.9 || got > 20.1 {
		t.Fatalf("exposure after reduce = %.2f want ~20", got)
	}
}
