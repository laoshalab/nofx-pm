package polymarket

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"nofx/prediction/types"
	"strconv"
)

type clobBookResponse struct {
	Bids []clobLevel `json:"bids"`
	Asks []clobLevel `json:"asks"`
	Mid  string      `json:"mid"`
}

type clobLevel struct {
	Price string `json:"price"`
	Size  string `json:"size"`
}

// GetMidPrice reads CLOB public midpoint for a token.
func (c *Client) GetMidPrice(tokenID string) (float64, error) {
	if tokenID == "" {
		return 0, fmt.Errorf("empty token_id")
	}
	u := fmt.Sprintf("%s/midpoint?token_id=%s", c.cfg.ClobURL, url.QueryEscape(tokenID))
	body, err := c.getJSON(u)
	if err != nil {
		return 0, err
	}
	var resp struct {
		Mid string `json:"mid"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return 0, err
	}
	return strconv.ParseFloat(resp.Mid, 64)
}

// GetOrderBook returns full CLOB book depth for a token.
func (c *Client) GetOrderBook(tokenID string) (*types.OrderBook, error) {
	if tokenID == "" {
		return nil, fmt.Errorf("empty token_id")
	}
	u := fmt.Sprintf("%s/book?token_id=%s", c.cfg.ClobURL, url.QueryEscape(tokenID))
	body, err := c.getJSON(u)
	if err != nil {
		return nil, err
	}
	var raw clobBookResponse
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	book := &types.OrderBook{
		TokenID: tokenID,
		Bids:    parseBookLevels(raw.Bids),
		Asks:    parseBookLevels(raw.Asks),
	}
	if raw.Mid != "" {
		book.Mid, _ = strconv.ParseFloat(raw.Mid, 64)
	}
	if book.Mid == 0 && len(book.Bids) > 0 && len(book.Asks) > 0 {
		book.Mid = (book.Bids[0].Price + book.Asks[0].Price) / 2
	}
	if book.Mid == 0 {
		book.Mid, _ = c.GetMidPrice(tokenID)
	}
	return book, nil
}

func parseBookLevels(levels []clobLevel) []types.BookLevel {
	out := make([]types.BookLevel, 0, len(levels))
	for _, lv := range levels {
		price, err1 := strconv.ParseFloat(lv.Price, 64)
		size, err2 := strconv.ParseFloat(lv.Size, 64)
		if err1 != nil || err2 != nil {
			continue
		}
		out = append(out, types.BookLevel{Price: price, Size: size})
	}
	return out
}

// BookMeta holds tick size and neg_risk from CLOB /book (required for signing).
type BookMeta struct {
	TickSize string
	NegRisk  bool
}

func (c *Client) GetBookMeta(tokenID string) (*BookMeta, error) {
	u := fmt.Sprintf("%s/book?token_id=%s", c.cfg.ClobURL, url.QueryEscape(tokenID))
	body, err := c.getJSON(u)
	if err != nil {
		return nil, err
	}
	var raw struct {
		TickSize string `json:"tick_size"`
		NegRisk  bool   `json:"neg_risk"`
	}
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, err
	}
	return &BookMeta{TickSize: raw.TickSize, NegRisk: raw.NegRisk}, nil
}

func roundToTick(price float64, tick float64) float64 {
	if tick <= 0 {
		return price
	}
	return math.Round(price/tick) * tick
}
