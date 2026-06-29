package trader

import (
	"errors"
	"testing"

	"nofx/store"
)

func TestSaveFailedCycleRecordPersistsError(t *testing.T) {
	st, err := store.New(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	const traderID = "pred-fail-1"
	if err := st.Prediction().CreateTrader(&store.PredictionTraderDB{
		ID:        traderID,
		UserID:    "u1",
		Name:      "fail-test",
		AIModelID: "model-1",
		Venue:     "polymarket",
	}); err != nil {
		t.Fatal(err)
	}

	pt := &PredictionTrader{
		traderDBID: traderID,
		st:         st,
		callCount:  3,
	}
	cycleErr := errors.New("AI parse failed")
	pt.saveFailedCycleRecord(nil, cycleErr)

	rows, err := st.Prediction().ListDecisions(traderID, 5)
	if err != nil {
		t.Fatal(err)
	}
	if len(rows) != 1 {
		t.Fatalf("expected 1 decision row, got %d", len(rows))
	}
	if rows[0].Success {
		t.Fatal("expected success=false")
	}
	if rows[0].ErrorMessage != cycleErr.Error() {
		t.Fatalf("error_message: %q", rows[0].ErrorMessage)
	}
	if rows[0].CycleNumber != 3 {
		t.Fatalf("cycle_number: got %d", rows[0].CycleNumber)
	}
}
