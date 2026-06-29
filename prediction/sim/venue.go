package sim

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"nofx/prediction/types"
	"nofx/store"
)

// MarketReader provides live Polymarket data without wallet access.
type MarketReader interface {
	VenueID() string
	GetMarketBySlug(slug string) (*types.Market, error)
	SearchMarkets(filter types.MarketFilter) ([]types.Market, error)
	GetOrderBook(tokenID string) (*types.OrderBook, error)
	GetMidPrice(tokenID string) (float64, error)
}

type storedOrder struct {
	OrderID      string
	TokenID      string
	Side         string
	Price        float64
	OriginalSize float64
	SizeMatched  float64
	Status       string
	Terminal     bool
	FilledUsd    float64
}

// Venue simulates Polymarket execution against live market data.
type Venue struct {
	market MarketReader
	cfg    Config
	ledger *Ledger

	store    *store.Store
	traderID string

	mu      sync.RWMutex
	orders  map[string]*storedOrder
	seq     atomic.Int64
	persist persistFn
}

func NewVenue(market MarketReader, cfg Config, ledger *Ledger) *Venue {
	if cfg.InitialBalanceUsd <= 0 {
		cfg.InitialBalanceUsd = DefaultConfig().InitialBalanceUsd
	}
	if cfg.FillModel == "" {
		cfg.FillModel = FillModelLimitCross
	}
	if ledger == nil {
		ledger = NewLedger(cfg.InitialBalanceUsd)
	}
	return &Venue{
		market: market,
		cfg:    cfg,
		ledger: ledger,
		orders: make(map[string]*storedOrder),
	}
}

func (v *Venue) Ledger() *Ledger { return v.ledger }

func (v *Venue) VenueID() string { return "polymarket_sim" }

func (v *Venue) GetCollateral() (availableUsdc, totalEquity float64, err error) {
	mids, _ := v.fetchMids()
	availableUsdc, totalEquity = v.ledger.MarkEquity(mids)
	return availableUsdc, totalEquity, nil
}

func (v *Venue) GetOutcomePositions() ([]types.OutcomePosition, error) {
	mids, _ := v.fetchMids()
	return v.ledger.OutcomePositions(mids), nil
}

func (v *Venue) GetMarketBySlug(slug string) (*types.Market, error) {
	return v.market.GetMarketBySlug(slug)
}

func (v *Venue) SearchMarkets(filter types.MarketFilter) ([]types.Market, error) {
	return v.market.SearchMarkets(filter)
}

func (v *Venue) GetOrderBook(tokenID string) (*types.OrderBook, error) {
	return v.market.GetOrderBook(tokenID)
}

func (v *Venue) GetMidPrice(tokenID string) (float64, error) {
	return v.market.GetMidPrice(tokenID)
}

func (v *Venue) PlaceLimitOrder(req types.LimitOrderReq) (*types.OrderResult, error) {
	if req.TokenID == "" {
		return nil, fmt.Errorf("empty token_id")
	}
	if req.Price <= 0 || req.Size <= 0 {
		return nil, fmt.Errorf("invalid price/size")
	}

	book, err := v.market.GetOrderBook(req.TokenID)
	if err != nil {
		return nil, fmt.Errorf("sim: order book: %w", err)
	}

	quote, err := simulateFill(v.cfg, req.Side, req.Price, req.Size, book)
	if err != nil {
		return &types.OrderResult{
			Status: "rejected",
			Error:  err.Error(),
		}, nil
	}

	slug, outcome, conditionID, negRisk := v.resolveTokenMeta(req)
	if req.NegRisk {
		negRisk = true
	}

	side := strings.ToUpper(strings.TrimSpace(req.Side))
	var fillErr error
	if side == "BUY" {
		fillErr = v.ledger.ApplyBuy(req.TokenID, slug, outcome, conditionID, negRisk, quote.FillShares, quote.FillPrice)
	} else {
		_, fillErr = v.ledger.ApplySell(req.TokenID, quote.FillShares, quote.FillPrice)
	}
	if fillErr != nil {
		return &types.OrderResult{
			Status: "rejected",
			Error:  fillErr.Error(),
		}, nil
	}

	filledUsd := quote.FillShares * quote.FillPrice
	id := fmt.Sprintf("sim-%d", v.seq.Add(1))
	order := &storedOrder{
		OrderID:      id,
		TokenID:      req.TokenID,
		Side:         side,
		Price:        quote.FillPrice,
		OriginalSize: req.Size,
		SizeMatched:  quote.FillShares,
		Status:       "matched",
		Terminal:     true,
		FilledUsd:    filledUsd,
	}
	v.mu.Lock()
	v.orders[id] = order
	v.mu.Unlock()

	v.recordFill(id, req.TokenID, slug, outcome, side, quote.FillPrice, quote.FillShares, filledUsd, req.CycleNumber, req.DecisionAction, req.Reasoning)

	if v.persist != nil {
		v.persist()
	}

	return &types.OrderResult{
		OrderID:      id,
		FilledShares: quote.FillShares,
		FilledUsd:    filledUsd,
		Status:       "matched",
	}, nil
}

func (v *Venue) CancelOrder(orderID string) error {
	if orderID == "" {
		return fmt.Errorf("empty order_id")
	}
	v.mu.Lock()
	defer v.mu.Unlock()
	o, ok := v.orders[orderID]
	if !ok {
		return nil
	}
	if o.Terminal {
		return nil
	}
	o.Status = "cancelled"
	o.Terminal = true
	return nil
}

func (v *Venue) GetOrderStatus(orderID string) (*types.OrderStatus, error) {
	if orderID == "" {
		return nil, fmt.Errorf("empty order_id")
	}
	v.mu.RLock()
	o, ok := v.orders[orderID]
	v.mu.RUnlock()
	if !ok {
		return &types.OrderStatus{
			OrderID: orderID,
			Status:  "unknown",
		}, nil
	}
	return &types.OrderStatus{
		OrderID:      o.OrderID,
		SizeMatched:  o.SizeMatched,
		OriginalSize: o.OriginalSize,
		Status:       o.Status,
		Terminal:     o.Terminal,
	}, nil
}

func (v *Venue) ListOpenOrders() ([]types.VenueOrder, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	out := make([]types.VenueOrder, 0)
	for _, o := range v.orders {
		if o.Terminal {
			continue
		}
		out = append(out, types.VenueOrder{
			OrderID:      o.OrderID,
			TokenID:      o.TokenID,
			Side:         o.Side,
			Price:        o.Price,
			OriginalSize: o.OriginalSize,
			SizeMatched:  o.SizeMatched,
			Status:       o.Status,
		})
	}
	return out, nil
}

func (v *Venue) FindRedeemablePositions() ([]types.RedeemablePosition, error) {
	positions := v.ledger.Positions()
	out := make([]types.RedeemablePosition, 0)
	for _, p := range positions {
		if p.MarketSlug == "" || p.Shares <= 0 {
			continue
		}
		m, err := v.market.GetMarketBySlug(p.MarketSlug)
		if err != nil || m == nil || !m.Closed {
			continue
		}
		mid, err := v.market.GetMidPrice(p.TokenID)
		if err != nil || mid < 0.95 {
			continue
		}
		out = append(out, types.RedeemablePosition{
			ConditionID: p.ConditionID,
			MarketSlug:  p.MarketSlug,
			Outcome:     p.Outcome,
			Shares:      p.Shares,
			NegRisk:     p.NegRisk,
		})
	}
	return out, nil
}

func (v *Venue) RedeemCondition(conditionID string, negRisk bool) (*types.RedeemResult, error) {
	if conditionID == "" {
		return nil, fmt.Errorf("empty condition_id")
	}
	for _, p := range v.ledger.Positions() {
		if p.ConditionID != conditionID || p.Shares <= 0 {
			continue
		}
		mid, err := v.market.GetMidPrice(p.TokenID)
		if err != nil {
			return nil, err
		}
		payout := 0.0
		if mid >= 0.95 {
			payout = 1.0
		}
		shares := p.Shares
		usd, err := v.ledger.RedeemPosition(p.TokenID, payout)
		if err != nil {
			return nil, err
		}
		v.recordFill(
			fmt.Sprintf("sim-redeem-%d", time.Now().UnixMilli()),
			p.TokenID, p.MarketSlug, p.Outcome, "REDEEM", payout, shares, usd,
			0, "redeem", "",
		)
		if v.persist != nil {
			v.persist()
		}
		return &types.RedeemResult{
			ConditionID: conditionID,
			Status:      "success",
			TxHash:      fmt.Sprintf("sim-redeem-%d", time.Now().UnixMilli()),
		}, nil
	}
	return nil, fmt.Errorf("sim: no redeemable position for condition %s", conditionID)
}

func (v *Venue) resolveTokenMeta(req types.LimitOrderReq) (slug, outcome, conditionID string, negRisk bool) {
	if req.MarketSlug != "" {
		slug = req.MarketSlug
	}
	if req.Outcome != "" {
		outcome = req.Outcome
	}
	if slug != "" {
		if m, err := v.market.GetMarketBySlug(slug); err == nil && m != nil {
			conditionID = m.ConditionID
			negRisk = m.NegRisk
			if outcome == "" {
				switch req.TokenID {
				case m.YesTokenID:
					outcome = "YES"
				case m.NoTokenID:
					outcome = "NO"
				}
			}
		}
	}
	if p, ok := v.ledger.PositionForToken(req.TokenID); ok {
		if slug == "" {
			slug = p.MarketSlug
		}
		if outcome == "" {
			outcome = p.Outcome
		}
		if conditionID == "" {
			conditionID = p.ConditionID
		}
		negRisk = p.NegRisk
	}
	return slug, outcome, conditionID, negRisk
}

func (v *Venue) fetchMids() (map[string]float64, error) {
	mids := make(map[string]float64)
	for _, p := range v.ledger.Positions() {
		if p.TokenID == "" {
			continue
		}
		if mid, err := v.market.GetMidPrice(p.TokenID); err == nil && mid > 0 {
			mids[p.TokenID] = mid
		}
	}
	return mids, nil
}
