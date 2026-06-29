package sim

import (
	"testing"

	"nofx/prediction/types"
)

type stubMarket struct {
	books map[string]*types.OrderBook
}

func (s *stubMarket) VenueID() string { return "polymarket" }

func (s *stubMarket) GetMarketBySlug(slug string) (*types.Market, error) {
	return &types.Market{
		Venue:       "polymarket",
		Slug:        slug,
		ConditionID: "cond-" + slug,
		YesTokenID:  "yes-" + slug,
		NoTokenID:   "no-" + slug,
	}, nil
}

func (s *stubMarket) SearchMarkets(types.MarketFilter) ([]types.Market, error) {
	return nil, nil
}

func (s *stubMarket) GetOrderBook(tokenID string) (*types.OrderBook, error) {
	if b, ok := s.books[tokenID]; ok {
		return b, nil
	}
	return nil, nil
}

func (s *stubMarket) GetMidPrice(tokenID string) (float64, error) {
	if b, ok := s.books[tokenID]; ok {
		return b.Mid, nil
	}
	return 0, nil
}

func TestSimBuyCrossesAsk(t *testing.T) {
	token := "yes-btc"
	market := &stubMarket{books: map[string]*types.OrderBook{
		token: {
			TokenID: token,
			Asks:    []types.BookLevel{{Price: 0.55, Size: 100}},
			Bids:    []types.BookLevel{{Price: 0.53, Size: 100}},
			Mid:     0.54,
		},
	}}
	v := NewVenue(market, DefaultConfig(), NewLedger(1000))

	res, err := v.PlaceLimitOrder(types.LimitOrderReq{
		TokenID:    token,
		Side:       "BUY",
		Price:      0.58,
		Size:       10,
		MarketSlug: "btc",
		Outcome:    "YES",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "matched" || res.FilledShares != 10 {
		t.Fatalf("unexpected result: %+v", res)
	}

	avail, equity := v.ledger.Cash(), 0.0
	_, equity = v.ledger.MarkEquity(map[string]float64{token: 0.54})
	if avail != 1000-5.5 {
		t.Fatalf("cash want 994.5 got %.2f", avail)
	}
	if equity < 999 {
		t.Fatalf("equity too low: %.2f", equity)
	}

	st, err := v.GetOrderStatus(res.OrderID)
	if err != nil || st.SizeMatched != 10 {
		t.Fatalf("status: %+v err=%v", st, err)
	}
}

func TestSimBuyInsufficientCash(t *testing.T) {
	token := "yes-btc"
	market := &stubMarket{books: map[string]*types.OrderBook{
		token: {TokenID: token, Asks: []types.BookLevel{{Price: 0.55, Size: 100}}, Mid: 0.55},
	}}
	v := NewVenue(market, DefaultConfig(), NewLedger(1))
	res, err := v.PlaceLimitOrder(types.LimitOrderReq{
		TokenID: token, Side: "BUY", Price: 0.58, Size: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "rejected" {
		t.Fatalf("expected rejected, got %+v", res)
	}
}

func TestSimSellRequiresShares(t *testing.T) {
	token := "yes-btc"
	market := &stubMarket{books: map[string]*types.OrderBook{
		token: {TokenID: token, Bids: []types.BookLevel{{Price: 0.55, Size: 100}}, Mid: 0.55},
	}}
	v := NewVenue(market, DefaultConfig(), NewLedger(1000))
	res, err := v.PlaceLimitOrder(types.LimitOrderReq{
		TokenID: token, Side: "SELL", Price: 0.50, Size: 5,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "rejected" {
		t.Fatalf("expected rejected sell without shares, got %+v", res)
	}
}

func TestSimBuyThenSell(t *testing.T) {
	token := "yes-btc"
	market := &stubMarket{books: map[string]*types.OrderBook{
		token: {
			TokenID: token,
			Asks:    []types.BookLevel{{Price: 0.40, Size: 100}},
			Bids:    []types.BookLevel{{Price: 0.38, Size: 100}},
			Mid:     0.39,
		},
	}}
	v := NewVenue(market, DefaultConfig(), NewLedger(100))

	if _, err := v.PlaceLimitOrder(types.LimitOrderReq{
		TokenID: token, Side: "BUY", Price: 0.45, Size: 10, MarketSlug: "btc", Outcome: "YES",
	}); err != nil {
		t.Fatal(err)
	}
	res, err := v.PlaceLimitOrder(types.LimitOrderReq{
		TokenID: token, Side: "SELL", Price: 0.38, Size: 10,
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Status != "matched" || res.FilledShares != 10 {
		t.Fatalf("sell result: %+v", res)
	}
	if v.ledger.Cash() != 99.8 { // 100 - 4 + 3.8
		t.Fatalf("cash want 99.8 got %.2f", v.ledger.Cash())
	}
}

func TestSimGetCollateral(t *testing.T) {
	token := "yes-btc"
	market := &stubMarket{books: map[string]*types.OrderBook{
		token: {TokenID: token, Asks: []types.BookLevel{{Price: 0.50, Size: 20}}, Mid: 0.50},
	}}
	v := NewVenue(market, DefaultConfig(), NewLedger(100))
	_, _ = v.PlaceLimitOrder(types.LimitOrderReq{
		TokenID: token, Side: "BUY", Price: 0.55, Size: 10, MarketSlug: "btc", Outcome: "YES",
	})
	avail, equity, err := v.GetCollateral()
	if err != nil {
		t.Fatal(err)
	}
	if avail != 95 {
		t.Fatalf("avail want 95 got %.2f", avail)
	}
	if equity != 100 {
		t.Fatalf("equity want 100 got %.2f", equity)
	}
}
