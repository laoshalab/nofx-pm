package engine

import (
	"testing"
	"time"

	predcfg "nofx/prediction/config"
	"nofx/prediction/types"
)

func TestEvaluateRulesBuyYes(t *testing.T) {
	ctx := &Context{
		CandidateMarkets: []MarketSnapshot{{
			Market: types.Market{
				Slug:        "btc-up",
				YesTokenID:  "y1",
				Liquidity:   5000,
				EndDate:     time.Now().Add(2 * time.Hour),
			},
			YesMid: 0.42,
			NoMid:  0.56,
		}},
	}
	strat := predcfg.DefaultStrategyConfig()
	strat.Rules.MaxBuyYesPrice = 0.45
	strat.Rules.DefaultSizeUsd = 15

	decisions := EvaluateRules(ctx, strat)
	if len(decisions) != 1 || decisions[0].Action != types.ActionBuyYes {
		t.Fatalf("got %+v", decisions)
	}
}

func TestEvaluateRulesWaitWhenNoSignal(t *testing.T) {
	ctx := &Context{
		CandidateMarkets: []MarketSnapshot{{
			Market: types.Market{Slug: "x", Liquidity: 5000},
			YesMid: 0.70,
			NoMid:  0.28,
		}},
	}
	strat := predcfg.DefaultStrategyConfig()
	strat.Rules.MaxBuyYesPrice = 0.45
	decisions := EvaluateRules(ctx, strat)
	if len(decisions) != 1 || decisions[0].Action != types.ActionWait {
		t.Fatalf("expected wait, got %+v", decisions)
	}
}

func TestEvaluateRulesDefaultEdgePct(t *testing.T) {
	ctx := &Context{
		CandidateMarkets: []MarketSnapshot{{
			Market: types.Market{
				Slug:       "btc-up",
				YesTokenID: "y1",
				Liquidity:  5000,
				EndDate:    time.Now().Add(2 * time.Hour),
			},
			YesMid: 0.42,
			NoMid:  0.56,
		}},
	}
	strat := predcfg.DefaultStrategyConfig()
	strat.Rules.MaxBuyYesPrice = 0.45
	strat.Rules.DefaultEdgePct = 0
	strat.Risk.MinEdgePct = 2.0

	decisions := EvaluateRules(ctx, strat)
	if len(decisions) != 1 || decisions[0].EdgePct < 2.0 {
		t.Fatalf("expected edge fallback >= 2.0, got %+v", decisions)
	}
}

func TestEvaluateRulesNoDualSide(t *testing.T) {
	ctx := &Context{
		CandidateMarkets: []MarketSnapshot{{
			Market: types.Market{
				Slug:       "dual",
				YesTokenID: "y1",
				NoTokenID:  "n1",
				Liquidity:  5000,
				EndDate:    time.Now().Add(2 * time.Hour),
			},
			YesMid: 0.40,
			NoMid:  0.40,
		}},
	}
	strat := predcfg.DefaultStrategyConfig()
	strat.Rules.MaxBuyYesPrice = 0.45
	strat.Rules.MaxBuyNoPrice = 0.45

	decisions := EvaluateRules(ctx, strat)
	buys := 0
	for _, d := range decisions {
		if d.Action == types.ActionBuyYes || d.Action == types.ActionBuyNo {
			buys++
		}
	}
	if buys > 1 {
		t.Fatalf("expected at most one buy per market, got %+v", decisions)
	}
}
