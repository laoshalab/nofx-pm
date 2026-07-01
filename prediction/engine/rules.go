package engine

import (
	"fmt"
	"math"
	"strings"
	"time"

	predcfg "nofx/prediction/config"
	"nofx/prediction/spot"
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

	var spotQuote *spot.Quote
	if sym := strings.TrimSpace(rules.SpotSymbol); sym != "" {
		if q, err := spot.FetchQuote(sym); err == nil {
			spotQuote = &q
		} else if ctx != nil && len(ctx.SpotQuotes) > 0 {
			key := spot.NormalizeSymbol(sym)
			if q, ok := ctx.SpotQuotes[key]; ok {
				spotQuote = &q
			} else {
				for _, v := range ctx.SpotQuotes {
					spotQuote = &v
					break
				}
			}
		}
	}

	var out []types.PredictionDecision
	slugAction := map[string]types.PredictionAction{}

	sellSize := rules.SellSizeUsd
	if sellSize <= 0 {
		sellSize = rules.DefaultSizeUsd
	}
	if ctx != nil {
		openCount := 0
		for _, p := range ctx.Positions {
			if p.Shares > 0 && p.MarketSlug != "" {
				openCount++
			}
		}
		maxMarkets := strat.Risk.MaxOpenMarkets
		if maxMarkets <= 0 {
			maxMarkets = 10
		}
		atCap := openCount >= maxMarkets

		for _, p := range ctx.Positions {
			if p.Shares <= 0 || p.TokenID == "" || p.MarketSlug == "" {
				continue
			}
			mark := p.MidPrice
			if mark <= 0 {
				mark = p.AvgCost
			}
			if mark <= 0 {
				continue
			}
			takeProfit := rules.TakeProfitMid > 0 && mark >= rules.TakeProfitMid
			trim := atCap && mark > p.AvgCost && p.AvgCost > 0
			if !takeProfit && !trim {
				continue
			}
			if _, dup := slugAction[p.MarketSlug]; dup {
				continue
			}
			reason := fmt.Sprintf("rule: take profit mid %.3f >= %.3f", mark, rules.TakeProfitMid)
			if trim && !takeProfit {
				reason = fmt.Sprintf("rule: trim at max open markets (%d); mid %.3f > cost %.3f", openCount, mark, p.AvgCost)
			}
			out = append(out, types.PredictionDecision{
				MarketSlug: p.MarketSlug,
				TokenID:    p.TokenID,
				Action:     types.ActionSell,
				SizeUsd:    sellSize,
				Reasoning:  reason,
			})
			slugAction[p.MarketSlug] = types.ActionSell
		}
	}

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
			if !spotAllowsBuyYes(m, spotQuote, rules.SpotMinChangePct) {
				continue
			}
			if prev, ok := slugAction[m.Slug]; ok && prev != types.ActionBuyYes {
				continue
			}
			price := roundPrice(snap.YesMid + 0.01)
			reason := fmt.Sprintf("rule: YES mid %.3f <= max %.3f", snap.YesMid, rules.MaxBuyYesPrice)
			if spotQuote != nil {
				reason += fmt.Sprintf("; spot %s 5m %+.2f%%", spotQuote.Symbol, spotQuote.Change5mPct)
			}
			out = append(out, types.PredictionDecision{
				MarketSlug: m.Slug,
				TokenID:    m.YesTokenID,
				Action:     types.ActionBuyYes,
				SizeUsd:    rules.DefaultSizeUsd,
				LimitPrice: price,
				Confidence: 75,
				EdgePct:    edgePct,
				Reasoning:  reason,
			})
			slugAction[m.Slug] = types.ActionBuyYes
		}
		if rules.MaxBuyNoPrice > 0 && snap.NoMid > 0 && snap.NoMid <= rules.MaxBuyNoPrice {
			if !spotAllowsBuyNo(m, spotQuote, rules.SpotMinChangePct) {
				continue
			}
			if prev, ok := slugAction[m.Slug]; ok && prev != types.ActionBuyNo {
				continue
			}
			price := roundPrice(snap.NoMid + 0.01)
			reason := fmt.Sprintf("rule: NO mid %.3f <= max %.3f", snap.NoMid, rules.MaxBuyNoPrice)
			if spotQuote != nil {
				reason += fmt.Sprintf("; spot %s 5m %+.2f%%", spotQuote.Symbol, spotQuote.Change5mPct)
			}
			out = append(out, types.PredictionDecision{
				MarketSlug: m.Slug,
				TokenID:    m.NoTokenID,
				Action:     types.ActionBuyNo,
				SizeUsd:    rules.DefaultSizeUsd,
				LimitPrice: price,
				Confidence: 75,
				EdgePct:    edgePct,
				Reasoning:  reason,
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

func isDirectionalMarket(m types.Market) bool {
	hay := strings.ToLower(m.Question + " " + m.Slug)
	return strings.Contains(hay, "up or down") ||
		strings.Contains(hay, "up-or-down") ||
		strings.Contains(hay, "up/down")
}

func spotAllowsBuyYes(m types.Market, q *spot.Quote, minPct float64) bool {
	if q == nil || minPct <= 0 || !isDirectionalMarket(m) {
		return true
	}
	return q.Change5mPct >= minPct
}

func spotAllowsBuyNo(m types.Market, q *spot.Quote, minPct float64) bool {
	if q == nil || minPct <= 0 || !isDirectionalMarket(m) {
		return true
	}
	return q.Change5mPct <= -minPct
}

// RulesFullDecision wraps rule output as a FullDecision for logging/storage.
func RulesFullDecision(decisions []types.PredictionDecision) *FullDecision {
	return &FullDecision{
		Timestamp: time.Now(),
		CoTTrace:  "rule engine (no AI)",
		Decisions: decisions,
	}
}
