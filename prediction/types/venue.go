package types

// PredictionVenue is the shared interface for prediction market execution.
// Crypto exchanges use trader/types.Trader instead — do not merge the two.
type PredictionVenue interface {
	VenueID() string

	GetCollateral() (availableUsdc, totalEquity float64, err error)
	GetOutcomePositions() ([]OutcomePosition, error)

	GetMarketBySlug(slug string) (*Market, error)
	SearchMarkets(filter MarketFilter) ([]Market, error)
	GetOrderBook(tokenID string) (*OrderBook, error)
	GetMidPrice(tokenID string) (float64, error)

	PlaceLimitOrder(req LimitOrderReq) (*OrderResult, error)
	CancelOrder(orderID string) error
	GetOrderStatus(orderID string) (*OrderStatus, error)

	ListOpenOrders() ([]VenueOrder, error)

	FindRedeemablePositions() ([]RedeemablePosition, error)
	RedeemCondition(conditionID string, negRisk bool) (*RedeemResult, error)
}
