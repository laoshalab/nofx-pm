package trader

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"nofx/mcp"
	"nofx/prediction/config"
	"nofx/prediction/engine"
	"nofx/prediction/risk"
	"nofx/prediction/telemetry"
	"nofx/prediction/types"
	"nofx/store"
)

// PredictionTrader is the parallel runtime to crypto AutoTrader (M2).
type PredictionTrader struct {
	id         string
	name       string
	cfg        config.TraderConfig
	venue      types.PredictionVenue
	engine     *engine.PredictionEngine
	riskGate   *risk.Gate
	mcpClient  mcp.AIClient
	st         *store.Store
	userID     string
	traderDBID string

	callCount      int
	startTime      time.Time
	dailyVolume    float64
	lastReset      time.Time
	recentDecisions []types.PredictionDecision

	lastCycleAt         time.Time
	lastCycleOK         bool
	lastCycleError      string
	lastDecisionCount   int
	lastExecutionCount  int

	isRunning      bool
	loopActive     bool
	isRunningMutex sync.RWMutex
	cycleMu        sync.Mutex // serializes runCycle (main ticker + fast loop)
	stopCh         chan struct{}
}

func NewPredictionTrader(cfg config.TraderConfig, venue types.PredictionVenue, client mcp.AIClient) (*PredictionTrader, error) {
	if venue == nil {
		return nil, fmt.Errorf("venue is nil")
	}
	strat := cfg.Strategy
	if strat.Risk.MaxOrderUsd == 0 {
		strat.Risk = risk.DefaultConfig()
		cfg.Strategy = strat
	}
	mode := strat.Mode
	if mode == "" {
		mode = config.ModeAI
	}
	needsAI := mode == config.ModeAI || mode == config.ModeHybrid
	if needsAI && client == nil {
		return nil, fmt.Errorf("AI client is nil")
	}
	if client != nil {
		client.SetAPIKey(cfg.APIKey, cfg.CustomURL, cfg.CustomModel)
	}
	return &PredictionTrader{
		id:        cfg.ID,
		name:      cfg.Name,
		cfg:       cfg,
		venue:     venue,
		engine:    engine.NewPredictionEngine(strat, venue),
		riskGate:  risk.NewGate(strat.Risk),
		mcpClient: client,
		startTime: time.Now(),
		lastReset: time.Now(),
		stopCh:    make(chan struct{}),
	}, nil
}

func (pt *PredictionTrader) ID() string   { return pt.id }
func (pt *PredictionTrader) Name() string { return pt.name }

// SetStore enables persisting decision records to SQLite.
func (pt *PredictionTrader) SetStore(st *store.Store, userID, traderID string) {
	pt.st = st
	pt.userID = userID
	pt.traderDBID = traderID
}

func (pt *PredictionTrader) Run() error {
	pt.isRunningMutex.Lock()
	if pt.isRunning {
		pt.isRunningMutex.Unlock()
		return fmt.Errorf("prediction trader already running")
	}
	pt.isRunning = true
	pt.loopActive = true
	pt.stopCh = make(chan struct{})
	pt.isRunningMutex.Unlock()

	interval := pt.cfg.ScanInterval
	if interval <= 0 {
		interval = pt.cfg.Strategy.ScanIntervalMin
	}
	if interval <= 0 {
		interval = 5
	}
	ticker := time.NewTicker(time.Duration(interval) * time.Minute)
	defer ticker.Stop()

	if sec := pt.cfg.Strategy.FastLoopSec; sec > 0 {
		go pt.runFastLoop(sec)
	}

	// First cycle immediately
	if _, err := pt.runCycle(); err != nil {
		pt.logWarnf("cycle error: %v", err)
	}

	for {
		select {
		case <-pt.stopCh:
			return nil
		case <-ticker.C:
			if _, err := pt.runCycle(); err != nil {
				pt.logWarnf("cycle error: %v", err)
			}
		}
	}
}

func (pt *PredictionTrader) Stop() {
	pt.isRunningMutex.Lock()
	defer pt.isRunningMutex.Unlock()
	if !pt.isRunning {
		return
	}
	pt.isRunning = false
	pt.loopActive = false
	close(pt.stopCh)
}

// IsRunning reports whether the periodic loop goroutine is active.
func (pt *PredictionTrader) IsRunning() bool {
	pt.isRunningMutex.RLock()
	defer pt.isRunningMutex.RUnlock()
	return pt.isRunning
}

// RunOnce executes a single cycle (for CLI / testing).
func (pt *PredictionTrader) RunOnce() (*engine.FullDecision, error) {
	return pt.runCycle()
}

func (pt *PredictionTrader) runCycle() (*engine.FullDecision, error) {
	pt.cycleMu.Lock()
	defer pt.cycleMu.Unlock()
	defer pt.recordSimSnapshot()

	cycleStart := time.Now()
	pt.callCount++
	telemetry.RecordTrader(pt.traderIDForTelemetry(), pt.name, "cycle", "info", "cycle started", "", map[string]interface{}{
		"cycle": pt.callCount,
	})

	pt.syncDailyVolumeFromStore()

	pt.tryAutoRedeem()

	runtimeMin := int(time.Since(pt.startTime).Minutes())
	ctx, err := engine.BuildContext(pt.venue, pt.engine, pt.callCount, runtimeMin, pt.recentDecisions)
	if err != nil {
		pt.recordCycleTelemetry(cycleStart, 0, 0, err)
		return nil, err
	}

	positions, _ := pt.venue.GetOutcomePositions()
	pt.riskGate.SetPositions(positions)
	pt.riskGate.SetDailyVolume(pt.dailyVolume)

	var decision *engine.FullDecision
	mode := pt.effectiveMode()
	switch mode {
	case config.ModeRules:
		decision = pt.runRulesCycle(ctx, false)
	case config.ModeHybrid:
		decision, err = pt.runHybridCycle(ctx)
	default:
		decision, err = pt.runAICycle(ctx, false)
	}
	if err != nil {
		pt.saveFailedCycleRecord(decision, err)
		pt.notifyCycleFailed(err)
		decCount := 0
		if decision != nil {
			decCount = len(decision.Decisions)
		}
		pt.recordCycleTelemetry(cycleStart, decCount, 0, err)
		return decision, err
	}
	if decision == nil {
		pt.recordCycleTelemetry(cycleStart, 0, 0, nil)
		return nil, nil
	}

	sorted := sortDecisions(decision.Decisions)
	executions := make([]types.ExecutionOutcome, 0, len(sorted))
	for _, d := range sorted {
		if pt.shouldAbortExecution() {
			break
		}
		dCopy := d
		out := pt.executeDecisionWithOutcome(&dCopy)
		executions = append(executions, out)
		pt.auditExecution(out)
		if out.Status == "error" || out.Status == "risk_blocked" || out.Status == "rejected" {
			pt.logWarnf("execute %s %s: %s", d.MarketSlug, d.Action, out.Message)
		}
	}
	decision.Executions = executions
	pt.saveDecisionRecord(decision, err)
	pt.syncOrdersAfterCycle()
	pt.recordCycleTelemetry(cycleStart, len(decision.Decisions), len(executions), nil)

	if decision != nil {
		pt.recentDecisions = append(pt.recentDecisions, decision.Decisions...)
		if len(pt.recentDecisions) > 20 {
			pt.recentDecisions = pt.recentDecisions[len(pt.recentDecisions)-20:]
		}
	}
	return decision, nil
}

func (pt *PredictionTrader) recordSimSnapshot() {
	if snap, ok := pt.venue.(interface {
		RecordEquitySnapshot(cycleNumber int) error
	}); ok {
		if err := snap.RecordEquitySnapshot(pt.callCount); err != nil {
			pt.logWarnf("sim snapshot: %v", err)
		}
	}
}

func (pt *PredictionTrader) shouldAbortExecution() bool {
	pt.isRunningMutex.RLock()
	defer pt.isRunningMutex.RUnlock()
	return pt.loopActive && !pt.isRunning
}

func (pt *PredictionTrader) saveDecisionRecord(decision *engine.FullDecision, cycleErr error) {
	if pt.st == nil || pt.traderDBID == "" || decision == nil {
		return
	}
	decJSON, _ := json.Marshal(decision.Decisions)
	execJSON, _ := json.Marshal(decision.Executions)
	rec := &store.PredictionDecisionDB{
		TraderID:            pt.traderDBID,
		CycleNumber:         pt.callCount,
		Timestamp:           decision.Timestamp,
		CoTTrace:            decision.CoTTrace,
		DecisionsJSON:       string(decJSON),
		ExecutionsJSON:      string(execJSON),
		RawResponse:         decision.RawResponse,
		Success:             cycleErr == nil,
		AIRequestDurationMs: decision.AIRequestDurationMs,
	}
	if cycleErr != nil {
		rec.ErrorMessage = cycleErr.Error()
	}
	_ = pt.st.Prediction().SaveDecision(rec)
}

func (pt *PredictionTrader) saveFailedCycleRecord(decision *engine.FullDecision, cycleErr error) {
	if cycleErr == nil || pt.st == nil || pt.traderDBID == "" {
		return
	}
	if decision == nil {
		decision = &engine.FullDecision{
			Timestamp: time.Now().UTC(),
			Decisions: []types.PredictionDecision{{
				Action:    types.ActionWait,
				Reasoning: cycleErr.Error(),
			}},
		}
	}
	pt.saveDecisionRecord(decision, cycleErr)
}
