package api

import (
	"net/http"
	"strconv"
	"time"

	predtrader "nofx/prediction/trader"
	"nofx/prediction/telemetry"
	"nofx/store"

	"github.com/gin-gonic/gin"
)

func (s *Server) handlePredictionLive(c *gin.Context) {
	userID := c.GetString("user_id")
	id := c.Param("id")

	row, err := s.store.Prediction().GetTrader(userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trader not found"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "80"))
	sinceID, _ := strconv.ParseInt(c.Query("since_id"), 10, 64)

	runtime := predtrader.RuntimeStatus{}
	if rs, ok := s.predictionManager.RuntimeStatusFor(id); ok {
		runtime = rs
	} else {
		runtime.IsRunning = row.IsRunning
	}

	loadErr := ""
	if le := s.predictionManager.GetLoadError(id); le != nil {
		loadErr = le.Error()
	}

	c.JSON(http.StatusOK, gin.H{
		"server_time": time.Now().UTC(),
		"trader":      predictionTraderSummary(row, loadErr),
		"runtime":     runtime,
		"connectivity": telemetry.GetConnectivity(),
		"feed":        telemetry.List(id, limit, sinceID),
	})
}

func (s *Server) handlePredictionLiveFeed(c *gin.Context) {
	traderID := c.Query("trader_id")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "80"))
	sinceID, _ := strconv.ParseInt(c.Query("since_id"), 10, 64)

	c.JSON(http.StatusOK, gin.H{
		"server_time":  time.Now().UTC(),
		"connectivity": telemetry.GetConnectivity(),
		"feed":         telemetry.List(traderID, limit, sinceID),
	})
}

func predictionTraderSummary(row *store.PredictionTraderDB, loadErr string) gin.H {
	return gin.H{
		"id":                    row.ID,
		"name":                  row.Name,
		"trading_mode":          row.EffectiveTradingMode(),
		"is_running":            row.IsRunning,
		"scan_interval_minutes": row.ScanIntervalMinutes,
		"ai_model_id":           row.AIModelID,
		"venue":                 row.Venue,
		"load_error":            loadErr,
	}
}
