package sim

import (
	"fmt"
	"strings"

	"nofx/prediction/types"
)

type fillQuote struct {
	FillPrice  float64
	FillShares float64
}

func simulateFill(cfg Config, side string, limitPrice, reqShares float64, book *types.OrderBook) (*fillQuote, error) {
	if limitPrice <= 0 || reqShares <= 0 {
		return nil, fmt.Errorf("invalid order price/size")
	}
	side = strings.ToUpper(strings.TrimSpace(side))
	model := cfg.FillModel
	if model == "" {
		model = FillModelLimitCross
	}

	switch model {
	case FillModelMidInstant:
		return fillAtMid(cfg, side, limitPrice, reqShares, book)
	default:
		return fillLimitCross(cfg, side, limitPrice, reqShares, book)
	}
}

func fillLimitCross(cfg Config, side string, limitPrice, reqShares float64, book *types.OrderBook) (*fillQuote, error) {
	if book == nil {
		return nil, fmt.Errorf("order book unavailable")
	}
	bps := cfg.SlippageBps
	if side == "BUY" {
		if len(book.Asks) > 0 && book.Asks[0].Price <= limitPrice+1e-9 {
			price := book.Asks[0].Price
			shares := reqShares
			if book.Asks[0].Size > 0 && book.Asks[0].Size < shares {
				shares = book.Asks[0].Size
			}
			return &fillQuote{FillPrice: applySlippage(price, 1, bps), FillShares: shares}, nil
		}
		if book.Mid > 0 && limitPrice >= book.Mid-1e-9 {
			return &fillQuote{FillPrice: applySlippage(book.Mid, 1, bps), FillShares: reqShares}, nil
		}
		return nil, fmt.Errorf("buy limit %.4f does not cross book", limitPrice)
	}

	if len(book.Bids) > 0 && book.Bids[0].Price >= limitPrice-1e-9 {
		price := book.Bids[0].Price
		shares := reqShares
		if book.Bids[0].Size > 0 && book.Bids[0].Size < shares {
			shares = book.Bids[0].Size
		}
		return &fillQuote{FillPrice: applySlippage(price, -1, bps), FillShares: shares}, nil
	}
	if book.Mid > 0 && limitPrice <= book.Mid+1e-9 {
		return &fillQuote{FillPrice: applySlippage(book.Mid, -1, bps), FillShares: reqShares}, nil
	}
	return nil, fmt.Errorf("sell limit %.4f does not cross book", limitPrice)
}

func fillAtMid(cfg Config, side string, limitPrice, reqShares float64, book *types.OrderBook) (*fillQuote, error) {
	if book == nil || book.Mid <= 0 {
		return nil, fmt.Errorf("mid price unavailable")
	}
	if side == "BUY" && limitPrice+1e-9 < book.Mid {
		return nil, fmt.Errorf("buy limit %.4f below mid %.4f", limitPrice, book.Mid)
	}
	if side == "SELL" && limitPrice > book.Mid+1e-9 {
		return nil, fmt.Errorf("sell limit %.4f above mid %.4f", limitPrice, book.Mid)
	}
	dir := 1
	if side == "SELL" {
		dir = -1
	}
	return &fillQuote{FillPrice: applySlippage(book.Mid, dir, cfg.SlippageBps), FillShares: reqShares}, nil
}

func applySlippage(price float64, direction int, bps int) float64 {
	if bps <= 0 || price <= 0 {
		return price
	}
	adj := float64(bps) / 10000.0
	if direction > 0 {
		return price * (1 + adj)
	}
	return price * (1 - adj)
}
