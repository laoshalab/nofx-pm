package trader

import (
	"testing"

	"nofx/prediction/types"
)

func TestSortDecisionsSellFirst(t *testing.T) {
	in := []types.PredictionDecision{
		{Action: types.ActionBuyYes, MarketSlug: "a"},
		{Action: types.ActionSell, MarketSlug: "b"},
		{Action: types.ActionWait, MarketSlug: "c"},
	}
	out := sortDecisions(in)
	if out[0].Action != types.ActionSell {
		t.Fatalf("expected sell first, got %s", out[0].Action)
	}
	if out[len(out)-1].Action != types.ActionWait {
		t.Fatalf("expected wait last, got %s", out[len(out)-1].Action)
	}
}
