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
