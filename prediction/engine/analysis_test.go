package engine

import (
	"testing"

	"nofx/prediction/types"
)

func TestParsePredictionResponse(t *testing.T) {
	raw := `<reasoning>BTC likely up</reasoning>
<decision>
[{"market_slug":"btc-up","action":"buy_yes","size_usd":10,"limit_price":0.55,"confidence":80,"edge_pct":3,"reasoning":"momentum"}]
</decision>`
	decisions, cot, err := parsePredictionResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if cot != "BTC likely up" {
		t.Fatalf("cot: %q", cot)
	}
	if len(decisions) != 1 || decisions[0].Action != types.ActionBuyYes {
		t.Fatalf("decisions: %+v", decisions)
	}
}

func TestParsePredictionResponseMultipleDecisions(t *testing.T) {
	raw := `<decision>
[{"market_slug":"a","action":"buy_yes","size_usd":10,"limit_price":0.55,"confidence":80,"edge_pct":3,"reasoning":"a"},
 {"market_slug":"b","action":"buy_no","size_usd":5,"limit_price":0.40,"confidence":75,"edge_pct":2.5,"reasoning":"b"}]
</decision>`
	decisions, _, err := parsePredictionResponse(raw)
	if err != nil {
		t.Fatal(err)
	}
	if len(decisions) != 2 {
		t.Fatalf("expected 2 decisions, got %d: %+v", len(decisions), decisions)
	}
}

func TestExtractJSONArray(t *testing.T) {
	raw := `prefix [{"a":1},{"b":2}] suffix`
	got := extractJSONArray(raw)
	if got != `[{"a":1},{"b":2}]` {
		t.Fatalf("got %q", got)
	}
}

func TestValidateDecisionRejectsBadAction(t *testing.T) {
	err := validateDecision(&types.PredictionDecision{Action: "long", MarketSlug: "x"})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveTokenIDs(t *testing.T) {
	ctx := &Context{
		CandidateMarkets: []MarketSnapshot{{
			Market: types.Market{Slug: "m1", YesTokenID: "yes1", NoTokenID: "no1"},
		}},
	}
	decisions := []types.PredictionDecision{
		{MarketSlug: "m1", Action: types.ActionBuyYes},
		{MarketSlug: "m1", Action: types.ActionBuyNo},
	}
	resolveTokenIDs(decisions, ctx)
	if decisions[0].TokenID != "yes1" || decisions[1].TokenID != "no1" {
		t.Fatalf("tokens: %+v", decisions)
	}
}

func TestResolveTokenIDsSellFromPosition(t *testing.T) {
	ctx := &Context{
		CandidateMarkets: []MarketSnapshot{{
			Market: types.Market{Slug: "m1", YesTokenID: "yes1", NoTokenID: "no1"},
		}},
		Positions: []types.OutcomePosition{
			{MarketSlug: "m1", Outcome: "NO", TokenID: "no1", Shares: 25},
		},
	}
	decisions := []types.PredictionDecision{
		{MarketSlug: "m1", Action: types.ActionSell},
	}
	resolveTokenIDs(decisions, ctx)
	if decisions[0].TokenID != "no1" {
		t.Fatalf("sell should use NO position token, got %q", decisions[0].TokenID)
	}
}
