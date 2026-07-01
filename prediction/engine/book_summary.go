package engine

import "nofx/prediction/types"

const bookDepthLevels = 3

// BookSummary is a compact order-book view for prompts and rules.
type BookSummary struct {
	BestBid  float64 `json:"best_bid,omitempty"`
	BestAsk  float64 `json:"best_ask,omitempty"`
	BidDepth float64 `json:"bid_depth,omitempty"`
	AskDepth float64 `json:"ask_depth,omitempty"`
}

func summarizeBook(book *types.OrderBook) BookSummary {
	if book == nil {
		return BookSummary{}
	}
	var out BookSummary
	if len(book.Bids) > 0 {
		out.BestBid = book.Bids[0].Price
	}
	if len(book.Asks) > 0 {
		out.BestAsk = book.Asks[0].Price
	}
	for i, lv := range book.Bids {
		if i >= bookDepthLevels {
			break
		}
		out.BidDepth += lv.Size
	}
	for i, lv := range book.Asks {
		if i >= bookDepthLevels {
			break
		}
		out.AskDepth += lv.Size
	}
	return out
}
