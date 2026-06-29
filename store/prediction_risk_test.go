package store

import "testing"

func TestRiskStatePersist(t *testing.T) {
	st, err := New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	const traderID = "risk-trader-1"
	day := "2026-06-28"

	if err := st.Prediction().SaveRiskState(traderID, day, 42.5); err != nil {
		t.Fatal(err)
	}
	vol, gotDay, err := st.Prediction().LoadRiskState(traderID)
	if err != nil {
		t.Fatal(err)
	}
	if gotDay != day || vol != 42.5 {
		t.Fatalf("got day=%s vol=%f", gotDay, vol)
	}

	if err := st.Prediction().SaveRiskState(traderID, day, 100); err != nil {
		t.Fatal(err)
	}
	vol, _, _ = st.Prediction().LoadRiskState(traderID)
	if vol != 100 {
		t.Fatalf("updated vol: %f", vol)
	}
}

func TestAuditLogAppend(t *testing.T) {
	st, err := New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	const traderID = "audit-trader-1"
	if err := st.Prediction().SaveAuditLog(&PredictionAuditLogDB{
		TraderID:  traderID,
		EventType: "filled",
		Action:    "buy_yes",
		AmountUsd: 10,
		Status:    "filled",
		Message:   "test fill",
	}); err != nil {
		t.Fatal(err)
	}
	rows, err := st.Prediction().ListAuditLogs(traderID, 10)
	if err != nil || len(rows) != 1 {
		t.Fatalf("audit rows: %v err=%v", rows, err)
	}
}
