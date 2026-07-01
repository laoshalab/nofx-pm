package trader

import (
	"fmt"
	"time"

	"nofx/prediction/config"
	"nofx/prediction/engine"
	"nofx/prediction/types"
)

func (pt *PredictionTrader) effectiveMode() config.EngineMode {
	m := pt.cfg.Strategy.Mode
	if m == "" {
		return config.ModeAI
	}
	return m
}

func (pt *PredictionTrader) runAICycle(ctx *engine.Context, persist bool) (*engine.FullDecision, error) {
	if pt.mcpClient == nil {
		return nil, fmt.Errorf("AI client not configured")
	}
	decision, err := engine.GetPredictionDecisions(ctx, pt.mcpClient, pt.engine)
	if persist && decision != nil {
		pt.saveDecisionRecord(decision, err)
	}
	return decision, err
}

func (pt *PredictionTrader) runRulesCycle(ctx *engine.Context, persist bool) *engine.FullDecision {
	decisions := engine.EvaluateRules(ctx, pt.cfg.Strategy)
	decision := engine.RulesFullDecision(decisions)
	if persist {
		pt.saveDecisionRecord(decision, nil)
	}
	return decision
}

func (pt *PredictionTrader) runHybridCycle(ctx *engine.Context) (*engine.FullDecision, error) {
	ruleDecision := pt.runRulesCycle(ctx, false)
	aiDecision, err := pt.runAICycle(ctx, false)
	if aiDecision == nil {
		return ruleDecision, err
	}
	merged := mergeDecisions(ruleDecision.Decisions, aiDecision.Decisions)
	aiDecision.Decisions = merged
	aiDecision.CoTTrace = fmt.Sprintf("hybrid\n--- rules ---\n%s\n--- AI ---\n%s",
		ruleDecision.CoTTrace, aiDecision.CoTTrace)
	return aiDecision, err
}

func mergeDecisions(rules, ai []types.PredictionDecision) []types.PredictionDecision {
	bySlug := map[string]types.PredictionDecision{}
	for _, d := range rules {
		if d.Action == types.ActionWait || d.Action == types.ActionHold {
			continue
		}
		if d.MarketSlug != "" {
			bySlug[d.MarketSlug] = d
		}
	}
	for _, d := range ai {
		if d.MarketSlug == "" {
			continue
		}
		bySlug[d.MarketSlug] = d // AI overrides rules for same market
	}
	out := make([]types.PredictionDecision, 0, len(bySlug))
	for _, d := range bySlug {
		out = append(out, d)
	}
	if len(out) == 0 {
		return []types.PredictionDecision{{Action: types.ActionWait, Reasoning: "hybrid: no signals"}}
	}
	return out
}

func (pt *PredictionTrader) tryAutoRedeem() {
	if !pt.canAutoRedeem() {
		return
	}
	items, err := pt.venue.FindRedeemablePositions()
	if err != nil || len(items) == 0 {
		return
	}
	for _, item := range items {
		res, err := pt.venue.RedeemCondition(item.ConditionID, item.NegRisk)
		if err != nil {
			pt.logWarnf("redeem %s: %v", item.MarketSlug, err)
			continue
		}
		pt.logInfof("redeem %s status=%s", item.MarketSlug, res.Status)
	}
}

func (pt *PredictionTrader) canAutoRedeem() bool {
	if pt.riskGate.Config().PreviewMode {
		return true
	}
	if lr, ok := pt.venue.(interface{ LiveRedeemReady() bool }); ok && lr.LiveRedeemReady() {
		return true
	}
	if id, ok := pt.venue.(interface{ VenueID() string }); ok {
		return id.VenueID() == "polymarket_sim"
	}
	return false
}

func (pt *PredictionTrader) runFastLoop(intervalSec int) {
	ticker := time.NewTicker(time.Duration(intervalSec) * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-pt.stopCh:
			return
		case <-ticker.C:
			if _, err := pt.runCycle(); err != nil {
				pt.logWarnf("fast-loop error: %v", err)
			}
		}
	}
}
