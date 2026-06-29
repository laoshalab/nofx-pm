package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"nofx/crypto"
	"nofx/logger"
	"nofx/manager"
	predcfg "nofx/prediction/config"
	"nofx/prediction/sim"
	"nofx/prediction/types"
	"nofx/store"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type createPredictionTraderRequest struct {
	Name                string                 `json:"name" binding:"required"`
	AIModelID           string                 `json:"ai_model_id" binding:"required"`
	Venue               string                 `json:"venue"`
	PrivateKey          string                 `json:"private_key"`
	ProxyAddress        string                 `json:"proxy_address"`
	SignatureType       int                    `json:"signature_type"`
	ScanIntervalMinutes int                    `json:"scan_interval_minutes"`
	PreviewMode         *bool                  `json:"preview_mode"`
	TradingMode         string                 `json:"trading_mode"`
	SimConfig           map[string]interface{} `json:"sim_config"`
	Strategy            predcfg.StrategyConfig `json:"strategy"`
}

type updatePredictionTraderRequest struct {
	Name                string                 `json:"name" binding:"required"`
	AIModelID           string                 `json:"ai_model_id" binding:"required"`
	ProxyAddress        string                 `json:"proxy_address"`
	SignatureType       int                    `json:"signature_type"`
	ScanIntervalMinutes int                    `json:"scan_interval_minutes"`
	PreviewMode         *bool                  `json:"preview_mode"`
	TradingMode         string                 `json:"trading_mode"`
	SimConfig           map[string]interface{} `json:"sim_config"`
	Strategy            predcfg.StrategyConfig `json:"strategy"`
	PrivateKey          string                 `json:"private_key,omitempty"`
}

func parseListPage(c *gin.Context, defaultLimit int) (limit, offset int) {
	limit, _ = strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(defaultLimit)))
	offset, _ = strconv.Atoi(c.DefaultQuery("offset", "0"))
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > 100 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

func writePagedJSON(c *gin.Context, items interface{}, limit, offset int, hasMore bool) {
	c.JSON(http.StatusOK, gin.H{
		"items":    items,
		"limit":    limit,
		"offset":   offset,
		"has_more": hasMore,
	})
}

func (s *Server) handlePredictionMarketSearch(c *gin.Context) {
	tag := c.DefaultQuery("tag", "crypto")
	keyword := c.Query("keyword")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	filter := types.MarketFilter{Tag: tag, Limit: limit}
	if keyword != "" {
		filter.Keywords = []string{keyword}
	}
	withMids := strings.EqualFold(c.Query("with_mids"), "true") || c.Query("with_mids") == "1"
	if withMids {
		markets, err := manager.SearchMarketsWithMids(filter)
		if err != nil {
			c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, markets)
		return
	}
	markets, err := manager.SearchMarkets(filter)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, markets)
}

func (s *Server) handlePredictionMarketDetail(c *gin.Context) {
	slug := c.Param("slug")
	m, yesMid, noMid, err := manager.GetMarketBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"market":  m,
		"yes_mid": yesMid,
		"no_mid":  noMid,
	})
}

func (s *Server) handlePredictionTraderList(c *gin.Context) {
	userID := c.GetString("user_id")
	if err := s.predictionManager.LoadUserTradersFromStore(s.store, userID); err != nil {
		logger.Warnf("prediction load: %v", err)
	}
	rows, err := s.store.Prediction().ListTraders(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		loadErr := ""
		if le := s.predictionManager.GetLoadError(row.ID); le != nil {
			loadErr = le.Error()
		}
		out = append(out, gin.H{
			"id":                    row.ID,
			"name":                  row.Name,
			"ai_model_id":           row.AIModelID,
			"venue":                 row.Venue,
			"scan_interval_minutes": row.ScanIntervalMinutes,
			"preview_mode":          row.PreviewMode,
			"trading_mode":          row.EffectiveTradingMode(),
			"is_running":            row.IsRunning,
			"show_in_competition":   row.ShowInCompetition,
			"load_error":            loadErr,
			"created_at":            row.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) handleGetPredictionTrader(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	row, err := s.store.Prediction().GetTrader(userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}
	var strat predcfg.StrategyConfig
	_ = store.ParseStrategyJSON(row.StrategyJSON, &strat)
	var simCfg map[string]interface{}
	_ = json.Unmarshal([]byte(row.SimConfigJSON), &simCfg)
	loadErr := ""
	if le := s.predictionManager.GetLoadError(row.ID); le != nil {
		loadErr = le.Error()
	}
	dailyVol, volumeDay, _ := s.store.Prediction().LoadRiskState(id)
	c.JSON(http.StatusOK, gin.H{
		"id":                    row.ID,
		"name":                  row.Name,
		"ai_model_id":           row.AIModelID,
		"venue":                 row.Venue,
		"proxy_address":         row.ProxyAddress,
		"signature_type":        row.SignatureType,
		"scan_interval_minutes": row.ScanIntervalMinutes,
		"preview_mode":          row.PreviewMode,
		"trading_mode":          row.EffectiveTradingMode(),
		"is_running":            row.IsRunning,
		"show_in_competition":   row.ShowInCompetition,
		"load_error":            loadErr,
		"strategy":              strat,
		"sim_config":            simCfg,
		"has_private_key":       row.PrivateKey.String() != "",
		"daily_volume_usd":      dailyVol,
		"volume_day":            volumeDay,
	})
}

func (s *Server) handleCreatePredictionTrader(c *gin.Context) {
	userID := c.GetString("user_id")
	var req createPredictionTraderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := s.store.AIModel().Get(userID, req.AIModelID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ai_model_id"})
		return
	}
	preview := true
	if req.PreviewMode != nil {
		preview = *req.PreviewMode
	}
	tradingMode := strings.TrimSpace(req.TradingMode)
	if tradingMode == "" {
		if preview {
			tradingMode = store.PredictionTradingModePreview
		} else {
			tradingMode = store.PredictionTradingModeLive
		}
	}
	if tradingMode == store.PredictionTradingModeLive {
		if denyLiveModeRequest(c, tradingMode) {
			return
		}
		if strings.TrimSpace(req.PrivateKey) == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "private_key required for live trading"})
			return
		}
		if rejectLiveBrowserPrivateKey(c, tradingMode, req.PrivateKey) {
			return
		}
	}
	if !store.IsValidTradingMode(tradingMode) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trading_mode; use simulation, preview, or live"})
		return
	}
	if rejectPreviewWithoutPrivateKey(c, tradingMode, nil, req.PrivateKey) {
		return
	}
	strat := req.Strategy.WithDefaults()
	strat.Risk.PreviewMode = tradingMode != store.PredictionTradingModeLive
	if tradingMode == store.PredictionTradingModeSimulation {
		strat.Risk.PreviewMode = false
	}

	venue := req.Venue
	if venue == "" {
		venue = "polymarket"
	}
	scan := req.ScanIntervalMinutes
	if scan <= 0 {
		scan = 5
	}

	row := &store.PredictionTraderDB{
		ID:                  uuid.New().String(),
		UserID:              userID,
		Name:                strings.TrimSpace(req.Name),
		AIModelID:           req.AIModelID,
		Venue:               venue,
		PrivateKey:          crypto.EncryptedString(strings.TrimSpace(req.PrivateKey)),
		ProxyAddress:        req.ProxyAddress,
		SignatureType:       req.SignatureType,
		StrategyJSON:        store.MustStrategyJSON(strat),
		ScanIntervalMinutes: scan,
		PreviewMode:         tradingMode != store.PredictionTradingModeLive,
		TradingMode:         tradingMode,
		SimConfigJSON:       store.MustStrategyJSON(req.SimConfig),
	}
	if err := s.store.Prediction().CreateTrader(row); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	_ = s.predictionManager.LoadUserTradersFromStore(s.store, userID)
	c.JSON(http.StatusCreated, gin.H{"id": row.ID, "name": row.Name})
}

func (s *Server) handleUpdatePredictionTrader(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	var req updatePredictionTraderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	row, err := s.store.Prediction().GetTrader(userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}
	if _, err := s.store.AIModel().Get(userID, req.AIModelID); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid ai_model_id"})
		return
	}
	row.Name = req.Name
	row.AIModelID = req.AIModelID
	row.ProxyAddress = req.ProxyAddress
	if req.SignatureType > 0 {
		row.SignatureType = req.SignatureType
	}
	if req.ScanIntervalMinutes > 0 {
		row.ScanIntervalMinutes = req.ScanIntervalMinutes
	}
	wasRunning := row.IsRunning
	if tm := strings.TrimSpace(req.TradingMode); tm != "" {
		if !store.IsValidTradingMode(tm) {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid trading_mode; use simulation, preview, or live"})
			return
		}
		if tm == store.PredictionTradingModeLive {
			if denyLiveModeRequest(c, tm) {
				return
			}
		}
		row.TradingMode = tm
		row.PreviewMode = tm != store.PredictionTradingModeLive
	} else if req.PreviewMode != nil && row.TradingMode != store.PredictionTradingModeSimulation {
		row.PreviewMode = *req.PreviewMode
		if *req.PreviewMode {
			row.TradingMode = store.PredictionTradingModePreview
		} else {
			if denyLiveModeRequest(c, store.PredictionTradingModeLive) {
				return
			}
			row.TradingMode = store.PredictionTradingModeLive
		}
	}
	if req.SimConfig != nil {
		row.SimConfigJSON = store.MustStrategyJSON(req.SimConfig)
	}
	if req.Strategy.Mode != "" || len(req.Strategy.StaticSlugs) > 0 ||
		req.Strategy.MarketSource.Type != "" || req.Strategy.MarketSource.Tag != "" ||
		req.Strategy.Risk.MaxOrderUsd > 0 || req.Strategy.FastLoopSec > 0 {
		strat := req.Strategy.WithDefaults()
		tm := row.EffectiveTradingMode()
		strat.Risk.PreviewMode = tm != store.PredictionTradingModeLive
		if tm == store.PredictionTradingModeSimulation {
			strat.Risk.PreviewMode = false
		}
		row.StrategyJSON = store.MustStrategyJSON(strat)
	}
	if rejectLiveBrowserPrivateKey(c, row.EffectiveTradingMode(), req.PrivateKey) {
		return
	}
	if rejectLiveWithoutPrivateKey(c, row, req.PrivateKey) {
		return
	}
	if rejectPreviewWithoutPrivateKey(c, row.TradingMode, row, req.PrivateKey) {
		return
	}
	if err := s.store.Prediction().UpdateTrader(row); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if req.PrivateKey != "" {
		_ = s.store.Prediction().UpdatePrivateKey(userID, id, req.PrivateKey)
	}
	s.predictionManager.RemoveTrader(id)
	if wasRunning {
		_ = s.store.Prediction().UpdateStatus(userID, id, false)
	}
	_ = s.predictionManager.LoadUserTradersFromStore(s.store, userID)
	c.JSON(http.StatusOK, gin.H{"success": true, "is_running": false})
}

func (s *Server) handleDeletePredictionTrader(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	if _, ok := s.ownedPredictionTrader(c, id); !ok {
		return
	}
	if pt, err := s.predictionManager.GetTrader(id); err == nil {
		pt.Stop()
	}
	s.predictionManager.RemoveTrader(id)
	if err := s.store.Prediction().DeleteTrader(userID, id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) handleStartPredictionTrader(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	row, ok := s.ownedPredictionTrader(c, id)
	if !ok {
		return
	}
	if denyLiveTradingDisabled(c, row) {
		return
	}
	_ = s.predictionManager.LoadUserTradersFromStore(s.store, userID)
	pt, ptErr := s.predictionManager.GetTrader(id)
	runtimeRunning := ptErr == nil && pt != nil && pt.IsRunning()
	if row.IsRunning && !runtimeRunning {
		_ = s.store.Prediction().UpdateStatus(userID, id, false)
		row.IsRunning = false
	}
	if runtimeRunning {
		if !row.IsRunning {
			_ = s.store.Prediction().UpdateStatus(userID, id, true)
		}
		c.JSON(http.StatusConflict, gin.H{"error": "trader is already running"})
		return
	}
	if row.IsRunning {
		c.JSON(http.StatusConflict, gin.H{"error": "trader is already running"})
		return
	}
	if ptErr != nil {
		if loadErr := s.predictionManager.GetLoadError(id); loadErr != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": loadErr.Error()})
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": ptErr.Error()})
		return
	}
	_ = s.store.Prediction().UpdateStatus(userID, id, true)
	go func() {
		if runErr := pt.Run(); runErr != nil {
			logger.Warnf("[prediction] run error: %v", runErr)
		}
		_ = s.store.Prediction().UpdateStatus(userID, id, false)
	}()
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) handleStopPredictionTrader(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	if _, ok := s.ownedPredictionTrader(c, id); !ok {
		return
	}
	if pt, err := s.predictionManager.GetTrader(id); err == nil {
		pt.Stop()
	}
	_ = s.store.Prediction().UpdateStatus(userID, id, false)
	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) handlePredictionDecisions(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	if _, err := s.store.Prediction().GetTrader(userID, id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}
	limit, offset := parseListPage(c, 30)
	includeWire := includeFullPreviewWire(c.Query("include_wire"))
	rows, hasMore, err := s.store.Prediction().ListDecisionsPage(id, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, row := range rows {
		var decisions []types.PredictionDecision
		var executions []types.ExecutionOutcome
		_ = json.Unmarshal([]byte(row.DecisionsJSON), &decisions)
		if row.ExecutionsJSON != "" && row.ExecutionsJSON != "[]" {
			_ = json.Unmarshal([]byte(row.ExecutionsJSON), &executions)
		}
		if !includeWire {
			executions = redactExecutionOutcomes(executions)
		}
		out = append(out, gin.H{
			"id":                     row.ID,
			"cycle_number":           row.CycleNumber,
			"timestamp":              row.Timestamp.Format(time.RFC3339),
			"cot_trace":              row.CoTTrace,
			"decisions":              decisions,
			"executions":             executions,
			"success":                row.Success,
			"error_message":          row.ErrorMessage,
			"ai_request_duration_ms": row.AIRequestDurationMs,
		})
	}
	writePagedJSON(c, out, limit, offset, hasMore)
}

func (s *Server) handlePredictionPositions(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	row, err := s.store.Prediction().GetTrader(userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}
	if !predictionWalletConfigured(row) {
		c.JSON(http.StatusOK, gin.H{
			"available_usdc":      0,
			"total_equity":        0,
			"positions":           []types.OutcomePosition{},
			"wallet_configured":   false,
			"wallet_setup_hint":   "Configure Polymarket private key or proxy address for preview/live positions",
		})
		return
	}
	venue := manager.BuildVenueForTrader(s.store, row)
	positions, err := venue.GetOutcomePositions()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	avail, equity, _ := venue.GetCollateral()
	c.JSON(http.StatusOK, gin.H{
		"available_usdc":    avail,
		"total_equity":      equity,
		"positions":         positions,
		"wallet_configured": true,
	})
}

func (s *Server) handlePredictionRunOnce(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	row, ok := s.ownedPredictionTrader(c, id)
	if !ok {
		return
	}
	if denyLiveTradingDisabled(c, row) {
		return
	}
	_ = s.predictionManager.LoadUserTradersFromStore(s.store, userID)
	pt, err := s.predictionManager.GetTrader(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	decision, err := pt.RunOnce()
	if err != nil {
		resp := gin.H{"error": err.Error()}
		if decision != nil {
			resp["cot"] = decision.CoTTrace
			resp["decisions"] = decision.Decisions
			execs := decision.Executions
			if execs == nil {
				execs = []types.ExecutionOutcome{}
			}
			resp["executions"] = redactExecutionOutcomes(execs)
		}
		c.JSON(http.StatusBadGateway, resp)
		return
	}
	executions := decision.Executions
	if executions == nil {
		executions = []types.ExecutionOutcome{}
	}
	executions = redactExecutionOutcomes(executions)
	c.JSON(http.StatusOK, gin.H{
		"cot":         decision.CoTTrace,
		"decisions":   decision.Decisions,
		"executions":  executions,
		"duration_ms": decision.AIRequestDurationMs,
	})
}

func (s *Server) handlePredictionRedeem(c *gin.Context) {
	id := c.Param("id")
	row, ok := s.ownedPredictionTrader(c, id)
	if !ok {
		return
	}
	if denyLiveTradingDisabled(c, row) {
		return
	}
	if row.EffectiveTradingMode() == store.PredictionTradingModeLive && !predictionLiveRedeemEnabled() {
		c.JSON(http.StatusNotImplemented, gin.H{
			"error": "live redeem is not configured; set POLYMARKET_BUILDER_* and POLYGON_RPC_URL",
		})
		return
	}
	venue := manager.BuildVenueForTrader(s.store, row)
	redeemer, ok := venue.(interface {
		FindRedeemablePositions() ([]types.RedeemablePosition, error)
		RedeemCondition(conditionID string, negRisk bool) (*types.RedeemResult, error)
	})
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": "venue does not support redeem"})
		return
	}
	items, err := redeemer.FindRedeemablePositions()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	results := make([]*types.RedeemResult, 0, len(items))
	for _, item := range items {
		res, err := redeemer.RedeemCondition(item.ConditionID, item.NegRisk)
		if err != nil {
			results = append(results, &types.RedeemResult{
				ConditionID: item.ConditionID,
				Status:      "error",
				Error:       err.Error(),
			})
			continue
		}
		results = append(results, res)
	}
	if snap, ok := venue.(interface {
		RecordEquitySnapshot(cycleNumber int) error
	}); ok {
		_ = snap.RecordEquitySnapshot(0)
	}
	c.JSON(http.StatusOK, gin.H{"redeemed": len(results), "results": results})
}

func (s *Server) handlePredictionSimPnL(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	row, err := s.store.Prediction().GetTrader(userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}
	if row.EffectiveTradingMode() != store.PredictionTradingModeSimulation {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trader is not in simulation mode"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "100"))
	snapshots, err := s.store.Prediction().ListSimSnapshots(id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	cash, initial, ok, _ := s.store.Prediction().LoadSimAccount(id)
	venue := manager.BuildVenueForTrader(s.store, row)
	avail, equity, _ := venue.GetCollateral()
	posValue := equity - avail
	if posValue < 0 {
		posValue = 0
	}
	if !ok {
		initial = avail
	}
	pnl := equity - initial
	pnlPct := 0.0
	if initial > 0 {
		pnlPct = (pnl / initial) * 100
	}

	outSnaps := make([]gin.H, 0, len(snapshots))
	for _, snap := range snapshots {
		outSnaps = append(outSnaps, gin.H{
			"cycle_number":   snap.CycleNumber,
			"cash_usdc":      snap.CashUsdc,
			"position_value": snap.PositionValue,
			"total_equity":   snap.TotalEquity,
			"total_pnl":      snap.TotalPnl,
			"total_pnl_pct":  snap.TotalPnlPct,
			"open_positions": snap.OpenPositions,
			"timestamp":      snap.Timestamp.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"initial_balance": initial,
		"cash_usdc":       cash,
		"available_usdc":  avail,
		"position_value":  posValue,
		"total_equity":    equity,
		"total_pnl":       pnl,
		"total_pnl_pct":   pnlPct,
		"snapshots":       outSnaps,
	})
}

func (s *Server) handlePredictionSimFills(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	row, err := s.store.Prediction().GetTrader(userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}
	if row.EffectiveTradingMode() != store.PredictionTradingModeSimulation {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trader is not in simulation mode"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	fills, err := s.store.Prediction().ListSimFills(id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	out := make([]gin.H, 0, len(fills))
	for _, f := range fills {
		out = append(out, gin.H{
			"id":              f.ID,
			"order_id":        f.OrderID,
			"token_id":        f.TokenID,
			"market_slug":     f.MarketSlug,
			"outcome":         f.Outcome,
			"side":            f.Side,
			"fill_price":      f.FillPrice,
			"fill_shares":     f.FillShares,
			"fill_usd":        f.FillUsd,
			"cycle_number":    f.CycleNumber,
			"decision_action": f.DecisionAction,
			"reasoning":       f.Reasoning,
			"timestamp":       f.Timestamp.Format(time.RFC3339),
		})
	}
	c.JSON(http.StatusOK, out)
}

func (s *Server) handlePredictionOrders(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	if _, err := s.store.Prediction().GetTrader(userID, id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}

	limit, offset := parseListPage(c, 50)
	includeWire := includeFullPreviewWire(c.Query("include_wire"))
	rows, hasMore, err := s.store.Prediction().ListOrderRecordsPage(id, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	writePagedJSON(c, formatOrderRecords(rows, includeWire), limit, offset, hasMore)
}

func (s *Server) handlePredictionAuditLogs(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	if _, err := s.store.Prediction().GetTrader(userID, id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}
	limit, offset := parseListPage(c, 50)
	rows, hasMore, err := s.store.Prediction().ListAuditLogsPage(id, limit, offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		out = append(out, gin.H{
			"id":           r.ID,
			"cycle_number": r.CycleNumber,
			"event_type":   r.EventType,
			"market_slug":  r.MarketSlug,
			"action":       r.Action,
			"order_id":     r.OrderID,
			"amount_usd":   r.AmountUsd,
			"status":       r.Status,
			"message":      r.Message,
			"created_at":   r.CreatedAt.Format(time.RFC3339),
		})
	}
	writePagedJSON(c, out, limit, offset, hasMore)
}

func (s *Server) handlePredictionOrdersSync(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	row, err := s.store.Prediction().GetTrader(userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}
	if row.EffectiveTradingMode() == store.PredictionTradingModeSimulation {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order sync is for preview/live traders"})
		return
	}

	venue := manager.BuildVenueForTrader(s.store, row)
	records, _ := s.store.Prediction().ListOrderRecords(id, 100)
	for _, rec := range records {
		if rec.IsPreview || rec.OrderID == "" {
			continue
		}
		if rec.Status == "filled" || rec.Status == "cancelled" || rec.Status == "canceled" {
			continue
		}
		st, err := venue.GetOrderStatus(rec.OrderID)
		if err != nil || st == nil {
			continue
		}
		status := strings.ToLower(st.Status)
		if st.SizeMatched > 0 && st.OriginalSize > 0 && st.SizeMatched >= st.OriginalSize {
			status = "filled"
		}
		_ = s.store.Prediction().UpdateOrderStatus(id, rec.OrderID, status)
	}

	if openOrders, err := venue.ListOpenOrders(); err == nil {
		for _, o := range openOrders {
			_ = s.store.Prediction().SaveOrderRecord(&store.PredictionOrderRecordDB{
				TraderID: id,
				OrderID:  o.OrderID,
				TokenID:  o.TokenID,
				Side:     o.Side,
				Price:    o.Price,
				Size:     o.OriginalSize,
				Status:   o.Status,
			})
		}
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
	includeWire := includeFullPreviewWire(c.Query("include_wire"))
	rows, err := s.store.Prediction().ListOrderRecords(id, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, formatOrderRecords(rows, includeWire))
}

func (s *Server) handlePredictionCancelOrder(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	orderID := strings.TrimSpace(c.Param("order_id"))
	if orderID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order_id required"})
		return
	}
	row, err := s.store.Prediction().GetTrader(userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}
	if row.EffectiveTradingMode() == store.PredictionTradingModeSimulation {
		c.JSON(http.StatusBadRequest, gin.H{"error": "cancel is for preview/live orders only"})
		return
	}
	rec, err := s.store.Prediction().GetOrderRecord(id, orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "order not found for this trader"})
		return
	}
	if rec.Status == "filled" || rec.Status == "cancelled" || rec.Status == "canceled" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "order is already terminal: " + rec.Status})
		return
	}
	venue := manager.BuildVenueForTrader(s.store, row)
	if err := venue.CancelOrder(orderID); err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	status := "cancelled"
	if rec.IsPreview {
		status = "preview"
	}
	_ = s.store.Prediction().UpdateOrderStatus(id, orderID, status)
	c.JSON(http.StatusOK, gin.H{"success": true, "order_id": orderID, "status": status})
}

func formatOrderRecords(rows []store.PredictionOrderRecordDB, includeWire bool) []gin.H {
	out := make([]gin.H, 0, len(rows))
	for _, r := range rows {
		item := gin.H{
			"id":           r.ID,
			"order_id":     r.OrderID,
			"cycle_number": r.CycleNumber,
			"market_slug":  r.MarketSlug,
			"token_id":     r.TokenID,
			"side":         r.Side,
			"price":        r.Price,
			"size":         r.Size,
			"status":       r.Status,
			"is_preview":   r.IsPreview,
			"created_at":   r.CreatedAt.Format(time.RFC3339),
		}
		if r.PreviewWire != "" {
			if includeWire {
				item["preview_wire"] = r.PreviewWire
			} else {
				item["preview_wire"] = redactPreviewWireField(r.PreviewWire)
			}
		}
		out = append(out, item)
	}
	return out
}

type simResetRequest struct {
	InitialBalanceUsd float64 `json:"initial_balance_usd"`
}

func (s *Server) handlePredictionSimReset(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")
	row, err := s.store.Prediction().GetTrader(userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}
	if row.EffectiveTradingMode() != store.PredictionTradingModeSimulation {
		c.JSON(http.StatusBadRequest, gin.H{"error": "trader is not in simulation mode"})
		return
	}

	var req simResetRequest
	_ = c.ShouldBindJSON(&req)

	simCfgMap := map[string]interface{}{}
	_ = store.ParseSimConfigJSON(row.SimConfigJSON, &simCfgMap)
	if req.InitialBalanceUsd > 0 {
		simCfgMap["initial_balance_usd"] = req.InitialBalanceUsd
		row.SimConfigJSON = store.MustStrategyJSON(simCfgMap)
		if err := s.store.Prediction().UpdateTrader(row); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
	}

	cfg := sim.DefaultConfig()
	_ = store.ParseSimConfigJSON(row.SimConfigJSON, &cfg)
	if err := sim.ResetSimAccount(s.store, id, cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	s.predictionManager.RemoveTrader(id)
	_ = s.predictionManager.LoadUserTradersFromStore(s.store, userID)

	c.JSON(http.StatusOK, gin.H{
		"success":         true,
		"initial_balance": cfg.InitialBalanceUsd,
	})
}
