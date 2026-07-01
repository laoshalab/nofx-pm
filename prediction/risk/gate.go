package risk

import (
	"fmt"
	"nofx/prediction/types"
)

// Config is prediction-market risk limits (no leverage).
type Config struct {
	MaxOpenMarkets       int     `json:"max_open_markets"`
	MaxOrderUsd          float64 `json:"max_order_usd"`
	MaxDailyVolumeUsd    float64 `json:"max_daily_volume_usd"`
	MaxPositionMarketUsd float64 `json:"max_position_market_usd"`
	MinPrice             float64 `json:"min_price"`
	MaxPrice             float64 `json:"max_price"`
	MinConfidence        int     `json:"min_confidence"`
	MinEdgePct           float64 `json:"min_edge_pct"`
	PreviewMode          bool    `json:"preview_mode"`
}

func DefaultConfig() Config {
	return Config{
		MaxOpenMarkets:       10,
		MaxOrderUsd:          50,
		MaxDailyVolumeUsd:    500,
		MaxPositionMarketUsd: 200,
		MinPrice:             0.05,
		MaxPrice:             0.95,
		MinConfidence:        70,
		MinEdgePct:           2.0,
		PreviewMode:          true,
	}
}

// FillDefaults returns cfg with zero-valued limits replaced by DefaultConfig().
func FillDefaults(cfg Config) Config {
	def := DefaultConfig()
	if cfg.MaxOpenMarkets <= 0 {
		cfg.MaxOpenMarkets = def.MaxOpenMarkets
	}
	if cfg.MaxOrderUsd <= 0 {
		cfg.MaxOrderUsd = def.MaxOrderUsd
	}
	if cfg.MaxDailyVolumeUsd <= 0 {
		cfg.MaxDailyVolumeUsd = def.MaxDailyVolumeUsd
	}
	if cfg.MaxPositionMarketUsd <= 0 {
		cfg.MaxPositionMarketUsd = def.MaxPositionMarketUsd
	}
	if cfg.MinPrice <= 0 {
		cfg.MinPrice = def.MinPrice
	}
	if cfg.MaxPrice <= 0 {
		cfg.MaxPrice = def.MaxPrice
	}
	if cfg.MinConfidence <= 0 {
		cfg.MinConfidence = def.MinConfidence
	}
	if cfg.MinEdgePct <= 0 {
		cfg.MinEdgePct = def.MinEdgePct
	}
	return cfg
}

type Gate struct {
	cfg              Config
	dailyVolume      float64
	openMarkets      int
	openMarketSlugs  map[string]bool
	marketExposure   map[string]float64
	positionsByToken map[string]types.OutcomePosition
}

func NewGate(cfg Config) *Gate {
	return &Gate{
		openMarketSlugs:  make(map[string]bool),
		marketExposure:   make(map[string]float64),
		positionsByToken: make(map[string]types.OutcomePosition),
		cfg:              cfg,
	}
}

func (g *Gate) Config() Config { return g.cfg }

func (g *Gate) SetDailyVolume(v float64) { g.dailyVolume = v }
func (g *Gate) SetOpenMarkets(n int)     { g.openMarkets = n }

// SetPositions refreshes per-market exposure and open-market set from live holdings.
func (g *Gate) SetPositions(positions []types.OutcomePosition) {
	g.openMarketSlugs = make(map[string]bool)
	g.marketExposure = make(map[string]float64)
	g.positionsByToken = make(map[string]types.OutcomePosition)
	for _, p := range positions {
		if p.Shares <= 0 || p.MarketSlug == "" {
			continue
		}
		g.openMarketSlugs[p.MarketSlug] = true
		if p.TokenID != "" {
			g.positionsByToken[p.TokenID] = p
		}
		price := p.MidPrice
		if price <= 0 {
			price = p.AvgCost
		}
		if price <= 0 {
			continue
		}
		g.marketExposure[p.MarketSlug] += p.Shares * price
	}
	g.openMarkets = len(g.openMarketSlugs)
}

// PositionForToken returns the cached open position for sell sizing.
func (g *Gate) PositionForToken(tokenID string) (types.OutcomePosition, bool) {
	if tokenID == "" {
		return types.OutcomePosition{}, false
	}
	p, ok := g.positionsByToken[tokenID]
	return p, ok
}

func (g *Gate) CheckPrice(price float64) error {
	if price < g.cfg.MinPrice || price > g.cfg.MaxPrice {
		return fmt.Errorf("price %.4f outside [%.2f, %.2f]", price, g.cfg.MinPrice, g.cfg.MaxPrice)
	}
	return nil
}

// CheckSellPrice allows exits on low/high marks outside the buy band.
func (g *Gate) CheckSellPrice(price float64) error {
	const sellMin, sellMax = 0.001, 0.999
	if price < sellMin || price > sellMax {
		return fmt.Errorf("sell price %.4f outside [%.3f, %.3f]", price, sellMin, sellMax)
	}
	return nil
}

func (g *Gate) CheckOrderUsd(usd float64) error {
	if usd <= 0 {
		return fmt.Errorf("order usd must be positive")
	}
	if usd > g.cfg.MaxOrderUsd {
		return fmt.Errorf("order usd %.2f > max %.2f", usd, g.cfg.MaxOrderUsd)
	}
	return nil
}

func (g *Gate) CheckConfidence(confidence int) error {
	if confidence < g.cfg.MinConfidence {
		return fmt.Errorf("confidence %d < min %d", confidence, g.cfg.MinConfidence)
	}
	return nil
}

func (g *Gate) CheckEdge(edgePct float64) error {
	if edgePct < g.cfg.MinEdgePct {
		return fmt.Errorf("edge %.2f%% < min %.2f%%", edgePct, g.cfg.MinEdgePct)
	}
	return nil
}

func (g *Gate) CheckOpenMarkets() error {
	if g.openMarkets >= g.cfg.MaxOpenMarkets {
		return fmt.Errorf("open markets %d >= max %d", g.openMarkets, g.cfg.MaxOpenMarkets)
	}
	return nil
}

// CheckOpenMarketsForBuy only blocks new markets when at the open-market cap.
func (g *Gate) CheckOpenMarketsForBuy(marketSlug string) error {
	if marketSlug != "" && g.openMarketSlugs[marketSlug] {
		return nil
	}
	return g.CheckOpenMarkets()
}

func (g *Gate) CheckMarketExposure(marketSlug string, addUsd float64) error {
	if g.cfg.MaxPositionMarketUsd <= 0 || marketSlug == "" {
		return nil
	}
	current := g.marketExposure[marketSlug]
	if current+addUsd > g.cfg.MaxPositionMarketUsd {
		return fmt.Errorf("market %s exposure %.2f + %.2f > max %.2f",
			marketSlug, current, addUsd, g.cfg.MaxPositionMarketUsd)
	}
	return nil
}

func (g *Gate) CheckDailyVolume(addUsd float64) error {
	if g.dailyVolume+addUsd > g.cfg.MaxDailyVolumeUsd {
		return fmt.Errorf("daily volume %.2f + %.2f > max %.2f", g.dailyVolume, addUsd, g.cfg.MaxDailyVolumeUsd)
	}
	return nil
}

// CheckDecision runs all applicable gates for a trade action.
func (g *Gate) CheckDecision(d *types.PredictionDecision) error {
	switch d.Action {
	case types.ActionHold, types.ActionWait, types.ActionRedeem:
		return nil
	case types.ActionSell:
		if d.SizeUsd <= 0 {
			return fmt.Errorf("sell size_usd must be positive")
		}
		if d.TokenID != "" {
			p, ok := g.positionsByToken[d.TokenID]
			if !ok || p.Shares <= 0 {
				return fmt.Errorf("no open position for sell")
			}
			mark := p.MidPrice
			if mark <= 0 {
				mark = p.AvgCost
			}
			if mark > 0 {
				maxUsd := p.Shares * mark * 1.01
				if d.SizeUsd > maxUsd {
					return fmt.Errorf("sell usd %.2f exceeds position ~%.2f", d.SizeUsd, p.Shares*mark)
				}
			}
		}
		if d.LimitPrice > 0 {
			if err := g.CheckSellPrice(d.LimitPrice); err != nil {
				return err
			}
		}
		if err := g.CheckDailyVolume(d.SizeUsd); err != nil {
			return err
		}
		return nil
	case types.ActionBuyYes, types.ActionBuyNo:
		if err := g.CheckOrderUsd(d.SizeUsd); err != nil {
			return err
		}
		if err := g.CheckPrice(d.LimitPrice); err != nil {
			return err
		}
		if err := g.CheckConfidence(d.Confidence); err != nil {
			return err
		}
		if err := g.CheckEdge(d.EdgePct); err != nil {
			return err
		}
		if err := g.CheckOpenMarketsForBuy(d.MarketSlug); err != nil {
			return err
		}
		if err := g.CheckMarketExposure(d.MarketSlug, d.SizeUsd); err != nil {
			return err
		}
		if err := g.CheckDailyVolume(d.SizeUsd); err != nil {
			return err
		}
		return nil
	default:
		return fmt.Errorf("unknown action: %s", d.Action)
	}
}

func (g *Gate) RecordFill(usd float64) {
	g.dailyVolume += usd
}

// AddMarketExposure updates in-memory exposure after a confirmed fill (same cycle).
func (g *Gate) AddMarketExposure(marketSlug string, usd float64) {
	if marketSlug == "" || usd <= 0 {
		return
	}
	if g.marketExposure == nil {
		g.marketExposure = make(map[string]float64)
	}
	g.marketExposure[marketSlug] += usd
	if !g.openMarketSlugs[marketSlug] {
		g.openMarketSlugs[marketSlug] = true
		g.openMarkets = len(g.openMarketSlugs)
	}
}

// ReduceMarketExposure updates in-memory exposure after a sell fill (same cycle).
func (g *Gate) ReduceMarketExposure(marketSlug string, usd float64) {
	if marketSlug == "" || usd <= 0 {
		return
	}
	if g.marketExposure == nil {
		return
	}
	current := g.marketExposure[marketSlug]
	current -= usd
	if current <= 1e-6 {
		delete(g.marketExposure, marketSlug)
		delete(g.openMarketSlugs, marketSlug)
		g.openMarkets = len(g.openMarketSlugs)
	} else {
		g.marketExposure[marketSlug] = current
	}
}
