package trader

import (
	"fmt"

	"nofx/prediction/types"
)

func (pt *PredictionTrader) executeRedeem(d *types.PredictionDecision) error {
	if d.MarketSlug != "" {
		m, err := pt.venue.GetMarketBySlug(d.MarketSlug)
		if err != nil {
			return err
		}
		if m.ConditionID == "" {
			return fmt.Errorf("market has no condition_id")
		}
		res, err := pt.venue.RedeemCondition(m.ConditionID, m.NegRisk)
		if err != nil {
			return err
		}
		pt.logInfof("redeem %s → %s", d.MarketSlug, res.Status)
		return nil
	}

	items, err := pt.venue.FindRedeemablePositions()
	if err != nil {
		return err
	}
	for _, item := range items {
		if _, err := pt.venue.RedeemCondition(item.ConditionID, item.NegRisk); err != nil {
			pt.logWarnf("redeem %s: %v", item.MarketSlug, err)
		}
	}
	return nil
}
