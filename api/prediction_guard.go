package api

import (
	"net/http"
	"strings"

	"nofx/config"
	"nofx/prediction/polymarket"
	"nofx/store"

	"github.com/gin-gonic/gin"
)

func predictionSecurity() (liveEnabled, allowBrowserPrivateKey bool) {
	cfg := config.Get()
	if cfg == nil {
		return false, false
	}
	return cfg.PredictionLiveEnabled, cfg.PredictionAllowBrowserPrivateKey
}

func predictionLiveRedeemEnabled() bool {
	liveEnabled, _ := predictionSecurity()
	if !liveEnabled {
		return false
	}
	return polymarket.LiveRedeemConfigured(polymarket.DefaultConfig())
}

func (s *Server) ownedPredictionTrader(c *gin.Context, id string) (*store.PredictionTraderDB, bool) {
	userID := c.GetString("user_id")
	row, err := s.store.Prediction().GetTrader(userID, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "trader not found"})
		return nil, false
	}
	return row, true
}

func denyLiveTradingDisabled(c *gin.Context, row *store.PredictionTraderDB) bool {
	if row == nil || row.EffectiveTradingMode() != store.PredictionTradingModeLive {
		return false
	}
	liveEnabled, _ := predictionSecurity()
	if liveEnabled {
		return false
	}
	c.JSON(http.StatusForbidden, gin.H{
		"error": "live prediction trading is disabled on this server (set PREDICTION_LIVE_ENABLED=true to enable)",
	})
	return true
}

func denyLiveModeRequest(c *gin.Context, tradingMode string) bool {
	if strings.TrimSpace(tradingMode) != store.PredictionTradingModeLive {
		return false
	}
	liveEnabled, _ := predictionSecurity()
	if liveEnabled {
		return false
	}
	c.JSON(http.StatusForbidden, gin.H{
		"error": "live prediction trading is disabled on this server (set PREDICTION_LIVE_ENABLED=true to enable)",
	})
	return true
}

func rejectLiveBrowserPrivateKey(c *gin.Context, tradingMode, privateKey string) bool {
	if strings.TrimSpace(privateKey) == "" {
		return false
	}
	if strings.TrimSpace(tradingMode) != store.PredictionTradingModeLive {
		return false
	}
	_, allowBrowser := predictionSecurity()
	if allowBrowser {
		return false
	}
	c.JSON(http.StatusForbidden, gin.H{
		"error": "uploading private keys for live trading via API is disabled; configure keys server-side or set PREDICTION_ALLOW_BROWSER_PRIVATE_KEY=true for development only",
	})
	return true
}

func rejectLiveWithoutPrivateKey(c *gin.Context, row *store.PredictionTraderDB, newPrivateKey string) bool {
	if row == nil || row.EffectiveTradingMode() != store.PredictionTradingModeLive {
		return false
	}
	hasKey := row.PrivateKey.String() != "" ||
		strings.TrimSpace(newPrivateKey) != "" ||
		polymarket.EnvPrivateKey() != ""
	if hasKey {
		return false
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "private key required for live trading"})
	return true
}

func rejectPreviewWithoutPrivateKey(c *gin.Context, tradingMode string, row *store.PredictionTraderDB, newPrivateKey string) bool {
	mode := strings.TrimSpace(tradingMode)
	if mode == "" && row != nil {
		mode = row.EffectiveTradingMode()
	}
	if mode != store.PredictionTradingModePreview {
		return false
	}
	hasKey := (row != nil && row.PrivateKey.String() != "") || strings.TrimSpace(newPrivateKey) != ""
	if hasKey {
		return false
	}
	c.JSON(http.StatusBadRequest, gin.H{"error": "private_key required for preview signing"})
	return true
}

// predictionWalletConfigured reports whether preview/live can query Polymarket wallet APIs.
func predictionWalletConfigured(row *store.PredictionTraderDB) bool {
	if row == nil {
		return false
	}
	if row.EffectiveTradingMode() == store.PredictionTradingModeSimulation {
		return true
	}
	return row.PrivateKey.String() != "" || strings.TrimSpace(row.ProxyAddress) != ""
}
