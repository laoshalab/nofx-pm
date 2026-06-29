package types

// PredictionAction is a venue-neutral trade intent (not crypto long/short).
type PredictionAction string

const (
	ActionBuyYes  PredictionAction = "buy_yes"
	ActionBuyNo   PredictionAction = "buy_no"
	ActionSell    PredictionAction = "sell"
	ActionHold    PredictionAction = "hold"
	ActionWait    PredictionAction = "wait"
	ActionRedeem  PredictionAction = "redeem"
)

// PredictionDecision is one AI or rule engine output for a single market.
type PredictionDecision struct {
	MarketSlug string           `json:"market_slug"`
	TokenID    string           `json:"token_id"`
	Action     PredictionAction `json:"action"`
	SizeUsd    float64          `json:"size_usd"`
	LimitPrice float64          `json:"limit_price"`
	Confidence int              `json:"confidence"`
	EdgePct    float64          `json:"edge_pct"`
	Reasoning  string           `json:"reasoning"`
}

// ExecutionOutcome is the result of executing one decision in a cycle.
type ExecutionOutcome struct {
	MarketSlug string           `json:"market_slug"`
	Action     PredictionAction `json:"action"`
	Status     string           `json:"status"` // skipped | preview | filled | posted | rejected | risk_blocked | redeem | error
	Message    string           `json:"message,omitempty"`
	OrderID    string           `json:"order_id,omitempty"`
	FillUsd    float64          `json:"fill_usd,omitempty"`
	PreviewWire string          `json:"preview_wire,omitempty"` // signed CLOB payload (preview mode)
}

// LimitOrderReq is a signed CLOB limit order request.
type LimitOrderReq struct {
	TokenID    string  `json:"token_id"`
	Side       string  `json:"side"` // BUY | SELL
	Price      float64 `json:"price"`
	Size       float64 `json:"size"` // shares
	NegRisk    bool    `json:"neg_risk"`
	MarketSlug string  `json:"market_slug,omitempty"`
	Outcome    string  `json:"outcome,omitempty"` // YES | NO
	// Metadata for sim fill audit (ignored by live CLOB wire).
	CycleNumber    int    `json:"cycle_number,omitempty"`
	DecisionAction string `json:"decision_action,omitempty"`
	Reasoning      string `json:"reasoning,omitempty"`
}

// OrderResult is returned after order submission.
type OrderResult struct {
	OrderID      string  `json:"order_id"`
	FilledShares float64 `json:"filled_shares"`
	FilledUsd    float64 `json:"filled_usd"`
	Status       string  `json:"status"`
	Error        string  `json:"error,omitempty"`
	PreviewWire  string  `json:"preview_wire,omitempty"`
}

// VenueOrder is an open or historical CLOB order.
type VenueOrder struct {
	OrderID      string  `json:"order_id"`
	TokenID      string  `json:"token_id"`
	MarketID     string  `json:"market_id,omitempty"`
	Side         string  `json:"side"`
	Price        float64 `json:"price"`
	OriginalSize float64 `json:"original_size"`
	SizeMatched  float64 `json:"size_matched"`
	Status       string  `json:"status"`
	CreatedAt    string  `json:"created_at,omitempty"`
}

// OrderStatus from CLOB polling.
type OrderStatus struct {
	OrderID      string  `json:"order_id"`
	SizeMatched  float64 `json:"size_matched"`
	OriginalSize float64 `json:"original_size"`
	Status       string  `json:"status"`
	Terminal     bool    `json:"terminal"`
}
