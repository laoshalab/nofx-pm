package engine

import (
	"fmt"
	"strings"
	"time"

	"nofx/logger"
	"nofx/prediction/config"
	"nofx/prediction/types"
)

// MarketSnapshot is one tradable market with live prices.
type MarketSnapshot struct {
	Market   types.Market `json:"market"`
	YesMid   float64      `json:"yes_mid"`
	NoMid    float64      `json:"no_mid"`
	Spread   float64      `json:"spread,omitempty"`
}

// Context is passed to the AI for one prediction cycle.
type Context struct {
	CurrentTime      string                     `json:"current_time"`
	RuntimeMinutes   int                        `json:"runtime_minutes"`
	CallCount        int                        `json:"call_count"`
	AvailableUsdc    float64                    `json:"available_usdc"`
	TotalEquity      float64                    `json:"total_equity"`
	WalletAvailable  bool                       `json:"wallet_available"`
	Positions        []types.OutcomePosition    `json:"positions"`
	CandidateMarkets []MarketSnapshot           `json:"candidate_markets"`
	RecentDecisions  []types.PredictionDecision `json:"recent_decisions,omitempty"`
}

// FullDecision is the complete AI output for one cycle.
type FullDecision struct {
	Timestamp           time.Time                  `json:"timestamp"`
	SystemPrompt        string                     `json:"system_prompt"`
	UserPrompt          string                     `json:"user_prompt"`
	CoTTrace            string                     `json:"cot_trace"`
	Decisions           []types.PredictionDecision `json:"decisions"`
	Executions          []types.ExecutionOutcome   `json:"executions,omitempty"`
	RawResponse         string                     `json:"raw_response"`
	AIRequestDurationMs int64                      `json:"ai_request_duration_ms"`
}

// PredictionEngine builds prompts and discovers markets.
type PredictionEngine struct {
	cfg   config.StrategyConfig
	venue types.PredictionVenue
}

func NewPredictionEngine(cfg config.StrategyConfig, venue types.PredictionVenue) *PredictionEngine {
	return &PredictionEngine{cfg: cfg, venue: venue}
}

func (e *PredictionEngine) Config() config.StrategyConfig { return e.cfg }

func (e *PredictionEngine) GetCandidateMarkets() ([]MarketSnapshot, error) {
	src := e.cfg.MarketSource
	var markets []types.Market
	switch src.Type {
	case config.MarketSourceStatic, "":
		var fetchErrs []string
		for _, slug := range e.cfg.StaticSlugs {
			m, err := e.venue.GetMarketBySlug(slug)
			if err != nil {
				fetchErrs = append(fetchErrs, fmt.Sprintf("%s: %v", slug, err))
				logger.Warnf("[prediction] static market %s: %v", slug, err)
				continue
			}
			markets = append(markets, *m)
		}
		if len(e.cfg.StaticSlugs) > 0 && len(markets) == 0 {
			return nil, fmt.Errorf("static market fetch failed: %s", strings.Join(fetchErrs, "; "))
		}
	case config.MarketSourceTagSearch:
		filter := types.MarketFilter{
			Tag:      src.Tag,
			Keywords: src.Keywords,
			Limit:    src.Limit,
		}
		if filter.Limit <= 0 {
			filter.Limit = 20
		}
		found, err := e.venue.SearchMarkets(filter)
		if err != nil {
			return nil, err
		}
		markets = found
	case config.MarketSourceMixed:
		for _, slug := range e.cfg.StaticSlugs {
			m, err := e.venue.GetMarketBySlug(slug)
			if err != nil {
				logger.Warnf("[prediction] static market %s: %v", slug, err)
				continue
			}
			markets = append(markets, *m)
		}
		filter := types.MarketFilter{Tag: src.Tag, Keywords: src.Keywords, Limit: src.Limit}
		found, _ := e.venue.SearchMarkets(filter)
		markets = append(markets, found...)
	default:
		filter := types.MarketFilter{Tag: "crypto", Limit: 20}
		found, err := e.venue.SearchMarkets(filter)
		if err != nil {
			return nil, err
		}
		markets = found
	}

	out := make([]MarketSnapshot, 0, len(markets))
	seen := map[string]bool{}
	for _, m := range markets {
		if seen[m.Slug] || m.Closed {
			continue
		}
		if e.cfg.MinLiquidityUsd > 0 && m.Liquidity < e.cfg.MinLiquidityUsd {
			continue
		}
		if e.cfg.MinHoursToExpiry > 0 && !m.EndDate.IsZero() {
			hours := time.Until(m.EndDate).Hours()
			if hours < e.cfg.MinHoursToExpiry {
				continue
			}
		}
		snap := MarketSnapshot{Market: m}
		if m.YesTokenID != "" {
			snap.YesMid, _ = e.venue.GetMidPrice(m.YesTokenID)
		}
		if m.NoTokenID != "" {
			snap.NoMid, _ = e.venue.GetMidPrice(m.NoTokenID)
		}
		if snap.YesMid > 0 && snap.NoMid > 0 {
			snap.Spread = snap.YesMid + snap.NoMid - 1
		}
		out = append(out, snap)
		seen[m.Slug] = true
	}
	return out, nil
}
