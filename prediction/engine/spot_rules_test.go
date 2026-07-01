package engine

import (
	"testing"

	"nofx/prediction/spot"
	"nofx/prediction/types"
)

func TestSpotAllowsBuyYesRequiresUpMove(t *testing.T) {
	m := types.Market{Question: "Bitcoin up or down", Slug: "btc-up-or-down"}
	q := spot.Quote{Symbol: "BTCUSDT", Change5mPct: 0.05}
	if spotAllowsBuyYes(m, &q, 0.1) {
		t.Fatal("expected block when 5m change below threshold")
	}
	q.Change5mPct = 0.15
	if !spotAllowsBuyYes(m, &q, 0.1) {
		t.Fatal("expected allow when 5m change above threshold")
	}
}

func TestSpotAllowsBuyNoRequiresDownMove(t *testing.T) {
	m := types.Market{Question: "Bitcoin up or down", Slug: "btc-up-or-down"}
	q := spot.Quote{Symbol: "BTCUSDT", Change5mPct: -0.05}
	if spotAllowsBuyNo(m, &q, 0.1) {
		t.Fatal("expected block when 5m change above -threshold")
	}
	q.Change5mPct = -0.2
	if !spotAllowsBuyNo(m, &q, 0.1) {
		t.Fatal("expected allow when 5m change below -threshold")
	}
}

func TestSummarizeBook(t *testing.T) {
	book := &types.OrderBook{
		Bids: []types.BookLevel{{Price: 0.44, Size: 100}, {Price: 0.43, Size: 50}},
		Asks: []types.BookLevel{{Price: 0.46, Size: 80}},
	}
	s := summarizeBook(book)
	if s.BestBid != 0.44 || s.BestAsk != 0.46 || s.BidDepth != 150 || s.AskDepth != 80 {
		t.Fatalf("summary: %+v", s)
	}
}
