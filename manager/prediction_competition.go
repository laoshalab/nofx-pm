package manager

import (
	"context"
	"sort"
	"time"

	"nofx/logger"
	"nofx/prediction/sim"
	"nofx/store"
)

// GetCompetitionData returns public leaderboard rows for prediction traders.
func (pm *PredictionManager) GetCompetitionData(st *store.Store) (map[string]interface{}, error) {
	pm.competitionCache.mu.RLock()
	if time.Since(pm.competitionCache.timestamp) < 30*time.Second && len(pm.competitionCache.data) > 0 {
		cached := make(map[string]interface{}, len(pm.competitionCache.data))
		for k, v := range pm.competitionCache.data {
			cached[k] = v
		}
		pm.competitionCache.mu.RUnlock()
		return cached, nil
	}
	pm.competitionCache.mu.RUnlock()

	rows, err := st.Prediction().ListCompetitionTraders()
	if err != nil {
		return nil, err
	}

	logger.Infof("🔄 Refreshing prediction competition data, trader count: %d", len(rows))
	traders := pm.fetchPredictionCompetitionRows(st, rows)

	sort.Slice(traders, func(i, j int) bool {
		pnlPctI, okI := traders[i]["total_pnl_pct"].(float64)
		pnlPctJ, okJ := traders[j]["total_pnl_pct"].(float64)
		if !okI {
			pnlPctI = 0
		}
		if !okJ {
			pnlPctJ = 0
		}
		return pnlPctI > pnlPctJ
	})

	totalCount := len(traders)
	limit := 50
	if len(traders) > limit {
		traders = traders[:limit]
	}

	out := map[string]interface{}{
		"traders":     traders,
		"count":       len(traders),
		"total_count": totalCount,
	}

	pm.competitionCache.mu.Lock()
	pm.competitionCache.data = out
	pm.competitionCache.timestamp = time.Now()
	pm.competitionCache.mu.Unlock()

	return out, nil
}

func (pm *PredictionManager) fetchPredictionCompetitionRows(st *store.Store, rows []*store.PredictionTraderDB) []map[string]interface{} {
	type result struct {
		index int
		data  map[string]interface{}
	}

	ch := make(chan result, len(rows))
	for i, row := range rows {
		go func(index int, row *store.PredictionTraderDB) {
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			defer cancel()

			done := make(chan map[string]interface{}, 1)
			go func() {
				done <- buildPredictionCompetitionRow(st, row)
			}()

			select {
			case data := <-done:
				ch <- result{index: index, data: data}
			case <-ctx.Done():
				ch <- result{index: index, data: buildPredictionCompetitionFallback(st, row)}
			}
		}(i, row)
	}

	ordered := make([]map[string]interface{}, len(rows))
	for range rows {
		res := <-ch
		ordered[res.index] = res.data
	}
	out := make([]map[string]interface{}, 0, len(ordered))
	for _, row := range ordered {
		if row != nil {
			out = append(out, row)
		}
	}
	return out
}

func buildPredictionCompetitionFallback(st *store.Store, row *store.PredictionTraderDB) map[string]interface{} {
	aiModel := resolvePredictionAIModel(st, row)
	return map[string]interface{}{
		"trader_id":       row.ID,
		"trader_name":     row.Name,
		"ai_model":        aiModel,
		"exchange":        row.Venue,
		"trading_mode":    row.EffectiveTradingMode(),
		"total_equity":    0.0,
		"total_pnl":       0.0,
		"total_pnl_pct":   0.0,
		"position_count":  0,
		"margin_used_pct": 0.0,
		"is_running":      row.IsRunning,
	}
}

func buildPredictionCompetitionRow(st *store.Store, row *store.PredictionTraderDB) map[string]interface{} {
	aiModel := resolvePredictionAIModel(st, row)
	venue := BuildVenueForTrader(st, row)

	avail, equity, _ := venue.GetCollateral()
	positions, _ := venue.GetOutcomePositions()
	posCount := len(positions)
	posValue := equity - avail
	if posValue < 0 {
		posValue = 0
	}

	initial := predictionInitialBalance(st, row)
	pnl := equity - initial
	pnlPct := 0.0
	if initial > 0 {
		pnlPct = (pnl / initial) * 100
	}
	marginPct := 0.0
	if equity > 0 {
		marginPct = (posValue / equity) * 100
	}

	return map[string]interface{}{
		"trader_id":       row.ID,
		"trader_name":     row.Name,
		"ai_model":        aiModel,
		"exchange":        row.Venue,
		"trading_mode":    row.EffectiveTradingMode(),
		"total_equity":    equity,
		"total_pnl":       pnl,
		"total_pnl_pct":   pnlPct,
		"position_count":  posCount,
		"margin_used_pct": marginPct,
		"is_running":      row.IsRunning,
	}
}

func resolvePredictionAIModel(st *store.Store, row *store.PredictionTraderDB) string {
	if m, err := st.AIModel().Get(row.UserID, row.AIModelID); err == nil && m.Provider != "" {
		return m.Provider
	}
	return "unknown"
}

func predictionInitialBalance(st *store.Store, row *store.PredictionTraderDB) float64 {
	return PredictionInitialBalanceForCharts(st, row)
}

// PredictionInitialBalanceForCharts returns the baseline equity for PnL % on charts.
func PredictionInitialBalanceForCharts(st *store.Store, row *store.PredictionTraderDB) float64 {
	if _, initial, ok, _ := st.Prediction().LoadSimAccount(row.ID); ok && initial > 0 {
		return initial
	}
	simCfg := sim.DefaultConfig()
	_ = store.ParseSimConfigJSON(row.SimConfigJSON, &simCfg)
	if simCfg.InitialBalanceUsd > 0 {
		return simCfg.InitialBalanceUsd
	}
	snaps, err := st.Prediction().ListSimSnapshots(row.ID, 1)
	if err == nil && len(snaps) > 0 {
		return snaps[0].TotalEquity - snaps[0].TotalPnl
	}
	return 0
}

// InvalidateCompetitionCache clears cached leaderboard data (e.g. after visibility toggle).
func (pm *PredictionManager) InvalidateCompetitionCache() {
	pm.competitionCache.mu.Lock()
	pm.competitionCache.data = nil
	pm.competitionCache.timestamp = time.Time{}
	pm.competitionCache.mu.Unlock()
}
