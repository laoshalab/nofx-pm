package store

import (
	"testing"
	"time"
)

func TestSimFillsAndSnapshots(t *testing.T) {
	st, err := New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	traderID := "sim-trader-1"

	if err := st.Prediction().SaveSimState(traderID, 10000, 10000, nil); err != nil {
		t.Fatal(err)
	}
	if err := st.Prediction().SaveSimFill(&PredictionSimFillDB{
		TraderID:   traderID,
		OrderID:    "sim-1",
		MarketSlug: "btc-up",
		Outcome:    "YES",
		Side:       "BUY",
		FillPrice:  0.5,
		FillShares: 10,
		FillUsd:    5,
		Timestamp:  time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.Prediction().SaveSimSnapshot(&PredictionSimSnapshotDB{
		TraderID:      traderID,
		CycleNumber:   1,
		CashUsdc:      9995,
		PositionValue: 5,
		TotalEquity:   10000,
		TotalPnl:      0,
		Timestamp:     time.Now().UTC().Add(-2 * time.Hour),
	}); err != nil {
		t.Fatal(err)
	}
	if err := st.Prediction().SaveSimSnapshot(&PredictionSimSnapshotDB{
		TraderID:      traderID,
		CycleNumber:   2,
		CashUsdc:      9990,
		PositionValue: 10,
		TotalEquity:   10000,
		TotalPnl:      0,
		Timestamp:     time.Now().UTC(),
	}); err != nil {
		t.Fatal(err)
	}

	fills, err := st.Prediction().ListSimFills(traderID, 10)
	if err != nil || len(fills) != 1 {
		t.Fatalf("fills: %v err=%v", fills, err)
	}
	snaps, err := st.Prediction().ListSimSnapshots(traderID, 1)
	if err != nil || len(snaps) != 1 {
		t.Fatalf("snapshots: %v err=%v", snaps, err)
	}
	if snaps[0].CycleNumber != 2 {
		t.Fatalf("expected latest snapshot cycle 2, got %d", snaps[0].CycleNumber)
	}
	snapsAll, err := st.Prediction().ListSimSnapshots(traderID, 10)
	if err != nil || len(snapsAll) != 2 {
		t.Fatalf("snapshots all: %v err=%v", snapsAll, err)
	}
	if snapsAll[0].CycleNumber != 1 || snapsAll[1].CycleNumber != 2 {
		t.Fatalf("expected chronological order [1,2], got [%d,%d]", snapsAll[0].CycleNumber, snapsAll[1].CycleNumber)
	}
	if err := st.Prediction().DeleteSimData(traderID); err != nil {
		t.Fatal(err)
	}
	_, _, hasAccount, _ := st.Prediction().LoadSimAccount(traderID)
	if hasAccount {
		t.Fatal("expected account deleted")
	}
}
