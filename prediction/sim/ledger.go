package sim

import (
	"fmt"
	"sync"

	"nofx/prediction/types"
)

type positionMeta struct {
	TokenID     string
	MarketSlug  string
	Outcome     string // YES | NO
	ConditionID string
	NegRisk     bool
	Shares      float64
	AvgCost     float64
}

// Ledger tracks virtual USDC and outcome token holdings.
type Ledger struct {
	mu      sync.RWMutex
	cash    float64
	initial float64
	byToken map[string]*positionMeta
}

func (l *Ledger) setInitial(initial float64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	if initial > 0 {
		l.initial = initial
	}
}

func NewLedger(initialCash float64) *Ledger {
	if initialCash <= 0 {
		initialCash = DefaultConfig().InitialBalanceUsd
	}
	return &Ledger{
		cash:    initialCash,
		initial: initialCash,
		byToken: make(map[string]*positionMeta),
	}
}

func (l *Ledger) Cash() float64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.cash
}

func (l *Ledger) InitialBalance() float64 {
	l.mu.RLock()
	defer l.mu.RUnlock()
	return l.initial
}

func (l *Ledger) SetCash(cash float64) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.cash = cash
}

// RestorePosition loads a saved holding without changing cash.
func (l *Ledger) RestorePosition(p positionMeta) {
	if p.TokenID == "" || p.Shares <= 0 {
		return
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	cp := p
	l.byToken[p.TokenID] = &cp
}

func (l *Ledger) Positions() []positionMeta {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]positionMeta, 0, len(l.byToken))
	for _, p := range l.byToken {
		if p.Shares > 1e-9 {
			out = append(out, *p)
		}
	}
	return out
}

func (l *Ledger) PositionForToken(tokenID string) (positionMeta, bool) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	p, ok := l.byToken[tokenID]
	if !ok || p.Shares <= 1e-9 {
		return positionMeta{}, false
	}
	return *p, true
}

func (l *Ledger) ApplyBuy(tokenID, slug, outcome, conditionID string, negRisk bool, shares, price float64) error {
	if shares <= 0 || price <= 0 {
		return fmt.Errorf("invalid buy: shares=%.4f price=%.4f", shares, price)
	}
	cost := shares * price
	l.mu.Lock()
	defer l.mu.Unlock()
	if cost > l.cash+1e-9 {
		return fmt.Errorf("insufficient cash: need %.2f have %.2f", cost, l.cash)
	}
	l.cash -= cost
	p := l.byToken[tokenID]
	if p == nil {
		p = &positionMeta{
			TokenID:     tokenID,
			MarketSlug:  slug,
			Outcome:     outcome,
			ConditionID: conditionID,
			NegRisk:     negRisk,
		}
		l.byToken[tokenID] = p
	} else {
		if slug != "" {
			p.MarketSlug = slug
		}
		if outcome != "" {
			p.Outcome = outcome
		}
		if conditionID != "" {
			p.ConditionID = conditionID
		}
		p.NegRisk = negRisk
	}
	totalCost := p.AvgCost*p.Shares + cost
	p.Shares += shares
	if p.Shares > 0 {
		p.AvgCost = totalCost / p.Shares
	}
	return nil
}

func (l *Ledger) ApplySell(tokenID string, shares, price float64) (float64, error) {
	if shares <= 0 || price <= 0 {
		return 0, fmt.Errorf("invalid sell: shares=%.4f price=%.4f", shares, price)
	}
	l.mu.Lock()
	defer l.mu.Unlock()
	p := l.byToken[tokenID]
	if p == nil || p.Shares+1e-9 < shares {
		held := 0.0
		if p != nil {
			held = p.Shares
		}
		return 0, fmt.Errorf("insufficient shares: need %.4f have %.4f", shares, held)
	}
	proceeds := shares * price
	p.Shares -= shares
	if p.Shares <= 1e-9 {
		delete(l.byToken, tokenID)
	}
	l.cash += proceeds
	return proceeds, nil
}

func (l *Ledger) RedeemPosition(tokenID string, payoutPerShare float64) (float64, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	p := l.byToken[tokenID]
	if p == nil || p.Shares <= 1e-9 {
		return 0, fmt.Errorf("no position for token %s", tokenID)
	}
	payout := p.Shares * payoutPerShare
	l.cash += payout
	delete(l.byToken, tokenID)
	return payout, nil
}

func (l *Ledger) OutcomePositions(mids map[string]float64) []types.OutcomePosition {
	l.mu.RLock()
	defer l.mu.RUnlock()
	out := make([]types.OutcomePosition, 0, len(l.byToken))
	for _, p := range l.byToken {
		if p.Shares <= 1e-9 {
			continue
		}
		op := types.OutcomePosition{
			TokenID:    p.TokenID,
			MarketSlug: p.MarketSlug,
			Outcome:    p.Outcome,
			Shares:     p.Shares,
			AvgCost:    p.AvgCost,
		}
		if mid, ok := mids[p.TokenID]; ok {
			op.MidPrice = mid
		}
		out = append(out, op)
	}
	return out
}

func (l *Ledger) MarkEquity(mids map[string]float64) (available, equity float64) {
	l.mu.RLock()
	defer l.mu.RUnlock()
	available = l.cash
	equity = l.cash
	for _, p := range l.byToken {
		if p.Shares <= 1e-9 {
			continue
		}
		price := p.AvgCost
		if mid, ok := mids[p.TokenID]; ok && mid > 0 {
			price = mid
		}
		equity += p.Shares * price
	}
	return available, equity
}
