package trader

import (
	"fmt"
	"strings"
	"time"

	"nofx/prediction/types"
	"nofx/store"
)

func sortDecisions(decisions []types.PredictionDecision) []types.PredictionDecision {
	priority := map[types.PredictionAction]int{
		types.ActionSell:   0,
		types.ActionRedeem: 1,
		types.ActionBuyYes: 2,
		types.ActionBuyNo:  2,
		types.ActionHold:   3,
		types.ActionWait:   4,
	}
	out := make([]types.PredictionDecision, len(decisions))
	copy(out, decisions)
	for i := 0; i < len(out); i++ {
		for j := i + 1; j < len(out); j++ {
			if priority[out[j].Action] < priority[out[i].Action] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}

func (pt *PredictionTrader) executeDecisionWithOutcome(d *types.PredictionDecision) types.ExecutionOutcome {
	out := types.ExecutionOutcome{
		MarketSlug: d.MarketSlug,
		Action:     d.Action,
	}
	switch d.Action {
	case types.ActionHold, types.ActionWait:
		out.Status = "skipped"
		return out
	case types.ActionRedeem:
		if err := pt.riskGate.CheckDecision(d); err != nil {
			out.Status = "risk_blocked"
			out.Message = err.Error()
			return out
		}
		if err := pt.executeRedeem(d); err != nil {
			out.Status = "error"
			out.Message = err.Error()
			return out
		}
		out.Status = "redeem"
		out.Message = "redeem submitted"
		return out
	case types.ActionBuyYes, types.ActionBuyNo, types.ActionSell:
		if err := pt.riskGate.CheckDecision(d); err != nil {
			out.Status = "risk_blocked"
			out.Message = err.Error()
			return out
		}
		return pt.executeTradeOutcome(d)
	default:
		out.Status = "error"
		out.Message = fmt.Sprintf("unknown action: %s", d.Action)
		return out
	}
}

func (pt *PredictionTrader) executeDecision(d *types.PredictionDecision) error {
	out := pt.executeDecisionWithOutcome(d)
	if out.Status == "error" || out.Status == "risk_blocked" || out.Status == "rejected" {
		if out.Message != "" {
			return fmt.Errorf("%s", out.Message)
		}
	}
	return nil
}

func (pt *PredictionTrader) executeTradeOutcome(d *types.PredictionDecision) types.ExecutionOutcome {
	out := types.ExecutionOutcome{
		MarketSlug: d.MarketSlug,
		Action:     d.Action,
	}
	if d.TokenID == "" {
		out.Status = "error"
		out.Message = fmt.Sprintf("missing token_id for %s", d.MarketSlug)
		return out
	}
	if d.LimitPrice <= 0 {
		out.Status = "error"
		out.Message = fmt.Sprintf("limit_price must be positive for %s", d.Action)
		return out
	}

	side := "BUY"
	shares := d.SizeUsd / d.LimitPrice
	if d.Action == types.ActionSell {
		side = "SELL"
	}

	negRisk := false
	m, err := pt.venue.GetMarketBySlug(d.MarketSlug)
	if err == nil {
		negRisk = m.NegRisk
	}

	req := types.LimitOrderReq{
		TokenID:        d.TokenID,
		Side:           side,
		Price:          d.LimitPrice,
		Size:           shares,
		NegRisk:        negRisk,
		MarketSlug:     d.MarketSlug,
		CycleNumber:    pt.callCount,
		DecisionAction: string(d.Action),
		Reasoning:      d.Reasoning,
	}
	switch d.Action {
	case types.ActionBuyYes:
		req.Outcome = "YES"
	case types.ActionBuyNo:
		req.Outcome = "NO"
	}

	isPreviewVenue := pt.riskGate.Config().PreviewMode
	if isPreviewVenue {
		pt.logInfof("PREVIEW %s %s slug=%s token=%s price=%.4f shares=%.2f usd=%.2f — %s",
			side, d.Action, d.MarketSlug, truncateToken(d.TokenID), d.LimitPrice, shares, d.SizeUsd, d.Reasoning)
	}

	result, err := pt.venue.PlaceLimitOrder(req)
	if err != nil {
		out.Status = "error"
		out.Message = err.Error()
		return out
	}
	if result == nil {
		out.Status = "skipped"
		return out
	}
	if result.Error != "" {
		out.Status = "rejected"
		out.Message = result.Error
		return out
	}
	out.OrderID = result.OrderID

	if result.Status == "preview" || strings.HasPrefix(result.OrderID, "preview-") {
		out.Status = "preview"
		out.PreviewWire = result.PreviewWire
		out.Message = fmt.Sprintf("%s %s @ %.4f, %d shares (signed, not posted)", side, d.Action, d.LimitPrice, int(shares))
		pt.saveOrderRecord(d, result, &out, shares)
		return out
	}

	if result.Status == "matched" || result.FilledShares > 0 {
		fillUsd := result.FilledUsd
		if fillUsd <= 0 {
			fillUsd = result.FilledShares * d.LimitPrice
		}
		pt.riskGate.RecordFill(fillUsd)
		pt.recordFillVolume(fillUsd)
		if d.Action == types.ActionBuyYes || d.Action == types.ActionBuyNo {
			pt.riskGate.AddMarketExposure(d.MarketSlug, fillUsd)
		}
		pt.logInfof("order %s filled $%.2f", result.OrderID, fillUsd)
		out.Status = "filled"
		out.FillUsd = fillUsd
		out.Message = fmt.Sprintf("filled $%.2f", fillUsd)
		pt.saveOrderRecord(d, result, &out, shares)
		return out
	}

	fillUsd, filled := pt.confirmedFillUsd(d, result)
	if filled {
		pt.riskGate.RecordFill(fillUsd)
		pt.recordFillVolume(fillUsd)
		if d.Action == types.ActionBuyYes || d.Action == types.ActionBuyNo {
			pt.riskGate.AddMarketExposure(d.MarketSlug, fillUsd)
		}
		pt.logInfof("order %s filled $%.2f", result.OrderID, fillUsd)
		out.Status = "filled"
		out.FillUsd = fillUsd
		out.Message = fmt.Sprintf("filled $%.2f", fillUsd)
		pt.saveOrderRecord(d, result, &out, shares)
		return out
	}

	pt.logInfof("order %s posted (not filled yet)", result.OrderID)
	out.Status = "posted"
	out.Message = "order posted, awaiting fill"
	pt.saveOrderRecord(d, result, &out, shares)
	return out
}

func (pt *PredictionTrader) saveOrderRecord(d *types.PredictionDecision, result *types.OrderResult, out *types.ExecutionOutcome, shares float64) {
	if pt.st == nil || pt.traderDBID == "" || result == nil || result.OrderID == "" {
		return
	}
	side := "BUY"
	if d.Action == types.ActionSell {
		side = "SELL"
	}
	isPreview := out.Status == "preview" || strings.HasPrefix(result.OrderID, "preview-")
	rec := &store.PredictionOrderRecordDB{
		TraderID:    pt.traderDBID,
		OrderID:     result.OrderID,
		CycleNumber: pt.callCount,
		MarketSlug:  d.MarketSlug,
		TokenID:     d.TokenID,
		Side:        side,
		Price:       d.LimitPrice,
		Size:        shares,
		Status:      out.Status,
		PreviewWire: result.PreviewWire,
		IsPreview:   isPreview,
	}
	_ = pt.st.Prediction().SaveOrderRecord(rec)
}

func (pt *PredictionTrader) confirmedFillUsd(d *types.PredictionDecision, result *types.OrderResult) (usd float64, filled bool) {
	if result.OrderID == "" || strings.HasPrefix(result.OrderID, "preview-") {
		return 0, false
	}

	const attempts = 4
	for attempt := 0; attempt < attempts; attempt++ {
		if attempt > 0 {
			time.Sleep(time.Duration(attempt) * 400 * time.Millisecond)
		}
		st, err := pt.venue.GetOrderStatus(result.OrderID)
		if err != nil {
			pt.logWarnf("order %s status poll failed (try %d/%d): %v", result.OrderID, attempt+1, attempts, err)
			continue
		}
		if st == nil || st.SizeMatched <= 0 {
			if st != nil && st.Terminal {
				break
			}
			continue
		}
		if st.OriginalSize > 0 {
			usd = d.SizeUsd * (st.SizeMatched / st.OriginalSize)
		} else {
			usd = d.SizeUsd
		}
		return usd, true
	}
	return 0, false
}

func truncateToken(id string) string {
	if len(id) <= 12 {
		return id
	}
	return id[:6] + "..." + id[len(id)-4:]
}
