package trader

import (
	"time"

	"nofx/prediction/telemetry"
)

// RuntimeStatus is a point-in-time snapshot of trader loop state for the live console.
type RuntimeStatus struct {
	IsRunning           bool      `json:"is_running"`
	CycleCount          int       `json:"cycle_count"`
	UptimeSec           int64     `json:"uptime_sec"`
	ScanIntervalMinutes int       `json:"scan_interval_minutes"`
	LastCycleAt         time.Time `json:"last_cycle_at,omitempty"`
	LastCycleOK         bool      `json:"last_cycle_ok"`
	LastCycleError      string    `json:"last_cycle_error,omitempty"`
	LastDecisionCount   int       `json:"last_decision_count"`
	LastExecutionCount  int       `json:"last_execution_count"`
	DailyVolumeUsd      float64   `json:"daily_volume_usd"`
}

func (pt *PredictionTrader) RuntimeStatus() RuntimeStatus {
	pt.isRunningMutex.RLock()
	running := pt.isRunning
	pt.isRunningMutex.RUnlock()

	interval := pt.cfg.ScanInterval
	if interval <= 0 {
		interval = pt.cfg.Strategy.ScanIntervalMin
	}
	if interval <= 0 {
		interval = 5
	}

	return RuntimeStatus{
		IsRunning:           running,
		CycleCount:          pt.callCount,
		UptimeSec:           int64(time.Since(pt.startTime).Seconds()),
		ScanIntervalMinutes: interval,
		LastCycleAt:         pt.lastCycleAt,
		LastCycleOK:         pt.lastCycleOK,
		LastCycleError:      pt.lastCycleError,
		LastDecisionCount:   pt.lastDecisionCount,
		LastExecutionCount:  pt.lastExecutionCount,
		DailyVolumeUsd:      pt.dailyVolume,
	}
}

func (pt *PredictionTrader) traderIDForTelemetry() string {
	if pt.traderDBID != "" {
		return pt.traderDBID
	}
	return pt.id
}

func (pt *PredictionTrader) recordCycleTelemetry(start time.Time, decisionCount, executionCount int, cycleErr error) {
	pt.lastCycleAt = time.Now().UTC()
	pt.lastDecisionCount = decisionCount
	pt.lastExecutionCount = executionCount
	pt.lastCycleOK = cycleErr == nil
	pt.lastCycleError = ""
	if cycleErr != nil {
		pt.lastCycleError = cycleErr.Error()
	}

	level := "info"
	msg := "cycle completed"
	detail := ""
	meta := map[string]interface{}{
		"cycle":       pt.callCount,
		"decisions":   decisionCount,
		"executions":  executionCount,
		"duration_ms": time.Since(start).Milliseconds(),
	}
	if cycleErr != nil {
		level = "error"
		msg = "cycle failed"
		detail = cycleErr.Error()
	}
	telemetry.RecordTrader(pt.traderIDForTelemetry(), pt.name, "cycle", level, msg, detail, meta)
}
