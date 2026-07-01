package types

import "time"

// Market is a normalized Polymarket (or other venue) tradable market.
type Market struct {
	Venue         string    `json:"venue"`
	ID            string    `json:"id"`
	Slug          string    `json:"slug"`
	Question      string    `json:"question"`
	ConditionID   string    `json:"condition_id,omitempty"`
	EndDate       time.Time `json:"end_date,omitempty"`
	Closed        bool      `json:"closed"`
	NegRisk       bool      `json:"neg_risk"`
	YesTokenID    string    `json:"yes_token_id"`
	NoTokenID     string    `json:"no_token_id"`
	Volume24h     float64   `json:"volume_24h,omitempty"`
	Liquidity     float64   `json:"liquidity,omitempty"`
	OutcomePrices []float64 `json:"outcome_prices,omitempty"` // [YES, NO] when resolved
}

// WinningOutcome returns YES, NO, or "" when resolution is unknown.
func (m *Market) WinningOutcome() string {
	if len(m.OutcomePrices) >= 2 {
		if m.OutcomePrices[0] >= 0.99 {
			return "YES"
		}
		if m.OutcomePrices[1] >= 0.99 {
			return "NO"
		}
	}
	return ""
}

// PayoutPerShare returns $1 for winning outcome shares, else $0 when resolved.
func (m *Market) PayoutPerShare(outcome string) (float64, bool) {
	winner := m.WinningOutcome()
	if winner == "" {
		return 0, false
	}
	if winner == outcome {
		return 1, true
	}
	return 0, true
}

// OutcomePosition is a held YES/NO token position.
type OutcomePosition struct {
	TokenID    string  `json:"token_id"`
	MarketSlug string  `json:"market_slug"`
	Outcome    string  `json:"outcome"` // "YES" | "NO"
	Shares     float64 `json:"shares"`
	AvgCost    float64 `json:"avg_cost"`
	MidPrice   float64 `json:"mid_price,omitempty"`
}

// OrderBook snapshot for one outcome token.
type OrderBook struct {
	TokenID string      `json:"token_id"`
	Bids    []BookLevel `json:"bids"`
	Asks    []BookLevel `json:"asks"`
	Mid     float64     `json:"mid"`
}

type BookLevel struct {
	Price float64 `json:"price"`
	Size  float64 `json:"size"`
}

// MarketFilter for discovery queries.
type MarketFilter struct {
	Tag      string
	Closed   *bool
	Limit    int
	Keywords []string
}
