package api

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"nofx/logger"
	"nofx/manager"
	"nofx/store"

	"github.com/gin-gonic/gin"
)

// handlePublicPredictionCompetition returns the public prediction-market leaderboard.
func (s *Server) handlePublicPredictionCompetition(c *gin.Context) {
	competition, err := s.predictionManager.GetCompetitionData(s.store)
	if err != nil {
		SafeInternalError(c, "Get prediction competition data", err)
		return
	}
	c.JSON(http.StatusOK, competition)
}

// handlePredictionEquityHistoryBatch returns sim equity curves for prediction traders.
func (s *Server) handlePredictionEquityHistoryBatch(c *gin.Context) {
	var requestBody struct {
		TraderIDs []string `json:"trader_ids"`
		Hours     int      `json:"hours"`
	}

	if err := c.ShouldBindJSON(&requestBody); err != nil {
		traderIDsParam := c.Query("trader_ids")
		if traderIDsParam == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "trader_ids required"})
			return
		}
		requestBody.TraderIDs = strings.Split(traderIDsParam, ",")
		for i := range requestBody.TraderIDs {
			requestBody.TraderIDs[i] = strings.TrimSpace(requestBody.TraderIDs[i])
		}
		if hoursParam := c.Query("hours"); hoursParam != "" {
			fmt.Sscanf(hoursParam, "%d", &requestBody.Hours)
		}
	}

	if len(requestBody.TraderIDs) > 20 {
		requestBody.TraderIDs = requestBody.TraderIDs[:20]
	}

	result := s.getPredictionEquityHistoryForTraders(requestBody.TraderIDs, requestBody.Hours)
	c.JSON(http.StatusOK, result)
}

func (s *Server) getPredictionEquityHistoryForTraders(traderIDs []string, hours int) map[string]interface{} {
	result := make(map[string]interface{})
	histories := make(map[string]interface{})
	errors := make(map[string]string)
	now := time.Now()

	for _, traderID := range traderIDs {
		if traderID == "" {
			continue
		}
		row, err := s.store.Prediction().GetTraderByID(traderID)
		if err != nil || row == nil || !row.ShowInCompetition {
			errors[traderID] = "Trader not found"
			continue
		}

		var snapshots []store.PredictionSimSnapshotDB
		if hours > 0 {
			start := now.Add(-time.Duration(hours) * time.Hour)
			snapshots, err = s.store.Prediction().ListSimSnapshotsByTimeRange(traderID, start, now, 500)
		} else {
			snapshots, err = s.store.Prediction().ListSimSnapshots(traderID, 500)
		}
		if err != nil {
			logger.Errorf("[API] prediction equity history %s: %v", traderID, err)
			errors[traderID] = "Failed to get historical data"
			continue
		}

		initial := manager.PredictionInitialBalanceForCharts(s.store, row)
		history := make([]map[string]interface{}, 0, len(snapshots)+1)
		var lastSnapshotTime time.Time
		for _, snap := range snapshots {
			pnlPct := snap.TotalPnlPct
			if initial > 0 && pnlPct == 0 && snap.TotalEquity != initial {
				pnlPct = (snap.TotalEquity - initial) / initial * 100
			}
			history = append(history, map[string]interface{}{
				"timestamp":     snap.Timestamp,
				"total_equity":  snap.TotalEquity,
				"total_pnl":     snap.TotalPnl,
				"total_pnl_pct": pnlPct,
				"balance":       snap.CashUsdc,
			})
			if snap.Timestamp.After(lastSnapshotTime) {
				lastSnapshotTime = snap.Timestamp
			}
		}

		if row.EffectiveTradingMode() == store.PredictionTradingModeSimulation || len(history) == 0 {
			venue := manager.BuildVenueForTrader(s.store, row)
			avail, equity, _ := venue.GetCollateral()
			if now.Sub(lastSnapshotTime) > 30*time.Second || len(history) == 0 {
				pnl := equity - initial
				pnlPct := 0.0
				if initial > 0 {
					pnlPct = (pnl / initial) * 100
				}
				history = append(history, map[string]interface{}{
					"timestamp":     now,
					"total_equity":  equity,
					"total_pnl":     pnl,
					"total_pnl_pct": pnlPct,
					"balance":       avail,
				})
			}
		}

		histories[traderID] = history
	}

	result["histories"] = histories
	result["count"] = len(histories)
	if len(errors) > 0 {
		result["errors"] = errors
	}
	return result
}

func (s *Server) handlePredictionShowInCompetition(c *gin.Context) {
	userID := c.GetString("user_id")
	traderID := c.Param("id")

	var req struct {
		ShowInCompetition bool `json:"show_in_competition"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if _, err := s.store.Prediction().GetTrader(userID, traderID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}
	if err := s.store.Prediction().UpdateShowInCompetition(userID, traderID, req.ShowInCompetition); err != nil {
		SafeInternalError(c, "Update show_in_competition", err)
		return
	}
	s.predictionManager.InvalidateCompetitionCache()
	c.JSON(http.StatusOK, gin.H{
		"trader_id":             traderID,
		"show_in_competition":   req.ShowInCompetition,
	})
}
