package manager

import (
	"fmt"
	"sync"
	"time"

	"nofx/logger"
	"nofx/mcp"
	"nofx/prediction/config"
	"nofx/prediction/polymarket"
	predtrader "nofx/prediction/trader"
	"nofx/prediction/types"
	"nofx/store"
)

// PredictionManager manages PredictionTrader runtime instances.
type PredictionManager struct {
	traders    map[string]*predtrader.PredictionTrader
	loadErrors map[string]error
	competitionCache struct {
		mu        sync.RWMutex
		data      map[string]interface{}
		timestamp time.Time
	}
	mu sync.RWMutex
}

func NewPredictionManager() *PredictionManager {
	return &PredictionManager{
		traders:    make(map[string]*predtrader.PredictionTrader),
		loadErrors: make(map[string]error),
	}
}

func (pm *PredictionManager) GetTrader(id string) (*predtrader.PredictionTrader, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	t, ok := pm.traders[id]
	if !ok {
		return nil, fmt.Errorf("prediction trader '%s' not loaded", id)
	}
	return t, nil
}

func (pm *PredictionManager) GetLoadError(id string) error {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	return pm.loadErrors[id]
}

// RuntimeStatusFor returns in-memory loop stats when the trader is loaded.
func (pm *PredictionManager) RuntimeStatusFor(id string) (predtrader.RuntimeStatus, bool) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()
	t, ok := pm.traders[id]
	if !ok {
		return predtrader.RuntimeStatus{}, false
	}
	return t.RuntimeStatus(), true
}

func (pm *PredictionManager) RemoveTrader(id string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()
	if t, ok := pm.traders[id]; ok {
		t.Stop()
		delete(pm.traders, id)
	}
	delete(pm.loadErrors, id)
}

// LoadUserTradersFromStore loads prediction traders for one user.
func (pm *PredictionManager) LoadUserTradersFromStore(st *store.Store, userID string) error {
	rows, err := st.Prediction().ListTraders(userID)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if _, exists := pm.traders[row.ID]; exists {
			continue
		}
		if err := pm.addTraderFromStore(st, row); err != nil {
			pm.mu.Lock()
			pm.loadErrors[row.ID] = err
			pm.mu.Unlock()
			logger.Warnf("[prediction] failed to load trader %s: %v", row.Name, err)
		} else {
			pm.mu.Lock()
			delete(pm.loadErrors, row.ID)
			pm.mu.Unlock()
		}
	}
	return nil
}

// LoadTradersFromStore loads all prediction traders (startup).
func (pm *PredictionManager) LoadTradersFromStore(st *store.Store) error {
	rows, err := st.Prediction().ListAllTraders()
	if err != nil {
		return err
	}
	for _, row := range rows {
		if _, exists := pm.traders[row.ID]; exists {
			continue
		}
		if err := pm.addTraderFromStore(st, row); err != nil {
			pm.loadErrors[row.ID] = err
			logger.Warnf("[prediction] failed to load trader %s: %v", row.Name, err)
		}
	}
	logger.Infof("✓ Loaded %d prediction traders", len(pm.traders))
	return nil
}

func (pm *PredictionManager) addTraderFromStore(st *store.Store, row *store.PredictionTraderDB) error {
	pm.mu.Lock()
	if _, exists := pm.traders[row.ID]; exists {
		pm.mu.Unlock()
		return fmt.Errorf("trader already exists")
	}
	pm.mu.Unlock()

	strat := config.DefaultStrategyConfig()
	_ = store.ParseStrategyJSON(row.StrategyJSON, &strat)
	strat = strat.WithDefaults()
	mode := row.EffectiveTradingMode()
	strat.Risk.PreviewMode = mode != store.PredictionTradingModeLive
	if mode == store.PredictionTradingModeSimulation {
		strat.Risk.PreviewMode = false
	}
	needsAI := strat.Mode != config.ModeRules

	var client mcp.AIClient
	ptCfg := config.TraderConfig{
		ID:           row.ID,
		Name:         row.Name,
		ScanInterval: row.ScanIntervalMinutes,
		Strategy:     strat,
	}
	if needsAI {
		aiModel, err := st.AIModel().Get(row.UserID, row.AIModelID)
		if err != nil {
			return fmt.Errorf("AI model %s: %w", row.AIModelID, err)
		}
		if !aiModel.Enabled {
			return fmt.Errorf("AI model %s disabled", row.AIModelID)
		}
		ptCfg.AIModel = aiModel.Provider
		switch aiModel.Provider {
		case "qwen", "deepseek":
			ptCfg.APIKey = string(aiModel.APIKey)
		default:
			ptCfg.APIKey = string(aiModel.APIKey)
			ptCfg.CustomURL = aiModel.CustomAPIURL
			ptCfg.CustomModel = aiModel.CustomModelName
		}
		client = mcp.NewAIClientByProvider(aiModel.Provider)
	}

	if err := ValidatePredictionTraderRow(row); err != nil {
		return err
	}
	venue := BuildVenueForTrader(st, row)

	pt, err := predtrader.NewPredictionTrader(ptCfg, venue, client)
	if err != nil {
		return err
	}
	pt.SetStore(st, row.UserID, row.ID)

	pm.mu.Lock()
	pm.traders[row.ID] = pt
	pm.mu.Unlock()

	if row.IsRunning {
		go func(p *predtrader.PredictionTrader, name, id, uid string) {
			logger.Infof("[prediction] auto-starting %s", name)
			if err := p.Run(); err != nil {
				logger.Warnf("[prediction] trader %s stopped: %v", name, err)
				_ = st.Prediction().UpdateStatus(uid, id, false)
			}
		}(pt, row.Name, row.ID, row.UserID)
	}
	return nil
}

// SearchMarkets is a read-only helper for API (no auth wallet required).
func SearchMarkets(filter types.MarketFilter) ([]types.Market, error) {
	client := polymarket.NewClient(polymarket.DefaultConfig())
	return client.SearchMarkets(filter)
}

// GetMarketBySlug returns market metadata + mids via public API.
func GetMarketBySlug(slug string) (*types.Market, float64, float64, error) {
	client := polymarket.NewClient(polymarket.DefaultConfig())
	m, err := client.GetMarketBySlug(slug)
	if err != nil {
		return nil, 0, 0, err
	}
	yesMid, noMid := marketMids(client, m)
	return m, yesMid, noMid, nil
}

// MarketWithMids is a market plus optional CLOB mid prices.
type MarketWithMids struct {
	types.Market
	YesMid float64 `json:"yes_mid,omitempty"`
	NoMid  float64 `json:"no_mid,omitempty"`
}

// SearchMarketsWithMids searches Gamma and enriches each result with CLOB mids.
func SearchMarketsWithMids(filter types.MarketFilter) ([]MarketWithMids, error) {
	markets, err := SearchMarkets(filter)
	if err != nil {
		return nil, err
	}
	client := polymarket.NewClient(polymarket.DefaultConfig())
	out := make([]MarketWithMids, 0, len(markets))
	for _, m := range markets {
		yesMid, noMid := marketMids(client, &m)
		out = append(out, MarketWithMids{Market: m, YesMid: yesMid, NoMid: noMid})
	}
	return out, nil
}

func marketMids(client *polymarket.Client, m *types.Market) (yesMid, noMid float64) {
	if m.YesTokenID != "" {
		yesMid, _ = client.GetMidPrice(m.YesTokenID)
	}
	if m.NoTokenID != "" {
		noMid, _ = client.GetMidPrice(m.NoTokenID)
	}
	return yesMid, noMid
}
