package engine

import (
	"fmt"
	"math"
	"time"

	predcfg "nofx/prediction/config"
	"nofx/prediction/types"
)

// EvaluateRules produces decisions from deterministic rules (no AI).
func EvaluateRules(ctx *Context, strat predcfg.StrategyConfig) []types.PredictionDecision {
	rules := strat.Rules
	if rules.DefaultSizeUsd <= 0 {
		rules.DefaultSizeUsd = 10
	}
	if rules.MaxSpread <= 0 {
		rules.MaxSpread = 0.05
	}
	edgePct := rules.DefaultEdgePct
	if edgePct <= 0 {
		edgePct = strat.Risk.MinEdgePct
	}
	if edgePct <= 0 {
		edgePct = 2.5
	}
	minLiq := rules.MinLiquidityUsd
	if minLiq <= 0 {
		minLiq = strat.MinLiquidityUsd
	}

	var out []types.PredictionDecision
	slugAction := map[string]types.PredictionAction{}
	for _, snap := range ctx.CandidateMarkets {
		m := snap.Market
		if m.Closed {
			continue
		}
		if minLiq > 0 && m.Liquidity < minLiq {
			continue
		}
		spread := math.Abs(snap.YesMid + snap.NoMid - 1)
		if spread > rules.MaxSpread {
			continue
		}
		if strat.MinHoursToExpiry > 0 && !m.EndDate.IsZero() {
			if time.Until(m.EndDate).Hours() < strat.MinHoursToExpiry {
				continue
			}
		}

		if rules.MaxBuyYesPrice > 0 && snap.YesMid > 0 && snap.YesMid <= rules.MaxBuyYesPrice {
			if prev, ok := slugAction[m.Slug]; ok && prev != types.ActionBuyYes {
				continue
			}
			price := roundPrice(snap.YesMid + 0.01)
			out = append(out, types.PredictionDecision{
				MarketSlug: m.Slug,
				TokenID:    m.YesTokenID,
				Action:     types.ActionBuyYes,
				SizeUsd:    rules.DefaultSizeUsd,
				LimitPrice: price,
				Confidence: 75,
				EdgePct:    edgePct,
				Reasoning:  fmt.Sprintf("rule: YES mid %.3f <= max %.3f", snap.YesMid, rules.MaxBuyYesPrice),
			})
			slugAction[m.Slug] = types.ActionBuyYes
		}
		if rules.MaxBuyNoPrice > 0 && snap.NoMid > 0 && snap.NoMid <= rules.MaxBuyNoPrice {
			if prev, ok := slugAction[m.Slug]; ok && prev != types.ActionBuyNo {
				continue
			}
			price := roundPrice(snap.NoMid + 0.01)
			out = append(out, types.PredictionDecision{
				MarketSlug: m.Slug,
				TokenID:    m.NoTokenID,
				Action:     types.ActionBuyNo,
				SizeUsd:    rules.DefaultSizeUsd,
				LimitPrice: price,
				Confidence: 75,
				EdgePct:    edgePct,
				Reasoning:  fmt.Sprintf("rule: NO mid %.3f <= max %.3f", snap.NoMid, rules.MaxBuyNoPrice),
			})
			slugAction[m.Slug] = types.ActionBuyNo
		}
	}
	if len(out) == 0 {
		out = []types.PredictionDecision{{
			Action:    types.ActionWait,
			Reasoning: "rule: no signals this cycle",
		}}
	}
	return out
}

func roundPrice(p float64) float64 {
	return math.Round(p*1000) / 1000
}

// RulesFullDecision wraps rule output as a FullDecision for logging/storage.
func RulesFullDecision(decisions []types.PredictionDecision) *FullDecision {
	return &FullDecision{
		Timestamp: time.Now(),
		CoTTrace:  "rule engine (no AI)",
		Decisions: decisions,
	}
}
