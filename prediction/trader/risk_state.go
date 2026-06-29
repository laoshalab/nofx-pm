package trader

import (
	"time"

	"nofx/prediction/telemetry"
	"nofx/prediction/types"
	"nofx/store"
)

func utcVolumeDay() string {
	return time.Now().UTC().Format("2006-01-02")
}

func (pt *PredictionTrader) syncDailyVolumeFromStore() {
	if pt.st == nil || pt.traderDBID == "" {
		pt.maybeResetDailyVolumeLocal()
		return
	}
	vol, day, err := pt.st.Prediction().LoadRiskState(pt.traderDBID)
	if err != nil {
		pt.logWarnf("load risk state: %v", err)
		pt.maybeResetDailyVolumeLocal()
		return
	}
	today := utcVolumeDay()
	if day != today {
		pt.dailyVolume = 0
		pt.lastReset = time.Now().UTC()
		return
	}
	pt.dailyVolume = vol
	pt.lastReset = time.Now().UTC()
}

func (pt *PredictionTrader) maybeResetDailyVolumeLocal() {
	if time.Since(pt.lastReset) > 24*time.Hour {
		pt.dailyVolume = 0
		pt.lastReset = time.Now().UTC()
	}
}

func (pt *PredictionTrader) persistDailyVolume() {
	if pt.st == nil || pt.traderDBID == "" {
		return
	}
	if err := pt.st.Prediction().SaveRiskState(pt.traderDBID, utcVolumeDay(), pt.dailyVolume); err != nil {
		pt.logWarnf("save risk state: %v", err)
	}
}

func (pt *PredictionTrader) recordFillVolume(usd float64) {
	if usd <= 0 {
		return
	}
	pt.dailyVolume += usd
	pt.persistDailyVolume()
}

func (pt *PredictionTrader) auditExecution(out types.ExecutionOutcome) {
	if pt.st == nil || pt.traderDBID == "" {
		return
	}
	eventType := out.Status
	if eventType == "" {
		return
	}
	skip := map[string]bool{"skipped": true, "hold": true, "wait": true}
	if skip[eventType] {
		return
	}

	amount := out.FillUsd
	msg := out.Message
	switch eventType {
	case "filled", "posted", "preview", "rejected", "risk_blocked", "error", "redeem":
		// keep
	default:
		eventType = "execution"
	}

	pt.logInfof("AUDIT %s %s %s order=%s usd=%.2f — %s",
		eventType, out.Action, out.MarketSlug, out.OrderID, amount, msg)

	telemetry.RecordTrader(pt.traderIDForTelemetry(), pt.name, "execute", auditLevel(eventType), eventType+" "+string(out.Action), msg, map[string]interface{}{
		"market_slug": out.MarketSlug,
		"order_id":    out.OrderID,
		"fill_usd":    amount,
		"status":      eventType,
	})

	_ = pt.st.Prediction().SaveAuditLog(&store.PredictionAuditLogDB{
		TraderID:    pt.traderDBID,
		CycleNumber: pt.callCount,
		EventType:   eventType,
		MarketSlug:  out.MarketSlug,
		Action:      string(out.Action),
		OrderID:     out.OrderID,
		AmountUsd:   amount,
		Status:      out.Status,
		Message:     msg,
	})
}

func auditLevel(eventType string) string {
	switch eventType {
	case "error", "rejected", "risk_blocked":
		return "error"
	case "preview", "posted":
		return "warn"
	default:
		return "info"
	}
}
