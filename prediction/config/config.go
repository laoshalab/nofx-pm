package config

import "nofx/prediction/risk"

// MarketSourceType controls how candidate markets are discovered.
type MarketSourceType string

const (
	MarketSourceStatic    MarketSourceType = "static"
	MarketSourceTagSearch MarketSourceType = "tag_search"
	MarketSourceMixed     MarketSourceType = "mixed"
)

// MarketSourceConfig for market discovery.
type MarketSourceConfig struct {
	Type     MarketSourceType `json:"type"`
	Tag      string           `json:"tag"`
	Keywords []string         `json:"keywords"`
	Limit    int              `json:"limit"`
}

// EngineMode selects AI vs rule-based decision making (M4).
type EngineMode string

const (
	ModeAI     EngineMode = "ai"
	ModeRules  EngineMode = "rules"
	ModeHybrid EngineMode = "hybrid"
)

// RuleConfig for FastLoop / rules mode.
type RuleConfig struct {
	MaxBuyYesPrice  float64 `json:"max_buy_yes_price"`
	MaxBuyNoPrice   float64 `json:"max_buy_no_price"`
	MinLiquidityUsd float64 `json:"min_liquidity_usd"`
	MaxSpread       float64 `json:"max_spread"`
	DefaultSizeUsd  float64 `json:"default_size_usd"`
	DefaultEdgePct  float64 `json:"default_edge_pct"`
}

// StrategyConfig is the prediction trader strategy (M2).
type StrategyConfig struct {
	Language         string             `json:"language"`
	Mode             EngineMode         `json:"mode"`
	Rules            RuleConfig         `json:"rules"`
	FastLoopSec      int                `json:"fast_loop_sec"`
	MarketSource     MarketSourceConfig `json:"market_source"`
	StaticSlugs      []string           `json:"static_slugs"`
	MinLiquidityUsd  float64            `json:"min_liquidity_usd"`
	MinHoursToExpiry float64            `json:"min_hours_to_expiry"`
	ScanIntervalMin  int                `json:"scan_interval_min"`
	Risk             risk.Config        `json:"risk"`
}

func DefaultStrategyConfig() StrategyConfig {
	return StrategyConfig{
		Language: "en",
		Mode:     ModeAI,
		Rules: RuleConfig{
			MaxSpread:      0.05,
			DefaultSizeUsd: 10,
			DefaultEdgePct: 2.5,
		},
		MarketSource: MarketSourceConfig{
			Type:  MarketSourceTagSearch,
			Tag:   "crypto",
			Limit: 20,
		},
		MinLiquidityUsd:  500,
		MinHoursToExpiry: 0.25,
		ScanIntervalMin:  5,
		Risk:             risk.DefaultConfig(),
	}
}

// WithDefaults fills zero-valued strategy/risk fields from DefaultStrategyConfig().
func (s StrategyConfig) WithDefaults() StrategyConfig {
	def := DefaultStrategyConfig()
	if s.Language == "" {
		s.Language = def.Language
	}
	if s.Mode == "" {
		s.Mode = def.Mode
	}
	if s.Rules.MaxSpread <= 0 {
		s.Rules.MaxSpread = def.Rules.MaxSpread
	}
	if s.Rules.DefaultSizeUsd <= 0 {
		s.Rules.DefaultSizeUsd = def.Rules.DefaultSizeUsd
	}
	if s.Rules.DefaultEdgePct <= 0 {
		s.Rules.DefaultEdgePct = def.Rules.DefaultEdgePct
	}
	if s.MarketSource.Type == "" {
		s.MarketSource.Type = def.MarketSource.Type
	}
	if s.MarketSource.Tag == "" {
		s.MarketSource.Tag = def.MarketSource.Tag
	}
	if s.MarketSource.Limit <= 0 {
		s.MarketSource.Limit = def.MarketSource.Limit
	}
	if s.MinLiquidityUsd <= 0 {
		s.MinLiquidityUsd = def.MinLiquidityUsd
	}
	if s.ScanIntervalMin <= 0 {
		s.ScanIntervalMin = def.ScanIntervalMin
	}
	s.Risk = risk.FillDefaults(s.Risk)
	return s
}

// TraderConfig is runtime config for PredictionTrader.
type TraderConfig struct {
	ID           string
	Name         string
	AIModel      string
	APIKey       string
	CustomURL    string
	CustomModel  string
	ScanInterval int // minutes; 0 = use strategy
	Strategy     StrategyConfig
}
