package sim

import (
	"time"

	"nofx/store"
)

// BindStore enables SQLite persistence for simulation state, fills, and snapshots.
func (v *Venue) BindStore(st *store.Store, traderID string) {
	if st == nil || traderID == "" {
		return
	}
	v.store = st
	v.traderID = traderID
	v.persist = func() {
		_ = saveLedger(st, traderID, v.ledger)
	}
}

type persistFn func()

func (v *Venue) recordFill(orderID, tokenID, slug, outcome, side string, price, shares, usd float64, cycleNumber int, decisionAction, reasoning string) {
	if v.store == nil || v.traderID == "" {
		return
	}
	_ = v.store.Prediction().SaveSimFill(&store.PredictionSimFillDB{
		TraderID:       v.traderID,
		OrderID:        orderID,
		TokenID:        tokenID,
		MarketSlug:     slug,
		Outcome:        outcome,
		Side:           side,
		FillPrice:      price,
		FillShares:     shares,
		FillUsd:        usd,
		CycleNumber:    cycleNumber,
		DecisionAction: decisionAction,
		Reasoning:      reasoning,
		Timestamp:      time.Now().UTC(),
	})
}

// RecordEquitySnapshot saves a point-in-time PnL snapshot (called after each cycle).
func (v *Venue) RecordEquitySnapshot(cycleNumber int) error {
	if v.store == nil || v.traderID == "" {
		return nil
	}
	cash, equity, err := v.GetCollateral()
	if err != nil {
		return err
	}
	posValue := equity - cash
	if posValue < 0 {
		posValue = 0
	}
	initial := v.ledger.InitialBalance()
	pnl := equity - initial
	pnlPct := 0.0
	if initial > 0 {
		pnlPct = (pnl / initial) * 100
	}
	return v.store.Prediction().SaveSimSnapshot(&store.PredictionSimSnapshotDB{
		TraderID:      v.traderID,
		CycleNumber:   cycleNumber,
		CashUsdc:      cash,
		PositionValue: posValue,
		TotalEquity:   equity,
		TotalPnl:      pnl,
		TotalPnlPct:   pnlPct,
		OpenPositions: len(v.ledger.Positions()),
		Timestamp:     time.Now().UTC(),
	})
}

func saveLedger(st *store.Store, traderID string, ledger *Ledger) error {
	positions := ledger.Positions()
	rows := make([]store.PredictionSimPositionDB, 0, len(positions))
	for _, p := range positions {
		rows = append(rows, store.PredictionSimPositionDB{
			TokenID:     p.TokenID,
			MarketSlug:  p.MarketSlug,
			Outcome:     p.Outcome,
			ConditionID: p.ConditionID,
			NegRisk:     p.NegRisk,
			Shares:      p.Shares,
			AvgCost:     p.AvgCost,
		})
	}
	return st.Prediction().SaveSimState(traderID, ledger.Cash(), ledger.InitialBalance(), rows)
}

// LoadLedgerFromStore hydrates a ledger from SQLite (no-op if account missing).
func LoadLedgerFromStore(st *store.Store, traderID string, ledger *Ledger) error {
	if st == nil || traderID == "" {
		return nil
	}
	cash, initial, ok, err := st.Prediction().LoadSimAccount(traderID)
	if err != nil {
		return err
	}
	if !ok {
		return nil
	}
	ledger.SetCash(cash)
	if initial > 0 {
		ledger.setInitial(initial)
	}
	rows, err := st.Prediction().ListSimPositions(traderID)
	if err != nil {
		return err
	}
	for _, row := range rows {
		if row.Shares <= 0 {
			continue
		}
		ledger.RestorePosition(positionMeta{
			TokenID:     row.TokenID,
			MarketSlug:  row.MarketSlug,
			Outcome:     row.Outcome,
			ConditionID: row.ConditionID,
			NegRisk:     row.NegRisk,
			Shares:      row.Shares,
			AvgCost:     row.AvgCost,
		})
	}
	return nil
}

// InitLedgerInStore creates initial sim account row if absent.
func InitLedgerInStore(st *store.Store, traderID string, cfg Config) error {
	if st == nil || traderID == "" {
		return nil
	}
	_, _, ok, err := st.Prediction().LoadSimAccount(traderID)
	if err != nil {
		return err
	}
	if ok {
		return nil
	}
	initial := cfg.InitialBalanceUsd
	if initial <= 0 {
		initial = DefaultConfig().InitialBalanceUsd
	}
	if err := st.Prediction().SaveSimState(traderID, initial, initial, nil); err != nil {
		return err
	}
	ledger := NewLedger(initial)
	v := &Venue{ledger: ledger, store: st, traderID: traderID}
	return v.RecordEquitySnapshot(0)
}

// ResetSimAccount wipes simulation data and re-initializes the ledger.
func ResetSimAccount(st *store.Store, traderID string, cfg Config) error {
	if st == nil || traderID == "" {
		return nil
	}
	if err := st.Prediction().DeleteSimData(traderID); err != nil {
		return err
	}
	_ = st.Prediction().DeleteDecisions(traderID)
	_ = st.Prediction().DeleteOrderRecords(traderID)
	_ = st.Prediction().DeleteAuditLogs(traderID)
	_ = st.Prediction().DeleteRiskState(traderID)
	initial := cfg.InitialBalanceUsd
	if initial <= 0 {
		initial = DefaultConfig().InitialBalanceUsd
	}
	if err := st.Prediction().SaveSimState(traderID, initial, initial, nil); err != nil {
		return err
	}
	ledger := NewLedger(initial)
	v := &Venue{ledger: ledger, store: st, traderID: traderID}
	return v.RecordEquitySnapshot(0)
}
