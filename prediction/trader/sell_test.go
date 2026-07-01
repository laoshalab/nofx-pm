package trader

import (
	"testing"

	"nofx/prediction/risk"
	"nofx/prediction/types"
)

func TestNormalizeSellDecisionCapsToPosition(t *testing.T) {
	g := risk.NewGate(risk.DefaultConfig())
	g.SetPositions([]types.OutcomePosition{
		{
			TokenID:    "tok1",
			MarketSlug: "m1",
			Shares:     100,
			MidPrice:   0.55,
		},
	})
	pt := &PredictionTrader{riskGate: g}
	d := &types.PredictionDecision{
		MarketSlug: "m1",
		TokenID:    "tok1",
		Action:     types.ActionSell,
		SizeUsd:    200,
	}
	if err := pt.normalizeSellDecision(d); err != nil {
		t.Fatal(err)
	}
	want := 100 * 0.55
	if d.SizeUsd > want+0.01 || d.SizeUsd < want-1 {
		t.Fatalf("size_usd = %.2f want ~%.2f", d.SizeUsd, want)
	}
	if d.LimitPrice <= 0 {
		t.Fatalf("expected limit price, got %.4f", d.LimitPrice)
	}
}

func TestNormalizeSellDecisionNoPosition(t *testing.T) {
	g := risk.NewGate(risk.DefaultConfig())
	pt := &PredictionTrader{riskGate: g}
	d := &types.PredictionDecision{
		TokenID: "missing",
		Action:  types.ActionSell,
		SizeUsd: 10,
	}
	if err := pt.normalizeSellDecision(d); err == nil {
		t.Fatal("expected error for missing position")
	}
}
