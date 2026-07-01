package trader

import (
	"fmt"
	"math"

	"nofx/prediction/types"
)

// normalizeSellDecision caps sell size to open position and fills missing price/token fields.
func (pt *PredictionTrader) normalizeSellDecision(d *types.PredictionDecision) error {
	if d == nil {
		return fmt.Errorf("nil decision")
	}
	if d.TokenID == "" && d.MarketSlug != "" {
		positions, _ := pt.venue.GetOutcomePositions()
		for _, p := range positions {
			if p.MarketSlug == d.MarketSlug && p.Shares > 0 && p.TokenID != "" {
				d.TokenID = p.TokenID
				break
			}
		}
	}
	if d.TokenID == "" {
		return fmt.Errorf("missing token_id for sell")
	}

	p, ok := pt.riskGate.PositionForToken(d.TokenID)
	if !ok || p.Shares <= 0 {
		return fmt.Errorf("no open position for token %s", truncateToken(d.TokenID))
	}
	if d.MarketSlug == "" {
		d.MarketSlug = p.MarketSlug
	}

	mark := p.MidPrice
	if mark <= 0 {
		mark = p.AvgCost
	}
	if mark <= 0 {
		return fmt.Errorf("no mark price for %s", d.MarketSlug)
	}

	positionUsd := p.Shares * mark
	if d.SizeUsd <= 0 || d.SizeUsd > positionUsd {
		d.SizeUsd = positionUsd
	}

	if d.LimitPrice <= 0 {
		d.LimitPrice = roundSellPrice(mark - 0.01)
		if d.LimitPrice < 0.001 {
			d.LimitPrice = 0.001
		}
	}

	maxShares := d.SizeUsd / d.LimitPrice
	if maxShares > p.Shares {
		maxShares = p.Shares
		d.SizeUsd = maxShares * d.LimitPrice
	}
	return nil
}

func roundSellPrice(p float64) float64 {
	return math.Round(p*1000) / 1000
}

// SellPosition executes a manual partial or full sell for one token.
func (pt *PredictionTrader) SellPosition(tokenID string, sizeUsd, limitPrice float64) types.ExecutionOutcome {
	positions, err := pt.venue.GetOutcomePositions()
	if err != nil {
		return types.ExecutionOutcome{Status: "error", Message: err.Error()}
	}
	pt.riskGate.SetPositions(positions)

	var pos *types.OutcomePosition
	for i := range positions {
		if positions[i].TokenID == tokenID && positions[i].Shares > 0 {
			pos = &positions[i]
			break
		}
	}
	if pos == nil {
		return types.ExecutionOutcome{Status: "error", Message: "position not found"}
	}

	d := &types.PredictionDecision{
		MarketSlug: pos.MarketSlug,
		TokenID:    tokenID,
		Action:     types.ActionSell,
		SizeUsd:    sizeUsd,
		LimitPrice: limitPrice,
		Reasoning:  "manual sell via API",
	}
	return pt.executeDecisionWithOutcome(d)
}
