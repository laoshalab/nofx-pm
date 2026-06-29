package types

// RedeemablePosition is a resolved market position awaiting redeem.
type RedeemablePosition struct {
	ConditionID string  `json:"condition_id"`
	MarketSlug  string  `json:"market_slug"`
	Outcome     string  `json:"outcome"`
	Shares      float64 `json:"shares"`
	NegRisk     bool    `json:"neg_risk"`
}

// RedeemResult is returned after a redeem attempt.
type RedeemResult struct {
	ConditionID string `json:"condition_id"`
	Status      string `json:"status"` // preview | submitted | success
	TxHash      string `json:"tx_hash,omitempty"`
	Error       string `json:"error,omitempty"`
}
