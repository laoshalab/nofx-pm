package trader

import (
	"testing"

	"nofx/prediction/risk"
	"nofx/prediction/types"
)

type stubVenue struct {
	venueID string
}

func (s stubVenue) VenueID() string { return s.venueID }
func (s stubVenue) GetCollateral() (float64, float64, error) {
	return 0, 0, nil
}
func (s stubVenue) GetOutcomePositions() ([]types.OutcomePosition, error) { return nil, nil }
func (s stubVenue) GetMarketBySlug(string) (*types.Market, error)         { return nil, nil }
func (s stubVenue) SearchMarkets(types.MarketFilter) ([]types.Market, error) {
	return nil, nil
}
func (s stubVenue) GetOrderBook(string) (*types.OrderBook, error) { return nil, nil }
func (s stubVenue) GetMidPrice(string) (float64, error)         { return 0, nil }
func (s stubVenue) PlaceLimitOrder(types.LimitOrderReq) (*types.OrderResult, error) {
	return nil, nil
}
func (s stubVenue) CancelOrder(string) error                           { return nil }
func (s stubVenue) GetOrderStatus(string) (*types.OrderStatus, error)  { return nil, nil }
func (s stubVenue) ListOpenOrders() ([]types.VenueOrder, error)        { return nil, nil }
func (s stubVenue) FindRedeemablePositions() ([]types.RedeemablePosition, error) {
	return nil, nil
}
func (s stubVenue) RedeemCondition(string, bool) (*types.RedeemResult, error) {
	return nil, nil
}

func TestCanAutoRedeemPreviewMode(t *testing.T) {
	pt := &PredictionTrader{
		riskGate: risk.NewGate(risk.Config{PreviewMode: true}),
		venue:    stubVenue{venueID: "polymarket"},
	}
	if !pt.canAutoRedeem() {
		t.Fatal("preview mode should allow auto-redeem stub")
	}
}

func TestCanAutoRedeemSimulation(t *testing.T) {
	pt := &PredictionTrader{
		riskGate: risk.NewGate(risk.Config{PreviewMode: false}),
		venue:    stubVenue{venueID: "polymarket_sim"},
	}
	if !pt.canAutoRedeem() {
		t.Fatal("simulation venue should allow auto-redeem")
	}
}

func TestCanAutoRedeemLivePolymarketBlocked(t *testing.T) {
	pt := &PredictionTrader{
		riskGate: risk.NewGate(risk.Config{PreviewMode: false}),
		venue:    stubVenue{venueID: "polymarket"},
	}
	if pt.canAutoRedeem() {
		t.Fatal("live polymarket should not auto-redeem")
	}
}
